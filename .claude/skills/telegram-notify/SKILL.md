---
name: telegram-notify
description: Optional — after a task/epic is committed and pushed, post that cycle's full summary to Telegram when (and only when) Telegram env vars are configured. If they are not set, skip silently.
---

# telegram-notify (optional)

Post the full summary of a finished work cycle to Telegram. This is **optional
convenience tooling** — it is never required to build, run, or contribute to the
project, and it must degrade to a no-op when not configured.

## Configuration (env only — no secrets in the repo)
Set these in your local `.env` (gitignored) or CI secrets:

- `TELEGRAM_BOT_TOKEN` — a bot token from @BotFather.
- `TELEGRAM_CHAT_ID` — the chat/channel id to post to.

If either is empty, notifications are **skipped silently**. Tokens live only in
the environment; never hardcode or commit them.

## When
As the last step after a cycle's work is committed and pushed — an extra
notification, not a replacement for commit/push.

## Message content
The same full summary you would report to the user (no truncation): what was
built (bullets), commit hash + checks (build/test), and the next step.

## How to send
Use any minimal sender that reads the token/chat from the environment and posts
`stdin` as the message text (chunking anything over Telegram's 4096-char limit
on whole-line boundaries). For example, a small Node or shell helper that calls
`https://api.telegram.org/bot<token>/sendMessage`. The sender is not part of this
repository; wire your own, or leave the env vars unset to disable.

## Rules
- Token/chat id come only from the environment; never commit a secret or a
  personal path/account.
- Notifications are optional and must no-op when unconfigured.
