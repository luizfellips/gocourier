# Future Improvements

[← Analysis index](README.md)

Planned evolution across short, medium, and long-term horizons.

---

## Short-term

- Real provider adapters (SendGrid, Twilio, FCM, generic webhook) behind `ChannelProvider`
- Automated purge job for expired `idempotency_keys` and published outbox rows
- Batch replay API for incident recovery
- Native JetStream consumer lag metrics in Prometheus
- Rate limiting per API key / tenant

---

## Medium-term

- JetStream clustering or managed NATS for HA
- Horizontal outbox sharding or leader-elected publisher
- Scheduler leader election for multi-instance deploy
- Tail sampling for OTEL traces in production
- Archival of old `audit_events` and `delivery_attempts` to cold storage

---

## Long-term architectural evolution

- **V2 hybrid scaling** (per [ADR-001](../adr/001-modular-monolith.md)): horizontally scaled worker pools per channel priority
- Extract notification dispatch to dedicated service while keeping ingest monolith
- Multi-region with Postgres read replicas and regional NATS streams
- Self-serve tenant dashboard with OAuth and webhook signing verification
- CDC-driven outbox if publish throughput exceeds poll-based publisher limits

---

**Next:** [Glossary](glossary.md) · [Tradeoffs](tradeoffs.md)
