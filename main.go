package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/store"
	"sync"
)

var (
	// flags
	port       = flag.Int("port", 11211, "server port")
	cap        = flag.Int("cap", 1024*1024*1024, "total capacity in bytes (including keys)")
	timeout    = flag.Int("timeout", 5, "maximum time in seconds an idle connection is open")
	maxValSize = flag.Int("max_val_size", 0, "max size of a value in bytes, <= 0 for no limit")
)

func main() {
	flag.Parse()
	glog.Infof("running server with port=%d cap=%d timeout=%d max_val_size", *port, *cap, *timeout, *maxValSize)
	glog.Infof("initializing storage engine...")
	server := &Server{
		port:       uint16(*port),
		se:         store.NewSimpleStorageEngine(store.NewLruEvictionPolicy(*cap)),
		lis:        nil,
		maxValSize: *maxValSize,
		timeout:    *timeout,
		mu:         sync.Mutex{}}
	err := server.Start()
	if err != nil {
		glog.Fatal(err)
	}
}
