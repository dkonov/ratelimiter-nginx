# Java rate limiter client

Minimal framework-independent Java client for the centralized nginx2 rate limiter. The implementation is in [`src/RateLimiter.java`](src/RateLimiter.java) and uses only the JDK `java.net.http.HttpClient`.

## Requirements

Java 21+ is recommended for the demo application.

## Install

Copy `RateLimiter.java` into your source tree and optionally add your package declaration:

```text
src/main/java/com/example/ratelimiter/RateLimiter.java
```

No Maven or Gradle dependency is required for this standalone source-file version.

## Create client

```java
RateLimiter rl = new RateLimiter("http://nginx2:8080", "demo");
```

Current transport settings:

- protocol: **HTTP/1.1**;
- connect timeout: 20 ms;
- request timeout: **50 ms**.

The client instance should be long-lived and reused so the JDK can reuse persistent connections.

## API

```java
RateLimiter.Decision d = rl.allow("order");
```

This checks the logical key `demo:order`.

```java
public record Decision(
    boolean allowed,
    int status,
    boolean bypass,
    boolean unavailable,
    String unavailableReason
) {}
```

## Decision semantics

| nginx2 result | allowed | status | bypass | unavailable | reason |
|---|---:|---:|---:|---:|---|
| `2xx` | true | 200 | false | false | empty |
| `429` | false | 429 | false | false | empty |
| `404` | false | 500 | false | false | empty |
| request/connect timeout | true | 200 | true | true | `timeout` |
| connection failure | true | 200 | true | true | `connect_error` |
| other I/O failure | true | 200 | true | true | `io_error` |
| nginx2 `5xx` | true | 200 | true | true | `5xx` |
| other unexpected limiter result | true | 200 | true | true | `other` |

Failures are **fail-open** by design.

## Plain Java example

```java
RateLimiter.Decision d = rl.allow("order");

if (!d.allowed()) {
    // Return d.status(), normally 429 or 500.
    return;
}

if (d.bypass()) {
    responseHeaders.add("X-RateLimit-Bypass", "true");
}
if (d.unavailable()) {
    responseHeaders.add("X-RateLimit-Unavailable", "true");
    responseHeaders.add("X-RateLimit-Unavailable-Reason", d.unavailableReason());
}

// Protected application work starts here.
```

## Spring MVC example

The client has no Spring dependency:

```java
@RestController
public class OrdersController {
    private final RateLimiter limiter =
        new RateLimiter("http://nginx2:8080", "demo");

    @GetMapping("/api/orders")
    public ResponseEntity<String> orders() {
        RateLimiter.Decision d = limiter.allow("order");

        if (!d.allowed()) {
            return ResponseEntity.status(d.status()).build();
        }

        ResponseEntity.BodyBuilder response = ResponseEntity.ok();
        if (d.bypass()) {
            response.header("X-RateLimit-Bypass", "true");
        }
        if (d.unavailable()) {
            response.header("X-RateLimit-Unavailable", "true");
            response.header("X-RateLimit-Unavailable-Reason", d.unavailableReason());
        }

        return response.body("ok");
    }
}
```

## Why HTTP/1.1 is forced

The limiter endpoint is a tiny synchronous request to standard NGINX. HTTP/1.1 avoids unnecessary protocol negotiation/fallback behavior and makes connection reuse predictable for this demo.

## Integration rule

Pass a stable logical policy name to `allow()` instead of the physical HTTP path:

```text
/api/java/order -> allow("order") -> demo:order
```

Go or Python services can use the same `demo:order` key and consume the same global limiter capacity.

## Operational metrics

Monitor at least:

- `bypass`;
- `unavailable`;
- `unavailable_reason` (`timeout`, `connect_error`, `io_error`, `5xx`, `other`).

A sustained non-zero bypass rate means effective application throughput can exceed the configured nginx2 rate.
