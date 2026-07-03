package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// httpClient is a small HTTP helper with a browser-like default User-Agent,
// which most live platforms require before they will answer their APIs.
type httpClient struct {
	userAgent string
	client    *http.Client
}

func newClient() *httpClient {
	return &httpClient{
		userAgent: defaultUA,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *httpClient) do(req *http.Request, headers map[string]string) ([]byte, error) {
	req.Header.Set("User-Agent", c.userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// Get performs a GET request and returns the response body.
func (c *httpClient) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request failed: %w", err)
	}
	return c.do(req, headers)
}

// Post performs a POST request and returns the response body.
func (c *httpClient) Post(ctx context.Context, url string, headers map[string]string, data []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request failed: %w", err)
	}
	return c.do(req, headers)
}
