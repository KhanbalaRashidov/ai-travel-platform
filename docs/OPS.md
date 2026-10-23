# Operations & Deployment

## Stack
Go (Fiber) API + Next.js web + PostgreSQL + Redis, as a modular monolith.
Everything runs via Docker Compose on the operator's machine (developed/tested
on macOS).

## Run locally (Mac)
```bash
cp .env.example .env          # fill the values you want (all optional to boot)
docker compose -f deploy/docker-compose.yml up -d --build
make migrate-up               # apply 0001..0003 to the LOCAL dev DB only
curl :8080/health             # {status:ok}
curl :8080/ready              # db + redis
# web: http://localhost:3000
```
The API loads the repo-root `.env` via `env_file` (no `--env-file` needed).
Migrations are **scripts**; apply them only to the ephemeral local dev Postgres,
never to a shared/production database.

## Configuration (.env)
See `.env.example` for the full list. Key groups:

- **Core:** `API_PORT`, `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET` (set a strong
  value outside dev), `JWT_TTL_HOURS`.
- **AI:** `LLM_PROVIDER` (`groq` default | `gemini` | `claude`) + the matching
  key (`GROQ_API_KEY` / `GEMINI_API_KEY` / `ANTHROPIC_API_KEY`), `AI_BUDGET_USD`.
  Unset → AI endpoints return 503; the rest runs.
- **Prices:** `PRICE_SOURCES` (optional overrides/extra sources on top of the
  built-in list). Real fetching also needs an AI key (for extraction).

Secrets live only in `.env` (gitignored); never commit keys.

## Headless rendering (extension point)
JS-rendered price pages are supported via the `pricefetch.Renderer` interface:
the source layer falls back to a headless renderer when an HTTP fetch returns an
empty/JS-only page. The default build ships HTTP-only (`NewHeadless()` → nil) so
the image stays lean. To enable, implement `Renderer` with a Chromium driver
(headless, `--no-sandbox` in containers) and wire it into `DefaultSources(...)`.

## Health, logs, metrics
- `GET /health` (liveness), `GET /ready` (DB + Redis readiness).
- Structured request logs via Fiber's logger; AI token/cost is logged per call.
- `GET /api/v1/analytics/stats` returns privacy-safe aggregate event counts.

## Backups & restore (Postgres)
Migrations ship as scripts and are applied with `make migrate-up` as part of
deploy (never against a shared DB from the app). Back up and verify restores:
```bash
# Nightly logical backup (cron):
docker compose -f deploy/docker-compose.yml exec -T postgres \
  pg_dump -U postgres ai_travel | gzip > backup_$(date +%F).sql.gz

# Restore (into a fresh DB) and verify before cutover:
gunzip -c backup_YYYY-MM-DD.sql.gz | \
  docker compose -f deploy/docker-compose.yml exec -T postgres psql -U postgres ai_travel
```
Test a restore on a throwaway database periodically (the launch checklist tracks
this). Redis and the fare cache are regenerable and need no backup. Keep dumps
encrypted and off-host.

## Scaling notes
- The planner engine is pure/stateless → horizontally scalable.
- Prices use a hot (Redis) → warm (DB) → cold (fetch) flywheel; repeat lookups
  avoid the network. Source fetches run in parallel per leg.
