import json, os
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
            self.send_response(204); self.end_headers(); return
        policy = POLICIES.get(self.path)
        if not policy:
            self.send_response(404); self.end_headers(); return
        allowed, status = RL.allow(policy)
        body = json.dumps({"backend": INSTANCE, "endpoint": self.path, "policy": "demo:"+policy, "allowed": allowed}).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers(); self.wfile.write(body)
    def log_message(self, fmt, *args): pass

ThreadingHTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
