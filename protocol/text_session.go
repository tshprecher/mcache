package protocol

import (
	"errors"
	"github.com/golang/glog"
	"github.com/tshprecher/mcache/store"
	"net"
)

type TextProtocolSession struct {
	conn          net.Conn
	messageBuffer MessageBuffer
	engine        store.StorageEngine
	alive         bool
}

func NewTextProtocolSession(conn net.Conn, engine store.StorageEngine) *TextProtocolSession {
	return &TextProtocolSession{
		conn:          conn,
		messageBuffer: NewTextProtocolMessageBuffer(conn, conn),
		engine:        engine,
		alive:         true,
	}
}

func (t *TextProtocolSession) Alive() bool {
	return t.alive
}

func (t *TextProtocolSession) closeOnError(err error) {
	// TODO: import glog and log this error=
	t.conn.Close()
	t.alive = false
}

func (t *TextProtocolSession) Serve() {
	if !t.alive {
		return
	}
	cmd, err := t.messageBuffer.Read()
	if err != nil {
		// write an error and close the connection
		// TODO: distinguish between recoverable and non recoverable errors?
		// TODO: add logging where appropriate
		glog.Errorf("command error: %s", err.Error())
		t.closeOnError(err)
		return
	}

	if cmd != nil {
		if cmd.storageCommand != nil {
			switch cmd.storageCommand.Typ {
			case SetCommand:
				err = t.serveSet(cmd.storageCommand)
			case AddCommand:
				err = errors.New("add not yet implemented")
			case ReplaceCommand:
				err = errors.New("replace not yet implemented")
			case AppendCommand:
				err = errors.New("append not yet implemented")
			case PrependCommand:
				err = errors.New("prepend not yet implemented")
			case CasCommand:
				err = errors.New("cas not yet implemented")
			}
		} else {
			err = errors.New("retrieval commands not yet implemented")
		}
	}
	if err != nil {
		t.closeOnError(err)
	}
	return
}

func (t *TextProtocolSession) serveSet(cmd *StorageCommand) error {
	ok := t.engine.Set(cmd.Key, store.Value{cmd.Flags, cmd.DataBlock})
	if ok {
		return t.messageBuffer.Write(TextStoredResponse{})
	} else {
		return t.messageBuffer.Write(TextNotStoredResponse{})
	}
}
