from dataclasses import dataclass
from enum import Enum
from http.client import HTTPConnection, HTTPSConnection
from threading import local
from urllib.parse import urlparse


class Outcome(str, Enum):
    ALLOWED = "allowed"
    DENIED = "denied"
    BYPASSED = "bypassed"
    UNAVAILABLE = "unavailable"
    CONFIG_ERROR = "config_error"


@dataclass(frozen=True)
class Decision:
    outcome: Outcome
    status: int


class RateLimiter:
    def __init__(self, url: str, service: str, timeout: float = 0.05, fail_open: bool = True):
        parsed = urlparse(url)
        if parsed.scheme not in ("http", "https") or not parsed.hostname:
            raise ValueError("invalid rate limiter URL")
        self._scheme = parsed.scheme
        self._host = parsed.hostname
        self._port = parsed.port or (443 if parsed.scheme == "https" else 80)
        base = parsed.path.rstrip("/")
        self._path = (base + "/check") if base else "/check"
        self._service = service
        self._timeout = timeout
        self._fail_open = fail_open
        self._local = local()

    def check(self, endpoint: str) -> Decision:
        try:
            conn = self._connection()
            conn.request(
                "POST",
                self._path,
                headers={
                    "X-RateLimit-Service": self._service,
                    "X-RateLimit-Endpoint": endpoint,
                },
            )
            response = conn.getresponse()
            status = response.status
            response.read()

            if 200 <= status < 300:
                return Decision(Outcome.ALLOWED, status)
            if status == 429:
                return Decision(Outcome.DENIED, 429)
            if status == 404:
                return Decision(Outcome.CONFIG_ERROR, 500)
            return self._unavailable()
        except Exception:
            self._reset_connection()
            return self._unavailable()

    def _connection(self):
        conn = getattr(self._local, "conn", None)
        if conn is None:
            cls = HTTPSConnection if self._scheme == "https" else HTTPConnection
            conn = cls(self._host, self._port, timeout=self._timeout)
            self._local.conn = conn
        return conn

    def _reset_connection(self):
        conn = getattr(self._local, "conn", None)
        if conn is not None:
            try:
                conn.close()
            finally:
                self._local.conn = None

    def _unavailable(self) -> Decision:
        if self._fail_open:
            return Decision(Outcome.BYPASSED, 200)
        return Decision(Outcome.UNAVAILABLE, 503)
