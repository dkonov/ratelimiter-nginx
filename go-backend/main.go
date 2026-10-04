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

        // A denied limiter decision stops this request here. Protected business
        // work below this point is not executed.
        if !d.Allowed {
            writeResponse(w, r, instance, policy, d, false)
            return
        }

        // Protected business work would execute here.
        executed := true
        writeResponse(w, r, instance, policy, d, executed)
    }
}

func writeResponse(w http.ResponseWriter, r *http.Request, instance, policy string, d ratelimiter.Decision, executed bool) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Backend-Executed", boolString(executed))
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
        "executed":           executed,
        "bypass":             d.Bypass,
        "unavailable":        d.Unavailable,
        "unavailable_reason": d.UnavailableReason,
    })
}

func boolString(v bool) string {
    if v {
        return "true"
    }
    return "false"
}

func env(k, d string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return d
}
