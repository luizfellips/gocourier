# Testing Guide

Comprehensive test suite for the event-driven notification platform. Tests validate correctness, idempotency, failure recovery, security, and operational readiness, not just happy paths.

## Prerequisites

| Requirement | Used by |
|-------------|---------|
| Go 1.22+ | All tests |
| Docker Desktop / Docker Engine | Integration, idempotency, replay, security, observability, failure, concurrency, chaos |
| [k6](https://k6.io/) | Load tests (`tests/performance/k6/`) |
| [vegeta](https://github.com/tsenart/vegeta) | Stress tests (`tests/performance/vegeta/`) |

Integration tests spin up **PostgreSQL 16** and **NATS JetStream 2.10** via [testcontainers-go](https://golang.testcontainers.org/). No local Postgres or NATS install required.

## Quick reference

See [commands.md](commands.md) for a full command reference (Makefile targets, `go test` equivalents, env vars, performance tools).

```bash
# From repo root
make test-unit          # fast, no Docker
make test-all           # unit + integration + idempotency + security + replay + observability
make coverage           # coverage report for internal/ + pkg/

# Tagged suites (Docker required unless noted)
make test-integration
make test-idempotency
make test-security
make test-replay
make test-observability
make test-failure
make test-concurrency   # 100 concurrent notifications
make test-chaos         # requires RUN_CHAOS=1

# Performance (staging stack must be running)
make ci-load            # k6 ingest load script
```

Skip slow suites during development:

```bash
go test -short ./...
```

## Directory layout

```
tests/
├── testkit/           # Shared harness: Postgres/NATS containers, stack wiring, fixtures
├── integration/     # End-to-end lifecycle, retry, scheduler, replay, outbox recovery
├── idempotency/     # Duplicate ingest + concurrent same-key tests
├── replay/          # DLQ → replay → success batch tests
├── concurrency/     # 100 / 1K / 10K parallel ingest (tiered by env var)
├── failure/         # Worker crash, broker outage, slow provider
├── chaos/           # Random worker termination, broker restart (staging only)
├── security/        # Auth, validation, injection, malformed payloads
├── observability/   # Audit trail completeness (Phase A — logs/metrics Phase B)
└── performance/
    ├── k6/          # HTTP load test scripts
    ├── vegeta/      # Sustained RPS targets
    └── SLO.md       # Latency / throughput targets
```

Unit tests live next to source code under `internal/` and `pkg/` (not under `tests/`).

## Build tags

Go build tags isolate suites so CI and local runs can target specific layers:

| Tag | Package(s) | Purpose |
|-----|------------|---------|
| *(none)* | `internal/...`, `pkg/...` | Unit tests — always run in CI |
| `integration` | `integration`, `idempotency`, `replay`, `observability` | Docker-backed E2E |
| `security` | `security` | HTTP auth and input validation |
| `concurrency` | `concurrency` | Parallel ingest at scale |
| `failure` | `failure` | Fault injection |
| `chaos` | `chaos` | Staging chaos experiments |

Example:

```bash
go test -tags=integration -timeout 20m -count=1 \
  ./tests/integration/... \
  ./tests/idempotency/... \
  ./tests/replay/... \
  ./tests/observability/...
```

## Test pyramid

```
                    ┌─────────────┐
                    │ chaos / DR  │  weekly, staging
                    ├─────────────┤
                    │ perf / load │  nightly
                    ├─────────────┤
                    │ concurrency │  nightly (100), weekly (1K+)
                    ├─────────────┤
                    │ integration │  every PR
                    ├─────────────┤
                    │    unit     │  every PR
                    └─────────────┘
```

## Suite descriptions

### Unit (`internal/`, `pkg/`)

Pure domain logic and application services with mocks.
Target **≥90% coverage** for `internal/domain` and `pkg/`.

| Area | Package | Examples |
|------|---------|----------|
| Event validation | `domain/notification` | schema version, required fields, key length |
| State machine | `domain/notification` | pending → queued → processing → terminal |
| Backoff | `pkg/retry` | exponential delay, cap, jitter |
| Circuit breaker | `pkg/circuitbreaker` | open / half-open / closed transitions |
| Ingest dedup | `application/ingest` | duplicate response, store conflict |
| Dispatch | `application/dispatch` | retry, DLQ, concurrent skip, dedup |
| Outbox | `application/outbox` | publish, skip already-published delivery |

```bash
go test -count=1 ./internal/... ./pkg/...
go test -coverprofile=coverage.out ./internal/domain/... ./pkg/...
go tool cover -func=coverage.out
```

### Integration (`tests/integration/`)

Full pipeline: **ingest → outbox → NATS → worker → provider → PostgreSQL**.

| Test | Validates |
|------|-----------|
| `TestFullLifecycle` | Happy path + sequential idempotency |
| `TestPermanentFailureMovesToDLQ` | Permanent provider error → DLQ |
| `TestTransientRetryThenSuccess` | Transient failure → retry → success |
| `TestSchedulerEnqueuesDueNotification` | Scheduled ingest → scheduler → outbox |
| `TestReplayFromDLQ` | DLQ delivery replayed to success |
| `TestOutboxRecoveryAfterRestart` | Pending outbox drained after flush |

### Idempotency (`tests/idempotency/`)

| Test | Validates |
|------|-----------|
| `TestSequentialDuplicateSubmits` | 100 duplicate ingests → one delivery |
| `TestConcurrentSameIdempotencyKey` | 10 goroutines, same key → one delivery ID |
| `TestSameKeyDifferentChannel` | Keys scoped per channel |
| `TestDispatchDedupOnRedelivery` | NATS redelivery → one provider send |

### Replay (`tests/replay/`)

Batch DLQ → replay → success for multiple deliveries. Uses shared `ingestPermanentFailure` helper.

### Security (`tests/security/`)

| Test | Validates |
|------|-----------|
| `TestUnauthorizedIngest` | 401 without API key |
| `TestMalformedJSON` | 400 on bad JSON |
| `TestOversizedIdempotencyKey` | 400 when key > 256 chars |
| `TestSQLInjectionInMetadata` | Parameterized queries; no table drop |

### Observability (`tests/observability/`)

Phase A: audit events written to PostgreSQL, terminal states recorded.

Phase B: OpenTelemetry + Prometheus metrics and trace propagation.

| Test | Validates |
|------|-----------|
| `TestAuditTrailCompleteness` | Audit rows after ingest + dispatch |
| `TestDeliveryTerminalStateRecorded` | Terminal status in DB |
| `TestMetricsEndpoint` | `GET /metrics` returns `notifications_received_total` |
| `TestPrometheusMetricAfterIngest` | Counter increments after ingest |
| `TestTraceContextInLogs` | W3C `traceparent` inject/extract |

```bash
go test -tags=integration -count=1 ./tests/observability/...
go test -count=1 ./pkg/telemetry/...
bash tests/observability/validate_telemetry.sh   # after docker compose up
```

See [docs/observability.md](../docs/observability.md) for the full validation checklist.

### Concurrency (`tests/concurrency/`)

| Test | Scale | Gate |
|------|-------|------|
| `TestConcurrent100` | 100 parallel ingests | Default in nightly CI |
| `TestConcurrent1000` | 1,000 | `RUN_LARGE_CONCURRENCY=1` |
| `TestConcurrent10000` | 10,000 | `RUN_LARGE_CONCURRENCY=1` |

Metrics JSON written to `CONCURRENCY_METRICS_DIR` (default: current directory).

```bash
RUN_LARGE_CONCURRENCY=1 go test -tags=concurrency -timeout 30m -run TestConcurrent1000 ./tests/concurrency/...
```

### Failure & chaos

```bash
make test-failure    # worker crash, broker outage, slow provider
RUN_CHAOS=1 make test-chaos   # staging only — worker SIGTERM, broker restart
```

Chaos and large-scale concurrency tests are **not** PR gates; they run on schedule against staging.

## testkit harness

`tests/testkit/` wires a disposable stack for integration-style tests:

```go
stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())
cancel := stack.RunBackground(ctx, 2, false) // outbox + worker
defer cancel()

resp, err := stack.Ingest.Ingest(ctx, testkit.SampleIngestRequest("my-test"))
stack.FlushAndDispatch(ctx, t, resp.DeliveryID)
```

| Helper | Purpose |
|--------|---------|
| `StartPostgres` | Postgres 16 + migrations |
| `StartNATS` | NATS JetStream |
| `StartStack` | Full wired services (ingest, dispatch, outbox, replay, scheduler) |
| `SampleIngestRequest` | Valid request with UUID idempotency key |
| `NewHTTPClient` | API client for httptest / live HTTP |
| `NewFixedClock` | Deterministic time for scheduler tests |

Configure via `StackConfig` (NATS redelivery, max attempts, circuit breaker threshold, worker concurrency).

## Mock provider failure injection

The mock provider (`internal/adapters/providers/mock`) simulates provider behavior via recipient address substrings:

| Recipient contains | Result |
|--------------------|--------|
| `fail-permanent` | Immediate DLQ (`ErrPermanent`) |
| `fail-transient` | Retry via NATS NAK (`ErrTransient`) |
| `fail-circuit` | Transient error (circuit breaker testing) |
| *(otherwise)* | Success |

Example:

```json
{"address": "fail-transient@example.com"}
```

The ops dashboard **Send Test Form** uses the same conventions for manual testing.

## Dashboard load test panel

The ops dashboard **Load Test Panel** (sidebar, below Send Test Form) drives the same ingest API as the k6 scripts for interactive spikes and failure scenarios. Open [http://localhost:3000](http://localhost:3000) with the stack running.

### Configuration

| Setting | Purpose |
|---------|---------|
| Recipient + presets | Happy / transient / permanent / circuit failure injection (same substrings as mock provider) |
| Burst count | Requests for parallel and mixed scenarios (1–500) |
| Duplicate count | Requests for duplicate-storm scenarios (1–500) |
| **Advanced → Channel mode** | Fixed channel, random from pool, or round-robin sweep |
| **Advanced → Channels in pool** | Which channels participate in random / sweep modes |
| **Advanced → Priority mode** | Fixed or random low / normal / high |
| **Advanced → Concurrency** | Max in-flight requests for parallel scenarios (1–100) |
| **Advanced → Stagger (ms)** | Delay between sequential submits (staggered burst) |

### Scenarios

| Scenario | Maps to k6 / behavior |
|----------|------------------------|
| Parallel burst | `concurrent.js` — unique keys, configurable concurrency |
| Duplicate storm (sequential) | `duplicate.js` — same key, one after another |
| Duplicate storm (parallel) | Same key, concurrent — idempotency race |
| Random channel burst | Per-request channel from selected pool |
| Random priority burst | Per-request priority |
| Mixed chaos | Random channel + priority |
| Retry storm | `retry_storm.js` — all `fail-transient@…` |
| Failure mix | Random happy / transient / permanent recipients |
| Staggered burst | Sequential with stagger delay |
| All-channels sweep | Round-robin across selected channels |

After a run, watch **Stats Bar**, **Deliveries**, Grafana worker panels, and Tempo traces. For sustained SLO baselines use k6 (`tests/load/`, `tests/performance/k6/`).

## Performance testing

Start the platform first:

```bash
cd deploy && docker compose up --build
```

Load test (k6):

```bash
k6 run tests/performance/k6/ingest_load.js
# override target
API_BASE_URL=http://localhost:3000 API_KEY=dev-api-key k6 run tests/performance/k6/ingest_load.js
```

Stress test (vegeta):

```bash
vegeta attack -duration=5m -rate=200 \
  -targets=tests/performance/vegeta/targets.txt | vegeta report
```

Initial SLO targets are documented in [performance/SLO.md](performance/SLO.md):

| Metric | Load target |
|--------|-------------|
| Ingest p95 | < 150 ms |
| Ingest p99 | < 300 ms |
| Sustained throughput | 500 req/s |
| Error rate | < 0.1 % |

## CI pipeline

[`.github/workflows/test.yml`](../.github/workflows/test.yml):

| Job | Trigger | Command |
|-----|---------|---------|
| `unit` | Every PR | `go test -race ./internal/... ./pkg/...` |
| `integration` | Every PR | `-tags=integration` suites |
| `security` | Every PR | `-tags=security` |
| `concurrency` | Nightly schedule | `TestConcurrent100` with `-race` |
| `weekly-chaos` | Nightly schedule | `-tags=chaos` with `RUN_CHAOS=1` |
| `nightly-load` | Nightly schedule | k6 against staging (placeholder) |

### Release gates

- Unit + integration + security green on PR
- Domain coverage ≥ 90 % (`internal/domain`)
- No race detector failures (Linux CI)
- Nightly perf within 20 % of baseline (when staging k6 is wired)

## Writing new tests

1. **Unit** — add `*_test.go` beside the package under test; use table-driven tests and testify assertions.
2. **Integration** — add `//go:build integration`, use `testkit.StartStack`, guard with `testing.Short()`.
3. **HTTP** — use `httptest` or `testkit.NewHTTPClient`.
4. **Failure scenarios** — use mock provider recipient conventions before adding Toxiproxy.
5. **Never** commit secrets; use `dev-api-key` / testcontainers defaults.

## Related docs

- [Architecture ADRs](../docs/adr/)
- [DLQ inspection runbook](../docs/runbooks/)
- [Replay runbook](../docs/runbooks/)
- [Performance SLOs](performance/SLO.md)
