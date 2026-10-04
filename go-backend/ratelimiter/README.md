# Go rate limiter client

Minimal framework-independent Go client for the centralized nginx2 rate limiter. The implementation is in [`ratelimiter.go`](ratelimiter.go) and uses only the Go standard library.

## Protocol

Before application work starts, the client synchronously asks nginx2 for permission:

```http
POST /check
X-RateLimit-Service: demo
X-RateLimit-Endpoint: order
```

The physical application URL and the logical limiter key are independent.

## Install

Copy the `ratelimiter` directory into your Go module:

```text
my-service/
├── go.mod
├── main.go
└── ratelimiter/
    └── ratelimiter.go
```

Then import it using your module path:

```go
import "my-service/ratelimiter"
```

No third-party dependencies are required.

## Create client

```go
rl := ratelimiter.New("http://nginx2:8080", "demo")
```

The client appends `/check` automatically. Current total request timeout: **50 ms**.

Create the client once and reuse it. `http.Client` provides connection pooling and keep-alive reuse.

## API

```go
Decision Allow(ctx context.Context, endpoint string)
```

Example:

```go
d := rl.Allow(r.Context(), "order")
```

This checks the logical key `demo:order`.

```go
type Decision struct {
    Allowed           bool
    Status            int
    Bypass            bool
    Unavailable       bool
    UnavailableReason string
}
```

## Decision semantics

| nginx2 result | Allowed | Status | Bypass | Unavailable | Reason |
|---|---:|---:|---:|---:|---|
| `2xx` | true | 200 | false | false | empty |
| `429` | false | 429 | false | false | empty |
| `404` | false | 500 | false | false | empty |
| timeout | true | 200 | true | true | `timeout` |
| connect failure | true | 200 | true | true | `connect_error` |
| other network I/O failure | true | 200 | true | true | `io_error` |
| nginx2 `5xx` | true | 200 | true | true | `5xx` |
| other unexpected limiter result | true | 200 | true | true | `other` |

Failures are **fail-open** by design: application work continues when nginx2 cannot provide a valid decision.

## net/http example

```go
func ordersHandler(rl *ratelimiter.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        d := rl.Allow(r.Context(), "order")

        if d.Bypass {
            w.Header().Set("X-RateLimit-Bypass", "true")
        }
        if d.Unavailable {
            w.Header().Set("X-RateLimit-Unavailable", "true")
            w.Header().Set("X-RateLimit-Unavailable-Reason", d.UnavailableReason)
        }

        if !d.Allowed {
            http.Error(w, http.StatusText(d.Status), d.Status)
            return
        }

        // Protected application work starts here.
        w.WriteHeader(http.StatusOK)
    }
}
```

## Integration rule

Use stable logical policy names rather than physical URLs. For example:

```text
/api/go/ordrer     -> demo:order
/api/java/order    -> demo:order
/api/python/order  -> demo:order
```

All callers using the same key consume the same global limiter capacity.

## Operational metrics

Monitor at least:

- `bypass` — requests allowed without a valid limiter decision;
- `unavailable` — limiter decision could not be obtained;
- `unavailable_reason` — `timeout`, `connect_error`, `io_error`, `5xx`, or `other`.

A sustained non-zero bypass rate indicates that the effective application throughput can exceed the configured nginx2 rate.
