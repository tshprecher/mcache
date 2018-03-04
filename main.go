package main

import (
	"fmt"
	"github.com/tshprecher/mcache/protocol"
	"github.com/tshprecher/mcache/store"
	"log"
	"net"
)

var storageEngine store.StorageEngine

func listen() {
	ln, err := net.Listen("tcp", ":11211")
	if err != nil {
		log.Fatalf("error when starting server: %v\n", err.Error())
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
	log.Print("session created")
	for session.Alive() {
		session.Serve()
	}
	log.Print("session completed")
}

func main() {
	fmt.Println("starting storage engine...")
	storageEngine = store.NewSimpleStorageEngine()
	fmt.Println("starting server...")
	listen()
}
