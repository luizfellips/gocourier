# ADR-002: NATS JetStream as Message Broker

## Status
Accepted

## Context
The platform needs durable queuing, at-least-once delivery, channel-based routing, and replay capability on a resource-constrained VPS.

## Decision
Use NATS JetStream with streams per channel and priority (`notifications.{channel}.{priority}`) and separate DLQ subjects (`dlq.{channel}`).

## Consequences
- Low operational footprint compared to Kafka
- Native persistence and consumer ack semantics
- PostgreSQL outbox remains the durability backstop if NATS publish fails
- Single-node JetStream is a single point of failure at MVP

## Alternatives Considered
- Kafka: excellent at scale but too heavy for solo VPS ops
- RabbitMQ: viable alternative with weaker native replay
- Redis Streams: not durable enough as primary queue
