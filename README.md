<p align="center">
  <img src="web/app/icon.svg" width="72" height="72" alt="AI Travel logo" />
</p>

<h1 align="center">AI Travel Platform</h1>

<p align="center">
  <b>Advisory-only, cost-optimal multi-country trip planner.</b><br/>
  Cheapest city sequence · every transport mode · real-source prices + links · day-by-day roadmap · live total cost.
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-Fiber-00ADD8" />
  <img alt="Next.js" src="https://img.shields.io/badge/Next.js-15-black" />
  <img alt="Postgres" src="https://img.shields.io/badge/PostgreSQL-16-336791" />
  <img alt="License" src="https://img.shields.io/badge/license-MIT-green" />
</p>

> No booking, no payment, no visa/legal advice — it plans and prices, you book where you like. Prices are real-source when a page is reachable, otherwise a clearly-labeled `~` estimate; the AI is never allowed to invent a price or a rating.

## Features

- **Cost-optimal routing** — cheapest multi-city / multi-country sequence (TSP: Held–Karp for small sets, NN + 2-opt for larger), evaluating round-trip and open-jaw and showing the cheaper as the plan plus the other as an alternative.
- **All transport modes** — flight, train, bus, ferry and car/taxi/transfer, per-leg selectable, with a round-trip return leg priced for the same mode.
- **Real-source prices + ready links** — multi-source fetch + AI extraction with a verbatim guardrail (no invented numbers); side-by-side source comparison, each with a **prefilled** deep-link (route + exact dates + passengers); unreal modes are marked "unavailable" and fall back to the cheapest feasible one.
- **Day-by-day roadmap** — structured, location-based itinerary (never raw markdown): connected timeline + map pins, every day filled with real places grouped by proximity, each with free/paid + `~`price + Google Maps link; editable (remove / add / swap).
- **Hotels, POIs & food** — per-city hotel tier selector with live re-pricing + multiple prefilled booking-site links; multi-source POIs (OSM + AI) with real-only ratings; an approximate per-city food estimate.
- **Country mode** — type a country, get AI-ranked city suggestions (reason + best season), pick what you like, and it flows into the normal plan.
- **Multilingual** — EN (default) / AZ / RU / TR UI, language-aware country autocomplete; AI replies in the chosen language.
- **LAN access** — open it from your phone on the same Wi-Fi with zero extra config.
- **Live cost** — transport / hotels / activities / food, total and per-person, updating as you change nights, order, tier or passengers; PDF/print export includes the full roadmap.

## Tech stack

- **API:** Go (Fiber) — modular monolith, split-ready domains (`services/{gateway,planner,connectors,knowledge,ai,pricefetch,dataflow}`).
- **Web:** Next.js 15 + TypeScript + Tailwind (`web/`), Leaflet maps.
- **DB:** PostgreSQL 16 (multi-tenant-ready). **Cache:** Redis.
- **LLM:** provider-agnostic — **Groq** (default, free) · Gemini · Claude. Runs token-less too (AI endpoints return 503 when no key).
- **Shared:** `pkg/{config,db,cache,auth,tenant,models,httpx}`.

## Quickstart

```bash
cp .env.example .env            # fill values (all optional to boot)
docker compose -f deploy/docker-compose.yml up -d   # Postgres + Redis + api + web
```

That's it — open **http://localhost:3000**. The API applies its embedded DB migrations automatically on boot (idempotent, fail-fast); no `make`/`psql` step. Health: `GET http://localhost:8080/health` · readiness: `GET /ready`.

Run pieces locally instead:

```bash
make run-api                         # API outside the container
cd web && npm install && npm run dev # web on http://localhost:3000
```

### Enable AI (optional, free)

Put a key in the repo-root `.env` (gitignored) and bring the stack up — nothing else. Default provider is **Groq** (free, OpenAI-compatible):

```bash
echo 'GROQ_API_KEY=...' >> .env      # LLM_PROVIDER=groq is the default
docker compose -f deploy/docker-compose.yml up -d
```

Switch provider by setting `LLM_PROVIDER` to `gemini` (+ `GEMINI_API_KEY`) or `claude` (+ `ANTHROPIC_API_KEY`). Without a key the stack still runs; AI endpoints return `503 "AI not configured"`.

## Configuration

All settings live in the environment — see [`.env.example`](.env.example) for the complete, documented list. Key ones:

| Variable | Purpose |
|---|---|
| `LLM_PROVIDER` | `groq` (default) · `gemini` · `claude` |
| `GROQ_API_KEY` / `GEMINI_API_KEY` / `ANTHROPIC_API_KEY` | provider key (in `.env` only) |
| `GROQ_MODEL` / `GROQ_FAST_MODEL` | big model (roadmap/chat) + fast model (price extraction) |
| `PRICE_SOURCES` | real-price sources (`name \| tos-note \| url-template`); empty = labeled estimates |
| `DATABASE_URL` / `REDIS_URL` | datastores (local dev Docker by default) |
| `CORS_ORIGINS` | browser allowlist (`*` in dev) |
| `NEXT_PUBLIC_API_BASE_URL` | empty = auto-derive from page host (LAN); set to pin a fixed API URL |
| `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` | optional run-summary notifications; skipped silently when unset |

Secrets go in `.env` only (gitignored) — never commit keys. Real prices are the operator's ToS responsibility; blocked/anti-bot pages are skipped, never bypassed.

## Access on your network (LAN)

The web app resolves the API from the page's own host at runtime (no hardcoded `localhost`), the gateway binds `0.0.0.0:8080`, the web binds `0.0.0.0:3000`, and CORS defaults to `*` — so a phone/tablet on the same Wi-Fi works after `up -d`:

1. Find the host's LAN IP — macOS `ipconfig getifaddr en0`, Linux `hostname -I | awk '{print $1}'`, Windows `ipconfig` → "IPv4 Address".
2. On the other device (same Wi-Fi), open `http://<lan-ip>:3000`; it calls the API at `http://<lan-ip>:8080` automatically.

To pin a fixed API, set `NEXT_PUBLIC_API_BASE_URL`; to restrict CORS, set `CORS_ORIGINS`. The LAN IP is not a secret.

## Monorepo layout

```
services/   Go app (gateway = HTTP entry; planner/connectors/knowledge/ai/pricefetch/dataflow = domains)
pkg/        shared libs (config, db, cache, auth, tenant, models, httpx)
db/         SQL migrations + scripts (auto-applied on boot)
deploy/     docker-compose + Dockerfiles
web/        Next.js frontend
```

## Contributing

Contributions are welcome. Please:

1. Fork and branch from `main`.
2. Keep both sides green — `go test ./...` and `cd web && npm run build`.
3. Follow the existing style; keep secrets in `.env`; never commit keys or personal data.
4. Open a PR with a clear description of the change and its rationale.

Migrations apply only to the local dev Docker Postgres — never point them at a shared/prod database.

## License

[MIT](LICENSE) — do anything, no warranty. © the AI Travel Platform contributors.

## Support

If this project is useful to you, consider supporting its development — see [`.github/FUNDING.yml`](.github/FUNDING.yml) (maintainer: set your own funding link there).
