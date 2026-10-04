package ratelimiter

import (
    "context"
    "net/http"
    "strings"
    "time"
)

type Client struct {
    url     string
    service string
    client  *http.Client
}

func New(baseURL, service string) *Client {
    return &Client{
        url: strings.TrimRight(baseURL, "/") + "/check",
        service: service,
        client: &http.Client{Timeout: 50 * time.Millisecond},
    }
}

func (c *Client) Allow(ctx context.Context, endpoint string) (bool, int) {
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, http.NoBody)
    if err != nil { return true, 200 }
    req.Header.Set("X-RateLimit-Service", c.service)
    req.Header.Set("X-RateLimit-Endpoint", endpoint)
    resp, err := c.client.Do(req)
    if err != nil { return true, 200 }
    defer resp.Body.Close()
    if resp.StatusCode == 429 { return false, 429 }
    if resp.StatusCode == 404 { return false, 500 }
    if resp.StatusCode >= 200 && resp.StatusCode < 300 { return true, 200 }
    return true, 200
}
