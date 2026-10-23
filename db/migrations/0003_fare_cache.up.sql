-- Durable (warm) fare cache: real fetched prices keyed by leg + date + display
-- currency, per tenant. Survives a Redis flush; the resolver reads it before
-- hitting the network and writes winning candidates back here.
CREATE TABLE IF NOT EXISTS fare_cache (
    tenant_id   TEXT        NOT NULL,
    fare_key    TEXT        NOT NULL,
    source      TEXT        NOT NULL,
    url         TEXT        NOT NULL DEFAULT '',
    amount      DOUBLE PRECISION NOT NULL,
    currency    TEXT        NOT NULL,
    fetched_at  TIMESTAMPTZ NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, fare_key)
);

CREATE INDEX IF NOT EXISTS idx_fare_cache_expires ON fare_cache (expires_at);
