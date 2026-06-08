# Runbook: Replay Procedure

## Overview

Replay re-queues a DLQ or failed delivery for another dispatch attempt. PostgreSQL is authoritative; replay does not duplicate succeeded deliveries.

## API replay (operator action)

```bash
curl -X POST http://localhost:8080/v1/notifications/<delivery-id>/replay \
  -H "X-API-Key: dev-api-key"
```

Expected response:

```json
{"delivery_id":"<uuid>","status":"queued"}
```

## Preconditions

- Delivery status must be `dlq` or `failed`
- Fix root cause before replay (provider outage, bad config, etc.)
- Do not replay deliveries with permanent validation errors unless payload was corrected in DB

## Verify replay

```sql
SELECT status, retry_count, updated_at FROM deliveries WHERE id = '<delivery-id>';
```

```sql
SELECT event_type, created_at FROM audit_events
WHERE delivery_id = '<delivery-id>' AND event_type = 'NotificationReplayed';
```

Wait for worker to process:

```sql
SELECT status FROM deliveries WHERE id = '<delivery-id>';
-- expect: succeeded
```

## Bulk replay (SQL + manual)

For incident recovery, query affected deliveries:

```sql
SELECT id FROM deliveries
WHERE status = 'dlq'
  AND updated_at > NOW() - INTERVAL '1 hour'
  AND channel = 'email';
```

Replay each via API. Automate with a script looping over IDs.

## Safety

- Replay sets `retry_count` to 0 and status to `queued`
- Workers skip dispatch if status is already `succeeded`
- All replays are recorded in `audit_events` with `NotificationReplayed`
