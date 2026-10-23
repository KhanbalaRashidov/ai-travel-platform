# Migrations

Plain SQL migrations, applied **only to the local dev Docker Postgres** (ephemeral — no real data). Never run against a shared or production database.

Naming: `NNNN_name.up.sql` / `NNNN_name.down.sql` (sequential).

Apply (needs `psql`, or run inside the Postgres container):
```bash
make migrate-up        # DB_URL defaults to the local dev DB
make migrate-down      # rollback
```

Or via the container directly:
```bash
docker compose -f deploy/docker-compose.yml exec -T postgres \
  psql -U postgres -d ai_travel < db/migrations/0001_init.up.sql
```

| Version | Tables |
|---|---|
| 0001_init | tenants, users (+ default tenant) |

Planned next (later epics): trips, cities, days, transport_legs, accommodations, cost_lines, plan_variants, info_links; knowledge base: destinations, cost_samples, route_cache, poi_cache, fx_rates, popularity_rank — all with `tenant_id` + `source` + `fetched_at` + `ttl`.
