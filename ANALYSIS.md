# Gocourier — System Analysis

A technical deep dive for experienced engineers who have never seen this codebase. This document explains **what** the system is, **why** it exists, **how** architectural choices trade off against alternatives, and **what** can be learned from studying it.

The analysis is split into focused documents under **[docs/analysis/](docs/analysis/)** for easier navigation. Use the index below or start with [docs/analysis/README.md](docs/analysis/README.md).

---

## Quick start

**In one sentence:** POST a notification → PostgreSQL persists it (transactional outbox) → NATS queues it → workers dispatch to channel providers with retry, circuit breaking, and DLQ on failure.

**Start here:** [Overview](docs/analysis/overview.md)

---

## Document index

| Section | Document |
|---------|----------|
| System Overview & Executive Summary | [overview.md](docs/analysis/overview.md) |
| Pain Points Addressed | [pain-points.md](docs/analysis/pain-points.md) |
| Architecture Overview | [architecture.md](docs/analysis/architecture.md) |
| Technology Choices | [technology-choices.md](docs/analysis/technology-choices.md) |
| Key Design Decisions | [design-decisions.md](docs/analysis/design-decisions.md) |
| System Workflows | [workflows.md](docs/analysis/workflows.md) |
| Engineering Concepts Demonstrated | [engineering-concepts.md](docs/analysis/engineering-concepts.md) |
| Use Cases | [use-cases.md](docs/analysis/use-cases.md) |
| Tradeoffs and Limitations | [tradeoffs.md](docs/analysis/tradeoffs.md) |
| What an Engineer Can Learn | [lessons-learned.md](docs/analysis/lessons-learned.md) |
| Future Improvements | [future-improvements.md](docs/analysis/future-improvements.md) |
| Glossary | [glossary.md](docs/analysis/glossary.md) |

---

## Related documentation

- [README.md](README.md) — Quick start and binary overview
- [docs/adr/](docs/adr/) — Architecture decision records
- [docs/observability.md](docs/observability.md) — Telemetry validation
- [docs/runbooks/](docs/runbooks/) — DLQ, replay, backup procedures
- [tests/README.md](tests/README.md) — Full testing guide and pyramid
- [tests/performance/SLO.md](tests/performance/SLO.md) — Performance targets
