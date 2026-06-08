# ADR-004: Effectively-Once via Idempotency Keys

## Status
Accepted

## Context
Duplicate notifications (e.g., duplicate order confirmation emails) harm user trust. The broker provides at-least-once delivery only.

## Decision
Implement effectively-once semantics using:
1. Unique constraint on `(idempotency_key, channel)` with TTL in PostgreSQL at ingest
2. Status check before dispatch (skip if already `succeeded`)
3. Append-only `delivery_attempts` audit trail

We do not claim end-to-end exactly-once.

## Consequences
- Duplicate API calls return the original `delivery_id` without re-dispatch
- Redelivered broker messages are safely deduplicated at the worker
- Requires durable idempotency storage, not in-memory caches alone

## Alternatives Considered
- Broker exactly-once: not available in NATS in the sense we need; would still require business-key dedup
- Full distributed exactly-once: rejected due to cost and complexity
