# ADR-003: PostgreSQL as System of Record + Transactional Outbox

## Status
Accepted

## Context
Notification deliveries require durable state, idempotency tracking, audit history, scheduling, and safe publishing to the message broker without dual-write bugs.

## Decision
PostgreSQL is the system of record for deliveries, attempts, idempotency keys, scheduled notifications, and an outbox table. A background publisher reads the outbox and publishes to NATS JetStream.

## Consequences
- Ingest and state transitions are ACID within a transaction
- Broker unavailability does not lose events; outbox rows accumulate and retry
- Write amplification from audit + outbox + attempts tables
- Replay is database-backed and authoritative

## Alternatives Considered
- Publish directly to broker on ingest: rejected due to dual-write risk
- Event sourcing: rejected as over-engineering for MVP
