# Test Commands

Reference for every command used to run tests in this project. Run all commands from the **repository root** unless noted otherwise.

## Windows (PowerShell)

`make` is not available by default on Windows. Use the `go test` commands below, or the helper script:

```powershell
.\scripts\test.ps1 unit
.\scripts\test.ps1 coverage
.\scripts\test.ps1 integration
.\scripts\test.ps1 all
```

**Quote paths that contain dots** — PowerShell splits `-coverprofile=coverage.out` into two arguments, so Go tries to load a package named `.out`:

```powershell
# Wrong — fails with "no required module provides package .out"
go test -coverprofile=coverage.out ./internal/... ./pkg/...

# Correct
go test -coverprofile="coverage.out" ./internal/... ./pkg/...
go tool cover -func="coverage.out"
```

**Race detector** often needs CGO on Windows. If you see `-race requires cgo`, omit `-race`:

```powershell
go test -count=1 ./internal/... ./pkg/...
```

Set environment variables in PowerShell:

```powershell
$env:RUN_CHAOS = "1"
$env:RUN_LARGE_CONCURRENCY = "1"
```

## Prerequisites

| Tool | Required for |
|------|----------------|
| Go 1.22+ | All commands |
| Docker | Integration, idempotency, replay, security, observability, failure, concurrency, chaos |
| Make | Makefile targets (optional on Windows — use the equivalent `go test` commands below) |
| k6 | `make ci-load` / load tests |
| vegeta | Stress tests (optional) |

Integration tests start disposable **Postgres** and **NATS** containers automatically. You do not need them installed locally.

---

## Makefile targets

These are the recommended entry points. Defined in [`Makefile`](../Makefile).

| Command | What it does |
|---------|----------------|
| `make test-unit` | Runs unit tests under `internal/` and `pkg/` with the race detector. Fast, no Docker. |
| `make test-integration` | Runs end-to-end tests in `tests/integration/` (lifecycle, retry, scheduler, replay, outbox recovery). Requires Docker. |
| `make test-idempotency` | Runs duplicate-ingest and concurrent same-key tests in `tests/idempotency/`. Requires Docker. |
| `make test-security` | Runs auth, validation, and injection tests in `tests/security/`. Requires Docker. |
| `make test-replay` | Runs DLQ → replay → success batch tests in `tests/replay/`. Requires Docker. |
| `make test-observability` | Runs audit-trail checks in `tests/observability/`. Requires Docker. |
| `make test-failure` | Runs fault-injection tests (worker crash, broker outage, slow provider) in `tests/failure/`. Requires Docker. |
| `make test-concurrency` | Runs concurrency tests in `tests/concurrency/` with the race detector. Requires Docker. Runs all tests in the package (including large-scale tests gated by env vars). |
| `make test-chaos` | Runs chaos experiments in `tests/chaos/` with `RUN_CHAOS=1`. Requires Docker. Intended for staging, not PR CI. |
| `make test-all` | Runs, in order: `test-unit`, `test-integration`, `test-idempotency`, `test-security`, `test-replay`, `test-observability`. Does **not** include failure, concurrency, or chaos. |
| `make coverage` | Runs unit tests with coverage profiling and prints a per-function summary. |
| `make ci-load` | Runs the k6 load script against a **running** API (`tests/performance/k6/ingest_load.js`). Requires k6 and a live stack. |

### Examples

```bash
# Daily development — fast feedback
make test-unit

# Before opening a PR — full CI-equivalent suite (Docker required)
make test-all

# Coverage report for domain and pkg code
make coverage
```

---

## Equivalent `go test` commands

Use these when Make is unavailable (e.g. Windows without Make) or when you need finer control.

### Unit tests

```bash
go test -count=1 ./internal/... ./pkg/...
```

Runs domain, application, and utility unit tests. No build tags, no Docker.

With race detector (Linux/macOS with CGO enabled):

```bash
go test -race -count=1 ./internal/... ./pkg/...
```

Skip slow tests during local iteration:

```bash
go test -short ./...
```

Integration and Docker-backed tests call `t.Skip` when `-short` is set.

### Coverage

```bash
go test -coverprofile=coverage.out ./internal/... ./pkg/...
go tool cover -func=coverage.out
```

PowerShell — **quote the profile path** (see [Windows section](#windows-powershell)):

```powershell
go test -coverprofile="coverage.out" ./internal/... ./pkg/...
go tool cover -func="coverage.out"
```

Open an HTML coverage report:

```bash
go tool cover -html=coverage.out
```

### Integration suite

Single command matching CI integration job:

```bash
go test -tags=integration -timeout 20m -count=1 \
  ./tests/integration/... \
  ./tests/idempotency/... \
  ./tests/replay/... \
  ./tests/observability/...
```

Individual packages:

```bash
go test -tags=integration -timeout 20m -count=1 ./tests/integration/...
go test -tags=integration -timeout 20m -count=1 ./tests/idempotency/...
go test -tags=integration -timeout 30m -count=1 ./tests/replay/...
go test -tags=integration -timeout 15m -count=1 ./tests/observability/...
```

### Security

```bash
go test -tags=security -timeout 15m -count=1 ./tests/security/...
```

### Concurrency

Default — runs `TestConcurrent100` and skips large tests unless env var is set:

```bash
go test -tags=concurrency -timeout 30m -count=1 ./tests/concurrency/...
```

Run only the 100-notification test:

```bash
go test -tags=concurrency -timeout 30m -count=1 -run TestConcurrent100 ./tests/concurrency/...
```

Large-scale tests (1K / 10K) — opt in explicitly:

```bash
RUN_LARGE_CONCURRENCY=1 go test -tags=concurrency -timeout 30m -count=1 -run TestConcurrent1000 ./tests/concurrency/...
RUN_LARGE_CONCURRENCY=1 go test -tags=concurrency -timeout 30m -count=1 -run TestConcurrent10000 ./tests/concurrency/...
```

Write concurrency metrics JSON to a directory:

```bash
CONCURRENCY_METRICS_DIR=./metrics go test -tags=concurrency -run TestConcurrent100 ./tests/concurrency/...
```

### Failure injection

```bash
go test -tags=failure -timeout 20m -count=1 ./tests/failure/...
```

### Chaos (staging only)

```bash
# Linux / macOS
RUN_CHAOS=1 go test -tags=chaos -timeout 20m -count=1 ./tests/chaos/...

# PowerShell
$env:RUN_CHAOS="1"; go test -tags=chaos -timeout 20m -count=1 ./tests/chaos/...
```

Without `RUN_CHAOS=1`, chaos tests skip themselves.

### Run a single test

```bash
go test -tags=integration -count=1 -run TestFullLifecycle -v ./tests/integration/...
go test -count=1 -run TestDispatchSkipsSucceeded -v ./internal/application/dispatch/...
```

---

## Performance commands

These hit a **running** deployment — start the stack first:

```bash
cd deploy && docker compose up --build
```

### k6 load test

Via Makefile:

```bash
make ci-load
```

Direct:

```bash
k6 run tests/performance/k6/ingest_load.js
```

With custom target and API key:

```bash
API_BASE_URL=http://localhost:3000 API_KEY=dev-api-key k6 run tests/performance/k6/ingest_load.js
```

Default script stages: ramp to 100 VUs, hold at 500 VUs for 5 minutes, ramp down. See [performance/SLO.md](performance/SLO.md) for targets.

### Dashboard load test panel

Interactive bursts from the ops UI at http://localhost:3000 — **Load Test Panel** in the left sidebar. Scenarios include parallel burst, duplicate storms (sequential/parallel), random channel/priority, retry storm, failure mix, staggered burst, and all-channels sweep. Full configuration reference: [README.md](README.md#dashboard-load-test-panel).

### vegeta stress test

```bash
vegeta attack -duration=5m -rate=200 \
  -targets=tests/performance/vegeta/targets.txt | vegeta report
```

Higher load / find breaking point:

```bash
echo "POST http://localhost:3000/v1/notifications" | \
  vegeta attack -rate=0 -max-workers=500 -duration=10m -body @tests/performance/vegeta/targets.txt
```

---

## Environment variables

| Variable | Used by | Effect |
|----------|---------|--------|
| `RUN_CHAOS=1` | `tests/chaos/` | Enables chaos experiments; otherwise tests skip |
| `RUN_LARGE_CONCURRENCY=1` | `tests/concurrency/` | Enables 1K and 10K concurrency tests |
| `CONCURRENCY_METRICS_DIR` | `tests/concurrency/` | Directory for JSON metrics output (default: `.`) |
| `API_BASE_URL` | k6 script | API base URL (default: `http://localhost:3000`) |
| `API_KEY` | k6 script | Ingest API key (default: `dev-api-key`) |
| `CGO_ENABLED=1` | `go test -race` | Required for race detector on some platforms |

---

## Common flags

| Flag | Purpose |
|------|---------|
| `-count=1` | Disable test result caching; always re-run |
| `-v` | Verbose output — print each test name |
| `-short` | Skip slow / Docker-backed tests |
| `-race` | Enable race detector (requires CGO) |
| `-timeout 20m` | Fail if package runs longer than limit (needed for Docker suites) |
| `-tags integration` | Include integration-tagged test files |
| `-run TestName` | Run only tests matching regex |
| `-coverprofile=coverage.out` | Write coverage data to file |

---

## CI commands

GitHub Actions (`.github/workflows/test.yml`) runs equivalent commands on every PR:

```bash
go test -race -count=1 -coverprofile=coverage.out ./internal/... ./pkg/...
go test -tags=integration -timeout 20m -count=1 ./tests/integration/... ./tests/idempotency/... ./tests/replay/... ./tests/observability/...
go test -tags=security -timeout 15m -count=1 ./tests/security/...
```

Nightly scheduled jobs additionally run:

```bash
go test -tags=concurrency -race -timeout 30m -count=1 -run TestConcurrent100 ./tests/concurrency/...
RUN_CHAOS=1 go test -tags=chaos -timeout 20m -count=1 ./tests/chaos/...
```

---

## Troubleshooting

| Problem | Likely cause | Fix |
|---------|--------------|-----|
| `Cannot connect to Docker` | Docker not running | Start Docker Desktop / daemon |
| Tests hang then timeout | Container pull or startup slow | Increase `-timeout`; check Docker resources |
| `-race requires cgo` | CGO disabled (common on Windows) | Run `go test` without `-race`, or set `CGO_ENABLED=1` with a C compiler |
| `RUN_CHAOS` tests skip | Env var not set | Export `RUN_CHAOS=1` before running |
| k6 connection refused | Stack not running | `docker compose up` in `deploy/` |
| `make: command not found` | Make not installed (Windows) | Use `go test` commands or `.\scripts\test.ps1` |
| `no required module provides package .out` | PowerShell split `coverage.out` | Use `-coverprofile="coverage.out"` (quoted) |

---

## See also

- [Testing guide (README.md)](README.md) — suite descriptions, testkit, mock provider conventions
- [Performance SLOs](performance/SLO.md) — latency and throughput targets
- [Root README](../README.md) — project setup and architecture
