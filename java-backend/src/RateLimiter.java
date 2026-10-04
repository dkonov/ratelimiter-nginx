import java.io.IOException;
import java.net.ConnectException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.net.http.HttpTimeoutException;
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
            .version(HttpClient.Version.HTTP_1_1)
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
                return new Decision(true, 200, false, false, "");
            }
            if (status == 429) {
                return new Decision(false, 429, false, false, "");
            }
            if (status == 404) {
                return new Decision(false, 500, false, false, "");
            }
            if (status >= 500) {
                return unavailableDecision("5xx");
            }

            return unavailableDecision("other");
        } catch (HttpTimeoutException e) {
            return unavailableDecision("timeout");
        } catch (ConnectException e) {
            return unavailableDecision("connect_error");
        } catch (IOException e) {
            return unavailableDecision("io_error");
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return unavailableDecision("other");
        } catch (Exception e) {
            return unavailableDecision("other");
        }
    }

    private Decision unavailableDecision(String reason) {
        return new Decision(true, 200, true, true, reason);
    }

    public record Decision(
        boolean allowed,
        int status,
        boolean bypass,
        boolean unavailable,
        String unavailableReason
    ) {}
}
