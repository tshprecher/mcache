package store

import (
	"github.com/golang/glog"
	"sync"
)

type Value struct {
	Flags     uint16
	CasUnique int64
	Bytes     []byte
}

type StorageEngine interface {
	Set(key string, value Value) (ok bool)
	Get(key string) (value Value, found bool)
	Cas(key string, value Value) (exists, notFound bool)
	Delete(key string) bool
}

func NewSimpleStorageEngine(ep EvictionPolicy) *SimpleStorageEngine {
	return &SimpleStorageEngine{map[string]Value{}, ep, 0, sync.Mutex{}}
}

// TODO: add memory limits
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
