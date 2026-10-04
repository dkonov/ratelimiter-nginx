# ratelimiter-nginx

Minimal demo of a centralized rate limiter built with pure NGINX.

## Topology

- `nginx1` — incoming router/load balancer.
- `nginx2` — centralized rate limiter.
- Go backend — 3 instances.
- Java backend — 3 instances.
- Python backend — 3 instances.
- `k6` — load generator and summary by endpoint/backend.

All application endpoints answer immediately. Before business handling, each backend synchronously calls `nginx2 /check` through its language-specific rate-limit library.

## Rate-limit policies

| Logical key | Limit | Physical endpoints |
|---|---:|---|
| `demo:order` | 100 r/s | `/api/java/order`, `/api/go/ordrer`, `/api/python/order` |
| `demo:common` | 300 r/s | `/api/java/common`, `/api/go/common` |
| `demo:search` | 10 r/s | `/api/python/search` |

The limit is shared by all backend instances and by all physical endpoints that use the same logical key.

## Start

```bash
docker compose up -d --build
```

## Quick checks

```bash
curl -i http://localhost:8080/api/java/order
curl -i http://localhost:8080/api/java/common
curl -i http://localhost:8080/api/go/ordrer
curl -i http://localhost:8080/api/go/common
curl -i http://localhost:8080/api/python/order
curl -i http://localhost:8080/api/python/search
```

Every response contains `X-Backend-Instance`, for example `java2`, `go1`, or `python3`.

## Load test

Default test intentionally exceeds all three shared limits:

- Java order: 60 r/s
- Go order: 60 r/s
- Python order: 60 r/s
- Java common: 200 r/s
- Go common: 200 r/s
- Python search: 20 r/s

Run for 30 seconds:

```bash
docker compose --profile load run --rm k6 run /scripts/load.js
```

Override any rate or duration:

```bash
docker compose --profile load run --rm \
  -e DURATION=60s \
  -e JAVA_ORDER_RATE=80 \
  -e GO_ORDRER_RATE=80 \
  -e PYTHON_ORDER_RATE=80 \
  -e JAVA_COMMON_RATE=250 \
  -e GO_COMMON_RATE=250 \
  -e PYTHON_SEARCH_RATE=30 \
  k6 run /scripts/load.js
```

The k6 script prints three tables:

1. Per physical endpoint: total / 200 / 429 / other.
2. Per logical policy: combined total / 200 / 429.
3. Per backend instance: total / 200 / 429 / other.
