import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from ratelimiter import Outcome, RateLimiter

INSTANCE = os.getenv("BACKEND_INSTANCE", "python")
LIMITER = RateLimiter(os.getenv("RATE_LIMITER_URL", "http://nginx2:8080"), "demo")
POLICIES = {
    "/api/python/order": "order",
    "/api/python/search": "search",
}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        policy = POLICIES.get(self.path)
        if policy is None:
            self.send_error(404)
            return

        decision = LIMITER.check(policy)
        self.send_response(decision.status if decision.outcome != Outcome.ALLOWED else 200)
        self.send_header("X-Backend-Instance", INSTANCE)

        if decision.outcome == Outcome.BYPASSED:
            self.send_header("X-RateLimit-Bypass", "true")

        if decision.outcome in (Outcome.ALLOWED, Outcome.BYPASSED):
            body = json.dumps({"backend": INSTANCE, "endpoint": self.path}).encode()
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        else:
            self.end_headers()

    def log_message(self, *_):
        pass


ThreadingHTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
