package main

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/protocol"
	"github.com/tshprecher/mcache/store"
	"io"
	"net"
	"sync"
)

// handleSession wraps a TextSession and polls TextSession.Serve().
// If an error occurs, the session is promptly closed.
func handleSession(session *protocol.TextSession) {
	glog.Infof("session started: addr=%v", session.RemoteAddr())
	for session.Alive() {
		err := session.Serve()
		if nerr, ok := err.(net.Error); ok {
			if !nerr.Temporary() {
				glog.Errorf("error serving: %v", nerr)
				session.Close()
			}
		} else if err != nil && err != io.EOF {
			glog.Errorf("error serving: %v", err)
			session.Close()
		}
	}
	glog.Infof("session ended: addr=%v", session.RemoteAddr())
}

type Server struct {
	port    uint16
	se      store.StorageEngine
	lis     net.Listener
	timeout int
	mu      sync.Mutex
}

func (s *Server) setListener(l net.Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lis = l
}

// Start opens up a port to listen, wraps each client connection
// inside a TextSession, and spins up a goroutine to serve that
// session.
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
		go handleSession(protocol.NewTextSession(conn, s.se, s.timeout))
	}
	return nil
}

// Stop stops the server from listening for new connections, but goroutines
// serving existing connections will continue to run until those sessions
// have ended.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lis == nil {
		return
	}
	glog.Info("stopping server...")
	s.lis.Close()
}
