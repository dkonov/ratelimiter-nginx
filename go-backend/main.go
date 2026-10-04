package main

import (
    "encoding/json"
    "net/http"
    "os"

    "demo/go-backend/ratelimiter"
)

func main() {
    instance := env("INSTANCE_NAME", "go")
    rl := ratelimiter.New(env("RATE_LIMITER_URL", "http://nginx2:8080"), "demo")

    mux := http.NewServeMux()
    mux.HandleFunc("/api/go/ordrer", wrap(instance, "order", rl))
    mux.HandleFunc("/api/go/common", wrap(instance, "common", rl))
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })

    http.ListenAndServe(":8080", mux)
}

func wrap(instance, policy string, rl *ratelimiter.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        d := rl.Allow(r.Context(), policy)

        w.Header().Set("Content-Type", "application/json")
        if d.Bypass {
            w.Header().Set("X-RateLimit-Bypass", "true")
        }
        if d.Unavailable {
            w.Header().Set("X-RateLimit-Unavailable", "true")
            w.Header().Set("X-RateLimit-Unavailable-Reason", d.UnavailableReason)
        }

        w.WriteHeader(d.Status)
        json.NewEncoder(w).Encode(map[string]any{
            "backend":            instance,
            "endpoint":           r.URL.Path,
            "policy":             "demo:" + policy,
            "allowed":            d.Allowed,
            "bypass":             d.Bypass,
            "unavailable":        d.Unavailable,
            "unavailable_reason": d.UnavailableReason,
        })
    }
}

func env(k, d string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return d
}
