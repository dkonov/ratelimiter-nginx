package ratelimiter

import (
    "context"
    "errors"
    "net"
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
    Allowed           bool
    Status            int
    Bypass            bool
    Unavailable       bool
    UnavailableReason string
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
        return unavailableDecision("other")
    }

    req.Header.Set("X-RateLimit-Service", c.service)
    req.Header.Set("X-RateLimit-Endpoint", endpoint)

    resp, err := c.client.Do(req)
    if err != nil {
        return unavailableDecision(classifyError(err))
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
    if resp.StatusCode >= 500 {
        return unavailableDecision("5xx")
    }

    return unavailableDecision("other")
}

func classifyError(err error) string {
    if errors.Is(err, context.DeadlineExceeded) {
        return "timeout"
    }

    var netErr net.Error
    if errors.As(err, &netErr) && netErr.Timeout() {
        return "timeout"
    }

    var opErr *net.OpError
    if errors.As(err, &opErr) {
        if opErr.Op == "dial" {
            return "connect_error"
        }
        return "io_error"
    }

    return "io_error"
}

func unavailableDecision(reason string) Decision {
    return Decision{
        Allowed:           true,
        Status:            200,
        Bypass:            true,
        Unavailable:       true,
        UnavailableReason: reason,
    }
}
