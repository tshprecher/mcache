package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io"
	"strconv"
	"strings"
)

var (
	// protocol errors
	invalidStorageCommand = NewClientProtocolError("storage commands must take exactly 5 or 6 terms")
	invalidDeleteCommand  = NewClientProtocolError("delete must take exactly 2 or 3 terms")
	invalidKey            = NewClientProtocolError("malformed key")
	invalidFlags          = NewClientProtocolError("malformed flags")
	invalidExpTime        = NewClientProtocolError("malformed exptime")
	invalidBytes          = NewClientProtocolError("malformed bytes")
	invalidCasUniq        = NewClientProtocolError("malformed cas_unique")
	noReplyExpected       = NewClientProtocolError("expected 'noreply' as last term")
	commandLineTooLong    = NewClientProtocolError(fmt.Sprintf("command line exceeding %d bytes", MaxCommandLength))

	commandNotFound = NewStdProtocolError()

	// map the input string to command type
	cmdStringToType = map[string]int{
		"set":     SetCommand,
		"add":     AddCommand,
		"replace": ReplaceCommand,
		"append":  AppendCommand,
		"prepend": PrependCommand,
		"cas":     CasCommand,
		"get":     GetCommand,
		"gets":    GetsCommand,
		"delete":  DelCommand,
	}

	_ MessageBuffer = &textProtocolMessageBuffer{}
)

type MessageBuffer interface {
	Read() (*Command, error)
	Write(r Response) error
}

type textProtocolMessageBuffer struct {
	wireIn      io.Reader
	wireOut     io.Writer
	curCmd      Command
	cmdHeader   *bytes.Buffer
	cmdType     int
	cmdComplete bool
}

func NewTextProtocolMessageBuffer(wireIn io.Reader, wireOut io.Writer) *textProtocolMessageBuffer {
	return &textProtocolMessageBuffer{
		wireIn:  wireIn,
		wireOut: wireOut,
		curCmd: Command{
			storageCommand:   nil,
			retrievalCommand: nil,
		},
		cmdHeader:   &bytes.Buffer{},
		cmdType:     -1,
		cmdComplete: false,
	}
}

func (t *textProtocolMessageBuffer) Write(r Response) (err error) {
	// TODO: handle the case where writing does *not* complete all bytes on the first attempt
	// perhaps spin until an error is reached and there's some sort of timeout and the close the connection?
	bytes := r.Bytes()
	n, err := t.wireOut.Write(bytes)
	if err != nil {
		return
	} else if n < len(bytes) {
		err = errors.New("could not write complete message to wire")
	}
	return
}

func (t *textProtocolMessageBuffer) Read() (cmd *Command, err error) {
	// if the header is not read, try reading it first
	if t.cmdType == -1 {
		err = t.readHeader()
		if err != nil {
			return
		}
	}

	// continue reading the body of the request
	if t.cmdType != -1 {
		t.readBody()
	}

	if t.cmdComplete {
		glog.Infof("received command: '%v'", string(t.cmdHeader.Bytes()))
		cmd = new(Command)
		*cmd = t.curCmd
		t.curCmd.storageCommand = nil
		t.curCmd.retrievalCommand = nil
		t.curCmd.deleteCommand = nil
		t.cmdComplete = false
		t.cmdType = -1
		t.cmdHeader.Truncate(0)
	}
	return
}

func (t *textProtocolMessageBuffer) readHeader() error {
	b := [1]byte{}
	n, err := t.wireIn.Read(b[0:1])
	if n == 0 && err != nil {
		return err
	}
	for n > 0 {
		//glog.Infof("DEBUG A: read header byte %v\n", b[0])
		if t.cmdHeader.Len() > MaxCommandLength {
			return commandLineTooLong
		}
		t.cmdHeader.Write(b[0:1])
		bytes := t.cmdHeader.Bytes()
		if len(bytes) >= 2 && bytes[len(bytes)-1] == '\n' && bytes[len(bytes)-2] == '\r' {
			// reached the end of the cmd header so parse it.
			t.cmdHeader.Truncate(t.cmdHeader.Len() - 2)
			return t.parseHeader(t.cmdHeader.Bytes())
		}
		n, err = t.wireIn.Read(b[0:1])
	}
	return nil
}

func (t *textProtocolMessageBuffer) parseHeader(bytes []byte) (err error) {
	terms := strings.Split(strings.TrimSpace(string(bytes)), " ")
	if len(terms) == 0 {
		err = commandNotFound
		return
	}
	typ, ok := cmdStringToType[terms[0]]
	if !ok {
		err = commandNotFound
		return
	}
	if IsStorageCommand(typ) {
		err = t.unpackStorageCommand(typ, terms)
	} else if IsRetrievalCommand(typ) {
		err = t.unpackRetrievalCommand(typ, terms)
	} else if IsDeleteCommand(typ) {
		err = t.unpackDeleteCommand(typ, terms)
	}
	return
}

func (t *textProtocolMessageBuffer) validateKey(key string) error {
	if len(key) > MaxKeyLength || !keyRegex.MatchString(key) {
		return invalidKey
	}
	return nil
}

func (t *textProtocolMessageBuffer) unpackDeleteCommand(typ int, terms []string) error {
	if len(terms) < 2 || len(terms) > 3 {
		return invalidDeleteCommand
	}
	key := terms[1]
	err := t.validateKey(key)
	if err != nil {
		return err
	}
	noReply := false
	if len(terms) == 3 {
		if terms[2] == "noreply" {
			noReply = true
		} else {
			return noReplyExpected
		}
	}
	t.cmdType = typ
	t.curCmd.deleteCommand = &DeleteCommand{
		Key:     key,
		NoReply: noReply,
	}
	return nil
}

func (t *textProtocolMessageBuffer) unpackRetrievalCommand(typ int, terms []string) error {
	keys := terms[1:]
	for _, k := range keys {
		err := t.validateKey(k)
		if err != nil {
			return err
		}
	}
	t.cmdType = typ
	t.curCmd.retrievalCommand = &RetrievalCommand{
		Typ:  typ,
		keys: keys,
	}
	return nil
}

func (t *textProtocolMessageBuffer) unpackStorageCommand(typ int, terms []string) error {
	if len(terms) < 5 || len(terms) > 7 {
		return invalidStorageCommand
	}
	key := terms[1]
	err := t.validateKey(key)
	if err != nil {
		return err
	}
	flags, err := strconv.ParseUint(terms[2], 10, 16)
	if err != nil {
		return invalidFlags
	}
	expTime, err := strconv.ParseInt(terms[3], 10, 32)
	if err != nil {
		return invalidExpTime
	}
	numBytes, err := strconv.ParseUint(terms[4], 10, 32)
	if err != nil {
		return invalidBytes
	}
	var casUnique int64
	if typ == CasCommand {
		casUnique, err = strconv.ParseInt(terms[5], 10, 64)
		if err != nil {
			return invalidCasUniq
		}
	}
	noReply := false
	if typ == CasCommand && len(terms) == 7 || typ != CasCommand && len(terms) == 6 {
		if terms[len(terms)-1] == "noreply" {
			noReply = true
		} else {
			return noReplyExpected
		}
	}

	t.cmdType = typ
	t.curCmd.storageCommand = &StorageCommand{
		Typ:       typ,
		Key:       key,
		Flags:     uint16(flags),
		ExpTime:   int32(expTime),
		NumBytes:  uint32(numBytes),
		CasUnique: int64(casUnique),
		NoReply:   noReply,

		// filled in when reading the body
		DataBlock: nil,
	}
	return nil
}

func (t *textProtocolMessageBuffer) readDataBlock() error {
	b := [1]byte{}
	for len(t.curCmd.storageCommand.DataBlock) < int(t.curCmd.storageCommand.NumBytes)+2 {
		//glog.Infof("DEBUG B: read body byte %v\n", b[0])
		n, _ := t.wireIn.Read(b[0:1])
		if n > 0 {
			t.curCmd.storageCommand.DataBlock = append(t.curCmd.storageCommand.DataBlock, b[0:1]...)
		} else {
			break
		}
	}
	lenDataBlock := len(t.curCmd.storageCommand.DataBlock)
	if lenDataBlock == int(t.curCmd.storageCommand.NumBytes)+2 {
		t.curCmd.storageCommand.DataBlock = t.curCmd.storageCommand.DataBlock[0 : lenDataBlock-2]
		t.cmdComplete = true
	}
	return nil
}

func (t *textProtocolMessageBuffer) readBody() error {
	if IsStorageCommand(t.cmdType) {
		return t.readDataBlock()
	} else if IsRetrievalCommand(t.cmdType) {
		// no body, so just set completion
		t.cmdComplete = true
	} else if IsDeleteCommand(t.cmdType) {
		// no body, so just set completion
		t.cmdComplete = true
	}
	return nil
}
