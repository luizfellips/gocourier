# Observability (Phase B)

OpenTelemetry tracing, Prometheus metrics, Grafana dashboards, and Tempo trace storage for the notification platform.

## Architecture

```
Client (web/k6) ──HTTP traceparent──► API ──OTLP──► OTEL Collector ──► Tempo
                                      │                    │
                                      ├── /metrics ────────┼──► Prometheus ──► Grafana
Worker / Scheduler ──OTLP + sidecar metrics (:9091/:9092)
Outbox ── traceparent in NATS headers ──► Worker (extract) ── linked trace
```

## Quick start

```bash
cd deploy
docker compose up --build
```

| URL | Service |
|-----|---------|
| http://localhost:3000 | Ops dashboard (web) |
| http://localhost:3001 | Grafana (admin/admin) |
| http://localhost:9090 | Prometheus |
| http://localhost:3200 | Tempo API |

## Validation checklist

1. All containers healthy: `docker compose ps`
2. Prometheus targets UP: http://localhost:9090/targets (api, worker, scheduler, nats-exporter)
3. Grafana dashboards: **Notification Platform** folder — API, Worker, Queue, System
4. Send notification via web → Tempo search by `delivery_id` shows api + worker spans
5. **Load Test Panel** (retry storm / failure mix) → worker retry and DLQ metrics move
6. `notifications_received_total` increases in Prometheus
7. `fail-transient` preset → `notifications_retried_total` increases
8. `fail-permanent` preset → `notifications_dlq_total` increases
9. Alert rules loaded: http://localhost:9090/rules
10. Integration tests: `go test -tags=integration ./tests/observability/...`

Post-compose script (bash):

```bash
bash tests/observability/validate_telemetry.sh
```

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | (empty) | OTLP gRPC endpoint; set to enable export |
| `OTEL_SERVICE_NAME` | binary name | Service name in traces |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | Trace sampler |
| `OTEL_SDK_DISABLED` | — | Set `true` to disable OTLP |
| `METRICS_ADDR` | `:9091` worker, `:9092` scheduler | Sidecar metrics port |

API exposes `/metrics` on `HTTP_ADDR` (:8080).

## Learning notes

### Problem: no cross-service causality

Log-only correlation IDs break across async boundaries (outbox → NATS → worker). W3C `traceparent` propagation plus OTLP export gives a single trace in Tempo for ingest through dispatch.

### HTTP instrumentation

`otelhttp` wraps the API handler for automatic span creation. Business counters (`notifications_received_total`, etc.) complement RED metrics for SLO dashboards.

### Async trace propagation

Outbox publish injects `traceparent` into NATS headers. Worker consume extracts context before dispatch so Tempo shows linked spans.

### Sidecar metrics

Worker and scheduler are not HTTP servers; they expose `/metrics` on dedicated ports for Prometheus scrape annotations (Kubernetes pattern).

### Trade-offs at scale

- No tail sampling in dev — all traces exported (costly at high volume)
- Dual path: direct Prometheus scrape + OTLP traces (metrics don't depend on collector uptime)
- JetStream lag is approximate via broker metrics; native consumer lag polling is future work

## Related docs

- [tests/README.md](../tests/README.md) — Phase B test commands
- [tests/load/README.md](../tests/load/README.md) — k6 load scenarios
