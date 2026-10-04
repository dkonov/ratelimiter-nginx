package ratelimiter;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.concurrent.atomic.AtomicLong;

public final class RateLimiter {
    public enum FailMode { OPEN, CLOSED }
    public enum Outcome { ALLOWED, DENIED, BYPASSED, UNAVAILABLE, CONFIG_ERROR }

    public record Decision(Outcome outcome, int statusCode, String error) {
        public boolean allowed() {
            return outcome == Outcome.ALLOWED || outcome == Outcome.BYPASSED;
        }
    }

    public record Stats(long allowed, long denied, long bypassed, long unavailable, long configError) {}

    private final URI checkUri;
    private final String service;
    private final FailMode failMode;
    private final Duration requestTimeout;
    private final HttpClient client;

    private final AtomicLong allowed = new AtomicLong();
    private final AtomicLong denied = new AtomicLong();
    private final AtomicLong bypassed = new AtomicLong();
    private final AtomicLong unavailable = new AtomicLong();
    private final AtomicLong configError = new AtomicLong();

    public RateLimiter(String url, String service) {
        this(url, service, FailMode.OPEN, Duration.ofMillis(20), Duration.ofMillis(50));
    }

    public RateLimiter(String url, String service, FailMode failMode,
                       Duration connectTimeout, Duration requestTimeout) {
        if (url == null || url.isBlank()) throw new IllegalArgumentException("ratelimiter: url is required");
        if (service == null || service.isBlank()) throw new IllegalArgumentException("ratelimiter: service is required");

        this.checkUri = URI.create(url.replaceAll("/+$", "") + "/check");
        this.service = service.trim();
        this.failMode = failMode;
        this.requestTimeout = requestTimeout;
        this.client = HttpClient.newBuilder()
                .connectTimeout(connectTimeout)
                .version(HttpClient.Version.HTTP_1_1)
                .build();
    }

    public Decision allow(String endpoint) {
        if (endpoint == null || endpoint.isBlank()) {
            configError.incrementAndGet();
            return new Decision(Outcome.CONFIG_ERROR, 500, "ratelimiter: endpoint is empty");
        }
        endpoint = endpoint.trim();

        HttpRequest request = HttpRequest.newBuilder(checkUri)
                .timeout(requestTimeout)
                .header("X-RateLimit-Service", service)
                .header("X-RateLimit-Endpoint", endpoint)
                .POST(HttpRequest.BodyPublishers.noBody())
                .build();

        try {
            HttpResponse<Void> response = client.send(request, HttpResponse.BodyHandlers.discarding());
            int status = response.statusCode();
            if (status >= 200 && status < 300) {
                allowed.incrementAndGet();
                return new Decision(Outcome.ALLOWED, status, null);
            }
            if (status == 429) {
                denied.incrementAndGet();
                return new Decision(Outcome.DENIED, status, null);
            }
            if (status == 404) {
                configError.incrementAndGet();
                return new Decision(Outcome.CONFIG_ERROR, 500,
                        "ratelimiter: unknown policy " + service + ":" + endpoint);
            }
            return onUnavailable("ratelimiter: unexpected status " + status);
        } catch (Exception ex) {
            return onUnavailable(ex.toString());
        }
    }

    public Stats stats() {
        return new Stats(allowed.get(), denied.get(), bypassed.get(), unavailable.get(), configError.get());
    }

    private Decision onUnavailable(String error) {
        unavailable.incrementAndGet();
        if (failMode == FailMode.OPEN) {
            bypassed.incrementAndGet();
            return new Decision(Outcome.BYPASSED, 200, error);
        }
        return new Decision(Outcome.UNAVAILABLE, 503, error);
    }
}
