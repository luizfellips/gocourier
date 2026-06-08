# Pain Points Addressed

[← Analysis index](README.md)

For each major pain point: what existed before, why common approaches fail, how this system improves the situation, and tradeoffs introduced.

---

## 1. Dual-write between database and message broker

**Problem before:** Accepting a notification in the DB and publishing to NATS in two steps means one can succeed and the other fail—events are lost or duplicated at the system boundary.

**Why common approaches fail:** "Publish first, then save" loses durability on DB failure. "Save first, then publish" loses queue visibility on broker failure. Two-phase commit across Postgres and NATS is impractical.

**How this system solves it:** **Transactional outbox pattern.** Ingest and outbox insert happen in a single PostgreSQL transaction. A background publisher drains the outbox independently.

**Tradeoffs:** Added latency (outbox poll interval, default 500ms); write amplification (extra outbox table); eventual consistency between DB commit and broker publish.

See also: [ADR-003](../adr/003-postgresql-outbox.md), [Technology Choices — Transactional outbox](technology-choices.md#transactional-outbox).

---

## 2. Duplicate notifications from retries and redelivery

**Problem before:** Clients retry HTTP on timeouts; NATS delivers at-least-once; workers may crash mid-processing—all can cause duplicate sends.

**Why common approaches fail:** In-memory dedup caches don't survive restarts. Broker dedup alone doesn't cover duplicate API calls with the same business key.

**How this system solves it:** **Effectively-once semantics** via:

- Unique constraint on `(idempotency_key, channel)` with TTL at ingest
- Status check before dispatch (`skip if succeeded`)
- Optimistic concurrency on status transitions (`UpdateIfStatus`)

**Tradeoffs:** Requires durable idempotency storage; TTL means keys can eventually be reused; does not claim end-to-end exactly-once (honest scope).

See also: [ADR-004](../adr/004-effectively-once.md).

---

## 3. Provider outages cascading into platform failure

**Problem before:** A failing email provider can cause unbounded retries, worker saturation, and retry storms.

**Why common approaches fail:** Blind infinite retry wastes resources and delays healthy traffic. Immediate DLQ loses recoverable transient failures.

**How this system solves it:** Classified errors (`ErrTransient` vs `ErrPermanent`), exponential backoff with jitter, max attempt cap (default 8), per-channel **circuit breakers**, and DLQ for exhausted retries.

**Tradeoffs:** Circuit breaker open state causes additional transient failures; tuning thresholds is environment-specific.

See also: [Workflows — Error handling](workflows.md#error-handling-flow), [DLQ runbook](../runbooks/dlq-inspection.md).

---

## 4. No visibility into async notification lifecycle

**Problem before:** "Did the email send?" requires correlating logs across services with no guaranteed trace continuity.

**Why common approaches fail:** Correlation IDs in logs break at async boundaries unless explicitly propagated through every hop.

**How this system solves it:** W3C `traceparent` injected into NATS headers at outbox publish, extracted at worker consume; append-only `audit_events` and `delivery_attempts`; Grafana dashboards and Prometheus business counters.

**Tradeoffs:** Full trace sampling is costly at high volume; JetStream consumer lag metrics are approximate in MVP.

See also: [Observability](../observability.md).

---

## 5. Scheduled / delayed notifications

**Problem before:** Delayed sends shouldn't block HTTP handlers or rely on broker delayed-message features alone.

**How this system solves it:** `scheduled_notifications` table + dedicated **scheduler** binary that polls due rows, transitions deliveries to `queued`, and enqueues outbox messages.

**Tradeoffs:** Poll-based scheduling (default 5s interval)—not sub-second precision; scheduler is a single logical process at MVP.

See also: [Workflows — Data processing lifecycle](workflows.md#data-processing-lifecycle).

---

**Next:** [Architecture](architecture.md) · [Design Decisions](design-decisions.md)
