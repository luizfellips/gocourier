-- deliveries: current state of notification deliveries
CREATE TABLE IF NOT EXISTS deliveries (
    id              UUID PRIMARY KEY,
    idempotency_key TEXT NOT NULL,
    tenant_id       TEXT NOT NULL DEFAULT 'default',
    channel         TEXT NOT NULL,
    priority        TEXT NOT NULL DEFAULT 'normal',
    recipient       JSONB NOT NULL,
    template        JSONB,
    payload         JSONB,
    status          TEXT NOT NULL,
    scheduled_at    TIMESTAMPTZ,
    correlation_id  TEXT,
    causation_id    TEXT,
    retry_count     INT NOT NULL DEFAULT 0,
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deliveries_status ON deliveries(status);
CREATE INDEX IF NOT EXISTS idx_deliveries_tenant ON deliveries(tenant_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_created ON deliveries(created_at);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    idempotency_key TEXT NOT NULL,
    channel         TEXT NOT NULL,
    delivery_id     UUID NOT NULL REFERENCES deliveries(id),
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (idempotency_key, channel)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON idempotency_keys(expires_at);

CREATE TABLE IF NOT EXISTS delivery_attempts (
    id                UUID PRIMARY KEY,
    delivery_id       UUID NOT NULL REFERENCES deliveries(id),
    attempt_number    INT NOT NULL,
    started_at        TIMESTAMPTZ NOT NULL,
    finished_at       TIMESTAMPTZ,
    success           BOOLEAN NOT NULL DEFAULT FALSE,
    error_message     TEXT,
    provider_response JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attempts_delivery ON delivery_attempts(delivery_id);

CREATE TABLE IF NOT EXISTS outbox (
    id          BIGSERIAL PRIMARY KEY,
    delivery_id UUID NOT NULL REFERENCES deliveries(id),
    subject     TEXT NOT NULL,
    payload     BYTEA NOT NULL,
    headers     JSONB NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'pending',
    attempts    INT NOT NULL DEFAULT 0,
    last_error  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(status, created_at) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS scheduled_notifications (
    delivery_id   UUID PRIMARY KEY REFERENCES deliveries(id),
    scheduled_at  TIMESTAMPTZ NOT NULL,
    processed     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduled_due ON scheduled_notifications(scheduled_at) WHERE processed = FALSE;

CREATE TABLE IF NOT EXISTS audit_events (
    id          BIGSERIAL PRIMARY KEY,
    delivery_id UUID NOT NULL REFERENCES deliveries(id),
    event_type  TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_delivery ON audit_events(delivery_id);
