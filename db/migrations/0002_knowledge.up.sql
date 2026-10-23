-- 0002_knowledge — self-growing knowledge base + price store.
-- Every row carries tenant_id + provenance (source, fetched_at) and, where it
-- is a cached price, an expires_at for freshness. Apply ONLY to the local dev
-- Docker Postgres (ephemeral). Never a shared/prod DB.

-- Destinations the system has learned about.
CREATE TABLE IF NOT EXISTS destinations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    country    text NOT NULL,
    city       text NOT NULL,
    lat        double precision,
    lng        double precision,
    summary    text NOT NULL DEFAULT '',
    source     text NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, country, city)
);

-- Crowdable/estimated cost samples per city + category (food, local transport...).
CREATE TABLE IF NOT EXISTS cost_samples (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    city       text NOT NULL,
    category   text NOT NULL,
    amount     numeric(12,2) NOT NULL,
    currency   text NOT NULL,
    estimated  boolean NOT NULL DEFAULT false,
    source     text NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz
);

-- Price store: cached transport quotes per leg+mode.
CREATE TABLE IF NOT EXISTS route_cache (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    from_city        text NOT NULL,
    to_city          text NOT NULL,
    mode             text NOT NULL,
    price_amount     numeric(12,2) NOT NULL,
    price_currency   text NOT NULL,
    duration_seconds integer NOT NULL DEFAULT 0,
    deep_link        text NOT NULL DEFAULT '',
    estimated        boolean NOT NULL DEFAULT false,
    source           text NOT NULL,
    fetched_at       timestamptz NOT NULL DEFAULT now(),
    expires_at       timestamptz,
    UNIQUE (tenant_id, from_city, to_city, mode)
);

-- POIs learned per city.
CREATE TABLE IF NOT EXISTS poi_cache (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    city       text NOT NULL,
    name       text NOT NULL,
    category   text NOT NULL DEFAULT '',
    lat        double precision,
    lng        double precision,
    source     text NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now()
);

-- Price store: cached FX rates.
CREATE TABLE IF NOT EXISTS fx_rates (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    base       text NOT NULL,
    quote      text NOT NULL,
    rate       double precision NOT NULL,
    source     text NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz,
    UNIQUE (tenant_id, base, quote)
);

-- Popular-destinations ranking (aggregate counts; privacy-safe).
CREATE TABLE IF NOT EXISTS popularity_rank (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    destination text NOT NULL,
    kind        text NOT NULL DEFAULT 'city',
    hits        bigint NOT NULL DEFAULT 0,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, destination, kind)
);

CREATE INDEX IF NOT EXISTS idx_route_cache_lookup ON route_cache (tenant_id, from_city, to_city, mode);
CREATE INDEX IF NOT EXISTS idx_popularity_hits ON popularity_rank (tenant_id, hits DESC);
