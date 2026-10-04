#!/usr/bin/env bash
set -euo pipefail
RATE_PYTHON="${1:-10}"
RATE_JAVA="${2:-10}"
DURATION="${3:-30s}"
COMPOSE="docker compose -f docker-compose.multilang.yml"

$COMPOSE up -d --build

echo '=== BEFORE ==='
bash scripts/multilang-metrics.sh

echo
echo "=== LOAD: python=${RATE_PYTHON}/s java=${RATE_JAVA}/s, shared limit=10/s ==="
$COMPOSE --profile load run --rm \
  -e RATE_PYTHON="$RATE_PYTHON" \
  -e RATE_JAVA="$RATE_JAVA" \
  -e DURATION="$DURATION" \
  k6 run /scripts/multilang-shared.js

echo
echo '=== AFTER ==='
bash scripts/multilang-metrics.sh
