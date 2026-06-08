# System Analysis

Technical deep dive for experienced engineers who have never seen this codebase. These documents explain **what** the system is, **why** it exists, **how** architectural choices trade off against alternatives, and **what** can be learned from studying it.

The full analysis is also available as a single file: [ANALYSIS.md](../../ANALYSIS.md) (index with links to the sections below).

## Contents

| Document | Topics |
|----------|--------|
| [Overview](overview.md) | What the system is, problems solved, executive summary |
| [Pain Points](pain-points.md) | Problems addressed, why alternatives fail, tradeoffs |
| [Architecture](architecture.md) | Components, data flow, dependencies, diagrams |
| [Technology Choices](technology-choices.md) | Stack decisions, alternatives, lessons |
| [Design Decisions](design-decisions.md) | ADRs and implementation decisions |
| [Workflows](workflows.md) | Request lifecycle, errors, scaling, deployment |
| [Engineering Concepts](engineering-concepts.md) | Patterns, reliability, security, testing |
| [Use Cases](use-cases.md) | Primary/secondary scenarios, edge cases, non-goals |
| [Tradeoffs & Limitations](tradeoffs.md) | MVP limits, debt, risks, bottlenecks |
| [Lessons Learned](lessons-learned.md) | What engineers can take away from this project |
| [Future Improvements](future-improvements.md) | Short, medium, and long-term evolution |
| [Glossary](glossary.md) | Terms, acronyms, domain concepts |

## Related documentation

- [Architecture ADRs](../adr/) — Formal decision records
- [Observability](../observability.md) — Telemetry setup and validation
- [Runbooks](../runbooks/) — DLQ, replay, backup procedures
- [Testing guide](../../tests/README.md) — Test pyramid and suites
- [Performance SLOs](../../tests/performance/SLO.md) — Latency and throughput targets
