package main

import (
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/tshprecher/mcache/store"
	"sync"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	// spin up a server first
	storageEngine = store.NewSimpleStorageEngine()

	server := &Server{11210, nil, sync.Mutex{}}
	go server.Start()
	time.Sleep(200 * time.Millisecond)

	mc1 := memcache.New("localhost:11210")
	err := mc1.Set(&memcache.Item{Key: "foo", Value: []byte("my value")})
	if err != nil {
		t.Error(err)
		return
	}
}
