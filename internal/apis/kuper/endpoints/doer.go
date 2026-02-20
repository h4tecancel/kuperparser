package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	Doer         Doer
	BaseURL      string
	ApplyHeaders func(*http.Request)
}

func New(doer Doer, baseURL string, applyHeaders func(*http.Request)) *Client {
	return &Client{
		Doer:         doer,
		BaseURL:      strings.TrimRight(baseURL, "/"),
		ApplyHeaders: applyHeaders,
	}
}

func (c *Client) newReq(ctx context.Context, method, path string) (*http.Request, error) {
	if c.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL is empty")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.ApplyHeaders != nil {
		c.ApplyHeaders(req)
	}
	return req, nil
}

func readLimited(resp *http.Response, limit int64) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

func decodeJSON[T any](b []byte, out *T) error {
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.UseNumber()
	return dec.Decode(out)
}
