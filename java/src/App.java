package demo;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.Executors;

public final class App {
    private static final String INSTANCE = System.getenv().getOrDefault("BACKEND_INSTANCE", "java");
    private static final RateLimiter LIMITER = new RateLimiter(
            System.getenv().getOrDefault("RATE_LIMITER_URL", "http://nginx2:8080"),
            "demo",
            true
    );

    public static void main(String[] args) throws Exception {
        HttpServer server = HttpServer.create(new InetSocketAddress(8080), 0);
        server.createContext("/api/java/order", protectedHandler("order", "/api/java/order"));
        server.createContext("/api/java/common", protectedHandler("common", "/api/java/common"));
        server.setExecutor(Executors.newFixedThreadPool(64));
        server.start();
    }

    private static HttpHandler protectedHandler(String policy, String endpoint) {
        return exchange -> {
            exchange.getResponseHeaders().set("X-Backend-Instance", INSTANCE);
            if (!exchange.getRequestMethod().equals("GET") || !exchange.getRequestURI().getPath().equals(endpoint)) {
                exchange.sendResponseHeaders(404, -1);
                exchange.close();
                return;
            }

            RateLimiter.Decision decision = LIMITER.check(policy);
            switch (decision.outcome()) {
                case ALLOWED -> respondOk(exchange, endpoint);
                case BYPASSED -> {
                    exchange.getResponseHeaders().set("X-RateLimit-Bypass", "true");
                    respondOk(exchange, endpoint);
                }
                default -> {
                    exchange.sendResponseHeaders(decision.status(), -1);
                    exchange.close();
                }
            }
        };
    }

    private static void respondOk(HttpExchange exchange, String endpoint) throws IOException {
        byte[] body = ("{\"backend\":\"" + INSTANCE + "\",\"endpoint\":\"" + endpoint + "\"}")
                .getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        exchange.sendResponseHeaders(200, body.length);
        exchange.getResponseBody().write(body);
        exchange.close();
    }
}
