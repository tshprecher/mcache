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

////

type Response interface {
	Bytes() []byte
}

////

type TextStoredResponse struct{}

func (_ TextStoredResponse) Bytes() []byte {
	return []byte("STORED\r\n")
}

type TextNotStoredResponse struct{}

func (_ TextNotStoredResponse) Bytes() []byte {
	return []byte("NOT_STORED\r\n")
}

type TextExistsResponse struct{}

func (_ TextExistsResponse) Bytes() []byte {
	return []byte("EXISTS\r\n")
}

type TextDeletedResponse struct{}

func (_ TextDeletedResponse) Bytes() []byte {
	return []byte("DELETED\r\n")
}

type TextNotFoundResponse struct{}

func (_ TextNotFoundResponse) Bytes() []byte {
	return []byte("NOT_FOUND\r\n")
}

type TextGetOrGetsResponse struct {
	values      map[string]store.Value
	withCasUniq bool
}

func (t TextGetOrGetsResponse) Bytes() []byte {
	buf := &bytes.Buffer{}
	for k, v := range t.values {
		if t.withCasUniq {
			buf.WriteString(fmt.Sprintf("VALUE %s %d %d %d\r\n", k, v.Flags, len(v.Bytes), v.CasUnique))
		} else {
			buf.WriteString(fmt.Sprintf("VALUE %s %d %d\r\n", k, v.Flags, len(v.Bytes)))
		}
		buf.Write(v.Bytes)
		buf.WriteString("\r\n")
	}
	buf.WriteString("END\r\n")
	return buf.Bytes()
}
