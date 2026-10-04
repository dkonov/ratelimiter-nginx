package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type Outcome string

const (
	Allowed     Outcome = "allowed"
	Denied      Outcome = "denied"
	Bypassed    Outcome = "bypassed"
	Unavailable Outcome = "unavailable"
	ConfigError Outcome = "config_error"
)

type Decision struct {
	Outcome Outcome
	Status  int
	Err     error
}

type Config struct {
	URL      string
	Service  string
	Timeout  time.Duration
	FailOpen bool
}

type Client struct {
	url      string
	service  string
	failOpen bool
	http     *http.Client
}

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.URL) == "" || strings.TrimSpace(cfg.Service) == "" {
		return nil, errors.New("ratelimiter: URL and Service are required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}
	transport := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 20 * time.Millisecond, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:        128,
		MaxIdleConnsPerHost: 128,
		IdleConnTimeout:     90 * time.Second,
	}
	return &Client{
		url:      strings.TrimRight(cfg.URL, "/") + "/check",
		service:  cfg.Service,
		failOpen: cfg.FailOpen,
		http:     &http.Client{Transport: transport, Timeout: timeout},
	}, nil
}

func (c *Client) Check(ctx context.Context, endpoint string) Decision {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, http.NoBody)
	if err != nil {
		return c.unavailable(err)
	}
	req.Header.Set("X-RateLimit-Service", c.service)
	req.Header.Set("X-RateLimit-Endpoint", endpoint)

	resp, err := c.http.Do(req)
	if err != nil {
		return c.unavailable(err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return Decision{Outcome: Allowed, Status: resp.StatusCode}
	case resp.StatusCode == http.StatusTooManyRequests:
		return Decision{Outcome: Denied, Status: http.StatusTooManyRequests}
	case resp.StatusCode == http.StatusNotFound:
		return Decision{Outcome: ConfigError, Status: http.StatusInternalServerError,
			Err: fmt.Errorf("ratelimiter: unknown policy %s:%s", c.service, endpoint)}
	default:
		return c.unavailable(fmt.Errorf("ratelimiter: unexpected status %d", resp.StatusCode))
	}
}

func (c *Client) Middleware(endpoint string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decision := c.Check(r.Context(), endpoint)
		switch decision.Outcome {
		case Allowed:
			next.ServeHTTP(w, r)
		case Bypassed:
			w.Header().Set("X-RateLimit-Bypass", "true")
			next.ServeHTTP(w, r)
		case Denied:
			w.WriteHeader(http.StatusTooManyRequests)
		case ConfigError:
			http.Error(w, "rate-limit configuration error", http.StatusInternalServerError)
		default:
			http.Error(w, "rate limiter unavailable", http.StatusServiceUnavailable)
		}
	})
}

func (c *Client) unavailable(err error) Decision {
	if c.failOpen {
		return Decision{Outcome: Bypassed, Status: http.StatusOK, Err: err}
	}
	return Decision{Outcome: Unavailable, Status: http.StatusServiceUnavailable, Err: err}
}
