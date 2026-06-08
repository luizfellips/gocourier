# Use Cases

[← Analysis index](README.md)

Real-world scenarios the system supports, edge cases, and explicit non-goals.

---

## Primary use cases

1. **Transactional email** — Order confirmation with idempotency key tied to order ID.
2. **SMS alert** — High-priority password reset with retry on carrier blip.
3. **Webhook fan-out** — Partner notification with audit trail for compliance.
4. **Scheduled reminder** — `scheduled_at` for "send in 24 hours" without cron in every app.

### Example ingest request

```bash
curl -X POST http://localhost:8080/v1/notifications \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dev-api-key" \
  -d '{
    "schema_version": "1.0",
    "idempotency_key": "order-123-confirm",
    "channel": "email",
    "priority": "normal",
    "recipient": {"address": "user@example.com"},
    "template": {"id": "welcome", "data": {}}
  }'
```

---

## Secondary use cases

- **Ops replay** — Re-queue DLQ batch after provider outage recovery ([replay runbook](../runbooks/replay.md)).
- **Load test / demo** — Mock providers + dashboard **Send Test Form** (single sends) + **Load Test Panel** (parallel burst, duplicate storms, random channel/priority, retry/failure mixes) + k6 scripts under `tests/load/`.
- **Observability walkthrough** — Trace single `delivery_id` from API through worker in Tempo.

---

## Edge cases

| Scenario | Behavior |
|----------|----------|
| Duplicate POST same idempotency key | Returns original `delivery_id`, no re-dispatch |
| Same key, different channel | Allowed—keys scoped per channel |
| NATS redelivery while processing | `UpdateIfStatus` — one worker wins, other skips |
| Broker down at ingest | Outbox rows accumulate; drain on recovery |
| Already succeeded + replay API | Replay rejected (status gate) |
| Circuit breaker open | Fast fail as transient; NATS retries |

### Mock failure injection (testing)

| Recipient contains | Result |
|--------------------|--------|
| `fail-permanent` | Immediate DLQ |
| `fail-transient` | Retry via NATS NAK |
| `fail-circuit` | Transient error (circuit breaker testing) |
| *(otherwise)* | Success |

---

## Non-goals and unsupported scenarios

- End-to-end exactly-once delivery (explicitly out of scope)
- Multi-region active-active
- Template rendering / content management (template ID passed through; rendering is client or future service)
- User preference management / unsubscribe handling
- Real provider integrations at MVP
- Sub-second scheduled delivery precision
- Multi-tenant billing or quota enforcement

---

**Next:** [Tradeoffs & Limitations](tradeoffs.md) · [Workflows](workflows.md)
