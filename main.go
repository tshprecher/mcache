package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/store"
	"sync"
)

var storageEngine store.StorageEngine

func main() {
	flag.Parse()
	glog.Infof("initializing storage engine...")
	storageEngine = store.NewSimpleStorageEngine()

	server := &Server{11211, nil, sync.Mutex{}}
	err := server.Start()
	if err != nil {
		glog.Fatal(err)
	}
}
