#!/usr/bin/env bash
set -euo pipefail

TEST="${1:-slow}"
RATE="${2:-300}"
DURATION="${3:-30s}"
BACKENDS=(backend1 backend2 backend3)

case "$TEST" in
  slow|work) ;;
  *)
    echo "Usage: $0 [slow|work] [rate] [duration]" >&2
    echo "Example: $0 slow 300 30s" >&2
    exit 2
    ;;
esac

metric_value() {
  local service="$1"
  local metric="$2"

  docker compose exec -T "$service" \
    wget -qO- http://127.0.0.1:8080/metrics 2>/dev/null \
    | awk -v m="$metric" '$1 == m { print $2; found=1 } END { if (!found) print 0 }'
}

total_metric() {
  local metric="$1"
  local total=0
  local backend value

  for backend in "${BACKENDS[@]}"; do
    value="$(metric_value "$backend" "$metric")"
    total=$((total + value))
  done

  echo "$total"
}

capture() {
  local prefix="$1"
  eval "${prefix}_allowed=$(total_metric ratelimiter_allowed_total)"
  eval "${prefix}_denied=$(total_metric ratelimiter_denied_total)"
  eval "${prefix}_bypass=$(total_metric ratelimiter_bypass_total)"
  eval "${prefix}_unavailable=$(total_metric ratelimiter_unavailable_total)"
  eval "${prefix}_config_error=$(total_metric ratelimiter_config_error_total)"
}

echo "=== BEFORE ==="
bash scripts/metrics.sh once
capture before

echo
echo "=== K6: test=$TEST rate=$RATE/s duration=$DURATION ==="
docker compose --profile load run --rm \
  -e RATE="$RATE" \
  -e DURATION="$DURATION" \
  k6 run "/scripts/${TEST}.js"

capture after

echo
echo "=== AFTER ==="
bash scripts/metrics.sh once

allowed=$((after_allowed - before_allowed))
denied=$((after_denied - before_denied))
bypass=$((after_bypass - before_bypass))
unavailable=$((after_unavailable - before_unavailable))
config_error=$((after_config_error - before_config_error))
decisions=$((allowed + denied))

echo
echo "=== DELTA FOR THIS RUN ==="
printf '%-18s %12d\n' 'allowed (200)' "$allowed"
printf '%-18s %12d\n' 'denied (429)' "$denied"
printf '%-18s %12d\n' 'bypass' "$bypass"
printf '%-18s %12d\n' 'unavailable' "$unavailable"
printf '%-18s %12d\n' 'config_error' "$config_error"
printf '%-18s %12d\n' 'decisions' "$decisions"

if (( decisions > 0 )); then
  awk -v a="$allowed" -v d="$denied" 'BEGIN {
    printf "allowed_rate       %11.2f%%\n", (a*100)/(a+d)
    printf "denied_rate        %11.2f%%\n", (d*100)/(a+d)
  }'
fi
