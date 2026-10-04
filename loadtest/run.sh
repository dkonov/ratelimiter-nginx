#!/usr/bin/env sh
set -eu

docker compose --profile load run --rm \
  -e DURATION="${DURATION:-30s}" \
  -e JAVA_ORDER_RPS="${JAVA_ORDER_RPS:-120}" \
  -e JAVA_COMMON_RPS="${JAVA_COMMON_RPS:-180}" \
  -e GO_ORDER_RPS="${GO_ORDER_RPS:-120}" \
  -e GO_COMMON_RPS="${GO_COMMON_RPS:-180}" \
  -e PYTHON_ORDER_RPS="${PYTHON_ORDER_RPS:-120}" \
  -e PYTHON_SEARCH_RPS="${PYTHON_SEARCH_RPS:-30}" \
  k6 run /scripts/load.js
