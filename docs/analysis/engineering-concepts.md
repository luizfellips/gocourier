# Engineering Concepts Demonstrated

[← Analysis index](README.md)

How major engineering concepts appear in this system, why they matter, and practical lessons.

---

## System design principles


| Concept                     | In this system                 | Why it matters                           |
| --------------------------- | ------------------------------ | ---------------------------------------- |
| **Single writer principle** | PostgreSQL owns delivery state | Avoids split-brain between cache and DB  |
| **Separation of concerns**  | Ingest ≠ dispatch ≠ schedule   | Each path scales and fails independently |
| **Fail closed auth**        | Missing API key → 401          | No anonymous ingest in production config |


**Lesson:** Bound consistency at the transaction; accept eventual consistency across the broker boundary explicitly.

---

## Scalability patterns

- **Queue-based load leveling** — API accepts fast; workers drain at provider pace.
- **Competing consumers** — Multiple worker goroutines/processes on shared durable consumers.
- **Priority subjects** — `notifications.{channel}.{priority}` enables future weighted consumption (not fully exploited at MVP).

**Lesson:** Scale the **slow step** (provider dispatch), not the accept path.

---

## Reliability patterns

- Transactional outbox
- At-least-once consumption with idempotent handlers
- Exponential backoff + jitter (`pkg/retry`)
- Circuit breaker per channel (`pkg/circuitbreaker`)
- DLQ + operator replay
- Outbox recovery after API restart (integration test: `TestOutboxRecoveryAfterRestart`)

**Lesson:** Reliability is **layered**, there is no single mechanism that delivers everything; compose patterns with clear semantics at each layer.

---

## Security considerations

- API key authentication (`X-API-Key` / Bearer)
- Input validation (schema version, key length ≤ 256, required fields)
- Parameterized SQL (security tests include SQL injection in metadata)
- Internal vs public Docker networks (Postgres/NATS not exposed on host by default)
- No secrets in repo (`dev-api-key` for local only)

**Gaps at MVP:** No mTLS, no OAuth, no per-tenant rate limiting, no PII encryption at rest.

---

## API design

- RESTful ingest with `202 Accepted` for new, `200 OK` for duplicate
- Idempotency key as client-controlled dedup token (Stripe-style semantics)
- Schema version field for forward compatibility
- Minimal surface: ingest, replay, dashboard read, health, metrics

**Lesson:** **HTTP status codes encode business outcomes** (duplicate vs created)—not just success/failure.

---

## Event-driven architecture

- Domain events in `audit_events` (NotificationReceived, DispatchSucceeded, etc.)
- NATS as transport, not source of truth
- Subject-based routing decouples producers from consumer topology

**Lesson:** **Events for observability and integration; database for truth.**

---

## Distributed systems concepts

- Dual-write problem and outbox solution
- At-least-once vs exactly-once honesty
- Clock skew handling via DB `TIMESTAMPTZ` and application `Clock` port (testable with `FixedClock`)
- Split-brain mitigation via status CAS

---

## Observability

- **Metrics:** Business counters (`notifications_`*), RED (`api_request_duration_seconds`), pool stats, broker publish/consume
- **Traces:** OTEL spans on ingest, outbox flush, dispatch, replay; W3C propagation through NATS
- **Logs:** Structured `slog` with trace correlation
- **Dashboards:** API, Worker, Queue, System folders in Grafana

**Lesson:** **Business metrics alongside infrastructure metrics**—SLOs are about notifications delivered, not just CPU.

See [Observability](../observability.md).

---

## Caching

**Not used** for idempotency or delivery state—by design. Durable dedup requires PostgreSQL.

**Lesson:** Don't cache what you can't afford to lose; idempotency is a durability concern.

---

## Data modeling


| Table                               | Role                     |
| ----------------------------------- | ------------------------ |
| `deliveries`                        | Current state (mutable)  |
| `delivery_attempts`, `audit_events` | Append-only history      |
| `idempotency_keys`                  | TTL-scoped dedup index   |
| `outbox`                            | Transient publish intent |
| `scheduled_notifications`           | Time-indexed work queue  |


**Lesson:** **Separate mutable state from append-only history**—simplifies audit, replay, and debugging.

---

## Testing strategies


| Layer                     | Approach                                                                       |
| ------------------------- | ------------------------------------------------------------------------------ |
| Unit                      | Table-driven tests beside domain/application code; ≥90% domain coverage target |
| Integration               | testcontainers Postgres + NATS; full pipeline tests                            |
| Idempotency / concurrency | Race detector (`-race`); 100–10K parallel ingests                              |
| Failure                   | Mock provider injection; Toxiproxy for broker outages                          |
| Chaos                     | Staging-only worker SIGTERM, broker restart                                    |
| Performance               | k6 load, vegeta stress; documented SLOs                                        |


**Lesson:** **Test the failure modes you document in runbooks**—DLQ, replay, outbox recovery are CI-gated, not aspirational.

See [Testing guide](../../tests/README.md).

---

**Next:** [Use Cases](use-cases.md) · [Lessons Learned](lessons-learned.md)