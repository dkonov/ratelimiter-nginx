package ratelimiter

import (
	"log"
	"net/http"
)

func (l *Limiter) Middleware(endpoint string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := l.Allow(r.Context(), endpoint)
		switch d.Outcome {
		case Allowed:
			next.ServeHTTP(w, r)
			return
		case Bypassed:
			w.Header().Set("X-RateLimit-Bypass", "true")
			next.ServeHTTP(w, r)
			return
		case Denied:
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		case ConfigError:
			if d.Err != nil {
				log.Printf("ratelimiter configuration error: %v", d.Err)
			}
			http.Error(w, "Rate limiter configuration error", http.StatusInternalServerError)
			return
		default:
			if d.Err != nil {
				log.Printf("ratelimiter unavailable: %v", d.Err)
			}
			http.Error(w, "Rate limiter unavailable", http.StatusServiceUnavailable)
			return
		}
	})
}
