package store

import (
	"github.com/golang/glog"
	"sync"
)

// A Value represents a stored value, including the raw bytes,
// flags, and cas_unique.
type Value struct {
	Flags     uint16
	CasUnique int64
	Bytes     []byte
}

// StorageEngine defines the operations of a generic in-memory
// key/value storage logic.
type StorageEngine interface {
	// Set writes or overwrites the Value in the store, returning
	// true if and only if the value is successfully written.
	Set(key string, value Value) (ok bool)

	// Get returns the Value requested by the key along with
	// a boolean indicating if the Value is found in the store.
	Get(key string) (value Value, found bool)

	// Cas overwrites a Value in the store if and only if the
	// Value has not been written to since the client's last
	// Get(). It does this by checking the provided Value's CasUnique
	// against the one currently in the store.
	//
	// If the Value does not exist in the store, nothing is written
	// and notFound=true. If the Value exists in the store but has
	// been written after this client's last call to Get(), nothing
	// is written and exists=true.
	Cas(key string, value Value) (exists, notFound bool)

	// Delete deletes the Value mapped to by the key, returning true
	// if and only if the key exists and the item is properly deleted.
	Delete(key string) bool
}

// NewSimpleStorageEngine takes an EvictionPolicy and returns a
// SimpleStorageEngine configured with that eviction policy.
func NewSimpleStorageEngine(ep EvictionPolicy) *SimpleStorageEngine {
	return &SimpleStorageEngine{map[string]Value{}, ep, 0, sync.Mutex{}}
}

// A SimpleStorageEngine is coarsely locked StorageEngine. To
// facilitate multiple clients concurrently reading and writing,
// it locks *all* operations. While this is not even close to the
// most efficient implementation of a StorageEngine, it is maybe
// the simplest.
//
// Note: because an EvictionPolicy is not specified to be
// threadsafe and may be written on all reads to the StorageEngine,
// simply using RWLock here without a proper write lock on the
// EvictionPolicy may improve performance, but it could cause contention
type SimpleStorageEngine struct {
	values       map[string]Value
	ep           EvictionPolicy
	curCasUnique int64
	mu           sync.Mutex
}

func (s *SimpleStorageEngine) insertWithEvictions(key string, value Value) bool {
	evict, ok := s.ep.Add(key, value)
	if !ok {
		glog.Warning("value exceeds total cache capacity")
		return false
	}
	for _, e := range evict {
		v := s.values[e]
		glog.Infof("evicting key '%s' (%d bytes)", e, kvSize(e, v))
		delete(s.values, e)
	}

	s.curCasUnique++
	value.CasUnique = s.curCasUnique
	s.values[key] = value
	return true
}

func (s *SimpleStorageEngine) Set(key string, value Value) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.insertWithEvictions(key, value)
}

func (s *SimpleStorageEngine) Get(key string) (value Value, found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, found = s.values[key]
	s.ep.Touch(key)
	return
}

func (s *SimpleStorageEngine) Cas(key string, value Value) (exists, notFound bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.values[key]
	if !ok {
		notFound = true
		return
	}
	if val.CasUnique != value.CasUnique {
		exists = true
		return
	}

	// TODO: handle error on out of space?
	s.insertWithEvictions(key, value)
	return
}

func (s *SimpleStorageEngine) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.values[key]
	s.ep.Remove(key)
	delete(s.values, key)
	return ok
}
