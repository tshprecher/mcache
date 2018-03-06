// Package protocol implements the memcache protocol. It currently
// just supports the text protocol.
// see: https://github.com/memcached/memcached/blob/master/doc/protocol.txt
package protocol

import (
	"bytes"
	"fmt"
	"github.com/tshprecher/mcache/store"
	"regexp"
)

const (
	MaxKeyLength = 250

	// the six storage commands
	SetCommand = iota
	AddCommand
	ReplaceCommand
	AppendCommand
	PrependCommand
	CasCommand

	// the two retrieval commands
	GetCommand
	GetsCommand

	// delete command
	DelCommand
)

var (
	keyRegex = regexp.MustCompile(`^[0-9a-zA-Z_]+$`)
)

func IsStorageCommand(typ int) bool {
	return typ == SetCommand || typ == AddCommand || typ == ReplaceCommand ||
		typ == AppendCommand || typ == PrependCommand || typ == CasCommand
}

func IsRetrievalCommand(typ int) bool {
	return typ == GetCommand || typ == GetsCommand
}

func IsDeleteCommand(typ int) bool {
	return typ == DelCommand
}

/////

type StorageCommand struct {
	// header fields
	Typ       int
	Key       string
	Flags     uint16
	ExpTime   int32
	NumBytes  uint32
	CasUnique int64
	NoReply   bool

	// data block
	DataBlock []byte
}

type RetrievalCommand struct {
	Typ  int
	keys []string
}

type DeleteCommand struct {
	Key     string
	NoReply bool
}

type Command struct {
	storageCommand   *StorageCommand
	retrievalCommand *RetrievalCommand
	deleteCommand    *DeleteCommand
}

// Response represents a complete memcache protocol message
// in response to a client command
type Response interface {
	// Bytes returns all the bytes for the intended message
	// to be written to the client
	Bytes() []byte
}

// A TextStoredResponse builds the "STORED" response
type TextStoredResponse struct{}

func (_ TextStoredResponse) Bytes() []byte { return []byte("STORED\r\n") }

// A TextNotStoredResponse builds the "NOT_STORED" response
type TextNotStoredResponse struct{}

func (_ TextNotStoredResponse) Bytes() []byte { return []byte("NOT_STORED\r\n") }

// A TextExistsResponse builds the "EXISTS" response
type TextExistsResponse struct{}

func (_ TextExistsResponse) Bytes() []byte { return []byte("EXISTS\r\n") }

// A TextDeletedResponse builds the "DELETED" response
type TextDeletedResponse struct{}

func (_ TextDeletedResponse) Bytes() []byte { return []byte("DELETED\r\n") }

// A TextNotFoundResponse builds the "NOT_FOUND" response
type TextNotFoundResponse struct{}

func (_ TextNotFoundResponse) Bytes() []byte { return []byte("NOT_FOUND\r\n") }

type TextErrorResponse struct{}

func (_ TextErrorResponse) Bytes() []byte { return []byte("ERROR\r\n") }

type TextClientErrorResponse struct{ msg string }

func (t TextClientErrorResponse) Bytes() []byte {
	return []byte(fmt.Sprintf("CLIENT_ERROR %s\r\n", t.msg))
}

type TextServerErrorResponse struct{ msg string }

func (t TextServerErrorResponse) Bytes() []byte {
	return []byte(fmt.Sprintf("SERVER_ERROR %s\r\n", t.msg))
}

// A TextGetOrGetsResponse builds responses for get and gets commands
// given a slice of Values to return to the client. The withCasUniq flag
// should be true if and only if the intended response is for a gets command.
type TextGetOrGetsResponse struct {
	pairs []struct {
		k string
		v store.Value
	}
	withCasUniq bool
}

func (t TextGetOrGetsResponse) Bytes() []byte {
	buf := &bytes.Buffer{}
	for _, p := range t.pairs {
		if t.withCasUniq {
			buf.WriteString(fmt.Sprintf("VALUE %s %d %d %d\r\n", p.k, p.v.Flags, len(p.v.Bytes), p.v.CasUnique))
		} else {
			buf.WriteString(fmt.Sprintf("VALUE %s %d %d\r\n", p.k, p.v.Flags, len(p.v.Bytes)))
		}
		buf.Write(p.v.Bytes)
		buf.WriteString("\r\n")
	}
	buf.WriteString("END\r\n")
	return buf.Bytes()
}
