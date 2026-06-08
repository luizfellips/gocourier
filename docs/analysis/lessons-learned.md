# What an Engineer Can Learn From This Project

[← Analysis index](README.md)

Summarized lessons across architecture, infrastructure, patterns, product thinking, operations, and common mistakes.

---

## Architectural lessons

- Modular monolith with strict ports/adapters prepares you for split without premature microservices.
- Transactional outbox is the standard answer to DB + broker consistency—know when to reach for it.
- Claim honest delivery semantics (effectively-once) instead of overstating guarantees.

---

## Infrastructure lessons

- Docker Compose can run a credible observability stack for learning and small production.
- Healthchecks and graceful shutdown are part of the architecture, not polish.
- Backup runbooks distinguish Postgres (authoritative) from NATS (recoverable via outbox).

---

## Design-pattern lessons

- State machine in domain layer (`Delivery.Queue`, `StartProcessing`, `MarkSucceeded`, …)
- Error taxonomy drives retry vs DLQ—classify at the adapter boundary
- Compare-and-swap on status for idempotent consumers

---

## Product-thinking lessons

- Idempotency keys are an API contract—document TTL and channel scoping for integrators
- Duplicate response (`200` + `duplicate: true`) is better UX than silent double-send or opaque errors
- Operator replay is a first-class feature, not a database hack

---

## Operational lessons

- Runbooks for DLQ inspection, replay, and backup are co-located with code
- Mock failure injection via recipient patterns makes demos and tests reproducible
- Business metrics (`notifications_dlq_total`) belong in alert rules, not just error logs

---

## Mistakes to avoid

- Publishing to NATS inside the ingest HTTP handler without outbox
- In-memory idempotency caches in production
- Claiming exactly-once because the broker ack'd once
- Skipping integration tests for outbox recovery and concurrent idempotency
- Adding microservices before modular boundaries exist in one codebase

---

**Next:** [Future Improvements](future-improvements.md) · [Engineering Concepts](engineering-concepts.md)
