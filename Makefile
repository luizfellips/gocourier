.PHONY: test-unit test-integration test-idempotency test-security test-concurrency test-failure test-chaos test-replay test-observability test-all coverage ci-load

test-unit:
	go test -race -count=1 ./internal/... ./pkg/...

test-integration:
	go test -tags=integration -timeout 20m -count=1 ./tests/integration/...

test-idempotency:
	go test -tags=integration -timeout 20m -count=1 ./tests/idempotency/...

test-security:
	go test -tags=security -timeout 15m -count=1 ./tests/security/...

test-concurrency:
	go test -tags=concurrency -race -timeout 30m -count=1 ./tests/concurrency/...

test-failure:
	go test -tags=failure -timeout 20m -count=1 ./tests/failure/...

test-chaos:
	RUN_CHAOS=1 go test -tags=chaos -timeout 20m -count=1 ./tests/chaos/...

test-replay:
	go test -tags=integration -timeout 30m -count=1 ./tests/replay/...

test-observability:
	go test -tags=integration -timeout 15m -count=1 ./tests/observability/...

test-all: test-unit test-integration test-idempotency test-security test-replay test-observability

coverage:
	go test -coverprofile=coverage.out ./internal/... ./pkg/...
	go tool cover -func=coverage.out

ci-load:
	k6 run tests/performance/k6/ingest_load.js
