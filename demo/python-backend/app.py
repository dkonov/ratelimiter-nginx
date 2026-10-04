import json
import os
import random
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from ratelimiter import FailMode, Outcome, RateLimiter

BACKEND_NAME = os.getenv("BACKEND_NAME", "python-backend")
LIMITER_KEY = os.getenv("RATE_LIMIT_KEY", "shared-global")

limiter = RateLimiter(
    os.getenv("RATE_LIMITER_URL", "http://nginx2:8080"),
    os.getenv("SERVICE_NAME", "demo"),
    fail_mode=FailMode(os.getenv("RATE_LIMITER_FAIL_MODE", "open")),
)


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(204)
            self.end_headers()
            return
        if self.path == "/metrics":
            s = limiter.stats()
            body = "".join(f"ratelimiter_{k}_total {v}\n" for k, v in s.items()).encode()
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; version=0.0.4")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if self.path != "/python/report":
            self._json(404, {"error": "not found"})
            return

        decision = limiter.allow(LIMITER_KEY)
        if decision.outcome is Outcome.DENIED:
            self._json(429, {"backend": BACKEND_NAME, "decision": "denied", "key": LIMITER_KEY})
            return
        if decision.outcome is Outcome.CONFIG_ERROR:
            self._json(500, {"backend": BACKEND_NAME, "error": decision.error})
            return
        if decision.outcome is Outcome.UNAVAILABLE:
            self._json(503, {"backend": BACKEND_NAME, "error": decision.error})
            return

        time.sleep(random.uniform(0.02, 0.08))
        headers = {"X-RateLimit-Bypass": "true"} if decision.outcome is Outcome.BYPASSED else {}
        self._json(200, {
            "backend": BACKEND_NAME,
            "language": "python",
            "endpoint": "/python/report",
            "rate_limit_key": LIMITER_KEY,
        }, headers)

    def log_message(self, fmt, *args):
        print(f"{BACKEND_NAME} - {fmt % args}")

    def _json(self, status, payload, extra_headers=None):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        for k, v in (extra_headers or {}).items():
            self.send_header(k, v)
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    server = ThreadingHTTPServer(("0.0.0.0", 8080), Handler)
    print(f"{BACKEND_NAME} listening on :8080, limiter-key={LIMITER_KEY}")
    server.serve_forever()
