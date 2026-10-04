# Multi-language clients and shared-key demo

The central limiter API is language-neutral:

```http
POST /check
X-RateLimit-Service: demo
X-RateLimit-Endpoint: shared-global
```

Reusable clients:

- Go: existing `ratelimiter/` package.
- Python: `clients/python/ratelimiter.py` (standard library only; one persistent connection per calling thread).
- Java 21+: `clients/java/.../RateLimiter.java` (JDK `HttpClient`, no third-party dependency).

All clients implement the same behavior: 2xx=allow, 429=deny, 404=config error, network/5xx=fail-open or fail-closed.

## Demo topology

```text
                 nginx1
           /python/   /java/
              |          |
       +------+--+   +---+------+
       |         |   |          |
    python1   python2 java1    java2
       \         \    /         /
        \         \  /         /
          POST /check to nginx2
          Service: demo
          Endpoint: shared-global
                    |
                10 requests/s
```

Physical application endpoints are different:

- Python: `GET /python/report`
- Java: `GET /java/orders`

But every instance sends the same logical limiter key `demo:shared-global`. Therefore the configured `10r/s` is shared globally across both applications and all four instances.

## Run

```bash
bash scripts/run-multilang.sh 10 10 30s
```

This generates 10 r/s to Python plus 10 r/s to Java against one global 10 r/s limiter. Expected steady-state total is approximately:

```text
shared_200  ~ 10/s
shared_429  ~ 10/s
```

The split of accepted requests between Python and Java is intentionally not guaranteed; the important invariant is the combined limit.

Direct checks:

```bash
curl -i http://localhost:8080/python/report
curl -i http://localhost:8080/java/orders
bash scripts/multilang-metrics.sh
```
