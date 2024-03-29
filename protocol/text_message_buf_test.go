package protocol

import (
	"bytes"
	"reflect"
	"testing"
)

type readResult struct {
	cmd *Command
	err error
}

func expectEquals(t *testing.T, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected value %v, received %v", expected, actual)
	}
}

func expectOneSubcommand(t *testing.T, received *Command) {
	if received == nil {
		return
	}
	count := 0
	if received.storageCommand != nil {
		count++
	}
	if received.retrievalCommand != nil {
		count++
	}
	if received.deleteCommand != nil {
		count++
	}
	if count != 1 {
		t.Errorf("expected one non-nil subcommand")
	}
}

func expectCommandEquals(t *testing.T, expected, actual *Command) {
	if expected == nil && actual == nil {
		return
	}
	if expected == nil {
		t.Errorf("expected command nil, received %v", *actual)
		return
	}
	if actual == nil {
		t.Errorf("expected command %v, received nil", *expected)
		return
	}

	if expected.storageCommand != nil {
		expectEquals(t, *expected.storageCommand, *actual.storageCommand)
	} else if expected.retrievalCommand != nil {
		expectEquals(t, *expected.retrievalCommand, *actual.retrievalCommand)
	} else if expected.deleteCommand != nil {
		expectEquals(t, *expected.deleteCommand, *actual.deleteCommand)
	}
}

func expectReadResultEquals(t *testing.T, expResult readResult, recResult readResult) {
	if (expResult.err != nil && recResult.err == nil) || (expResult.err == nil && recResult.err != nil) ||
		(expResult.err != nil && recResult.err != nil && expResult.err.Error() != recResult.err.Error()) {
		t.Errorf("expected err %v, received %v\n", expResult.err, recResult.err)
	}
	expectCommandEquals(t, expResult.cmd, recResult.cmd)
}

func testTextRead(t *testing.T, buf MessageBuffer, wireIn *bytes.Buffer, packets [][]byte, expectedResults []readResult) {
	res := readResult{}
	for p := range packets {
		wireIn.Write(packets[p])
		res = readResult{}
		res.cmd, res.err = buf.Read()
		expectOneSubcommand(t, res.cmd)
		expectReadResultEquals(t, expectedResults[p], res)
	}
}

func TestTextReadSplitPackets(t *testing.T) {
	packets := [][]byte{
		[]byte("set my_key"),
		[]byte(" 3 2 1"),
		[]byte("\r"),
		[]byte("\n"),
		[]byte("1\r\n"),
	}
	expResults := []readResult{
		readResult{},
		readResult{},
		readResult{},
		readResult{},
		readResult{
			cmd: &Command{
				storageCommand: &StorageCommand{
					Typ:       SetCommand,
					Key:       "my_key",
					Flags:     3,
					ExpTime:   2,
					NumBytes:  1,
					NoReply:   false,
					DataBlock: []byte("1"),
				},
			},
			err: nil,
		},
	}
	wireIn, wireOut := &bytes.Buffer{}, &bytes.Buffer{}
	buf := NewTextProtocolMessageBuffer(wireIn, wireOut, 1024)
	testTextRead(t, buf, wireIn, packets, expResults)
}

func TestTextReadRetrievalCommand(t *testing.T) {
	packets := [][]byte{
		[]byte("get key key2\r\n"),
	}
	expResults := []readResult{
		readResult{
			cmd: &Command{
				retrievalCommand: &RetrievalCommand{
					Typ:  GetCommand,
					keys: []string{"key", "key2"},
				},
			},
			err: nil,
		},
	}
	wireIn, wireOut := &bytes.Buffer{}, &bytes.Buffer{}
	buf := NewTextProtocolMessageBuffer(wireIn, wireOut, 1024)
	testTextRead(t, buf, wireIn, packets, expResults)
}

func TestTextReadStorageCommand(t *testing.T) {
	packets := [][]byte{
		[]byte("set my_key 3 2 1\r\n1\r\n"),
	}
	expResults := []readResult{
		readResult{
			cmd: &Command{
				storageCommand: &StorageCommand{
					Typ:       SetCommand,
					Key:       "my_key",
					Flags:     3,
					ExpTime:   2,
					NumBytes:  1,
					NoReply:   false,
					DataBlock: []byte("1"),
				},
			},
			err: nil,
		},
	}
	wireIn, wireOut := &bytes.Buffer{}, &bytes.Buffer{}
	buf := NewTextProtocolMessageBuffer(wireIn, wireOut, 1024)
	testTextRead(t, buf, wireIn, packets, expResults)
}

func TestTextReadMultiple(t *testing.T) {
	packets := [][]byte{
		[]byte("set my_key 3 2 1\r\n1\r\n"),
		[]byte("set my_key2 3 2 1\r"),
		[]byte("\n2\r\n"),
	}
	expResults := []readResult{
		readResult{
			cmd: &Command{
				storageCommand: &StorageCommand{
					Typ:       SetCommand,
					Key:       "my_key",
					Flags:     3,
					ExpTime:   2,
					NumBytes:  1,
					NoReply:   false,
					DataBlock: []byte("1"),
				},
			},
			err: nil,
		},
		readResult{},
		readResult{
			cmd: &Command{
				storageCommand: &StorageCommand{
					Typ:       SetCommand,
					Key:       "my_key2",
					Flags:     3,
					ExpTime:   2,
					NumBytes:  1,
					NoReply:   false,
					DataBlock: []byte("2"),
				},
			},
			err: nil,
		},
	}
	wireIn, wireOut := &bytes.Buffer{}, &bytes.Buffer{}
	buf := NewTextProtocolMessageBuffer(wireIn, wireOut, 1024)
	testTextRead(t, buf, wireIn, packets, expResults)
}

func TestTextReadDeleteCommand(t *testing.T) {
	packets := [][]byte{
		[]byte("delete my_key\r\n"),
	}
	expResults := []readResult{
		readResult{
			cmd: &Command{
				deleteCommand: &DeleteCommand{
					Key:     "my_key",
					NoReply: false,
				},
			},
			err: nil,
		},
	}
	wireIn, wireOut := &bytes.Buffer{}, &bytes.Buffer{}
	buf := NewTextProtocolMessageBuffer(wireIn, wireOut, 1024)
	testTextRead(t, buf, wireIn, packets, expResults)
}

func TestTextWrite(t *testing.T) {
	wireIn, wireOut := &bytes.Buffer{}, &bytes.Buffer{}
	buf := NewTextProtocolMessageBuffer(wireIn, wireOut, 1024)

	resp := TextStoredResponse{}
	buf.Write(resp)
	if wireOut.Len() != len(resp.Bytes()) {
		t.Errorf("expected %v bytes written, received %v", len(resp.Bytes()), wireOut.Len())
	}

	resp2 := TextExistsResponse{}
	buf.Write(resp2)
	if wireOut.Len() != len(resp.Bytes())+len(resp2.Bytes()) {
		t.Errorf("expected %v bytes written, received %v", len(resp.Bytes())+len(resp2.Bytes()), wireOut.Len())
	}

	bytes := make([]byte, wireOut.Len())
	wireOut.Read(bytes)

	if string(bytes) != "STORED\r\nEXISTS\r\n" {
		t.Errorf("expected bytes written value %v, received %v", []byte("STORED\r\nEXISTS\r\n"), bytes)
	}
}
