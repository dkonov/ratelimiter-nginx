# ratelimiter-nginx

Centralized global rate limiter built on standard OSS NGINX, with minimal clients for Go, Java and Python.

No PostgreSQL, Redis, OpenResty, Lua, njs or third-party NGINX modules are required.

## Architecture

```text
                         nginx1
                 incoming router / LB
                          |
          +---------------+---------------+
          |               |               |
       Go x3           Java x3         Python x3
          |               |               |
          +---------------+---------------+
                          |
                 synchronous POST /check
                          |
                        nginx2
                 centralized rate limiter
```

Every backend asks nginx2 for permission before executing the endpoint:

```http
POST /check
X-RateLimit-Service: demo
X-RateLimit-Endpoint: order
```

The HTTP path and rate-limit key are intentionally independent. Different applications and languages can therefore share one global limit.

## Policies

| Application endpoint | Rate-limit key | NGINX rate | Burst |
|---|---|---:|---:|
| `/api/java/order` | `demo:order` | 100 r/s | 10 |
| `/api/go/ordrer` | `demo:order` | 100 r/s | 10 |
| `/api/python/order` | `demo:order` | 100 r/s | 10 |
| `/api/java/common` | `demo:common` | 300 r/s | 30 |
| `/api/go/common` | `demo:common` | 300 r/s | 30 |
| `/api/python/search` | `demo:search` | 10 r/s | 2 |

All entries with the same logical key share one global bucket. For example Java, Go and Python `order` traffic compete for the same `demo:order = 100 r/s` limit.

The limiter uses `burst + nodelay`, so short request microbursts are accepted without queueing while sustained traffic is still constrained by the configured rate.

> NGINX `limit_req` is a leaky-bucket rate limiter, not an exact wall-clock fixed-window counter.

## Backend instances

Docker Compose starts nine backend processes:

```text
go1 go2 go3
java1 java2 java3
python1 python2 python3
```

nginx1 load-balances each language group using standard NGINX upstream round-robin.

## Requirements

Install:

- Git
- Docker Engine
- Docker Compose plugin (`docker compose`)

Check:

```bash
git --version
docker --version
docker compose version
```

## Download and install

Recommended method:

```bash
cd /opt
git clone https://github.com/dkonov/ratelimiter-nginx.git
cd ratelimiter-nginx
```

Start the complete stack:

```bash
docker compose up -d --build
```

Check containers:

```bash
docker compose ps
```

nginx1 is exposed on:

```text
http://127.0.0.1:8080
```

nginx2 is exposed on:

```text
http://127.0.0.1:8081
```

### Updating an existing installation

```bash
cd /opt/ratelimiter-nginx
git pull
docker compose up -d --build
```

### ZIP download

If Git is not available:

```bash
wget -O ratelimiter-nginx.zip \
  https://github.com/dkonov/ratelimiter-nginx/archive/refs/heads/main.zip
unzip ratelimiter-nginx.zip
cd ratelimiter-nginx-main
docker compose up -d --build
```

A ZIP directory is not a Git repository, so `git pull`, `git fetch` and `git reset` do not work inside it.

## Smoke test

```bash
curl http://127.0.0.1:8080/api/java/order
curl http://127.0.0.1:8080/api/java/common
curl http://127.0.0.1:8080/api/go/ordrer
curl http://127.0.0.1:8080/api/go/common
curl http://127.0.0.1:8080/api/python/order
curl http://127.0.0.1:8080/api/python/search
```

A normal allowed response looks like:

```json
{
  "backend": "go2",
  "endpoint": "/api/go/ordrer",
  "policy": "demo:order",
  "allowed": true,
  "bypass": false,
  "unavailable": false
}
```

## Client behavior

All three clients use the same decision model:

| nginx2 result | Backend behavior |
|---|---|
| `2xx` | allow request |
| `429` | return `429` |
| `404` | configuration error, return `500` |
| timeout / connection error / nginx2 `5xx` | fail-open: continue request with `200` |

The limiter timeout is intentionally **50 ms**.

Fail-open responses are explicitly marked:

```http
X-RateLimit-Bypass: true
X-RateLimit-Unavailable: true
```

and their JSON body contains:

```json
{
  "bypass": true,
  "unavailable": true
}
```

In the current implementation every `unavailable` decision is also a `bypass`, because the clients use fail-open behavior.

## Language client documentation

- [Go client](go-backend/ratelimiter/README.md)
- [Python client](python-backend/README.md)
- [Java client](java-backend/README.md)

Each client is framework-agnostic and only performs the small synchronous `/check` request. Application routing remains the responsibility of the application/framework.

## Load test

The repository contains a k6 load generator. The helper script starts/rebuilds the application stack, waits for nginx1 health, and then runs k6 using host networking.

Default test:

```bash
sh loadtest/run.sh
```

60-second test:

```bash
DURATION=60s sh loadtest/run.sh
```

Override traffic independently for each application endpoint:

```bash
DURATION=60s \
JAVA_ORDER_RPS=120 \
GO_ORDER_RPS=120 \
PYTHON_ORDER_RPS=120 \
JAVA_COMMON_RPS=180 \
GO_COMMON_RPS=180 \
PYTHON_SEARCH_RPS=30 \
sh loadtest/run.sh
```

## k6 metrics

Per logical policy:

```text
order_200
order_429
order_bypass
order_unavailable
order_unexpected

common_200
common_429
common_bypass
common_unavailable
common_unexpected

search_200
search_429
search_bypass
search_unavailable
search_unexpected
```

Per endpoint:

```text
java_order_200
java_order_429
java_order_bypass
java_order_unavailable

go_order_...
python_order_...
java_common_...
go_common_...
python_search_...
```

Per backend instance:

```text
java_order_java1
java_order_java2
java_order_java3

go_order_go1
go_order_go2
go_order_go3

python_order_python1
python_order_python2
python_order_python3
```

`*_200` includes fail-open responses. To estimate requests explicitly allowed by nginx2:

```text
limiter_allowed ~= *_200 - *_bypass
```

## Direct nginx2 check

Allowed/limited decision can also be tested without a backend:

```bash
curl -i -X POST http://127.0.0.1:8081/check \
  -H 'X-RateLimit-Service: demo' \
  -H 'X-RateLimit-Endpoint: order'
```

Known policy returns `204` or `429` depending on rate state.

Unknown policy:

```bash
curl -i -X POST http://127.0.0.1:8081/check \
  -H 'X-RateLimit-Service: demo' \
  -H 'X-RateLimit-Endpoint: unknown'
```

returns `404`.

## Add a new policy

1. Choose a stable logical key, for example `demo:payments`.
2. Register it in `nginx2/nginx.conf` `map`.
3. Add or reuse a `limit_req_zone`.
4. Add an internal limiter location using `limit_req ... burst=N nodelay`.
5. Call the language client with the logical endpoint name (`payments`) from the application endpoint.

The logical limiter key should represent the resource being protected, not necessarily the physical HTTP path.

## Troubleshooting

Container state:

```bash
docker compose ps -a
```

nginx1 logs:

```bash
docker compose logs --tail=100 nginx1
```

nginx2 logs:

```bash
docker compose logs --tail=100 nginx2
```

All logs:

```bash
docker compose logs --tail=100
```

Rebuild everything cleanly:

```bash
docker compose down --remove-orphans
docker compose up -d --build
```
