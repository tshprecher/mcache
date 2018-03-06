package store

import (
	"testing"
)

func TestLruTouchNonExisting(t *testing.T) {
	p := NewLruEvictionPolicy(16)
	ok := p.Touch("non_existing")
	if ok {
		t.Errorf("expected unsuccessful touch")
	}
}

func TestLruTouchExisting(t *testing.T) {
	p := NewLruEvictionPolicy(16)

	node1 := &kvListNode{"key1", Value{0, 0, []byte{0}}, nil, nil}
	node2 := &kvListNode{"key2", Value{0, 0, []byte{0}}, nil, nil}
	node3 := &kvListNode{"key3", Value{0, 0, []byte{0}}, nil, nil}
	p.kvMap["key1"] = node1
	p.kvMap["key2"] = node2
	p.kvMap["key3"] = node3

	// wire up the linked list next pointers
	p.sentinel.next = node1
	node1.next = node2
	node2.next = node3
	node3.next = p.sentinel

	// wire up the linked list prev pointers
	node3.prev = node2
	node2.prev = node1
	node1.prev = p.sentinel
	p.sentinel.prev = node3

	// touch key2
	ok := p.Touch("key2")
	if !ok {
		t.Errorf("expected successful touch")
	}
	if !(p.sentinel.next == node2 && node2.next == node1 && node1.next == node3 && node3.next == p.sentinel) ||
		!(node3.prev == node1 && node1.prev == node2 && node2.prev == p.sentinel && p.sentinel.prev == node3) {
		t.Errorf("expected order key2, key1, key3 but received %v, %v, %v",
			p.sentinel.next.key, p.sentinel.next.next.key, p.sentinel.next.next.next.key)
	}

	// touch key3
	ok = p.Touch("key3")
	if !ok {
		t.Errorf("expected successful touch")
	}
	if !(p.sentinel.next == node3 && node3.next == node2 && node2.next == node1 && node1.next == p.sentinel) ||
		!(node1.prev == node2 && node2.prev == node3 && node3.prev == p.sentinel && p.sentinel.prev == node1) {
		t.Errorf("expected order key3, key2, key1 but received %v, %v, %v",
			p.sentinel.next.key, p.sentinel.next.next.key, p.sentinel.next.next.next.key)
	}
}

func TestLruDelete(t *testing.T) {
	p := NewLruEvictionPolicy(32)
	p.Add("key1", Value{0, 0, []byte{0}})
	p.Add("key2", Value{0, 0, []byte{0}})

	if len(p.kvMap) != 2 {
		t.Errorf("expected 2 elements in kvMap, received %d", len(p.kvMap))
	}

	p.Remove("unknown")
	if len(p.kvMap) != 2 {
		t.Errorf("expected 2 elements in kvMap, received %d", len(p.kvMap))
	}

	p.Remove("key1")
	if len(p.kvMap) != 1 {
		t.Errorf("expected 1 element in kvMap, received %d", len(p.kvMap))
	}

	p.Remove("key2")
	if len(p.kvMap) != 0 {
		t.Errorf("expected 0 elements in kvMap, received %d", len(p.kvMap))
	}

	if p.sentinel.prev != p.sentinel {
		t.Error("expected 0 elements in list")
	}
}

func TestLruEviction(t *testing.T) {
	p := NewLruEvictionPolicy(32)

	ev, sp := p.Add("key1", Value{0, 0, []byte{0}})
	if ev != nil || sp == false {
		t.Errorf("expected no evictions and can add")
	}
	if p.Used() != 15 {
		t.Errorf("expected 15 used bytes, received %d", p.Used())
	}

	ev, sp = p.Add("key2", Value{0, 0, []byte{0}})
	if ev != nil || sp == false {
		t.Errorf("expected no evictions and can add")
	}
	if p.Used() != 30 {
		t.Errorf("expected 30 used bytes, received %d", p.Used())
	}

	ev, sp = p.Add("key3", Value{0, 0, []byte{1, 2, 3}})
	if len(ev) != 1 {
		t.Errorf("expected 1 eviction, received %d", len(ev))
	}
	if ev[0] != "key1" {
		t.Errorf("expected eviction of key, received eviction of %s", ev[0])
	}
	if sp == false {
		t.Errorf("expected can add")
	}
	if p.Used() != 32 {
		t.Errorf("expected 32 used bytes, received %d", p.Used())
	}
}
