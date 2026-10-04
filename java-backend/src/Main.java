import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;

public class Main {
    static final String INSTANCE = System.getenv().getOrDefault("INSTANCE_NAME", "java");
    static final RateLimiter RL = new RateLimiter(
        System.getenv().getOrDefault("RATE_LIMITER_URL", "http://nginx2:8080"),
        "demo"
    );

    public static void main(String[] args) throws Exception {
        HttpServer s = HttpServer.create(new InetSocketAddress(8080), 0);
        s.createContext("/api/java/order", handler("order"));
        s.createContext("/api/java/common", handler("common"));
        s.createContext("/health", x -> {
            x.sendResponseHeaders(204, -1);
            x.close();
        });
        s.setExecutor(java.util.concurrent.Executors.newVirtualThreadPerTaskExecutor());
        s.start();
    }

    static HttpHandler handler(String policy) {
        return x -> {
            RateLimiter.Decision d = RL.allow(policy);

            // A denied limiter decision stops this request here. Protected
            // business work below this point is not executed.
            if (!d.allowed()) {
                writeResponse(x, policy, d, false);
                return;
            }

            // Protected business work would execute here.
            boolean executed = true;
            writeResponse(x, policy, d, executed);
        };
    }

    static void writeResponse(HttpExchange x, String policy, RateLimiter.Decision d, boolean executed) throws IOException {
        String body = String.format(
            "{\"backend\":\"%s\",\"endpoint\":\"%s\",\"policy\":\"demo:%s\",\"allowed\":%s,\"executed\":%s,\"bypass\":%s,\"unavailable\":%s,\"unavailable_reason\":\"%s\"}",
            INSTANCE,
            x.getRequestURI().getPath(),
            policy,
            d.allowed(),
            executed,
            d.bypass(),
            d.unavailable(),
            d.unavailableReason()
        );
        byte[] b = body.getBytes(StandardCharsets.UTF_8);

        x.getResponseHeaders().set("Content-Type", "application/json");
        x.getResponseHeaders().set("X-Backend-Executed", Boolean.toString(executed));
        if (d.bypass()) {
            x.getResponseHeaders().set("X-RateLimit-Bypass", "true");
        }
        if (d.unavailable()) {
            x.getResponseHeaders().set("X-RateLimit-Unavailable", "true");
            x.getResponseHeaders().set("X-RateLimit-Unavailable-Reason", d.unavailableReason());
        }

        x.sendResponseHeaders(d.status(), b.length);
        x.getResponseBody().write(b);
        x.close();
    }
}
