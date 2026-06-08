# Gocourier

Notification delivery platform built in Go. Ingests events via HTTP, queues through PostgreSQL outbox + NATS JetStream, and dispatches to email, SMS, push, and webhook channels via pluggable providers.

## Quick Start

```bash
cd deploy
docker compose up --build
```

**Dashboard:** open [http://localhost:3000](http://localhost:3000) — shadcn + TanStack Query + Zustand wired to the Go API. Use **Send Test Form** for single notifications and **Load Test Panel** for burst / duplicate / mixed-channel scenarios (documented in [tests/README.md](tests/README.md#dashboard-load-test-panel)).

**Observability:** Grafana [http://localhost:3001](http://localhost:3001) (admin/admin), Prometheus [http://localhost:9090](http://localhost:9090). See [docs/observability.md](docs/observability.md).

**Dev frontend only** (API must be running on :8080):

```bash
cd web && npm install && npm run dev
```

Opens [http://localhost:5173](http://localhost:5173) with Vite proxy to the API. Default API key: `dev-api-key`.

Send a notification (or use the dashboard UI):

```bash
curl -X POST http://localhost:8080/v1/notifications \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dev-api-key" \
  -d '{
    "schema_version": "1.0",
    "idempotency_key": "test-1",
    "channel": "email",
    "priority": "normal",
    "recipient": {"address": "user@example.com"},
    "template": {"id": "welcome", "data": {}}
  }'
```

## Architecture

- **API** — HTTP ingest, idempotency, transactional outbox
- **Outbox publisher** — background publish to NATS JetStream
- **Worker** — dispatch with retry, circuit breaker, DLQ
- **Scheduler** — delayed notification processing
- **PostgreSQL** — system of record
- **NATS JetStream** — durable work queue
- **OpenTelemetry + Prometheus + Grafana + Tempo** — traces, metrics, dashboards (Phase B)

See [ANALYSIS.md](ANALYSIS.md) (or [docs/analysis/](docs/analysis/)) for the full system deep dive, [docs/adr/](docs/adr/) for architecture decisions, [docs/observability.md](docs/observability.md) for telemetry validation, and [docs/runbooks/](docs/runbooks/) for operations.

## Development

See [tests/commands.md](tests/commands.md) for all test commands and [tests/README.md](tests/README.md) for the full testing guide.

```bash
make test-unit          # unit tests (internal + pkg)
make test-all           # unit + integration suites
make coverage           # coverage report
```

Quick tagged runs (Docker required):

```bash
make test-integration
make test-idempotency
make test-security
make ci-load             # k6 — see tests/load/ and tests/performance/k6/ingest_load.js
```

## Binaries

| Command | Purpose |
|---------|---------|
| `cmd/api` | HTTP API + outbox publisher |
| `cmd/worker` | NATS consumer + dispatch |
| `cmd/scheduler` | Scheduled notification processor |
