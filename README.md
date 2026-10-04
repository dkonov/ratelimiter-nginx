# ratelimiter-nginx

Minimal demo of a centralized rate limiter built only on standard NGINX.

## Topology

- `nginx1` — incoming router/load balancer.
- `nginx2` — central rate limiter.
- 3 Go instances: `go1..go3`.
- 3 Java instances: `java1..java3`.
- 3 Python instances: `python1..python3`.
- `k6` — load generator.

Each backend has a tiny language-specific rate-limit client. Before executing an endpoint it sends synchronous `POST /check` to nginx2 with:

```text
X-RateLimit-Service: demo
X-RateLimit-Endpoint: <order|common|search>
```

## Endpoints and shared policies

| Endpoint | Policy | Limit |
|---|---|---:|
| `/api/java/order` | `demo:order` | 100 r/s |
| `/api/go/ordrer` | `demo:order` | 100 r/s |
| `/api/python/order` | `demo:order` | 100 r/s |
| `/api/java/common` | `demo:common` | 300 r/s |
| `/api/go/common` | `demo:common` | 300 r/s |
| `/api/python/search` | `demo:search` | 10 r/s |

The limits are global by logical key. For example all 9 backend processes that call `demo:order` compete for one shared 100 r/s limit.

## Start

```bash
docker compose up -d --build
```

Smoke test:

```bash
curl http://localhost:8080/api/java/order
curl http://localhost:8080/api/go/ordrer
curl http://localhost:8080/api/python/order
curl http://localhost:8080/api/java/common
curl http://localhost:8080/api/go/common
curl http://localhost:8080/api/python/search
```

## Load test

```bash
sh loadtest/run.sh
```

Override per-endpoint rates independently:

```bash
DURATION=30s \
JAVA_ORDER_RPS=50 GO_ORDER_RPS=50 PYTHON_ORDER_RPS=50 \
JAVA_COMMON_RPS=200 GO_COMMON_RPS=200 \
PYTHON_SEARCH_RPS=30 \
sh loadtest/run.sh
```

k6 prints counters for every endpoint/status and every backend instance, for example:

```text
java_order_200
java_order_429
java_order_java1
java_order_java2
java_order_java3
...
python_search_python1
python_search_python2
python_search_python3
```

This makes it visible both how the global limiter behaves and how nginx1 distributes traffic between instances.
