import http.client
import socket
import threading
from urllib.parse import urlparse


class RateLimiter:
    def __init__(self, base_url, service="demo", timeout=0.05):
        u = urlparse(base_url)
        self.host = u.hostname
        self.port = u.port or 80
        self.service = service
        self.timeout = timeout
        self.local = threading.local()

    def _conn(self):
        c = getattr(self.local, "conn", None)
        if c is None:
            c = http.client.HTTPConnection(self.host, self.port, timeout=self.timeout)
            self.local.conn = c
        return c

    def allow(self, endpoint):
        try:
            c = self._conn()
            c.request("POST", "/check", headers={
                "X-RateLimit-Service": self.service,
                "X-RateLimit-Endpoint": endpoint,
            })
            r = c.getresponse()
            r.read()

            if 200 <= r.status < 300:
                return True, 200, False, False, ""
            if r.status == 429:
                return False, 429, False, False, ""
            if r.status == 404:
                return False, 500, False, False, ""
            if 500 <= r.status < 600:
                return True, 200, True, True, "5xx"

            return True, 200, True, True, "other"
        except (socket.timeout, TimeoutError):
            self.local.conn = None
            return True, 200, True, True, "timeout"
        except (ConnectionRefusedError, socket.gaierror):
            self.local.conn = None
            return True, 200, True, True, "connect_error"
        except (OSError, http.client.HTTPException):
            self.local.conn = None
            return True, 200, True, True, "io_error"
        except Exception:
            self.local.conn = None
            return True, 200, True, True, "other"
