# Load tests (k6)

Scenarios for Phase B observability validation. Run against the docker-compose stack (web proxy on port 3000).

For interactive runs from the browser, use the ops dashboard **Load Test Panel** (see [tests/README.md](../README.md#dashboard-load-test-panel) for scenarios and configuration).

## Prerequisites

- [k6](https://k6.io/docs/get-started/installation/) installed
- Stack running: `cd deploy && docker compose up --build`

## Scenarios

| Script | Purpose |
|--------|---------|
| `normal.js` | Steady 200 req/s for 2 minutes |
| `burst.js` | Spike to 2000 req/s then ramp down |
| `retry_storm.js` | All recipients `fail-transient@...` |
| `duplicate.js` | Same idempotency key repeated |
| `concurrent.js` | 100 VUs with unique keys |

## Commands

```bash
k6 run tests/load/normal.js
k6 run tests/load/burst.js
k6 run tests/load/retry_storm.js
k6 run tests/load/duplicate.js
k6 run tests/load/concurrent.js
```

Environment overrides:

```bash
API_BASE_URL=http://localhost:3000 API_KEY=dev-api-key k6 run tests/load/normal.js
```

## Observability checks

After a load run:

1. Prometheus: `notifications_received_total` increases
2. Grafana worker dashboard: retry/DLQ panels move with failure scenarios
3. Tempo: search traces by `delivery_id` from dashboard

See also `tests/performance/k6/ingest_load.js` for sustained SLO baseline.
