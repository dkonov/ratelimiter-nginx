# Pure NGINX centralized rate limiter demo

No PostgreSQL, Redis, OpenResty, Lua, njs or third-party NGINX modules.

## Architecture

```text
client -> nginx1 -> backend1/backend2/backend3
                         |
                         +-- synchronous POST /check --> nginx2
                                                        |-- 204 allow
                                                        `-- 429 deny
```

All three backend instances use `SERVICE_NAME=demo`, so the limit for a policy such as `demo:work` is global across all backend instances.

## Policies

Defined in `nginx2/nginx.conf`:

| service:endpoint | rate | burst | demo endpoint |
|---|---:|---:|---|
| demo:slow | 100 r/s | 20 | `/api/slow` |
| demo:search | 500 r/s | 100 | `/api/search` |
| demo:work | 1000 r/s | 200 | `/api/work` |

NGINX `limit_req` uses a leaky-bucket model. This is intentionally **not** an exact wall-clock fixed-window counter.

## Start

```bash
docker compose up -d --build
```

Check:

```bash
curl -i http://localhost:8080/api/work
curl -i http://localhost:8080/api/search
curl -i http://localhost:8080/api/slow
```

Direct limiter check:

```bash
curl -i -X POST http://localhost:8081/check \
  -H 'X-RateLimit-Service: demo' \
  -H 'X-RateLimit-Endpoint: work'
```

Unknown policy returns 404:

```bash
curl -i -X POST http://localhost:8081/check \
  -H 'X-RateLimit-Service: demo' \
  -H 'X-RateLimit-Endpoint: does-not-exist'
```

## Load test

1000 r/s policy, generate 1200 arrivals/s:

```bash
docker compose --profile load run --rm \
  -e RATE=1200 -e DURATION=30s \
  k6 run /scripts/work.js
```

Slow backend endpoint: 1-2 seconds of work, limiter 100 r/s:

```bash
docker compose --profile load run --rm \
  -e RATE=120 -e DURATION=30s \
  k6 run /scripts/slow.js
```

## Observe backend goroutines and limiter counters

```bash
for b in backend1 backend2 backend3; do
  echo "=== $b ==="
  docker compose exec -T "$b" wget -qO- http://127.0.0.1:8080/debug/stats
  echo
done
```

Prometheus-like metrics from any backend:

```bash
docker compose exec -T backend1 wget -qO- http://127.0.0.1:8080/metrics
```

NGINX limiter decisions:

```bash
docker compose logs -f nginx2
```

The access log contains `limit_req_status="PASSED|REJECTED|..."`.

## Fail-open test

Stop nginx2:

```bash
docker compose stop nginx2
```

Requests continue through the backend because the default is `RATE_LIMITER_FAIL_MODE=open`:

```bash
curl -i http://localhost:8080/api/work
```

The response contains:

```text
X-RateLimit-Bypass: true
```

and `ratelimiter_bypass_total` increases.

Restore:

```bash
docker compose start nginx2
```

For fail-closed set:

```yaml
RATE_LIMITER_FAIL_MODE: closed
```

Then an unavailable nginx2 produces backend HTTP 503.

## Add a new backend endpoint

1. In the backend call middleware with a stable logical endpoint name:

```go
mux.Handle("/api/orders", limiter.Middleware("orders", ordersHandler))
```

2. Register the policy in `nginx2/nginx.conf`:

```nginx
map "$http_x_ratelimit_service:$http_x_ratelimit_endpoint" $rl_uri {
    ...
    "my-service:orders" /_rl/500;
}
```

If the desired rate class already exists, nothing else is required. If a new rate is needed, add a new `limit_req_zone` and a matching internal location.

## Library behavior

The reusable package is `ratelimiter/`.

- 2xx from nginx2 -> allow.
- 429 -> deny and return 429 to the caller.
- 404 -> configuration error, backend returns 500. It is **not** bypassed.
- timeout / connection error / nginx2 5xx:
  - fail-open -> execute backend work and add `X-RateLimit-Bypass: true`;
  - fail-closed -> return 503.
- HTTP transport uses keep-alive and connection pooling.
- Default total limiter timeout: 50 ms.
- Default connect timeout: 20 ms.
