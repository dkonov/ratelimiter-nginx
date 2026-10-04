# Go client

The reusable Go client remains in the repository root package `ratelimiter/`.
It intentionally depends only on the Go standard library.

Minimal use:

```go
limiter, _ := ratelimiter.New(ratelimiter.Config{
    URL:      "http://nginx2:8080",
    Service:  "demo",
    FailMode: ratelimiter.FailOpen,
})

decision := limiter.Allow(ctx, "shared-global")
if decision.Outcome == ratelimiter.Denied {
    // return HTTP 429
}
```

The string passed to `Allow` is a logical rate-limit key, not necessarily the application's HTTP path.
