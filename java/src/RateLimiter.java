package demo;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

public final class RateLimiter {
    public enum Outcome { ALLOWED, DENIED, BYPASSED, UNAVAILABLE, CONFIG_ERROR }
    public record Decision(Outcome outcome, int status) {}

    private final URI checkUri;
    private final String service;
    private final boolean failOpen;
    private final HttpClient client;

    public RateLimiter(String baseUrl, String service, boolean failOpen) {
        this.checkUri = URI.create(baseUrl.replaceAll("/+$", "") + "/check");
        this.service = service;
        this.failOpen = failOpen;
        this.client = HttpClient.newBuilder()
                .connectTimeout(Duration.ofMillis(20))
                .build();
    }

    public Decision check(String endpoint) {
        try {
            HttpRequest request = HttpRequest.newBuilder(checkUri)
                    .timeout(Duration.ofMillis(50))
                    .header("X-RateLimit-Service", service)
                    .header("X-RateLimit-Endpoint", endpoint)
                    .POST(HttpRequest.BodyPublishers.noBody())
                    .build();

            int status = client.send(request, HttpResponse.BodyHandlers.discarding()).statusCode();
            if (status >= 200 && status < 300) return new Decision(Outcome.ALLOWED, status);
            if (status == 429) return new Decision(Outcome.DENIED, 429);
            if (status == 404) return new Decision(Outcome.CONFIG_ERROR, 500);
            return unavailable();
        } catch (Exception ignored) {
            return unavailable();
        }
    }

    private Decision unavailable() {
        return failOpen
                ? new Decision(Outcome.BYPASSED, 200)
                : new Decision(Outcome.UNAVAILABLE, 503);
    }
}
