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

type Decision struct {
    Allowed     bool
    Status      int
    Bypass      bool
    Unavailable bool
}

func New(baseURL, service string) *Client {
    return &Client{
        url:     strings.TrimRight(baseURL, "/") + "/check",
        service: service,
        client:  &http.Client{Timeout: 50 * time.Millisecond},
    }
}

func (c *Client) Allow(ctx context.Context, endpoint string) Decision {
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, http.NoBody)
    if err != nil {
        return unavailableDecision()
    }

    req.Header.Set("X-RateLimit-Service", c.service)
    req.Header.Set("X-RateLimit-Endpoint", endpoint)

    resp, err := c.client.Do(req)
    if err != nil {
        return unavailableDecision()
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        return Decision{Allowed: true, Status: 200}
    }
    if resp.StatusCode == 429 {
        return Decision{Allowed: false, Status: 429}
    }
    if resp.StatusCode == 404 {
        return Decision{Allowed: false, Status: 500}
    }

    return unavailableDecision()
}

func unavailableDecision() Decision {
    return Decision{
        Allowed:     true,
        Status:      200,
        Bypass:      true,
        Unavailable: true,
    }
}
