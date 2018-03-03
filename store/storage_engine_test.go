package store

import (
	"testing"
)

func expectBoolEquals(t *testing.T, exp, rec bool) {
	if exp != rec {
		t.Errorf("expected bool %v, received %v", exp, rec)
	}
}

func expectValueEquals(t *testing.T, exp, rec []byte) {
	if exp == nil && rec == nil {
		return
	}
	if exp == nil || rec == nil || len(exp) != len(rec) {
		t.Errorf("expected value %v, received %v", exp, rec)
	}
	for b := range exp {
		if exp[b] != rec[b] {
			t.Errorf("expected value %v, received %v", exp, rec)
		}
	}
}

func testCommon(t *testing.T, s StorageEngine) {
	// no key should exist
	value, found := s.Get("key1")
	expectBoolEquals(t, false, found)
	expectValueEquals(t, nil, value)

	// set key1, then read it
	set := s.Set("key1", []byte("value1"))
	expectBoolEquals(t, true, set)
	value, found = s.Get("key1")
	expectBoolEquals(t, true, found)
	expectValueEquals(t, []byte("value1"), value)

	// overwrite key1, then read it
	set = s.Set("key1", []byte("value2"))
	expectBoolEquals(t, true, set)
	value, found = s.Get("key1")
	expectBoolEquals(t, true, found)
	expectValueEquals(t, []byte("value2"), value)

	// deleting a key that does not exist returns false
	deleted := s.Delete("key_not_existing")
	expectBoolEquals(t, false, deleted)

	// set second key, then read it
	set = s.Set("key2", []byte("value3"))
	expectBoolEquals(t, true, set)
	value, found = s.Get("key2")
	expectBoolEquals(t, true, found)
	expectValueEquals(t, []byte("value3"), value)

	// read first key to make sure it's not modified
	value, found = s.Get("key1")
	expectBoolEquals(t, true, found)
	expectBoolEquals(t, true, set)
	expectValueEquals(t, []byte("value2"), value)

	// successfully delete both keys
	deleted = s.Delete("key1")
	expectBoolEquals(t, true, deleted)
	deleted = s.Delete("key2")
	expectBoolEquals(t, true, deleted)

	// read key1 and key2 to verify no data is stored
	value, found = s.Get("key1")
	expectBoolEquals(t, false, found)
	expectValueEquals(t, nil, value)
	value, found = s.Get("key2")
	expectBoolEquals(t, false, found)
	expectValueEquals(t, nil, value)
}

func TestSimpleStorageEngineCommon(t *testing.T) {
	testCommon(t, NewSimpleStorageEngine())
}
