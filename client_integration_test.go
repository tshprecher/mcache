package main

import (
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/tshprecher/mcache/store"
	"sync"
	"testing"
	"time"
)

func itemValuesEqual(item1, item2 *memcache.Item) bool {
	if item1 == item2 {
		return true
	}
	if item1 == nil || item2 == nil {
		return false
	}
	return item1.Flags == item2.Flags && item1.Key == item2.Key &&
		string(item1.Value) == string(item2.Value)

}

// TODO: test case where value is empty (no bytes)
func TestIntegration(t *testing.T) {
	// spin up a server
	se := store.NewSimpleStorageEngine()
	server := &Server{11210, se, nil, sync.Mutex{}}
	go server.Start()
	time.Sleep(500 * time.Millisecond)

	testSetAndGet(t)

	*se = *store.NewSimpleStorageEngine()
	testGets(t)

	*se = *store.NewSimpleStorageEngine()
	testMultipleSessions(t)

	*se = *store.NewSimpleStorageEngine()
	testDelete(t)

	*se = *store.NewSimpleStorageEngine()
	testDelete(t)

	*se = *store.NewSimpleStorageEngine()
	testCas(t)
}

func testSetAndGet(t *testing.T) {
	mc := memcache.New("localhost:11210")

	item1 := &memcache.Item{Key: "foo", Flags: 3, Value: []byte("my value")}
	err := mc.Set(item1)
	if err != nil {
		t.Error(err)
	}

	item, err := mc.Get("bar")
	if err != memcache.ErrCacheMiss {
		t.Errorf("expected cache miss, received %v", err)
	}
	if item != nil {
		t.Errorf("expected nil item, received %v", item)
	}

	item, err = mc.Get("foo")
	if err != nil {
		t.Errorf("expected nil error")
	}
	if !itemValuesEqual(item, item1) {
		t.Errorf("expected %v to equal $v", *item, *item1)
	}
}

func testGets(t *testing.T) {
	mc := memcache.New("localhost:11210")
	fooItem := &memcache.Item{Key: "foo", Flags: 3, Value: []byte("my value")}
	barItem := &memcache.Item{Key: "bar", Flags: 2, Value: []byte("my value 2")}

	mc.Set(fooItem)
	mc.Set(barItem)

	res, err := mc.GetMulti([]string{"bar", "foo"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 2 {
		t.Errorf("expected two results, received %d", len(res))
	}
	if !itemValuesEqual(res["foo"], fooItem) {
		t.Errorf("expected %v to equal $v", res["foo"], *fooItem)
	}
	if !itemValuesEqual(res["bar"], barItem) {
		t.Errorf("expected %v to equal $v", res["bar"], *barItem)
	}
}

func testMultipleSessions(t *testing.T) {
	mc1 := memcache.New("localhost:11210")
	mc2 := memcache.New("localhost:11210")

	fooItem := &memcache.Item{Key: "foo", Flags: 3, Value: []byte("my value")}
	barItem := &memcache.Item{Key: "bar", Flags: 3, Value: []byte("my value 2")}
	mc1.Set(fooItem)
	mc2.Set(barItem)

	item1, _ := mc1.Get("foo")
	item2, _ := mc2.Get("bar")

	if !itemValuesEqual(item1, fooItem) {
		t.Errorf("expected %v to equal $v", *item1, *fooItem)
	}
	if !itemValuesEqual(item2, barItem) {
		t.Errorf("expected %v to equal $v", *item2, *barItem)
	}
}

func testDelete(t *testing.T) {
	mc := memcache.New("localhost:11210")
	fooItem, _ := mc.Get("foo")
	if fooItem != nil {
		t.Errorf("expected nil item, received %v", *fooItem)
	}

	mc.Set(&memcache.Item{Key: "foo", Flags: 3, Value: []byte("my value")})

	fooItem, _ = mc.Get("foo")
	if fooItem == nil {
		t.Errorf("expected non-nil item")
	}

	mc.Delete("foo")
	fooItem, _ = mc.Get("foo")
	if fooItem != nil {
		t.Errorf("expected nil item, received %v", *fooItem)
	}
}

func testCas(t *testing.T) {
	mc := memcache.New("localhost:11210")

	err := mc.CompareAndSwap(&memcache.Item{Key: "foo", Flags: 0, Value: nil})
	if err != memcache.ErrCacheMiss {
		t.Errorf("expected cache miss, received %v", err)
	}

	mc.Set(&memcache.Item{Key: "foo", Flags: 3, Value: []byte("my value")})
	fooItem, _ := mc.Get("foo")

	mc.Set(&memcache.Item{Key: "foo", Flags: 3, Value: []byte("my value overwritten")})
	fooItem2, _ := mc.Get("foo")

	err = mc.CompareAndSwap(fooItem)
	if err != memcache.ErrCASConflict {
		t.Errorf("expected cas conflict, received %v", err)
	}

	err = mc.CompareAndSwap(fooItem2)
	if err != nil {
		t.Errorf("expected nil error, received %v", err)
	}
}

