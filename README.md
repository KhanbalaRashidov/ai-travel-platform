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

## Local dev / deploy — one command
```bash
cp .env.example .env            # fill values (all optional to boot)
docker compose -f deploy/docker-compose.yml up -d   # Postgres + Redis + api + web
```
That's it. The api applies its embedded DB migrations automatically on boot
(idempotent, fail-fast) — no `make`/`psql` step. Run pieces locally if you want:
```bash
make run-api                    # API outside the container
cd web && npm install && npm run dev
```

Health: `GET http://localhost:8080/health` · Readiness (db+redis): `GET /ready`

### Open it from another device on the same Wi‑Fi (LAN)
The web app resolves the API from the **page's own host** at runtime (no hardcoded
`localhost`), the gateway binds `0.0.0.0:8080`, the web binds `0.0.0.0:3000`, and
CORS defaults to `*` — so a phone/tablet on the same Wi‑Fi can use it with no extra
config after `up -d`:

1. Find the host machine's LAN IP:
   - macOS: `ipconfig getifaddr en0` (or `en1` on Wi‑Fi)
   - Linux: `hostname -I | awk '{print $1}'`
   - Windows: `ipconfig` → "IPv4 Address"
2. On the other device (same Wi‑Fi), open `http://<lan-ip>:3000`.
   The browser automatically calls the API at `http://<lan-ip>:8080`.

Notes:
- To pin a fixed API URL instead, set `NEXT_PUBLIC_API_BASE_URL` (inlined at build);
  a custom API port can be set with `NEXT_PUBLIC_API_PORT` (default `8080`).
- To restrict CORS, set `CORS_ORIGINS` (comma-separated) in `.env`.
- The LAN IP is not a secret; real secrets stay in the gitignored `.env`.
- Running locally (not Docker): `cd web && npm run dev` already binds `0.0.0.0`.

### AI (Groq by default, free — or Gemini / Claude)
Put the key in the **repo-root `.env`** (gitignored) and bring the stack up — nothing else. Default provider is **Groq** (free, OpenAI-compatible):
```bash
echo 'GROQ_API_KEY=...' >> .env                  # LLM_PROVIDER=groq is the default
docker compose -f deploy/docker-compose.yml up -d
```
To switch provider, set `LLM_PROVIDER` to `gemini` (+ `GEMINI_API_KEY`) or `claude` (+ `ANTHROPIC_API_KEY`) in `.env`. The api service loads the repo-root `.env` via `env_file` (no `--env-file`, no `${...}` interpolation warnings). Verify:
```bash
docker compose -f deploy/docker-compose.yml logs api | grep -i AI             # -> "AI enabled: provider=groq model=..."
docker compose -f deploy/docker-compose.yml exec api printenv | grep -E 'GROQ|GEMINI|LLM_PROVIDER'
```
Without the selected provider's key the stack still runs; the AI endpoints return 503 ("AI not configured"). Explicit `environment:` values (DATABASE_URL/REDIS_URL/JWT_*) override anything in `.env`, so the container keeps its service-DNS URLs.

Transient provider overload (Gemini 503/429 "high demand") is retried a few times with exponential backoff + jitter; if it persists the endpoint returns 503 "AI service is busy, try again shortly" and the UI shows a retry. Set `GEMINI_FALLBACK_MODEL` to have a second model tried when the primary stays overloaded.

### Real prices (multi-source scraping + AI extraction)
Set `PRICE_SOURCES` in `.env` (one `name | tos-note | url-template` per line; see `.env.example`) to fetch real prices from public pages. The pipeline: HTTP fetch (realistic UA) → the LLM extracts `{price,currency}` from the page text → a **verbatim guardrail** drops any number not present literally → FX-normalize and pick the cheapest with `source + fetched_at` → cache hot in Redis. Blocked/anti-bot pages (401/403/429/CAPTCHA) are **skipped gracefully — never bypassed**; if every source fails the plan falls back to the clearly-labeled `~ indicative` estimate. Scraping is the operator's ToS responsibility.

JS-rendered pages are an extension point: the source layer already falls back to a headless `Renderer` when an HTTP fetch returns an empty/JS-only page. To enable it, implement `pricefetch.Renderer` with a browser automation library (e.g. a Chromium driver launched headless with `--no-sandbox` for containers) and pass it to `DefaultSources(...)` in the router (today `NewHeadless()` returns nil, so the default build/image stay lean — HTTP fetching only).

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
