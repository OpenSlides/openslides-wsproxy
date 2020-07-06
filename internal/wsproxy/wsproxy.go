package wsproxy

import (
	"context"
	"fmt"
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

	wsConn := newWSConnection(ctx, ws.geturler)

	readDone := make(chan struct{})
	// Read on connection
	go func() {
		defer close(readDone)

		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				wsConn.eventError(err)
				return
			}
			if messageType == websocket.BinaryMessage {
				wsConn.eventError(fmt.Errorf("binary messages not supported"))
			}
			if err := wsConn.fromClient(p); err != nil {
				wsConn.eventError(err)
			}
		}
	}()

	if err := sendLoop(conn, wsConn.toClient()); err != nil {
		log.Printf("Error writing to client: %v", err)
	}
	<-readDone
}

func sendLoop(conn *websocket.Conn, out <-chan []byte) error {
	for p := range out {
		if err := conn.WriteMessage(websocket.TextMessage, p); err != nil {
			return fmt.Errorf("sending message: %w", err)
		}
	}
	return nil
}
