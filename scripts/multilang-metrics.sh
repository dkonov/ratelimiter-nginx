#!/usr/bin/env bash
set -euo pipefail
COMPOSE="docker compose -f docker-compose.multilang.yml"
SERVICES=(python1 python2 java1 java2)

metric() {
  local service="$1" name="$2"
  $COMPOSE exec -T "$service" sh -c 'wget -qO- http://127.0.0.1:8080/metrics' 2>/dev/null \
    | awk -v m="$name" '$1==m {print $2; found=1} END {if(!found) print 0}'
}

printf '%-10s %10s %10s %10s %12s %10s\n' service allowed denied bypass unavailable cfg_error
TA=0; TD=0; TB=0; TU=0; TC=0
for s in "${SERVICES[@]}"; do
  a=$(metric "$s" ratelimiter_allowed_total)
  d=$(metric "$s" ratelimiter_denied_total)
  b=$(metric "$s" ratelimiter_bypassed_total)
  u=$(metric "$s" ratelimiter_unavailable_total)
  c=$(metric "$s" ratelimiter_config_error_total)
  printf '%-10s %10d %10d %10d %12d %10d\n' "$s" "$a" "$d" "$b" "$u" "$c"
  TA=$((TA+a)); TD=$((TD+d)); TB=$((TB+b)); TU=$((TU+u)); TC=$((TC+c))
done
printf '%-10s %10d %10d %10d %12d %10d\n' TOTAL "$TA" "$TD" "$TB" "$TU" "$TC"
