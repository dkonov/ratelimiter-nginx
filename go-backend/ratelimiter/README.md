# Go rate limiter client

Minimal framework-independent Go client for the centralized nginx2 rate limiter.

The implementation is in [`ratelimiter.go`](ratelimiter.go) and depends only on the Go standard library.

## Purpose

Before application work starts, the client synchronously asks nginx2 whether the logical rate-limit key still has capacity.

Protocol:

```http
POST /check
X-RateLimit-Service: demo
X-RateLimit-Endpoint: order
```

The client does not know or care which HTTP framework the application uses.

## Install into another Go application

Copy the `ratelimiter` directory into your module, for example:

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

## Create client

```go
rl := ratelimiter.New(
    "http://nginx2:8080",
    "demo",
)
```

Arguments:

- `baseURL` — nginx2 base URL.
- `service` — logical service part of the rate-limit key.

The client appends `/check` automatically.

Current total request timeout: **50 ms**.

## API

```go
Decision Allow(ctx context.Context, endpoint string)
```

Example:

```go
d := rl.Allow(r.Context(), "order")
```

This produces the logical key:

```text
demo:order
```

`Decision`:

```go
type Decision struct {
    Allowed     bool
    Status      int
    Bypass      bool
    Unavailable bool
}
```

## Decision semantics

| nginx2 result | Allowed | Status | Bypass | Unavailable |
|---|---:|---:|---:|---:|
| `2xx` | true | 200 | false | false |
| `429` | false | 429 | false | false |
| `404` | false | 500 | false | false |
| timeout / connection error / `5xx` | true | 200 | true | true |

The last case is **fail-open**: application work is allowed when nginx2 cannot provide a decision.

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
        }

        if !d.Allowed {
            http.Error(w, http.StatusText(d.Status), d.Status)
            return
        }

        // Application work starts only after the limiter decision.
        w.WriteHeader(http.StatusOK)
    }
}
```

## Integration rule

Use a stable logical policy name rather than passing the physical URL as the limiter key.

For example these unrelated HTTP paths can intentionally share one policy:

```text
/api/go/ordrer     -> demo:order
/api/java/order    -> demo:order
/api/python/order  -> demo:order
```

That is how the global cross-application limit is achieved.

## Production considerations

- Keep the client instance long-lived; do not create one per request.
- The standard `http.Client` reuses connections automatically.
- The current library is fail-open by design.
- `Bypass=true` should be monitored because it means the request was not explicitly allowed by nginx2.
- A `404` is treated as configuration error, not as fail-open.
