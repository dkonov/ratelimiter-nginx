# Python rate limiter client

Minimal framework-independent Python client for the centralized nginx2 rate limiter. The implementation is in [`ratelimiter.py`](ratelimiter.py) and uses only the Python standard library.

## Install

Copy `ratelimiter.py` into your project:

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
rl = RateLimiter("http://nginx2:8080", "demo")
```

Arguments:

- `base_url` — nginx2 base URL;
- `service` — logical service part of the limiter key;
- `timeout` — request timeout in seconds, default **0.05** (50 ms).

Create one client per application process and reuse it.

## API

```python
allowed, status, bypass, unavailable, unavailable_reason = rl.allow("order")
```

This checks the logical key `demo:order`.

Return values:

- `allowed` — whether protected work may continue;
- `status` — HTTP status to return when processing is stopped;
- `bypass` — request was allowed without a successful nginx2 decision;
- `unavailable` — nginx2 could not provide a usable decision;
- `unavailable_reason` — empty on normal decisions, otherwise one of the documented failure reasons.

## Decision semantics

| nginx2 result | allowed | status | bypass | unavailable | reason |
|---|---:|---:|---:|---:|---|
| `2xx` | true | 200 | false | false | empty |
| `429` | false | 429 | false | false | empty |
| `404` | false | 500 | false | false | empty |
| timeout | true | 200 | true | true | `timeout` |
| DNS/connect failure | true | 200 | true | true | `connect_error` |
| socket/HTTP I/O failure | true | 200 | true | true | `io_error` |
| nginx2 `5xx` | true | 200 | true | true | `5xx` |
| other unexpected limiter result | true | 200 | true | true | `other` |

Failures are **fail-open** by design.

## Example

```python
allowed, status, bypass, unavailable, reason = rl.allow("order")

if not allowed:
    # Return status, usually 429 or 500.
    return

if bypass:
    response_headers["X-RateLimit-Bypass"] = "true"
if unavailable:
    response_headers["X-RateLimit-Unavailable"] = "true"
    response_headers["X-RateLimit-Unavailable-Reason"] = reason

# Protected application work starts here.
```

## Connection behavior

The client stores a persistent `HTTPConnection` per Python thread using `threading.local()` and resets that connection after transport failures.

## Integration rule

The argument passed to `allow()` is a logical policy name, not necessarily the physical URL:

```text
/api/python/order -> allow("order") -> demo:order
```

Go and Java services can use the same `demo:order` key and share the same global limit.

## Operational metrics

Monitor at least:

- `bypass`;
- `unavailable`;
- `unavailable_reason` (`timeout`, `connect_error`, `io_error`, `5xx`, `other`).

A sustained non-zero bypass rate means application throughput can exceed the configured nginx2 rate.
