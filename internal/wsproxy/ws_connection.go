package wsproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type wsConnection struct {
	ctx      context.Context
	getURLer GetURLer
	out      chan []byte

	connsMu sync.Mutex
	conns   map[int]func()
}

func newWSConnection(ctx context.Context, getURLer GetURLer) *wsConnection {
	c := &wsConnection{
		ctx:      ctx,
		getURLer: getURLer,
		out:      make(chan []byte, 1),
		conns:    make(map[int]func()),
	}
	return c
}

// fromClient processes a message from the client to the proxy.
func (c *wsConnection) fromClient(msg []byte) error {
	var v struct {
		Cmd string `json:"cmd"`
	}
	if err := json.Unmarshal(msg, &v); err != nil {
		return fmt.Errorf("decoding command name: %w", err)
	}

	var cmd command
	switch v.Cmd {
	case "connect":
		cmd = &cmdConnect{getURLer: c.getURLer}
	case "close":
		cmd = new(cmdClose)
	case "":
		return fmt.Errorf("given object needs attribute `cmd`")
	default:
		return fmt.Errorf("unknown command `%s`", v.Cmd)
	}

	if err := json.Unmarshal(msg, &cmd); err != nil {
		return fmt.Errorf("decoding cmd: %w", err)
	}

	if err := cmd.Call(c); err != nil {
		return fmt.Errorf("calling command: %w", err)
	}
	return nil
}

func (c *wsConnection) registerConn(id int, cancel func()) {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()
	c.conns[id] = cancel
}

func (c *wsConnection) CloseConn(id int) {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()
	c.conns[id]()
}

func (c *wsConnection) Close() {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	for _, close := range c.conns {
		close()
	}
	close(c.out)
}

// toClient returns a channel where messages to the client can be received from.
func (c *wsConnection) toClient() <-chan []byte {
	return c.out
}

func (c *wsConnection) eventConnected(id int) {
	c.event(`{"event":"connected","id":%d}`, id)
}

func (c *wsConnection) eventColse(id int, code int, reason json.RawMessage) {
	c.event(`{"event":"close","id":%d,"code":%d,"reason":%s}`, id, code, reason)
}

func (c *wsConnection) eventData(id int, data json.RawMessage) {
	c.event(`{"event":"data","id":%d,"data":%s}`, id, data)
}

func (c *wsConnection) eventError(err error) {
	c.event(`{"event":"error","reason":"%s"}`, err.Error())
}

func (c *wsConnection) event(format string, a ...interface{}) {
	defer func() {
		// This happens when c.out was closed. This is not clean. Is there a
		// better way to do this?
		recover()
	}()
	c.out <- []byte(fmt.Sprintf(format, a...))
}
