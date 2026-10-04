# Java rate limiter client

Minimal framework-independent Java client for the centralized nginx2 rate limiter.

The implementation is in [`src/RateLimiter.java`](src/RateLimiter.java) and uses only the JDK HTTP client.

## Requirements

Java 21+ is recommended for the demo application. The `RateLimiter` client itself uses standard `java.net.http.HttpClient` APIs.

## Install into another Java application

Copy `RateLimiter.java` into your source tree and optionally place it in your own package.

Example:

```text
src/main/java/com/example/ratelimiter/RateLimiter.java
```

If you add a package declaration, import it normally from application code.

No Maven/Gradle dependency is required for this standalone source-file version.

## Create client

```java
RateLimiter rl = new RateLimiter(
    "http://nginx2:8080",
    "demo"
);
```

Arguments:

- `baseUrl` — nginx2 base URL.
- `service` — logical service part of the rate-limit key.

Current settings:

- connect timeout: 20 ms
- request timeout: **50 ms**

## API

```java
RateLimiter.Decision decision = rl.allow("order");
```

This checks the logical key:

```text
demo:order
```

Decision record:

```java
public record Decision(
    boolean allowed,
    int status,
    boolean bypass,
    boolean unavailable
) {}
```

## Decision semantics

| nginx2 result | allowed | status | bypass | unavailable |
|---|---:|---:|---:|---:|
| `2xx` | true | 200 | false | false |
| `429` | false | 429 | false | false |
| `404` | false | 500 | false | false |
| timeout / connection error / `5xx` | true | 200 | true | true |

The last case is **fail-open**.

## Plain Java example

```java
RateLimiter.Decision d = rl.allow("order");

if (!d.allowed()) {
    // Return d.status(), normally 429 or 500.
    return;
}

// Execute application work here.
```

If the application exposes the limiter decision to callers, propagate diagnostic headers:

```java
if (d.bypass()) {
    responseHeaders.add("X-RateLimit-Bypass", "true");
}
if (d.unavailable()) {
    responseHeaders.add("X-RateLimit-Unavailable", "true");
}
```

## Spring MVC example

The client does not depend on Spring. A typical wrapper can look like:

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
        }

        return response.body("ok");
    }
}
```

## Connection behavior

`HttpClient` is created once and reused. Keep the `RateLimiter` instance long-lived rather than creating it per request.

The JDK client manages connection reuse internally.

## Integration rule

Pass a stable logical policy name to `allow()` instead of coupling the limiter directly to a physical HTTP path.

Example:

```text
/api/java/order -> allow("order") -> demo:order
```

Another Go or Python service can use the same `demo:order` key and consume the same global limiter capacity.

## Production considerations

- Reuse one `RateLimiter` instance per application process or component.
- Monitor `bypass` and `unavailable` decisions.
- The current client is intentionally fail-open for timeout/network/5xx failures.
- `404` is treated as configuration error and returned as application `500`.
- Keep limiter calls outside the business work itself: ask for permission first, then perform the protected work.
