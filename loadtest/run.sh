#!/usr/bin/env sh
set -eu

fail() {
  echo
  echo "ERROR: nginx1 did not become ready"
  echo "=== docker compose ps ==="
  docker compose ps -a || true
  echo
  echo "=== nginx1 logs ==="
  docker compose logs --tail=100 nginx1 || true
  echo
  echo "=== nginx2 logs ==="
  docker compose logs --tail=100 nginx2 || true
  exit 1
}

health_ok() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsS http://127.0.0.1:8080/health >/dev/null 2>&1
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O /dev/null http://127.0.0.1:8080/health >/dev/null 2>&1
  else
    docker compose exec -T nginx1 wget -q -O /dev/null http://127.0.0.1:8080/health >/dev/null 2>&1
  fi
}

echo "=== starting application stack ==="
docker compose up -d --build

ready=0
i=0
while [ "$i" -lt 60 ]; do
  if health_ok; then
    ready=1
    break
  fi
  i=$((i + 1))
  sleep 1
done

[ "$ready" -eq 1 ] || fail

echo
 echo "=== application is ready ==="
docker compose ps

echo
 echo "=== starting k6 ==="
docker run --rm --network host \
  -v "$(pwd)/loadtest:/scripts:ro" \
  -e TARGET_URL="http://127.0.0.1:8080" \
  -e DURATION="${DURATION:-30s}" \
  -e JAVA_ORDER_RPS="${JAVA_ORDER_RPS:-120}" \
  -e JAVA_COMMON_RPS="${JAVA_COMMON_RPS:-180}" \
  -e GO_ORDER_RPS="${GO_ORDER_RPS:-120}" \
  -e GO_COMMON_RPS="${GO_COMMON_RPS:-180}" \
  -e PYTHON_ORDER_RPS="${PYTHON_ORDER_RPS:-120}" \
  -e PYTHON_SEARCH_RPS="${PYTHON_SEARCH_RPS:-30}" \
  grafana/k6:latest run /scripts/load.js
