package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type FailMode int

const (
	FailOpen FailMode = iota
	FailClosed
)

type Outcome int

const (
	Allowed Outcome = iota
	Denied
	Bypassed
	Unavailable
	ConfigError
)

type Decision struct {
	Outcome    Outcome
	StatusCode int
	Err        error
}

func (d Decision) IsAllowed() bool {
	return d.Outcome == Allowed || d.Outcome == Bypassed
}

type Config struct {
	URL            string
	Service        string
	Timeout        time.Duration
	ConnectTimeout time.Duration
	FailMode       FailMode
	Client         *http.Client
}

type Stats struct {
	Allowed     uint64 `json:"allowed"`
	Denied      uint64 `json:"denied"`
	Bypassed    uint64 `json:"bypassed"`
	Unavailable uint64 `json:"unavailable"`
	ConfigError uint64 `json:"config_error"`
}

type Limiter struct {
	checkURL string
	service  string
	failMode FailMode
	client   *http.Client

	allowed     atomic.Uint64
	denied      atomic.Uint64
	bypassed    atomic.Uint64
	unavailable atomic.Uint64
	configError atomic.Uint64
}

func New(cfg Config) (*Limiter, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, errors.New("ratelimiter: URL is required")
	}
	if strings.TrimSpace(cfg.Service) == "" {
		return nil, errors.New("ratelimiter: service is required")
	}

	base, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("ratelimiter: invalid URL: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/check"

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}
	connectTimeout := cfg.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = 20 * time.Millisecond
	}

	client := cfg.Client
	if client == nil {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   connectTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          256,
			MaxIdleConnsPerHost:   128,
			MaxConnsPerHost:       0,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   connectTimeout,
			ResponseHeaderTimeout: timeout,
			ExpectContinueTimeout: 1 * time.Second,
		}
		client = &http.Client{
			Transport: transport,
			Timeout:   timeout,
		}
	}

	return &Limiter{
		checkURL: base.String(),
		service:  cfg.Service,
		failMode: cfg.FailMode,
		client:   client,
	}, nil
}

func (l *Limiter) Allow(ctx context.Context, endpoint string) Decision {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		l.configError.Add(1)
		return Decision{Outcome: ConfigError, StatusCode: http.StatusInternalServerError, Err: errors.New("ratelimiter: endpoint is empty")}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.checkURL, http.NoBody)
	if err != nil {
		return l.onUnavailable(err)
	}
	req.Header.Set("X-RateLimit-Service", l.service)
	req.Header.Set("X-RateLimit-Endpoint", endpoint)
	req.Header.Set("Connection", "keep-alive")

	resp, err := l.client.Do(req)
	if err != nil {
		return l.onUnavailable(err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		l.allowed.Add(1)
		return Decision{Outcome: Allowed, StatusCode: resp.StatusCode}

	case resp.StatusCode == http.StatusTooManyRequests:
		l.denied.Add(1)
		return Decision{Outcome: Denied, StatusCode: resp.StatusCode}

	case resp.StatusCode == http.StatusNotFound:
		l.configError.Add(1)
		return Decision{
			Outcome:    ConfigError,
			StatusCode: http.StatusInternalServerError,
			Err:        fmt.Errorf("ratelimiter: unknown policy %s:%s", l.service, endpoint),
		}

	default:
		return l.onUnavailable(fmt.Errorf("ratelimiter: unexpected status %d", resp.StatusCode))
	}
}

func (l *Limiter) onUnavailable(err error) Decision {
	l.unavailable.Add(1)
	if l.failMode == FailOpen {
		l.bypassed.Add(1)
		return Decision{Outcome: Bypassed, StatusCode: http.StatusOK, Err: err}
	}
	return Decision{Outcome: Unavailable, StatusCode: http.StatusServiceUnavailable, Err: err}
}

func (l *Limiter) Stats() Stats {
	return Stats{
		Allowed:     l.allowed.Load(),
		Denied:      l.denied.Load(),
		Bypassed:    l.bypassed.Load(),
		Unavailable: l.unavailable.Load(),
		ConfigError: l.configError.Load(),
	}
}
