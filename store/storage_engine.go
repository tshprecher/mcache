package store

import (
	"sync"
)

type Value struct {
	Flags     uint16
	CasUnique int64
	Bytes     []byte
}

type StorageEngine interface {
	// TODO: consider working with immutable []byte so callers
	// cannot alter data
	Set(key string, value Value) (ok bool)
	Get(key string) (value Value, found bool)
	Cas(key string, value Value) (exists, notFound bool)
	Delete(key string) bool
}

func NewSimpleStorageEngine() *SimpleStorageEngine {
	return &SimpleStorageEngine{map[string]Value{}, 0, sync.Mutex{}}
}

// TODO: add memory limits
type SimpleStorageEngine struct {
	values       map[string]Value
	curCasUnique int64
	mu           sync.Mutex
}

func (s *SimpleStorageEngine) Set(key string, value Value) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.curCasUnique++
	value.CasUnique = s.curCasUnique
	s.values[key] = value
	return true
}

func (s *SimpleStorageEngine) Get(key string) (value Value, found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, found = s.values[key]
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
	s.curCasUnique++
	value.CasUnique = s.curCasUnique
	s.values[key] = value
	return
}

func (s *SimpleStorageEngine) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.values[key]
	delete(s.values, key)
	return ok
}
