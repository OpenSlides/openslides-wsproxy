package wsproxy

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestConnectionFromClientToClient(t *testing.T) {
	b := newBackendMock()
	defer b.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := newWSConnection(ctx, b, *http.DefaultClient)

	t.Run("Open connection", func(t *testing.T) {
		data := []byte(`{"cmd":"connect","id":2,"url":"/some/path"}`)
		expected := []byte(`{"event":"connected","id":2}`)

		if err := c.fromClient(data); err != nil {
			t.Fatalf("fromClient returned err: %v", err)
		}

		timer := time.NewTimer(time.Millisecond)
		defer timer.Stop()
		var received []byte
		var ok bool
		select {
		case received, ok = <-c.toClient():
			if !ok {
				t.Errorf("toClient channel was unexpected closed")
			}
		case <-timer.C:
			t.Errorf("toClient did not return for one millisecond")
		}

		if b.ConnectionCount() != 1 {
			t.Errorf("got %d connections to backend, expected 1", b.ConnectionCount())
		}

		if !bytes.Equal(received, expected) {
			t.Errorf("toClient returned `%s`, expected `%s`", received, expected)
		}
	})

	t.Run("Receive data", func(t *testing.T) {
		expected := []byte(`{"event":"data","id":2,"data":"some data"}`)
		b.Send([]byte(`"some data"`))

		timer := time.NewTimer(time.Millisecond)
		defer timer.Stop()
		var received []byte
		var ok bool
		select {
		case received, ok = <-c.toClient():
			if !ok {
				t.Errorf("toClient channel was unexpected closed")
			}
		case <-timer.C:
			t.Errorf("toClient did not return for one millisecond")
		}

		if !bytes.Equal(received, expected) {
			t.Errorf("toClient returned `%s`, expected `%s`", received, expected)
		}
	})

	t.Run("Disconnect", func(t *testing.T) {
		data := []byte(`{"cmd":"close","id":2}`)
		expected := []byte(`{"event":"close","id":2,"code":200,"reason":null}`)

		if err := c.fromClient(data); err != nil {
			t.Fatalf("fromClient returned err: %v", err)
		}

		timer := time.NewTimer(time.Millisecond)
		defer timer.Stop()
		var received []byte
		var ok bool
		select {
		case received, ok = <-c.toClient():
			if !ok {
				t.Errorf("toClient channel was unexpected closed")
			}
		case <-timer.C:
			t.Errorf("toClient did not return for one millisecond")
		}

		time.Sleep(time.Millisecond)
		if b.ConnectionCount() != 0 {
			t.Errorf("got %d connections to backend, expected 0", b.ConnectionCount())
		}

		if !bytes.Equal(received, expected) {
			t.Errorf("toClient returned `%s`, expected `%s`", received, expected)
		}
	})

}

type backendMock struct {
	mu              sync.Mutex
	connectionCount int

	srv *httptest.Server
	c   chan []byte
}

func newBackendMock() *backendMock {
	b := new(backendMock)
	b.c = make(chan []byte, 1)
	b.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.(http.Flusher).Flush()

		b.mu.Lock()
		b.connectionCount++
		b.mu.Unlock()

		defer func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			b.connectionCount--
		}()

		for {
			select {
			case <-r.Context().Done():
				log.Println("exit server")
				return
			case msg := <-b.c:
				log.Printf("sending %s", msg)
				msg = append(msg, '\n')
				w.Write(msg)
				w.(http.Flusher).Flush()
			}
		}
	}))
	return b
}

func (b *backendMock) Close() {
	b.srv.Close()
}

func (b *backendMock) GetURL(_ string) string {
	return b.srv.URL
}

func (b *backendMock) Send(msg []byte) {
	b.c <- msg
}

func (b *backendMock) ConnectionCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.connectionCount
}
