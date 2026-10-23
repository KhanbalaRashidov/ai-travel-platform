# Launch Checklist

## Secrets & config
- [ ] `JWT_SECRET` set to a strong, unique value (not the dev default).
- [ ] AI key set for the chosen `LLM_PROVIDER` (or accept AI-off / 503).
- [ ] `PRICE_SOURCES` reviewed; operator accepts each source's ToS.
- [ ] `.env` is NOT committed; `.env.example` lists every key.
- [ ] `NEXT_PUBLIC_API_BASE_URL` points at the deployed API.

## Data
- [ ] Migrations `0001..0003` applied to the target DB (scripts only).
- [ ] Redis reachable; `GET /ready` returns ok.
- [ ] Starter destinations seeded (automatic on first boot).

## Security
- [ ] CORS restricted to the web origin in production (`CORS_ORIGINS`).
- [ ] Rate limits enabled on AI/scrape endpoints.
- [ ] No secrets in code/commits/logs (key masking verified).
- [ ] TLS terminated at the proxy; security headers set.

## Quality
- [ ] `go test ./...` green; `npm run build` green.
- [ ] `/plan` opens reliably (ErrorBoundary + null guards).
- [ ] Prices show `source` + `fetched_at`; estimates marked `~ indicative`.
- [ ] Itinerary day count matches the trip's real nights.
- [ ] i18n: AZ/EN/RU/TR render; LLM replies follow the UI language.

## Legal / UX
- [ ] "Advisory only, indicative prices, no booking" disclaimer visible.
- [ ] `docs/LEGAL.md` linked from the footer.
- [ ] Attribution shown for OpenStreetMap map tiles.

## Ops
- [ ] Health checks wired to the orchestrator/monitor.
- [ ] Structured logs shipped; AI token/cost + analytics observable.
- [ ] Backups for Postgres configured.
