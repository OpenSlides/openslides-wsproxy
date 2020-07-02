package wsproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type command interface {
	Call(ctx context.Context, out chan<- []byte) error
}

// GetURLer returns a full url for a url path.
type GetURLer interface {
	GetURL(url string) string
}

type cmdConnect struct {
	getURLer GetURLer
	url      string
	id       int
	body     string
}

func (c *cmdConnect) UnmarshalJSON(data []byte) error {
	var v struct {
		URL  string `json:"url"`
		ID   int    `json:"id"`
		Body string `json:"body"`
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

func (c *cmdConnect) Call(ctx context.Context, out chan<- []byte) error {
	var body io.Reader
	if c.body != "" {
		body = strings.NewReader(c.body)
	}

	log.Println(c.url)

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

		event := map[string]interface{}{
			"reason": string(body),
			"code":   resp.StatusCode,
		}
		if err := sendEvent(out, "close", c.id, event); err != nil {
			return fmt.Errorf("send error event: %w", err)
		}
		return nil
	}

	if err := sendEvent(out, "connected", c.id, nil); err != nil {
		return fmt.Errorf("sending connected event: %w", err)
	}

	go func() {
		// Read resp.Body until it closes or the context is done.
		defer resp.Body.Close()
		defer func() {
			event := map[string]interface{}{
				"reason": nil,
				"code":   resp.StatusCode,
			}
			if err := sendEvent(out, "close", c.id, event); err != nil {
				log.Printf("sending event to client: %v", err)
				return
			}
		}()

		msgChan := readerToChan(resp.Body)
		for {
			select {
			case msg, ok := <-msgChan:
				if !ok {
					return
				}

				event := map[string]interface{}{
					"data": json.RawMessage(msg),
				}
				if err := sendEvent(out, "data", c.id, event); err != nil {
					log.Printf("sending event to client: %v", err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

type cmdClose struct{}
