package main

import (
	"bufio"
	"github.com/tshprecher/mcache/store"
	"net"
	"sync"
	"testing"
	"time"
)

func TestIntegrationProtocol(t *testing.T) {
	se := store.NewSimpleStorageEngine(store.NewLruEvictionPolicy(1024))
	server := &Server{11209, se, nil, 2, sync.Mutex{}}
	go server.Start()
	time.Sleep(500 * time.Millisecond)

	testProtoGetAndSet(t)

	*se = *store.NewSimpleStorageEngine(store.NewLruEvictionPolicy(1024))
	testProtoDelete(t)

	*se = *store.NewSimpleStorageEngine(store.NewLruEvictionPolicy(1024))
	testProtoCas(t)
}

func expectResponse(t *testing.T, exp string, rec string) {
	if string(exp) != rec {
		t.Errorf("expected response %#v, received %#v", exp, rec)
	}
}

func testMessages(t *testing.T, conn net.Conn, messageLines, responseLines []string) {
	buf := bufio.NewReader(conn)
	for _, m := range messageLines {
		conn.Write([]byte(m))
	}
	// TODO: depending on time can be flaky
	time.Sleep(500 * time.Millisecond)
	rep, err := buf.ReadString('\n')
	r := 0
	for err == nil {
		//t.Logf("rep -> %#v", string(rep))
		if r < len(responseLines) {
			expectResponse(t, responseLines[r], rep)
		} else {
			t.Errorf("unexpected response line %#v", rep)
			return
		}
		r++
		if buf.Buffered() == 0 {
			if r < len(responseLines) {
				t.Errorf("expected response line %#v", responseLines[r])
			}
			break
		}
		rep, err = buf.ReadString('\n')
	}
}

func testProtoGetAndSet(t *testing.T) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "localhost:11209")
	if err != nil {
		t.Error(err)
	}
	conn, _ := net.DialTCP("tcp", nil, tcpAddr)
	defer conn.Close()

	testMessages(t, conn,
		[]string{
			"get key\r\n",
			"set key 3 0 0\r\n\r\n",
			"set key2 3 0 1\r\n2\r\n",
			"set key3 3 0 1 noreply\r\n3\r\n",
			"get key\r\n",
			"gets key2\r\n",
			"get key key3\r\n",
			"gets key key3\r\n",
		},
		[]string{
			"END\r\n",
			"STORED\r\n",
			"STORED\r\n",

			"VALUE key 3 0\r\n",
			"\r\n",
			"END\r\n",

			"VALUE key2 3 1 2\r\n",
			"2\r\n",
			"END\r\n",

			"VALUE key 3 0\r\n",
			"\r\n",
			"VALUE key3 3 1\r\n",
			"3\r\n",
			"END\r\n",

			"VALUE key 3 0 1\r\n",
			"\r\n",
			"VALUE key3 3 1 3\r\n",
			"3\r\n",
			"END\r\n",
		},
	)
}

func testProtoDelete(t *testing.T) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "localhost:11209")
	if err != nil {
		t.Error(err)
	}
	conn, _ := net.DialTCP("tcp", nil, tcpAddr)
	defer conn.Close()

	testMessages(t, conn,
		[]string{
			"delete key\r\n",
			"set key 3 0 1\r\n1\r\n",
			"delete key\r\n",
			"set key 3 0 1\r\n1\r\n",
			"delete key noreply\r\n",
			"delete key\r\n",
		},
		[]string{
			"NOT_FOUND\r\n",
			"STORED\r\n",
			"DELETED\r\n",
			"STORED\r\n",
			"NOT_FOUND\r\n",
		},
	)
}

func testProtoCas(t *testing.T) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "localhost:11209")
	if err != nil {
		t.Error(err)
	}
	conn, _ := net.DialTCP("tcp", nil, tcpAddr)
	defer conn.Close()

	testMessages(t, conn,
		[]string{
			"set key 3 0 1\r\n1\r\n",
			"cas key 3 0 1 0\r\n2\r\n",

			"cas key 3 0 1 1\r\n2\r\n",
			"gets key\r\n",

			"cas key 3 0 1 2 noreply\r\n3\r\n",
			"gets key\r\n",
		},
		[]string{
			"STORED\r\n",
			"EXISTS\r\n",
			"STORED\r\n",

			"VALUE key 3 1 2\r\n",
			"2\r\n",
			"END\r\n",

			"VALUE key 3 1 3\r\n",
			"3\r\n",
			"END\r\n",
		},
	)
}
