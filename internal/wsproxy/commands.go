package wsproxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type cmdConnect struct {
	getURLer GetURLer
	url      string
	id       int
	body     json.RawMessage
}

func (c *cmdConnect) UnmarshalJSON(data []byte) error {
	var v struct {
		URL  string          `json:"url"`
		ID   int             `json:"id"`
		Body json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if v.ID == 0 || v.URL == "" {
		return fmt.Errorf("connect command requires the parameters url and id")
	}

	url := c.getURLer.GetURL(v.URL)
	if url == "" {
		return fmt.Errorf("unknown url `%s`", v.URL)
	}

	c.url = url
	c.id = v.ID
	c.body = v.Body
	return nil
}

func (c *cmdConnect) Call(conn *wsConnection) error {
	var body io.Reader
	if c.body != nil {
		body = bytes.NewReader(c.body)
	}

	ctx, cancel := context.WithCancel(conn.ctx)
	conn.registerConn(c.id, cancel)

	req, err := http.NewRequestWithContext(ctx, "GET", c.url, body)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request to `%s`: %w", c.url, err)
	}

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}

		conn.eventColse(c.id, resp.StatusCode, body)
		return nil
	}

	conn.eventConnected(c.id)

	go func() {
		// Read resp.Body until it closes or the context is done.
		defer resp.Body.Close()
		defer func() {
			conn.eventColse(c.id, resp.StatusCode, []byte("null"))
		}()

		msgChan := readerToChan(resp.Body)
		for {
			select {
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				conn.eventData(c.id, msg)

			case <-conn.ctx.Done():
				return
			}
		}
	}()
	return nil
}

type cmdClose struct {
	id int
}

func (c *cmdClose) UnmarshalJSON(data []byte) error {
	var v struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if v.ID == 0 {
		return fmt.Errorf("close command requires the parameters id")
	}

	c.id = v.ID
	return nil
}

func (c *cmdClose) Call(conn *wsConnection) error {
	conn.CloseConn(c.id)
	return nil
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
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("scanner: %v", err)
			return
		}
	}()
	return c
}
