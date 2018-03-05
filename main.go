package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/protocol"
	"github.com/tshprecher/mcache/store"
	"net"
)

var storageEngine store.StorageEngine

func listen() {
	ln, err := net.Listen("tcp", ":11211")
	if err != nil {
		glog.Fatalf("error when starting server: %v\n", err.Error())
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			// TODO: log error with glog
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	session := protocol.NewTextProtocolSession(conn, storageEngine)
	glog.Infof("session started: addr=%v", conn.RemoteAddr())
	for session.Alive() {
		err := session.Serve()
		if err != nil {
			glog.Errorf("error serving: %v", err)
		}
		session.Close()
	}
	glog.Infof("session ended: addr=%v", conn.RemoteAddr())

}

func main() {
	flag.Parse()
	fmt.Println("starting storage engine...")
	storageEngine = store.NewSimpleStorageEngine()
	fmt.Println("starting server...")
	listen()
}
