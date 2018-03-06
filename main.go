package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/store"
	"sync"
)

var (
	// flags
	port    = flag.Int("port", 11211, "server port")
	cap     = flag.Int("cap", 1024*1024*1024, "total capacity in bytes")
	timeout = flag.Int("timeout", 5, "maximum time in seconds an idle connection is open")
)

func main() {
	flag.Parse()
	glog.Infof("running server with port=%d cap=%d timeout=%d", *port, *cap, *timeout)
	glog.Infof("initializing storage engine...")
	server := &Server{
		port:    uint16(*port),
		se:      store.NewSimpleStorageEngine(store.NewLruEvictionPolicy(*cap)),
		lis:     nil,
		timeout: *timeout,
		mu:      sync.Mutex{}}
	err := server.Start()
	if err != nil {
		glog.Fatal(err)
	}
}
