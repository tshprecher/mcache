package protocol

import (
	"errors"
	"github.com/tshprecher/mcache/store"
	"net"
	"time"
)

// A TextSession manages the connection and protocol logic by
// reading commands from a MessageBuffer, handling the business logic,
// and writing responses back to the client.
type TextSession struct {
	conn          net.Conn
	messageBuffer MessageBuffer
	engine        store.StorageEngine
	alive         bool
	maxValSize    int
	timeout       int
	lastActive    time.Time
}

// NewTextSession returns a new TextSession given the established
// connection, an existing StorageEngine, and a timeout in seconds. If no
// command is read within the given timeout period, the connection and session
// are closed.
func NewTextSession(conn net.Conn, engine store.StorageEngine, maxValSize, timeout int) *TextSession {
	return &TextSession{
		conn:          conn,
		messageBuffer: NewTextProtocolMessageBuffer(conn, conn, maxValSize),
		engine:        engine,
		alive:         true,
		timeout:       timeout,
		lastActive:    time.Now(),
	}
}

// RemoteAddr returns conn.RemoteAddr() of the underlying connection.
func (t *TextSession) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

// Alive returns true if and only if the session is still servicing requests.
func (t *TextSession) Alive() bool {
	return t.alive
}

// Close closes the underlying connection and designates this session as dead.
func (t *TextSession) Close() error {
	err := t.conn.Close()
	t.alive = false
	return err
}

// Serve attempts to read a command, handle it, and write the response
// or error back to the client. It returns nil if and only if no command can
// be processed or a command has successfully been processed. It is intended
// to be polled.
func (t *TextSession) Serve() error {
	if !t.alive {
		return errors.New("cannot serve a dead session")
	}
	cmd, err := t.messageBuffer.Read()
	if cmd == nil && time.Since(t.lastActive) >= time.Duration(t.timeout*1e9) {
		return errors.New("session timed out")
	}
	if err != nil {
		if perr, ok := err.(*ErrorResponse); ok {
			t.messageBuffer.Write(perr)
		}
		return err
	}

	if cmd != nil {
		t.lastActive = time.Now()
		if cmd.storageCommand != nil {
			switch cmd.storageCommand.Typ {
			case SetCommand:
				err = t.serveSet(cmd.storageCommand)
			case AddCommand:
				err = NewServerErrorResponse("add not yet implemented")
			case ReplaceCommand:
				err = NewServerErrorResponse("replace not yet implemented")
			case AppendCommand:
				err = NewServerErrorResponse("append not yet implemented")
			case PrependCommand:
				err = NewServerErrorResponse("prepend not yet implemented")
			case CasCommand:
				err = t.serveCas(cmd.storageCommand)
			}
		} else if cmd.retrievalCommand != nil {
			switch cmd.retrievalCommand.Typ {
			case GetCommand, GetsCommand:
				err = t.serveGetAndGets(cmd.retrievalCommand)
			}
		} else if cmd.deleteCommand != nil {
			err = t.serveDelete(cmd.deleteCommand)
		} else {
			panic("no command set")
		}
	}
	if perr, ok := err.(*ErrorResponse); ok {
		t.messageBuffer.Write(perr)
	}

	return err
}

// serveSet handles the protocol logic for the 'set' command
func (t *TextSession) serveSet(cmd *StorageCommand) error {
	ok := t.engine.Set(cmd.Key, store.Value{cmd.Flags, 0, cmd.DataBlock})
	if ok && !cmd.NoReply {
		return t.messageBuffer.Write(TextStoredResponse{})
	} else if !ok && !cmd.NoReply {
		return t.messageBuffer.Write(TextNotStoredResponse{})
	}
	return nil
}

// serveCas handles the protocol logic for the 'cas' command
func (t *TextSession) serveCas(cmd *StorageCommand) error {
	exists, notFound := t.engine.Cas(cmd.Key, store.Value{cmd.Flags, cmd.CasUnique, cmd.DataBlock})
	if exists && !cmd.NoReply {
		return t.messageBuffer.Write(TextExistsResponse{})
	} else if notFound && !cmd.NoReply {
		return t.messageBuffer.Write(TextNotFoundResponse{})
	} else if !exists && !notFound && !cmd.NoReply {
		return t.messageBuffer.Write(TextStoredResponse{})
	}
	return nil
}

// serveGetAndGets handles the protocol logic for the 'get' and 'gets' commands.
func (t *TextSession) serveGetAndGets(cmd *RetrievalCommand) error {
	results := []struct {
		k string
		v store.Value
	}{}
	for _, k := range cmd.keys {
		v, ok := t.engine.Get(k)
		if ok {
			results = append(results, struct {
				k string
				v store.Value
			}{k, v})
		}
	}
	return t.messageBuffer.Write(TextGetOrGetsResponse{pairs: results, withCasUniq: cmd.Typ == GetsCommand})
}

// serveDelete handles the protocol logic for the 'delete' command
func (t *TextSession) serveDelete(cmd *DeleteCommand) error {
	ok := t.engine.Delete(cmd.Key)
	if ok && !cmd.NoReply {
		return t.messageBuffer.Write(TextDeletedResponse{})
	} else if !ok && !cmd.NoReply {
		return t.messageBuffer.Write(TextNotFoundResponse{})
	}
	return nil
}
