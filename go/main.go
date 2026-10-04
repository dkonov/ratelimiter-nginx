package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"example.com/ratelimiter-demo-go/ratelimiter"
)

func main() {
	instance := env("BACKEND_INSTANCE", "go")
	limiter, err := ratelimiter.New(ratelimiter.Config{
		URL:      env("RATE_LIMITER_URL", "http://nginx2:8080"),
		Service:  "demo",
		Timeout:  50 * time.Millisecond,
		FailOpen: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/go/ordrer", withBackend(instance, limiter.Middleware("order", jsonHandler(instance, "/api/go/ordrer"))))
	mux.Handle("/api/go/common", withBackend(instance, limiter.Middleware("common", jsonHandler(instance, "/api/go/common"))))

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func withBackend(instance string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend-Instance", instance)
		next.ServeHTTP(w, r)
	})
}

func jsonHandler(instance, endpoint string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"backend": instance, "endpoint": endpoint})
	})
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
