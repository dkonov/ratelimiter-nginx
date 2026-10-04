package demo;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import ratelimiter.RateLimiter;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.concurrent.Executors;
import java.util.concurrent.ThreadLocalRandom;

public final class Main {
    private static final String BACKEND_NAME = env("BACKEND_NAME", "java-backend");
    private static final String LIMITER_KEY = env("RATE_LIMIT_KEY", "shared-global");
    private static final RateLimiter LIMITER = new RateLimiter(
            env("RATE_LIMITER_URL", "http://nginx2:8080"),
            env("SERVICE_NAME", "demo"),
            "closed".equalsIgnoreCase(env("RATE_LIMITER_FAIL_MODE", "open"))
                    ? RateLimiter.FailMode.CLOSED : RateLimiter.FailMode.OPEN,
            Duration.ofMillis(20), Duration.ofMillis(50)
    );

    public static void main(String[] args) throws Exception {
        HttpServer server = HttpServer.create(new InetSocketAddress(8080), 0);
        server.createContext("/health", exchange -> send(exchange, 204, ""));
        server.createContext("/metrics", Main::metrics);
        server.createContext("/java/orders", Main::orders);
        server.setExecutor(Executors.newVirtualThreadPerTaskExecutor());
        System.out.printf("%s listening on :8080, limiter-key=%s%n", BACKEND_NAME, LIMITER_KEY);
        server.start();
    }

    private static void orders(HttpExchange exchange) throws IOException {
        if (!"GET".equals(exchange.getRequestMethod())) {
            send(exchange, 405, "{\"error\":\"method not allowed\"}");
            return;
        }
        RateLimiter.Decision decision = LIMITER.allow(LIMITER_KEY);
        switch (decision.outcome()) {
            case DENIED -> { send(exchange, 429, json("denied")); return; }
            case CONFIG_ERROR -> { send(exchange, 500, json("config_error")); return; }
            case UNAVAILABLE -> { send(exchange, 503, json("unavailable")); return; }
            case BYPASSED -> exchange.getResponseHeaders().set("X-RateLimit-Bypass", "true");
            default -> { }
        }

        try {
            Thread.sleep(ThreadLocalRandom.current().nextInt(20, 81));
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
        String body = "{\"backend\":\"" + BACKEND_NAME + "\",\"language\":\"java\"," +
                "\"endpoint\":\"/java/orders\",\"rate_limit_key\":\"" + LIMITER_KEY + "\"}";
        send(exchange, 200, body);
    }

    private static void metrics(HttpExchange exchange) throws IOException {
        RateLimiter.Stats s = LIMITER.stats();
        String body = "ratelimiter_allowed_total " + s.allowed() + "\n" +
                "ratelimiter_denied_total " + s.denied() + "\n" +
                "ratelimiter_bypassed_total " + s.bypassed() + "\n" +
                "ratelimiter_unavailable_total " + s.unavailable() + "\n" +
                "ratelimiter_config_error_total " + s.configError() + "\n";
        exchange.getResponseHeaders().set("Content-Type", "text/plain; version=0.0.4");
        send(exchange, 200, body);
    }

    private static String json(String decision) {
        return "{\"backend\":\"" + BACKEND_NAME + "\",\"decision\":\"" + decision +
                "\",\"key\":\"" + LIMITER_KEY + "\"}";
    }

    private static void send(HttpExchange exchange, int status, String body) throws IOException {
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        if (status != 204 && exchange.getResponseHeaders().getFirst("Content-Type") == null) {
            exchange.getResponseHeaders().set("Content-Type", "application/json");
        }
        exchange.sendResponseHeaders(status, status == 204 ? -1 : bytes.length);
        if (status != 204) exchange.getResponseBody().write(bytes);
        exchange.close();
    }

    private static String env(String name, String def) {
        String value = System.getenv(name);
        return value == null || value.isBlank() ? def : value;
    }
}
