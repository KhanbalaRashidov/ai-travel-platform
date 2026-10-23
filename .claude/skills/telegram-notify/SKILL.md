---
name: telegram-notify
description: Hər task/epic tamamlananda (commit+push+Trello-dan SONRA) həmin iş dövrünün TAM hesabatını — chat-da verilən eyni üslub və məzmunla, qısaldılmamış — mövcud Node.js Telegram bridge ilə Telegram-a göndər.
---

# telegram-notify

Hər iş dövrünün (task/epic) sonunda görülən işin **tam hesabatını** Telegram-a göndər. Yeni bot/token yaratma — **mövcud bridge**-i təkrar işlət.

## Nə vaxt
Bir epic/task tamamlanıb **commit + push + Trello kartı Done**-a keçiriləndən SONRA. Yəni iş dövrü bitən kimi, son addım kimi.

## Mesajın məzmunu
Chat-da istifadəçiyə verdiyin **eyni tam hesabat**, olduğu kimi (qısaltma YOX):
- `<Epic> icra olundu ✅ və push edildi`
- `Commit:` hash + yoxlama (gofmt / vet / test nəticələri)
- `Nə quruldu` bölməsi (bullet-lar)
- `Trello → Done` kartları
- varsa `🎓 növbəti kapsul` (AZ izah + interview EN termin/cümlə)
- `Növbəti addım`

## Necə göndərilir
Bridge: `C:\Users\XANBALA\Desktop\telegram-claude-bot` (polling bot + `send.js` outbound CLI).
`send.js` həmin layihənin MÖVCUD `.env`-ini oxuyur (`TELEGRAM_BOT_TOKEN` + `ALLOWED_CHAT_IDS`) — token burada **saxlanmır/təkrarlanmır**. Mətn stdin-dən verilir, >4096 olsa ~3800-lik bütöv-sətir hissələrə bölünüb "(1/N)" başlığı ilə ardıcıl göndərilir (truncate yox), plain text.

Addımlar:
1. Hesabat mətnini bir fayla yaz (məs. scratchpad-də `report.txt`).
2. Göndər:
   ```bash
   node "C:\Users\XANBALA\Desktop\telegram-claude-bot\send.js" < report.txt
   ```
   (və ya `echo "..." | node ".../send.js"`).
3. `send: delivered N message(s)` çıxışını yoxla; uğursuzsa xətanı istifadəçiyə bildir.

## Qaydalar
- Token/chat_id yalnız bridge-in `.env`-indən gəlir; kodda hardcode yox, yeni secret yaratma.
- `send.js` bridge layihəsinə aiddir (ayrı repo) — onu commit etmə. Travel repo-da yalnız bu SKILL.md commit olunur.
- Bu bildiriş commit/push/Trello-nu ƏVƏZ ETMİR — onlardan sonra əlavə addımdır.
