import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

public final class RateLimiter {
    private final URI checkUri;
    private final String service;
    private final HttpClient client;

    public RateLimiter(String baseUrl, String service) {
        this.checkUri = URI.create(baseUrl.replaceAll("/+$", "") + "/check");
        this.service = service;
        this.client = HttpClient.newBuilder()
            .connectTimeout(Duration.ofMillis(20))
            .build();
    }

    public Decision allow(String endpoint) {
        try {
            HttpRequest req = HttpRequest.newBuilder(checkUri)
                .timeout(Duration.ofMillis(50))
                .header("X-RateLimit-Service", service)
                .header("X-RateLimit-Endpoint", endpoint)
                .POST(HttpRequest.BodyPublishers.noBody())
                .build();

            int status = client.send(req, HttpResponse.BodyHandlers.discarding()).statusCode();

            if (status >= 200 && status < 300) {
                return new Decision(true, 200, false, false);
            }
            if (status == 429) {
                return new Decision(false, 429, false, false);
            }
            if (status == 404) {
                return new Decision(false, 500, false, false);
            }

            return unavailableDecision();
        } catch (Exception e) {
            return unavailableDecision();
        }
    }

    private Decision unavailableDecision() {
        return new Decision(true, 200, true, true);
    }

    public record Decision(boolean allowed, int status, boolean bypass, boolean unavailable) {}
}
