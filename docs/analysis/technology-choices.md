# Technology Choices

[← Analysis index](README.md)

For each significant technology, framework, library, pattern, or infrastructure component: what it is, why it was chosen, alternatives, tradeoffs, and lessons.

---

## Go 1.25

**What it is:** Statically typed, compiled language with strong concurrency primitives.

**Why chosen:** Single binary deployment, low memory footprint on a VPS, excellent standard library for HTTP and tooling, first-class race detector for concurrency tests.

**Alternatives:** Node.js (weaker typing for domain logic), Java (heavier runtime), Rust (steeper iteration cost for solo developer).

**Tradeoffs:** Smaller ecosystem for some SaaS SDKs vs Python/Node; explicit error handling verbosity.

**Lessons:** Go suits **ops-friendly, resource-constrained backend services** where deploy simplicity and concurrency correctness matter more than rapid CRUD scaffolding.

**Appropriate when:** Single-binary deploys, high concurrency, small teams. **Inappropriate when:** Heavy ML/data science integration or rapid UI prototyping is the primary goal.

---

## Hexagonal architecture (Ports & Adapters)

**What it is:** Domain and application logic depend on interfaces (`ports`); infrastructure (`adapters/postgres`, `adapters/nats`, `adapters/providers`) implements them.

**Why chosen:** Swap Postgres/NATS/mock providers in tests without changing business rules; keeps `internal/application` free of SQL and NATS imports.

**Alternatives:** Layered MVC (often leaks infrastructure into services), microservices per channel (rejected in [ADR-001](../adr/001-modular-monolith.md)).

**Tradeoffs:** More packages and boilerplate; indirection can feel heavy for very small projects.

**Lessons:** **Ports pay off when you have multiple adapters and a serious test pyramid.** The `testkit` harness wires real Postgres + NATS against the same ports production uses.

---

## PostgreSQL 16

**What it is:** ACID relational database used as system of record.

**Why chosen:** Transactional outbox requires ACID; JSONB for flexible recipient/template payloads; mature backup/restore story on a VPS.

**Alternatives:** CockroachDB (overkill for single-node MVP), DynamoDB (harder transactional outbox), event sourcing store (rejected—append-only audit is sufficient).

**Tradeoffs:** Write amplification (deliveries + idempotency + outbox + attempts + audit); vertical scaling limits; connection pool tuning needed under load.

**Lessons:** PostgreSQL as **single source of truth + outbox** is a well-trodden pattern for "reliable enough" event-driven systems without Kafka complexity.

---

## NATS JetStream

**What it is:** Lightweight message broker with durable streams, consumer acks, and subject-based routing.

**Why chosen:** Low ops footprint vs Kafka; native persistence; subject patterns map cleanly to `notifications.{channel}.{priority}`; fits solo VPS deployment.

**Alternatives:** Kafka (excellent at scale, too heavy for MVP ops), RabbitMQ (viable, weaker native replay), Redis Streams (less durable as primary queue).

**Tradeoffs:** Single-node JetStream is SPOF at MVP; at-least-once only; cluster setup is future work.

**Lessons:** **Match broker complexity to team size and scale.** JetStream is appropriate for thousands of msg/s on a VPS; revisit at multi-region or millions/sec.

See also: [ADR-002](../adr/002-nats-jetstream.md).

---

## Transactional outbox

**What it is:** Pattern where message publish intent is stored in the same DB transaction as business data; a separate process publishes to the broker.

**Why chosen:** Eliminates dual-write between Postgres and NATS ([ADR-003](../adr/003-postgresql-outbox.md)).

**Alternatives:** Change Data Capture (Debezium)—more moving parts; direct publish on ingest—unsafe.

**Tradeoffs:** Eventual consistency lag; outbox table growth; publisher must be idempotent (implemented via `HasPublishedForDelivery`).

**Lessons:** The outbox is the **minimum viable correctness layer** between OLTP and async messaging. Always design the publisher for retries and duplicates.

---

## Idempotency keys (effectively-once)

**What it is:** Client-supplied business key scoped per channel, stored with unique constraint and TTL.

**Why chosen:** Duplicate API calls and broker redelivery must not double-send ([ADR-004](../adr/004-effectively-once.md)).

**Alternatives:** Distributed exactly-once (Kafka transactions + DB—complex); dedup only at broker (insufficient).

**Tradeoffs:** Clients must generate keys; TTL (default 24h) allows key reuse after expiry; cross-channel same key is allowed by design.

**Lessons:** **Honest semantics:** claim "effectively-once" at the business level, not mathematically exactly-once end-to-end.

---

## OpenTelemetry + Prometheus + Grafana + Tempo

**What it is:** OTEL for traces (via OTLP gRPC), Prometheus for metrics (direct scrape + business counters), Tempo for trace storage, Grafana for visualization.

**Why chosen:** Cross-service causality across async boundary; RED + business metrics for SLOs; industry-standard tooling.

**Alternatives:** Datadog/New Relic (cost, vendor lock-in for solo project); logs-only (insufficient for async).

**Tradeoffs:** No tail sampling in dev (all traces exported); dual metrics path (Prometheus scrape + OTEL); collector is another failure point for traces only.

**Lessons:** **Propagate trace context through message headers**, not just HTTP. Outbox → NATS → worker is where most systems lose observability.

See also: [Observability](../observability.md).

---

## pgx/v5

**What it is:** High-performance PostgreSQL driver for Go with connection pooling.

**Why chosen:** Native Go, pool support, query tracing hooks for telemetry.

**Alternatives:** database/sql + lib/pq (older), GORM (unnecessary ORM layer for this schema).

**Tradeoffs:** Raw SQL in adapters; manual migration management.

**Lessons:** For bounded schemas with heavy transactional logic, **explicit SQL beats ORM magic**.

---

## testcontainers-go

**What it is:** Spins up real Postgres and NATS in Docker during integration tests.

**Why chosen:** Tests validate actual SQL, JetStream ack semantics, and race conditions—not mocks of mocks.

**Alternatives:** Embedded Postgres (platform quirks), heavy mocking (misses integration bugs).

**Tradeoffs:** Requires Docker; slower CI; longer timeouts.

**Lessons:** **Invest in a testkit** that mirrors production wiring—pays dividends for idempotency and outbox recovery tests.

See also: [Testing guide](../../tests/README.md).

---

## React ops dashboard (web/)

**What it is:** Vite + React frontend with shadcn UI, TanStack Query, Zustand—proxied to Go API.

**Why chosen:** Operator visibility, manual test sends (mock failure injection via recipient patterns), and an interactive **Load Test Panel** for burst / duplicate / multi-channel scenarios without leaving the browser.

**Alternatives:** CLI-only ops (harder for demos and on-call); Retool (external dependency).

**Tradeoffs:** Separate build/deploy artifact; not multi-tenant admin at MVP.

**Lessons:** A **thin ops UI** accelerates validation of retry/DLQ/replay flows beyond curl scripts.

---

**Next:** [Design Decisions](design-decisions.md) · [Engineering Concepts](engineering-concepts.md)
