# Glossary

[← Analysis index](README.md)

Important technical terms, acronyms, services, and domain-specific concepts.

| Term | Definition |
|------|------------|
| **API (binary)** | `cmd/api` — HTTP server, ingest, replay, dashboard API, outbox publisher |
| **Audit event** | Append-only row in `audit_events` recording domain occurrences (received, succeeded, DLQ, replayed) |
| **Channel** | Notification medium: `email`, `sms`, `push`, or `webhook` |
| **ChannelProvider** | Port interface implemented by mock (and future real) dispatch adapters |
| **Circuit breaker** | Per-channel failure gate that opens after threshold failures in a sliding window |
| **Delivery** | A single notification request tracked end-to-end with UUID `delivery_id` |
| **DLQ** | Dead Letter Queue — terminal failure path; NATS stream `DLQ` with subjects `dlq.{channel}` |
| **Effectively-once** | Duplicate requests and redeliveries do not cause duplicate provider sends; not mathematical exactly-once |
| **Idempotency key** | Client-supplied deduplication token, unique per `(key, channel)` within TTL |
| **Ingest** | Application service accepting and persisting new notification requests |
| **JetStream** | NATS persistence layer providing streams, consumers, and ack semantics |
| **Modular monolith** | Single codebase/deployment unit with enforced internal module boundaries |
| **NATS** | Message broker used for work queue and DLQ publish |
| **NAK** | Negative acknowledgment — tells JetStream to redeliver the message |
| **Outbox** | Table + publisher pattern bridging PostgreSQL transactions to NATS |
| **Outbox publisher** | Background loop in API binary flushing `outbox` rows to NATS |
| **Port** | Go interface in `internal/ports` defining adapter contracts |
| **Priority** | Routing hint: `low`, `normal`, or `high` — encoded in NATS subject |
| **Provider** | External or mock service that actually sends the notification |
| **Replay** | Operator action re-queuing a DLQ delivery for another dispatch attempt |
| **Scheduler** | `cmd/scheduler` — processes due `scheduled_notifications` into the outbox |
| **Subject** | NATS address string, e.g. `notifications.email.normal` |
| **System of record** | PostgreSQL — authoritative source for delivery state and history |
| **Transactional outbox** | Insert outbox row in same DB transaction as business data |
| **Transient error** | Retryable failure (provider blip, circuit open) — maps to NATS NAK |
| **Permanent error** | Non-retryable failure (validation) — immediate DLQ |
| **Worker** | `cmd/worker` — NATS consumer running dispatch service |
| **W3C traceparent** | Standard HTTP header format for distributed trace propagation |
| **OTEL / OpenTelemetry** | Observability framework for traces and metrics export |
| **Tempo** | Grafana trace backend storing OTEL spans |
| **Prometheus** | Metrics TSDB scraped from `/metrics` endpoints |
| **testkit** | Test harness spinning up Postgres + NATS and wiring production-like stack |
| **ADR** | Architecture Decision Record — documented in [docs/adr/](../adr/) |
| **SLO** | Service Level Objective — latency/throughput targets in [tests/performance/SLO.md](../../tests/performance/SLO.md) |
| **Tenant ID** | Metadata field for multi-tenant scoping (default: `default`) |
| **Correlation ID** | Metadata field linking notification to upstream business transaction |

---

**Back to:** [Analysis index](README.md)
