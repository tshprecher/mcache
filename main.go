package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/store"
	"sync"
)

func main() {
	flag.Parse()
	glog.Infof("initializing storage engine...")
	server := &Server{11211, store.NewSimpleStorageEngine(), nil, sync.Mutex{}}
	err := server.Start()
	if err != nil {
		glog.Fatal(err)
	}
}
