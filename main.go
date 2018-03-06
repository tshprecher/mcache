package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/store"
	"sync"
)

var (
	// flags
	port = flag.Int("port", 11211, "server port")
	cap  = flag.Int("cap", 1024*1024*1024, "total capacity in bytes")
)

func main() {
	flag.Parse()
	glog.Infof("initializing storage engine...")
	server := &Server{uint16(*port), store.NewSimpleStorageEngine(store.NewLruEvictionPolicy(*cap)), nil, sync.Mutex{}}
	err := server.Start()
	if err != nil {
		glog.Fatal(err)
	}
}
