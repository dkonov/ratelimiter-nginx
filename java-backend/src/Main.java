import com.sun.net.httpserver.*;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;

public class Main {
    static final String INSTANCE = System.getenv().getOrDefault("INSTANCE_NAME", "java");
    static final RateLimiter RL = new RateLimiter(System.getenv().getOrDefault("RATE_LIMITER_URL", "http://nginx2:8080"), "demo");

    public static void main(String[] args) throws Exception {
        HttpServer s = HttpServer.create(new InetSocketAddress(8080), 0);
        s.createContext("/api/java/order", handler("order"));
        s.createContext("/api/java/common", handler("common"));
        s.createContext("/health", x -> { x.sendResponseHeaders(204, -1); x.close(); });
        s.setExecutor(java.util.concurrent.Executors.newVirtualThreadPerTaskExecutor());
        s.start();
    }

    static HttpHandler handler(String policy) {
        return x -> {
            RateLimiter.Decision d = RL.allow(policy);
            String body = String.format("{\"backend\":\"%s\",\"endpoint\":\"%s\",\"policy\":\"demo:%s\",\"allowed\":%s}", INSTANCE, x.getRequestURI().getPath(), policy, d.allowed());
            byte[] b = body.getBytes(StandardCharsets.UTF_8);
            x.getResponseHeaders().set("Content-Type", "application/json");
            x.sendResponseHeaders(d.status(), b.length);
            x.getResponseBody().write(b); x.close();
        };
    }
}
