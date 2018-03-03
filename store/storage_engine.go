package store

import (
	"sync"
)

type StorageEngine interface {
	// TODO: consider working with immutable []byte so callers
	// cannot alter data
	Set(key string, value []byte) bool
	Get(key string) (value []byte, found bool)
	Delete(key string) bool
}

func NewSimpleStorageEngine() StorageEngine {
	return &SimpleStorageEngine{map[string][]byte{}, sync.Mutex{}}
}

// TODO: add memory limits
type SimpleStorageEngine struct {
	values map[string][]byte
	mu     sync.Mutex
}

func (s *SimpleStorageEngine) Set(key string, value []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return true
}

func (s *SimpleStorageEngine) Get(key string) (value []byte, found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, found = s.values[key]
	return
}

func (s *SimpleStorageEngine) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.values[key]
	delete(s.values, key)
	return ok
}
