import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from ratelimiter import RateLimiter

INSTANCE = os.getenv("INSTANCE_NAME", "python")
RL = RateLimiter(os.getenv("RATE_LIMITER_URL", "http://nginx2:8080"), "demo")
POLICIES = {
    "/api/python/order": "order",
    "/api/python/search": "search",
}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(204)
            self.end_headers()
            return

        policy = POLICIES.get(self.path)
        if not policy:
            self.send_response(404)
            self.end_headers()
            return

        allowed, status, bypass, unavailable, unavailable_reason = RL.allow(policy)

        # A denied limiter decision stops this request here. Protected business
        # work below this point is not executed.
        if not allowed:
            self.write_result(policy, status, allowed, False, bypass, unavailable, unavailable_reason)
            return

        # Protected business work would execute here.
        executed = True
        self.write_result(policy, status, allowed, executed, bypass, unavailable, unavailable_reason)

    def write_result(self, policy, status, allowed, executed, bypass, unavailable, unavailable_reason):
        body = json.dumps({
            "backend": INSTANCE,
            "endpoint": self.path,
            "policy": "demo:" + policy,
            "allowed": allowed,
            "executed": executed,
            "bypass": bypass,
            "unavailable": unavailable,
            "unavailable_reason": unavailable_reason,
        }).encode()

        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("X-Backend-Executed", "true" if executed else "false")
        if bypass:
            self.send_header("X-RateLimit-Bypass", "true")
        if unavailable:
            self.send_header("X-RateLimit-Unavailable", "true")
            self.send_header("X-RateLimit-Unavailable-Reason", unavailable_reason)
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        pass


ThreadingHTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
