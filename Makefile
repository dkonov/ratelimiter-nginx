.PHONY: up down logs test-work test-slow stats limiter-logs

up:
	docker compose up -d --build

down:
	docker compose down --remove-orphans

logs:
	docker compose logs -f nginx1 nginx2 backend1 backend2 backend3

limiter-logs:
	docker compose logs -f nginx2

test-work:
	docker compose --profile load run --rm -e RATE=1200 -e DURATION=30s k6 run /scripts/work.js

test-slow:
	docker compose --profile load run --rm -e RATE=120 -e DURATION=30s k6 run /scripts/slow.js

stats:
	@for b in backend1 backend2 backend3; do \
		echo "=== $$b ==="; \
		docker compose exec -T $$b wget -qO- http://127.0.0.1:8080/debug/stats; echo; \
	done
