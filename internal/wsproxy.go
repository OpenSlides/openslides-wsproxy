package wsproxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WSProxy holds the state of the proxy.
type WSProxy struct {
	geturler GetURLer
}

// New returns an initialized WSProxy.
func New(geturler GetURLer) *WSProxy {
	return &WSProxy{
		geturler: geturler,
	}
}

func (ws *WSProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// TODO
		log.Println(err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	out := make(chan []byte, 1)
	defer close(out)

	done := make(chan struct{})
	// Read on connection
	go func() {
		defer close(done)

		if err := ws.receive(ctx, conn, out); err != nil {
			// TODO: Only send some errors to the client or "internal error"
			event := map[string]interface{}{
				"reason": err.Error(),
			}
			log.Println(err)
			if err := sendEvent(out, "error", 0, event); err != nil {
				log.Printf("Error: send event: %v", err)
			}
		}
	}()

	if err := sendLoop(conn, out); err != nil {
		log.Printf("Error writing to client: %v", err)
	}
	<-done
}

func sendLoop(conn *websocket.Conn, out <-chan []byte) error {
	for p := range out {
		if err := conn.WriteMessage(websocket.TextMessage, p); err != nil {
			return fmt.Errorf("sending message: %w", err)
		}
	}
	return nil
}

// receive reads from r until EOF is reached.
func (ws *WSProxy) receive(ctx context.Context, conn *websocket.Conn, out chan<- []byte) error {
	// TODO: handle ctx closed.

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if messageType == websocket.BinaryMessage {
			return fmt.Errorf("binary messages not supported")
		}

		var v struct {
			Cmd string `json:"cmd"`
		}
		if err := json.Unmarshal(p, &v); err != nil {
			return fmt.Errorf("decoding command name: %w", err)
		}

		var cmd command
		switch v.Cmd {
		case "connect":
			cmd = &cmdConnect{getURLer: ws.geturler}
		case "close":
			// TODO
			//cmd = new(cmdClose)
		case "":
			return fmt.Errorf("given object needs attribute `cmd`")
		default:
			return fmt.Errorf("unknown command `%s`", v.Cmd)
		}

		if err := json.Unmarshal(p, &cmd); err != nil {
			return fmt.Errorf("decoding cmd: %w", err)
		}

		if err := cmd.Call(ctx, out); err != nil {
			return fmt.Errorf("calling command: %w", err)
		}
	}
}

func readerToChan(r io.Reader) <-chan []byte {
	c := make(chan []byte, 1)
	go func() {
		defer close(c)
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			c <- scanner.Bytes()
		}
		if err := scanner.Err(); err != nil {
			// TODO handle error
			return
		}
	}()
	return c
}

func sendEvent(out chan<- []byte, name string, id int, event map[string]interface{}) error {
	if event == nil {
		event = make(map[string]interface{}, 2)
	}
	event["event"] = name
	if id != 0 {
		event["id"] = id
	}
	b, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encoding event: %w", err)
	}
	out <- b
	return nil
}
