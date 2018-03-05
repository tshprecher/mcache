package main

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/protocol"
	"net"
	"sync"
)

func handleConn(conn net.Conn) {
	session := protocol.NewTextProtocolSession(conn, storageEngine)
	glog.Infof("session started: addr=%v", conn.RemoteAddr())
	for session.Alive() {
		err := session.Serve()
		if nerr, ok := err.(net.Error); ok && !nerr.Temporary() { // != nil /* && err != io.EOF*/ {
			glog.Errorf("error serving: %v", nerr)
			session.Close()
		}
	}
	glog.Infof("session ended: addr=%v", conn.RemoteAddr())
}

type Server struct {
	port     uint16
	listener net.Listener
	mu       sync.Mutex
}

func (s *Server) setListener(l net.Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listener = l
}

func (s *Server) Start() error {
	glog.Info("starting server...")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	s.setListener(listener)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			glog.Warningf(err.Error())
			s.listener = nil
			break
		}
		go handleConn(conn)
	}
	return nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return
	}
	glog.Info("stopping server...")
	s.listener.Close()
}
