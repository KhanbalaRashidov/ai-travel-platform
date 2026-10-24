# Deploy (single server, behind nginx)

This deploys the whole stack with Docker Compose and fronts it with **nginx that
you install and configure on the server by hand**. The nginx config and the real
`.env` are intentionally **not** in this repo — only `.env.example` is committed.

The app's containers bind to `127.0.0.1` only (web `WEB_PORT=3100`, api
`API_PORT=8090`), and Postgres/Redis are not published at all, so nothing clashes
with services already on the box (e.g. an existing Postgres on `5432`) and nothing
is reachable publicly except through nginx. The API runs its embedded DB
migrations automatically on boot against its own fresh database (idempotent).

## 1. Clone

```bash
git clone https://github.com/KhanbalaRashidov/ai-travel-platform.git
cd ai-travel-platform
```

## 2. Configure env

```bash
cp .env.example .env
```

Edit `.env` (it is gitignored — never commit it). For a public deploy behind
nginx on, say, `travel.myalfasoft.com`:

```ini
APP_ENV=production

# Host ports (loopback only; nginx fronts them). Change if they clash.
BIND_HOST=127.0.0.1
API_PORT=8090
WEB_PORT=3100

# Auth — set a real secret
JWT_SECRET=<a-long-random-string>

# AI (optional). Groq is free; leave empty to run without AI (503 on AI routes).
LLM_PROVIDER=groq
GROQ_API_KEY=<your-key-or-empty>

# Browser calls the API same-origin through nginx:
NEXT_PUBLIC_API_BASE=same-origin
NEXT_PUBLIC_SITE_URL=https://travel.myalfasoft.com

# Tighten CORS to your domain (same-origin needs no CORS, but set it anyway):
CORS_ORIGINS=https://travel.myalfasoft.com
```

> `NEXT_PUBLIC_*` are inlined into the web bundle at **build** time, so set them
> before the build step below (they flow in as compose build args).

## 3. Build & start

Run from the repo root so the root `.env` drives port interpolation:

```bash
docker compose --env-file .env -f deploy/docker-compose.yml up -d --build
```

Verify locally on the server:

```bash
curl -s http://127.0.0.1:8090/health     # {"status":"ok",...}
curl -sI http://127.0.0.1:3100/          # 200 from the web app
```

## 4. Install nginx by hand (example — do NOT commit this)

Put a server block like the following at `/etc/nginx/sites-available/travel.conf`
(or your distro's conf.d), then enable it. This file lives **on the server only**,
never in the repo:

```nginx
server {
    listen 80;
    server_name travel.myalfasoft.com;
    # certbot will manage the 80->443 redirect + TLS once a cert is issued.

    # Web app (Next.js)
    location / {
        proxy_pass         http://127.0.0.1:3100;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_set_header   Upgrade           $http_upgrade;
        proxy_set_header   Connection        "upgrade";
    }

    # API (Go gateway) — same-origin /api/* so the browser never needs CORS
    location /api/ {
        proxy_pass         http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_read_timeout 120s;   # AI calls can take tens of seconds
    }
}
```

Enable + reload, then issue TLS with certbot:

```bash
sudo ln -s /etc/nginx/sites-available/travel.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d travel.myalfasoft.com     # provisions + redirects to HTTPS
sudo systemctl reload nginx
```

## 5. DNS

Point an `A` record for `travel.myalfasoft.com` at the server's public IP. Once it
propagates, certbot (step 4) can validate and the site is live.

## 6. Open

Visit `https://travel.myalfasoft.com`. The browser loads the web app and calls
`/api/v1/*` on the same origin; nginx proxies those to the API container.

## Updating

```bash
cd ai-travel-platform
git pull
docker compose --env-file .env -f deploy/docker-compose.yml up -d --build
```

## Notes

- Secrets live only in `.env` on the server (gitignored); never commit keys.
- To expose Postgres for inspection, set `POSTGRES_PORT` in `.env` and uncomment
  its (loopback) mapping in `deploy/docker-compose.yml`. Leave it unset otherwise.
- Migrations target the stack's own containerized Postgres — never point
  `DATABASE_URL` at a shared/production database.
