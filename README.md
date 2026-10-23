# AI Travel Platform

AI-based **informational** multi-country trip planner (advisory only — no payments, booking, visa, or commerce). Produces a cost-optimal multi-country roadmap (cheapest city sequence + per-leg flight/train/bus + hotel tier + day-by-day plan) with a live total cost, powered by a self-growing knowledge base built from open/free sources + scraping + AI.

> Status: **E1 scaffold** (modular monolith). Trello: https://trello.com/b/9X8mMBUL

## Stack
- **API:** Go (Fiber) — modular monolith, split-ready packages (`services/{gateway,planner,connectors,knowledge,ai}`)
- **Web:** Next.js + TypeScript + Tailwind (`web/`)
- **DB:** PostgreSQL (multi-tenant-ready — `tenant_id` on every row)
- **Cache:** Redis
- **Shared:** `pkg/{config,db,cache,auth,tenant,models,httpx}`

## Architecture
Clean layering: **handler → service → repository** (DB access only in the repository layer).

## Local dev
```bash
cp .env.example .env            # fill values
docker compose -f deploy/docker-compose.yml up -d   # Postgres + Redis (+ api, web)
# apply migrations to the LOCAL dev DB only (ephemeral):
make migrate-up
# run API locally (alternative to the api container):
make run-api
# run web:
cd web && npm install && npm run dev
```

Health: `GET http://localhost:8080/health` · Readiness (db+redis): `GET /ready`

### AI (Gemini by default, free — or Claude)
Put the key in the **repo-root `.env`** (gitignored) and bring the stack up — nothing else. Default provider is **Gemini** (free):
```bash
echo 'GEMINI_API_KEY=...' >> .env                # LLM_PROVIDER=gemini is the default
docker compose -f deploy/docker-compose.yml up -d
```
To use Claude instead, set `LLM_PROVIDER=claude` and `ANTHROPIC_API_KEY=...` in `.env`. The api service loads the repo-root `.env` via `env_file` (no `--env-file`, no `${...}` interpolation warnings). Verify:
```bash
docker compose -f deploy/docker-compose.yml logs api | grep -i AI             # -> "AI enabled: provider=gemini model=..."
docker compose -f deploy/docker-compose.yml exec api printenv | grep -E 'GEMINI|LLM_PROVIDER'
```
Without the selected provider's key the stack still runs; the AI endpoints return 503 ("AI not configured"). Explicit `environment:` values (DATABASE_URL/REDIS_URL/JWT_*) override anything in `.env`, so the container keeps its service-DNS URLs.

Transient provider overload (Gemini 503/429 "high demand") is retried a few times with exponential backoff + jitter; if it persists the endpoint returns 503 "AI service is busy, try again shortly" and the UI shows a retry. Set `GEMINI_FALLBACK_MODEL` to have a second model tried when the primary stays overloaded.

## Monorepo layout
```
services/   Go app (gateway = HTTP entry; planner/connectors/knowledge/ai = split-ready domains)
pkg/        shared libs (config, db, cache, auth, tenant, models, httpx)
db/         SQL migrations + scripts
deploy/     docker-compose + Dockerfiles
web/        Next.js frontend
```

## Rules
- Secrets only in `.env` (gitignored). Never commit secrets/keys/*.pem.
- Migrations are scripts; apply ONLY to the local dev Docker Postgres. Never touch a shared/prod DB.
- Data from open/free sources + ToS-aware scraping; not dependent on gated APIs.
