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

func expectMaxOneSubcommand(t *testing.T, received *Command) {
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
	if count != 1 {
		t.Errorf("expected at most one non-nil subcommand")
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
		panic("TODO: implement retrieval equals")
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
		expectMaxOneSubcommand(t, res.cmd)
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
		readResult{nil, nil},
		readResult{nil, nil},
		readResult{nil, nil},
		readResult{nil, nil},
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
	buf := NewTextProtocolMessageBuffer(wireIn, wireOut)
	testTextRead(t, buf, wireIn, packets, expResults)
}

func TestTextReadSetCommand(t *testing.T) {
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
	buf := NewTextProtocolMessageBuffer(wireIn, wireOut)
	testTextRead(t, buf, wireIn, packets, expResults)
}

func TestTextReadMultiple(t *testing.T) {
	t.Errorf("implement me")
}

func TestTextWrite(t *testing.T) {
	wireIn, wireOut := &bytes.Buffer{}, &bytes.Buffer{}
	buf := NewTextProtocolMessageBuffer(wireIn, wireOut)

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
