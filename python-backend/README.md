# Python rate limiter client

Minimal framework-independent Python client for the centralized nginx2 rate limiter.

The implementation is in [`ratelimiter.py`](ratelimiter.py) and uses only the Python standard library.

## Install into another Python application

Copy `ratelimiter.py` into your project, for example:

```text
my_service/
├── app.py
└── ratelimiter.py
```

Import it:

```python
from ratelimiter import RateLimiter
```

No pip packages are required.

## Create client

```python
rl = RateLimiter(
    "http://nginx2:8080",
    "demo",
)
```

Arguments:

- `base_url` — nginx2 base URL.
- `service` — logical service part of the key.
- `timeout` — request timeout in seconds; default is **0.05** (50 ms).

## API

```python
allowed, status, bypass, unavailable = rl.allow("order")
```

This checks:

```text
demo:order
```

Return values:

- `allowed` — whether application work may continue.
- `status` — HTTP status the application should return when it stops processing.
- `bypass` — request was allowed without a successful nginx2 decision.
- `unavailable` — nginx2 was unreachable, timed out or returned an unexpected 5xx-class result.

## Decision semantics

| nginx2 result | allowed | status | bypass | unavailable |
|---|---:|---:|---:|---:|
| `2xx` | true | 200 | false | false |
| `429` | false | 429 | false | false |
| `404` | false | 500 | false | false |
| timeout / connection error / `5xx` | true | 200 | true | true |

The last case is **fail-open**.

## Standard-library HTTP example

```python
allowed, status, bypass, unavailable = rl.allow("order")

if not allowed:
    # Return status, usually 429 or 500.
    return

# Execute application work here.
```

If the application exposes diagnostics to callers, propagate these headers on fail-open:

```python
if bypass:
    response_headers["X-RateLimit-Bypass"] = "true"
if unavailable:
    response_headers["X-RateLimit-Unavailable"] = "true"
```

## Flask example

The library has no Flask dependency, but it can be wrapped easily:

```python
from flask import Flask, jsonify, make_response
from ratelimiter import RateLimiter

app = Flask(__name__)
rl = RateLimiter("http://nginx2:8080", "demo")

@app.get("/api/orders")
def orders():
    allowed, status, bypass, unavailable = rl.allow("order")

    if not allowed:
        return "", status

    response = make_response(jsonify(ok=True), 200)
    if bypass:
        response.headers["X-RateLimit-Bypass"] = "true"
    if unavailable:
        response.headers["X-RateLimit-Unavailable"] = "true"
    return response
```

## FastAPI example

```python
from fastapi import FastAPI, Response
from ratelimiter import RateLimiter

app = FastAPI()
rl = RateLimiter("http://nginx2:8080", "demo")

@app.get("/api/orders")
def orders(response: Response):
    allowed, status, bypass, unavailable = rl.allow("order")

    if not allowed:
        response.status_code = status
        return

    if bypass:
        response.headers["X-RateLimit-Bypass"] = "true"
    if unavailable:
        response.headers["X-RateLimit-Unavailable"] = "true"

    return {"ok": True}
```

## Connection behavior

The client stores a persistent `HTTPConnection` per Python thread using `threading.local()`.

This avoids opening a new TCP connection for every request while remaining safe for threaded servers.

## Integration rule

The argument passed to `allow()` is a logical limiter key, not necessarily the physical HTTP path.

Example:

```text
/api/python/order -> allow("order") -> demo:order
```

A Java or Go service can use the same `demo:order` key and share the same global limit.

## Production considerations

- Create one `RateLimiter` object per application process, not per request.
- Keep the 50 ms timeout visible in operational metrics.
- Monitor `bypass` and `unavailable`; sustained non-zero values mean nginx2 is not reliably answering.
- `404` indicates a missing policy in nginx2 configuration and is returned as application `500`.
