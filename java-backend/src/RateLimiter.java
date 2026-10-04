import java.net.URI;
import java.net.http.*;
import java.time.Duration;

public final class RateLimiter {
    private final URI checkUri;
    private final String service;
    private final HttpClient client;

    public RateLimiter(String baseUrl, String service) {
        this.checkUri = URI.create(baseUrl.replaceAll("/+$", "") + "/check");
        this.service = service;
        this.client = HttpClient.newBuilder().connectTimeout(Duration.ofMillis(20)).build();
    }

    public Decision allow(String endpoint) {
        try {
            HttpRequest req = HttpRequest.newBuilder(checkUri)
                .timeout(Duration.ofMillis(50))
                .header("X-RateLimit-Service", service)
                .header("X-RateLimit-Endpoint", endpoint)
                .POST(HttpRequest.BodyPublishers.noBody()).build();
            int s = client.send(req, HttpResponse.BodyHandlers.discarding()).statusCode();
            if (s >= 200 && s < 300) return new Decision(true, 200);
            if (s == 429) return new Decision(false, 429);
            if (s == 404) return new Decision(false, 500);
            return new Decision(true, 200);
        } catch (Exception e) {
            return new Decision(true, 200);
        }
    }
    public record Decision(boolean allowed, int status) {}
}
