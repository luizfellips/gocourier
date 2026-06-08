# ADR-001: Modular Monolith over Microservices

## Status
Accepted

## Context
The platform is built by a solo developer, deployed on a single VPS via Docker Compose, and must be debuggable under production incidents without a large ops team.

## Decision
Start with a modular monolith: one Go module, clear package boundaries (`domain`, `application`, `ports`, `adapters`), and separate `cmd/` binaries for API, worker, and scheduler that share the same codebase.

## Consequences
- Faster iteration and simpler deployment
- Single-process debugging is straightforward
- Horizontal scaling requires splitting workers later (V2 hybrid model)
- Blast radius is larger than isolated microservices

## Alternatives Considered
- Event-driven microservices: rejected for MVP due to operational overhead
- CQRS + Event Sourcing: rejected; append-only audit in PostgreSQL is sufficient
