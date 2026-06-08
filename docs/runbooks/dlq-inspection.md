# Runbook: DLQ Inspection

## When to use

Notifications land in DLQ after permanent provider errors or max retry exhaustion.

## Identify DLQ deliveries

```sql
SELECT id, channel, idempotency_key, retry_count, last_error, updated_at
FROM deliveries
WHERE status = 'dlq'
ORDER BY updated_at DESC
LIMIT 50;
```

## Inspect attempt history

```sql
SELECT attempt_number, success, error_message, started_at, finished_at
FROM delivery_attempts
WHERE delivery_id = '<delivery-uuid>'
ORDER BY attempt_number;
```

## Inspect audit trail

```sql
SELECT event_type, payload, created_at
FROM audit_events
WHERE delivery_id = '<delivery-uuid>'
ORDER BY created_at;
```

## NATS DLQ stream

DLQ messages are published to subjects `dlq.{channel}` in the `DLQ` JetStream stream.

```bash
docker compose -f deploy/docker-compose.yml exec nats \
  nats stream view DLQ --subject dlq.email
```

## Common causes

| Error pattern | Likely cause | Action |
|---------------|--------------|--------|
| `invalid recipient` | Bad payload | Fix producer, do not replay |
| `provider unavailable` | Transient outage exhausted retries | Replay after provider recovery |
| `circuit breaker open` | Sustained provider failures | Wait for cooldown, fix provider |
| `no provider for channel` | Misconfiguration | Fix worker provider registration |

## Replay

See [replay procedure](replay.md).
