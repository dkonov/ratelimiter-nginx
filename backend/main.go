package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"ratelimiter-nginx/ratelimiter"
)

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func envInt(name string, def int) int {
	v, err := strconv.Atoi(env(name, strconv.Itoa(def)))
	if err != nil {
		return def
	}
	return v
}

func main() {
	backendName := env("BACKEND_NAME", "backend")
	serviceName := env("SERVICE_NAME", "demo")
	limiterURL := env("RATE_LIMITER_URL", "http://nginx2:8080")
	failMode := ratelimiter.FailOpen
	if env("RATE_LIMITER_FAIL_MODE", "open") == "closed" {
		failMode = ratelimiter.FailClosed
	}

	limiter, err := ratelimiter.New(ratelimiter.Config{
		URL:            limiterURL,
		Service:        serviceName,
		Timeout:        time.Duration(envInt("RATE_LIMITER_TIMEOUT_MS", 50)) * time.Millisecond,
		ConnectTimeout: time.Duration(envInt("RATE_LIMITER_CONNECT_TIMEOUT_MS", 20)) * time.Millisecond,
		FailMode:       failMode,
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.Handle("/api/work", limiter.Middleware("work", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		minMS := envInt("WORK_MIN_MS", 5)
		maxMS := envInt("WORK_MAX_MS", 20)
		sleepRandom(minMS, maxMS)
		respondJSON(w, map[string]any{
			"backend":  backendName,
			"endpoint": "work",
			"ok":       true,
		})
	})))

	mux.Handle("/api/search", limiter.Middleware("search", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sleepRandom(10, 40)
		respondJSON(w, map[string]any{
			"backend":  backendName,
			"endpoint": "search",
			"ok":       true,
		})
	})))

	mux.Handle("/api/slow", limiter.Middleware("slow", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sleepRandom(1000, 2000)
		respondJSON(w, map[string]any{
			"backend":  backendName,
			"endpoint": "slow",
			"ok":       true,
		})
	})))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/debug/stats", func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		respondJSON(w, map[string]any{
			"backend":          backendName,
			"goroutines":       runtime.NumGoroutine(),
			"gomaxprocs":       runtime.GOMAXPROCS(0),
			"heap_alloc_bytes": m.HeapAlloc,
			"rate_limiter":     limiter.Stats(),
		})
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		s := limiter.Stats()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "ratelimiter_allowed_total %d\n", s.Allowed)
		fmt.Fprintf(w, "ratelimiter_denied_total %d\n", s.Denied)
		fmt.Fprintf(w, "ratelimiter_bypass_total %d\n", s.Bypassed)
		fmt.Fprintf(w, "ratelimiter_unavailable_total %d\n", s.Unavailable)
		fmt.Fprintf(w, "ratelimiter_config_error_total %d\n", s.ConfigError)
		fmt.Fprintf(w, "backend_goroutines %d\n", runtime.NumGoroutine())
	})

	srv := &http.Server{
		Addr:              env("LISTEN_ADDR", ":8080"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("%s listening on %s, service=%s, limiter=%s", backendName, srv.Addr, serviceName, limiterURL)
	log.Fatal(srv.ListenAndServe())
}

func sleepRandom(minMS, maxMS int) {
	if maxMS < minMS {
		maxMS = minMS
	}
	d := minMS
	if maxMS > minMS {
		d += rand.Intn(maxMS - minMS + 1)
	}
	time.Sleep(time.Duration(d) * time.Millisecond)
}

func respondJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
