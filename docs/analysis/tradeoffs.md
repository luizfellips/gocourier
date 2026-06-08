# Tradeoffs and Limitations

[← Analysis index](README.md)

Current limitations, technical debt, risks, bottlenecks, and when a different architecture would be preferable.

---

## Current limitations

- Single-node NATS JetStream (SPOF)
- Mock providers only—no production vendor SLAs
- Poll-based outbox (500ms) and scheduler (5s)
- Single scheduler instance—no leader election
- Idempotency TTL (24h default)—keys reusable after expiry
- JetStream consumer lag metric approximate
- No automated outbox/idempotency row cleanup

---

## Technical debt

- Bulk replay is manual API loop, not batch endpoint
- `WorkerChannels` config exists but all channels share consumer filter
- Dashboard auth not separated from ingest API keys
- Nightly load test against staging marked placeholder in CI workflow

---

## Risks

| Risk | Impact | Mitigation path |
|------|--------|-----------------|
| Postgres disk growth (audit + attempts) | Slow queries, backup size | Archival policy, partitioning |
| Outbox publisher lag under spike | Increased delivery latency | Tune batch/interval; scale API |
| Provider retry storm | Worker saturation | Circuit breaker tuning; rate limits |
| NATS data loss without backup | In-flight messages lost | Weekly JetStream volume backup; outbox republish |

See [Backup runbook](../runbooks/backup-restore.md).

---

## Bottlenecks

1. PostgreSQL write rate on high ingest + audit + attempts
2. Single outbox publisher loop
3. Provider rate limits (when real adapters added)
4. Scheduler poll for large scheduled backlogs

---

## When a different architecture would be preferable

| Scenario | Better fit |
|----------|------------|
| **Millions of notifications per minute** | Kafka/Pulsar, sharded outbox, dedicated dispatch fleet |
| **Global low-latency** | Regional queues, geo-routed providers |
| **Heavy transformation pipeline** | Stream processing (Flink) between ingest and dispatch |
| **Large platform team** | Managed services (SQS + Lambda, SNS, etc.) may reduce ops burden vs self-hosted NATS |

---

**Next:** [Future Improvements](future-improvements.md) · [Lessons Learned](lessons-learned.md)
