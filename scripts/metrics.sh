#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-once}"
INTERVAL="${2:-1}"
BACKENDS=(backend1 backend2 backend3)

metric_value() {
  local service="$1"
  local metric="$2"

  docker compose exec -T "$service" \
    wget -qO- http://127.0.0.1:8080/metrics 2>/dev/null \
    | awk -v m="$metric" '$1 == m { print $2; found=1 } END { if (!found) print 0 }'
}

print_header() {
  printf '%-9s %12s %12s %12s %12s %12s %12s\n' \
    'backend' 'allowed' 'denied' 'bypass' 'unavailable' 'cfg_error' 'goroutines'
}

print_snapshot() {
  local total_allowed=0
  local total_denied=0
  local total_bypass=0
  local total_unavailable=0
  local total_cfg_error=0
  local total_goroutines=0

  echo
  echo "=== $(date '+%Y-%m-%d %H:%M:%S') ==="
  print_header

  for backend in "${BACKENDS[@]}"; do
    local allowed denied bypass unavailable cfg_error goroutines

    allowed="$(metric_value "$backend" ratelimiter_allowed_total)"
    denied="$(metric_value "$backend" ratelimiter_denied_total)"
    bypass="$(metric_value "$backend" ratelimiter_bypass_total)"
    unavailable="$(metric_value "$backend" ratelimiter_unavailable_total)"
    cfg_error="$(metric_value "$backend" ratelimiter_config_error_total)"
    goroutines="$(metric_value "$backend" backend_goroutines)"

    printf '%-9s %12d %12d %12d %12d %12d %12d\n' \
      "$backend" "$allowed" "$denied" "$bypass" "$unavailable" "$cfg_error" "$goroutines"

    total_allowed=$((total_allowed + allowed))
    total_denied=$((total_denied + denied))
    total_bypass=$((total_bypass + bypass))
    total_unavailable=$((total_unavailable + unavailable))
    total_cfg_error=$((total_cfg_error + cfg_error))
    total_goroutines=$((total_goroutines + goroutines))
  done

  printf '%-9s %12d %12d %12d %12d %12d %12d\n' \
    'TOTAL' "$total_allowed" "$total_denied" "$total_bypass" \
    "$total_unavailable" "$total_cfg_error" "$total_goroutines"

  local total_decisions=$((total_allowed + total_denied))
  if (( total_decisions > 0 )); then
    awk -v a="$total_allowed" -v d="$total_denied" 'BEGIN {
      printf "decision_rate: allowed=%.2f%% denied=%.2f%% total=%d\n", \
        (a*100)/(a+d), (d*100)/(a+d), a+d
    }'
  fi
}

case "$MODE" in
  once)
    print_snapshot
    ;;
  watch)
    while true; do
      clear 2>/dev/null || true
      print_snapshot
      sleep "$INTERVAL"
    done
    ;;
  *)
    echo "Usage: $0 [once|watch] [interval_seconds]" >&2
    echo "Examples:" >&2
    echo "  $0" >&2
    echo "  $0 once" >&2
    echo "  $0 watch 1" >&2
    exit 2
    ;;
esac
