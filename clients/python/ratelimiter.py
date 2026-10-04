from __future__ import annotations

import http.client
import ssl
from dataclasses import dataclass
from enum import Enum
from threading import Lock, local
from typing import Optional
from urllib.parse import urlsplit


class FailMode(str, Enum):
    OPEN = "open"
    CLOSED = "closed"


class Outcome(str, Enum):
    ALLOWED = "allowed"
    DENIED = "denied"
    BYPASSED = "bypassed"
    UNAVAILABLE = "unavailable"
    CONFIG_ERROR = "config_error"


@dataclass(frozen=True)
class Decision:
    outcome: Outcome
    status_code: int
    error: Optional[str] = None

    @property
    def allowed(self) -> bool:
        return self.outcome in (Outcome.ALLOWED, Outcome.BYPASSED)


class RateLimiter:
    """Small synchronous client for the central NGINX rate limiter.

    No third-party dependencies. Each calling thread keeps its own persistent
    HTTP connection, so normal threaded application servers get keep-alive
    without serializing all limiter checks through one socket.
    """

    def __init__(
        self,
        url: str,
        service: str,
        *,
        fail_mode: FailMode | str = FailMode.OPEN,
        timeout: float = 0.050,
    ) -> None:
        if not url.strip():
            raise ValueError("ratelimiter: url is required")
        if not service.strip():
            raise ValueError("ratelimiter: service is required")

        parsed = urlsplit(url.rstrip("/"))
        if parsed.scheme not in ("http", "https") or not parsed.hostname:
            raise ValueError("ratelimiter: invalid url")

        self._scheme = parsed.scheme
        self._host = parsed.hostname
        self._port = parsed.port or (443 if parsed.scheme == "https" else 80)
        base_path = parsed.path.rstrip("/")
        self._check_path = (base_path + "/check") or "/check"
        self._service = service.strip()
        self._fail_mode = FailMode(fail_mode)
        self._timeout = timeout
        self._local = local()
        self._lock = Lock()
        self._stats = {
            "allowed": 0,
            "denied": 0,
            "bypassed": 0,
            "unavailable": 0,
            "config_error": 0,
        }

    def allow(self, endpoint: str) -> Decision:
        endpoint = endpoint.strip()
        if not endpoint:
            self._inc("config_error")
            return Decision(Outcome.CONFIG_ERROR, 500, "ratelimiter: endpoint is empty")

        headers = {
            "X-RateLimit-Service": self._service,
            "X-RateLimit-Endpoint": endpoint,
            "Connection": "keep-alive",
            "Content-Length": "0",
        }

        try:
            conn = self._connection()
            conn.request("POST", self._check_path, body=None, headers=headers)
            response = conn.getresponse()
            status = response.status
            response.read()
        except (OSError, http.client.HTTPException) as exc:
            self._drop_connection()
            return self._on_unavailable(str(exc))

        if 200 <= status < 300:
            self._inc("allowed")
            return Decision(Outcome.ALLOWED, status)
        if status == 429:
            self._inc("denied")
            return Decision(Outcome.DENIED, status)
        if status == 404:
            self._inc("config_error")
            return Decision(Outcome.CONFIG_ERROR, 500, f"ratelimiter: unknown policy {self._service}:{endpoint}")
        return self._on_unavailable(f"ratelimiter: unexpected status {status}")

    def stats(self) -> dict[str, int]:
        with self._lock:
            return dict(self._stats)

    def _connection(self):
        conn = getattr(self._local, "connection", None)
        if conn is None:
            if self._scheme == "https":
                conn = http.client.HTTPSConnection(self._host, self._port, timeout=self._timeout, context=ssl.create_default_context())
            else:
                conn = http.client.HTTPConnection(self._host, self._port, timeout=self._timeout)
            self._local.connection = conn
        return conn

    def _drop_connection(self) -> None:
        conn = getattr(self._local, "connection", None)
        if conn is not None:
            try:
                conn.close()
            finally:
                self._local.connection = None

    def _inc(self, name: str) -> None:
        with self._lock:
            self._stats[name] += 1

    def _on_unavailable(self, error: str) -> Decision:
        self._inc("unavailable")
        if self._fail_mode is FailMode.OPEN:
            self._inc("bypassed")
            return Decision(Outcome.BYPASSED, 200, error)
        return Decision(Outcome.UNAVAILABLE, 503, error)
