package main

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/protocol"
	"github.com/tshprecher/mcache/store"
	"net"
	"sync"
)

func handleSession(session *protocol.TextSession) {
	glog.Infof("session started: addr=%v", session.RemoteAddr())
	for session.Alive() {
		err := session.Serve()
		if nerr, ok := err.(net.Error); ok && !nerr.Temporary() {
			glog.Errorf("error serving: %v", nerr)
			session.Close()
		}
	}
	glog.Infof("session ended: addr=%v", session.RemoteAddr())
}

type Server struct {
	port uint16
	se   store.StorageEngine
	lis  net.Listener
	mu   sync.Mutex
}

func (s *Server) setListener(l net.Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lis = l
}

func (s *Server) Start() error {
	glog.Info("starting server...")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	s.setListener(listener)
	for {
		conn, err := s.lis.Accept()
		if err != nil {
			glog.Warningf(err.Error())
			s.lis = nil
			break
		}
		go handleSession(protocol.NewTextProtocolSession(conn, s.se))
	}
	return nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lis == nil {
		return
	}
	glog.Info("stopping server...")
	s.lis.Close()
}
