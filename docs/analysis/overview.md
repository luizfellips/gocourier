# Overview

[← Analysis index](README.md)

## System Overview

### What this system is

**Gocourier** is a notification delivery platform written in Go. It accepts notification requests over HTTP, persists them durably, routes them through a message queue, and dispatches them to downstream channels (email, SMS, push, webhook) via pluggable provider adapters. At MVP, all providers are mock implementations; the pipeline is production-shaped but vendor-agnostic.

The system is organized as a **modular monolith**: one Go module with clear package boundaries (`domain`, `application`, `ports`, `adapters`) and three deployable binaries (`api`, `worker`, `scheduler`) that share the same codebase and database.

### The core problem(s) it solves

1. **Reliable multi-channel notification delivery** — Applications need to send user-facing messages without coupling directly to email/SMS/push vendors or reimplementing retry, scheduling, and failure handling in every service.

2. **Duplicate suppression** — Retries, network timeouts, and at-least-once message brokers can cause the same notification to be sent multiple times. Duplicate order confirmations or password resets erode user trust.

3. **Safe async handoff** — Writing to a database and publishing to a message broker in separate steps creates classic dual-write bugs. The system must guarantee that accepted notifications eventually reach the queue even if the broker is temporarily unavailable.

4. **Operational visibility under failure** — When notifications fail, operators need audit trails, dead-letter queues (DLQs), replay tools, and metrics—not just log grep.

### Why these problems matter

Notification delivery sits on the critical path between business events (order placed, password reset requested) and user experience. Failures are visible, support-heavy, and often legally sensitive (marketing opt-out, transactional email compliance). A naive "call SendGrid from the API handler" approach breaks down quickly under retries, partial outages, scheduled sends, and cross-service idempotency requirements.

### Who the intended users are

| Audience | How they interact |
|----------|-------------------|
| **Application developers** | POST `/v1/notifications` with idempotency keys, channel, recipient, and template |
| **Platform / SRE operators** | Grafana dashboards, Prometheus alerts, DLQ runbooks, replay API |
| **Solo maintainer / small team** | Single VPS deployment via Docker Compose, debuggable without a large ops org |

### Business and technical value provided

- **Business:** Consistent delivery semantics across channels; reduced duplicate notifications; delayed/scheduled sends; incident recovery via replay.
- **Technical:** Reference implementation of transactional outbox, effectively-once idempotency, hexagonal architecture, and observability patterns suitable for a resource-constrained deployment.

---

## Executive Summary

In plain English: **you POST a notification request; the API saves it in PostgreSQL and returns immediately; a background process publishes it to NATS; workers pick it up and call the appropriate channel provider; failures retry with backoff or land in a DLQ for manual replay.**

### Main workflow (user input → final output)

1. Client sends JSON to `POST /v1/notifications` with an `idempotency_key`, `channel`, `recipient`, and `template`.
2. API validates the request, checks for duplicate idempotency keys, and atomically inserts the delivery record, idempotency key, audit event, and (for immediate sends) an outbox row—or a scheduled row if `scheduled_at` is in the future.
3. API returns `202 Accepted` with a `delivery_id` (or `200 OK` if duplicate).
4. Outbox publisher (co-located with API) polls pending outbox rows and publishes `{delivery_id}` to NATS JetStream subject `notifications.{channel}.{priority}`.
5. Worker consumers receive messages, load delivery state from PostgreSQL, skip if already succeeded, transition to `processing`, and invoke the channel provider.
6. On success: status → `succeeded`, attempt recorded, audit event appended.
7. On transient failure: exponential backoff retry via NATS NAK; on permanent failure or max retries: status → `dlq`, message published to `dlq.{channel}`.

### Key differentiators compared to simpler alternatives

| Simpler approach | What this system adds |
|------------------|----------------------|
| Direct provider call in HTTP handler | Async decoupling, backpressure via queue, worker scaling |
| Fire-and-forget to a queue | Transactional outbox—no lost events if broker is down at ingest time |
| Broker-only deduplication | Business-key idempotency in PostgreSQL survives broker redelivery |
| Single binary, no separation | Independent scaling/restart of API, workers, and scheduler |
| Log-only debugging | OpenTelemetry traces across HTTP → outbox → NATS → worker; Prometheus SLO metrics |

---

**Next:** [Pain Points](pain-points.md) · [Architecture](architecture.md)
