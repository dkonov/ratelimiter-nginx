import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;
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

            String body = String.format(
                "{\"backend\":\"%s\",\"endpoint\":\"%s\",\"policy\":\"demo:%s\",\"allowed\":%s,\"bypass\":%s,\"unavailable\":%s}",
                INSTANCE,
                x.getRequestURI().getPath(),
                policy,
                d.allowed(),
                d.bypass(),
                d.unavailable()
            );
            byte[] b = body.getBytes(StandardCharsets.UTF_8);

            x.getResponseHeaders().set("Content-Type", "application/json");
            if (d.bypass()) {
                x.getResponseHeaders().set("X-RateLimit-Bypass", "true");
            }
            if (d.unavailable()) {
                x.getResponseHeaders().set("X-RateLimit-Unavailable", "true");
            }

            x.sendResponseHeaders(d.status(), b.length);
            x.getResponseBody().write(b);
            x.close();
        };
    }
}
