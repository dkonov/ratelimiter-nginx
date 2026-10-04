package ratelimiter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAllow204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-RateLimit-Service"); got != "demo" {
			t.Fatalf("service header = %q", got)
		}
		if got := r.Header.Get("X-RateLimit-Endpoint"); got != "work" {
			t.Fatalf("endpoint header = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	l, err := New(Config{URL: srv.URL, Service: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	d := l.Allow(context.Background(), "work")
	if d.Outcome != Allowed {
		t.Fatalf("outcome = %v, err=%v", d.Outcome, d.Err)
	}
}

func TestDeny429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	l, _ := New(Config{URL: srv.URL, Service: "demo"})
	d := l.Allow(context.Background(), "work")
	if d.Outcome != Denied || d.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("decision = %+v", d)
	}
}

func Test404IsConfigError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	l, _ := New(Config{URL: srv.URL, Service: "demo", FailMode: FailOpen})
	d := l.Allow(context.Background(), "missing")
	if d.Outcome != ConfigError {
		t.Fatalf("decision = %+v", d)
	}
}

func TestFailOpen(t *testing.T) {
	l, _ := New(Config{
		URL:      "http://127.0.0.1:1",
		Service:  "demo",
		FailMode: FailOpen,
		Timeout:  20 * time.Millisecond,
	})
	d := l.Allow(context.Background(), "work")
	if d.Outcome != Bypassed || !d.IsAllowed() {
		t.Fatalf("decision = %+v", d)
	}
}

func TestFailClosed(t *testing.T) {
	l, _ := New(Config{
		URL:      "http://127.0.0.1:1",
		Service:  "demo",
		FailMode: FailClosed,
		Timeout:  20 * time.Millisecond,
	})
	d := l.Allow(context.Background(), "work")
	if d.Outcome != Unavailable || d.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("decision = %+v", d)
	}
}
