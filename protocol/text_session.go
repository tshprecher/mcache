package protocol

import (
	"errors"
	"github.com/tshprecher/mcache/store"
	"net"
)

// TODO: honor noreply
type TextSession struct {
	conn          net.Conn
	messageBuffer MessageBuffer
	engine        store.StorageEngine
	alive         bool
}

func NewTextProtocolSession(conn net.Conn, engine store.StorageEngine) *TextSession {
	return &TextSession{
		conn:          conn,
		messageBuffer: NewTextProtocolMessageBuffer(conn, conn),
		engine:        engine,
		alive:         true,
	}
}

func (t *TextSession) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

func (t *TextSession) Alive() bool {
	return t.alive
}

func (t *TextSession) Close() error {
	err := t.conn.Close()
	t.alive = false
	return err
}

func (t *TextSession) Serve() error {
	if !t.alive {
		return errors.New("cannot serve a dead session")
	}
	cmd, err := t.messageBuffer.Read()
	if err != nil {
		// write an error and close the connection
		// TODO: distinguish between recoverable and non recoverable errors?
		// TODO: add logging where appropriate
		return err
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
	return err
}

func (t *TextSession) serveSet(cmd *StorageCommand) error {
	ok := t.engine.Set(cmd.Key, store.Value{cmd.Flags, 0, cmd.DataBlock})
	if ok {
		return t.messageBuffer.Write(TextStoredResponse{})
	} else {
		// TODO: this should never fail. if it does write a server error
		return t.messageBuffer.Write(TextNotStoredResponse{})
	}
}

func (t *TextSession) serveCas(cmd *StorageCommand) error {
	exists, notFound := t.engine.Cas(cmd.Key, store.Value{cmd.Flags, cmd.CasUnique, cmd.DataBlock})
	if exists {
		return t.messageBuffer.Write(TextExistsResponse{})
	} else if notFound {
		return t.messageBuffer.Write(TextNotFoundResponse{})
	} else {
		return t.messageBuffer.Write(TextStoredResponse{})
	}
}

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

func (t *TextSession) serveDelete(cmd *DeleteCommand) error {
	ok := t.engine.Delete(cmd.Key)
	if ok {
		return t.messageBuffer.Write(TextDeletedResponse{})
	} else {
		return t.messageBuffer.Write(TextNotFoundResponse{})
	}
}
