# Key Design Decisions

[← Analysis index](README.md)

Important architectural and implementation decisions with context, alternatives, pros/cons, long-term implications, and lessons. Formal ADRs live in [docs/adr/](../adr/).

---

## ADR-001: Modular monolith over microservices


|                  |                                                                                                       |
| ---------------- | ----------------------------------------------------------------------------------------------------- |
| **Decision**     | One module, three binaries, clear package boundaries                                                  |
| **Context**      | Solo developer, single VPS, must debug incidents without a platform team                              |
| **Alternatives** | Microservices per channel; CQRS + event sourcing                                                      |
| **Pros**         | Fast iteration, simple deploy, shared types, single repo debugging                                    |
| **Cons**         | Larger blast radius; worker scaling requires process duplication not independent services             |
| **Long-term**    | V2 hybrid: split workers horizontally; extract hot paths only when metrics justify                    |
| **Lesson**       | **Start monolithic, enforce modular boundaries** so splitting later is a deploy change, not a rewrite |


→ [Full ADR](../adr/001-modular-monolith.md)

---

## ADR-002: NATS JetStream as message broker


|                  |                                                                                                                         |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------- |
| **Decision**     | JetStream streams `NOTIFICATIONS` and `DLQ` with subject-per-channel-priority routing                                   |
| **Context**      | Durable queue, replay, low VPS resource usage                                                                           |
| **Alternatives** | Kafka, RabbitMQ, Redis Streams                                                                                          |
| **Pros**         | Small footprint, explicit ack/NAK, file-backed persistence                                                              |
| **Cons**         | Single-node SPOF; cluster ops deferred                                                                                  |
| **Long-term**    | JetStream clustering or managed NATS when availability SLO tightens                                                     |
| **Lesson**       | **Subject design is API design**: `notifications.email.high` enables future priority consumers without schema migration |


→ [Full ADR](../adr/002-nats-jetstream.md)

---

## ADR-003: PostgreSQL + transactional outbox


|                  |                                                                                                             |
| ---------------- | ----------------------------------------------------------------------------------------------------------- |
| **Decision**     | Postgres is authoritative; outbox bridges to NATS                                                           |
| **Context**      | Dual-write safety, audit, scheduling, idempotency in one store                                              |
| **Alternatives** | Direct broker publish; full event sourcing                                                                  |
| **Pros**         | ACID ingest; broker outage buffers in outbox; replay is DB-backed                                           |
| **Cons**         | Extra table and publisher loop; outbox lag under load                                                       |
| **Long-term**    | Outbox cleanup/archival policy; possibly CDC if publish volume dominates                                    |
| **Lesson**       | **The database transaction boundary is your consistency boundary**—put everything that must agree inside it |


→ [Full ADR](../adr/003-postgresql-outbox.md)

---

## ADR-004: Effectively-once via idempotency keys


|                  |                                                                                                               |
| ---------------- | ------------------------------------------------------------------------------------------------------------- |
| **Decision**     | `(idempotency_key, channel)` uniqueness + dispatch status gates                                               |
| **Context**      | At-least-once broker; client retries                                                                          |
| **Alternatives** | Broker exactly-once; distributed transactions                                                                 |
| **Pros**         | Predictable duplicate API behavior; safe NATS redelivery                                                      |
| **Cons**         | Storage + TTL management; client responsibility for keys                                                      |
| **Long-term**    | Background job to purge expired idempotency rows                                                              |
| **Lesson**       | **Idempotency is a product contract**, not an implementation detail: document key semantics for API consumers |


→ [Full ADR](../adr/004-effectively-once.md)

---

## ADR-005: Mock-first provider adapters


|                  |                                                                                                              |
| ---------------- | ------------------------------------------------------------------------------------------------------------ |
| **Decision**     | `ChannelProvider` port with mock implementations for all channels                                            |
| **Context**      | Validate pipeline before vendor integration                                                                  |
| **Alternatives** | Real providers at MVP; generic HTTP adapter only                                                             |
| **Pros**         | Deterministic failure injection; no external test dependencies                                               |
| **Cons**         | Production provider quirks (rate limits, webhooks) not yet exercised                                         |
| **Long-term**    | Real adapters behind same interface; provider-specific circuit tuning                                        |
| **Lesson**       | **Define the port before the adapter**—mock providers that simulate failure modes are test assets, not stubs |


→ [Full ADR](../adr/005-mock-providers.md)

---

## Separate scheduler binary


|                  |                                                                                   |
| ---------------- | --------------------------------------------------------------------------------- |
| **Decision**     | `cmd/scheduler` polls `scheduled_notifications` independently of API and worker   |
| **Context**      | Scheduled sends shouldn't run in HTTP request path                                |
| **Alternatives** | Cron inside API; broker delayed messages                                          |
| **Pros**         | Independent restart/scaling; clear responsibility                                 |
| **Cons**         | Another process to deploy and monitor; poll granularity limits precision          |
| **Long-term**    | Leader election if multiple schedulers; or migrate to precise delay queues        |
| **Lesson**       | **Time-based work deserves its own runtime**, not a goroutine tucked into the API |


---

## Optimistic concurrency on dispatch (`UpdateIfStatus`)


|                  |                                                                                    |
| ---------------- | ---------------------------------------------------------------------------------- |
| **Decision**     | Worker updates delivery only if status matches expected previous state             |
| **Context**      | Concurrent workers or redelivered messages could double-process                    |
| **Alternatives** | Pessimistic row locks; distributed locks (Redis)                                   |
| **Pros**         | No external lock service; skips concurrent dispatch cleanly                        |
| **Cons**         | Silent skip on conflict, thus relying on NATS redelivery for the "losing" worker   |
| **Long-term**    | Metrics on concurrent skip rate                                                    |
| **Lesson**       | **Compare-and-swap on state machines** is often enough for at-least-once consumers |


---

**Next:** [Workflows](workflows.md) · [Pain Points](pain-points.md)