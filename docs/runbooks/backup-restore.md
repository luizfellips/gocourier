# Runbook: Backup and Restore

## Overview

PostgreSQL is the system of record. NATS JetStream holds hot queue data. Back up PostgreSQL daily; snapshot NATS JetStream storage weekly or before upgrades.

## PostgreSQL Backup

### Manual backup

```bash
docker compose -f deploy/docker-compose.yml exec postgres \
  pg_dump -U gocourier -d gocourier -Fc -f /tmp/gocourier.dump

docker compose -f deploy/docker-compose.yml cp postgres:/tmp/gocourier.dump ./backups/
```

### Automated backup (cron on VPS)

```bash
0 2 * * * docker compose -f /opt/go-gocourier/deploy/docker-compose.yml exec -T postgres \
  pg_dump -U gocourier -d gocourier -Fc | gzip > /backups/gocourier-$(date +\%F).dump.gz
```

Copy off-site with `rclone` or object storage sync.

## PostgreSQL Restore

```bash
docker compose -f deploy/docker-compose.yml stop api worker scheduler

docker compose -f deploy/docker-compose.yml exec -T postgres \
  pg_restore -U gocourier -d gocourier --clean --if-exists < ./backups/gocourier.dump

docker compose -f deploy/docker-compose.yml start api worker scheduler
```

**RPO:** up to 24 hours with daily backups.
**RTO:** ~30–60 minutes depending on dump size.

## NATS JetStream Backup

JetStream data lives in the `natsdata` Docker volume.

```bash
docker compose -f deploy/docker-compose.yml stop worker api scheduler nats
docker run --rm -v deploy_natsdata:/data -v $(pwd)/backups:/backup alpine \
  tar czf /backup/nats-$(date +%F).tar.gz -C /data .
docker compose -f deploy/docker-compose.yml start nats worker api scheduler
```

## Recovery Notes

- If PostgreSQL is restored but NATS is not, pending outbox rows will re-publish on API startup.
- Deliveries in `succeeded` state will not be re-dispatched (effectively-once).
- After restore, verify health: `curl http://localhost:8080/health`
