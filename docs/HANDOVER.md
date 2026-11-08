# Grupon handover

Grupon is a polished **demonstration** of a local-deals voucher marketplace for Tashkent, based on the Dealo PRD (`docs/DEALO-REQUIREMENTS.md`). It has no real users, no real money and no legal or compliance delivery. One owner can try every role: buyer, merchant owner, manager, cashier and admin (see `docs/DEMO.md`).

Stack: Go API and worker (`services/api`), Next.js web (`apps/web`: buyer site, `/merchant/*`, `/admin/*`), PostgreSQL, OpenAPI contract in `packages/api-spec/openapi.yaml`.

## Deviations from the PRD (DEV-01 … DEV-12)

| ID | Deviation | Why / what to know |
|---|---|---|
| DEV-01 | Go API + Next.js web instead of Django + DRF + aiogram + Django Admin. | The admin panel is part of the Next.js app. |
| DEV-02 | No real PSP, fiscal receipt, MXIK or package code. A mock provider (demo only), a manual-confirmation mode and a PSP port exist. | A real provider is the first launch task. |
| DEV-03 | No hosting at all. The demo runs on the owner's PC and is shared with an ngrok tunnel. | Hosting is not included. A real launch needs a server and a domain, in Uzbekistan for personal data (see below). |
| DEV-04 | Legal gates G1–G9 do not apply. | Demo only. |
| DEV-05 | Support is contacts only (phone, Telegram, email from settings). No ticket system or SLA (FR-08.4). | |
| DEV-06 | No Telegram Mini App and no Telegram bot. | Dropped by decision. The unfinished work is kept only at tag `archive/telegram-wip`. |
| DEV-07 | Category capability flags exist per category only, not per category and city. | See migration 00025. |
| DEV-08 | Languages are uz (Latin), ru, en. No Cyrillic Uzbek. | |
| DEV-09 | `DEMO_MODE=true` shows the sign-in code on the page, enables mock-payment buttons and shows a header badge. | Must be off in a real launch. |
| DEV-10 | Reviews are included (PRD marks them later). | |
| DEV-11 | No booking, CRM, referral, advertising, loyalty or gift codes. | |
| DEV-12 | Dockerfiles and `docker-compose.prod.yml` exist but are not verified. | Deployment and CI use native runtimes. |

## Design notes you may be asked about

- **Voucher QR token (FR-04.2).** The token is `base64url("gv2:<voucher id>:<window>") "." base64url(HMAC-SHA256)`. The window is the number of 30-second intervals since the Unix epoch, so a screenshot goes stale; the previous window is also accepted for clock skew. The voucher page refreshes the QR using the server expiry. The human backup code (10 characters, `XXXXX-XXXXX`) is derived from the voucher id and `VOUCHER_QR_SECRET`, so no plain code is stored. If that secret leaks, every backup code leaks; a real launch should store random hashed codes.
- **Pagination (API-04).** Lists use `page` and `page_size` (default 24, maximum 100) and return `total`. There are no cursors.
- **409 and 422 (S-10).** 409 means the request conflicts with the current state (for example `invalid_deal_state`, `voucher_on_hold`, `merchant_not_approved`, `documents_locked`). 422 means the input or a rule is violated (validation errors, an expired or refunded voucher at redemption). Errors are `application/problem+json` with a stable `code`.
- **Money.** `int64` tiyin, UZS only. The ledger is append-only and double-entry. `audit_logs` is append-only too (database trigger).
- **Price proof.** Submitting a deal needs the business to have a price proof document. Demo merchants get a generated placeholder PDF marked DEMO, so the demo needs no upload.
- **Rate limits and Origin check.** Cookie-authenticated changes from a browser origin other than `SITE_URL` or `ALLOWED_ORIGINS` are refused (`origin_not_allowed`).

## Not included (interfaces ready)

- Real PSP integration, reconciliation (AT-08), chargebacks and real-money procedures.
- Fiscalization, MXIK/VAT data, receipt compliance.
- Bank-account change protection and its delayed approval (AT-17).
- Legal, regulatory, privacy and merchant-contract work; a real pilot launch.
- Telegram bot, Telegram OTP, notifications and Mini App (dropped).
- Ticket system and SLA, booking, CRM, ads, loyalty, gifts, Cyrillic Uzbek, category × city flags.
- Admin screens for managing categories, cities and users (these exist only through seed data).
- Email or SMS notifications for redemptions and expiry reminders (an in-app notification exists).

Everything marked ➖ in `docs/AUDIT.md` is in this list or in the deviation table above.

## Known limitations

- The payout hold from the burst fraud rule stops the whole merchant's payouts, not only the burst amount.
- Fraud rules `off_hours` and `distance` only record signals (visible on the admin fraud page); only the burst rule holds payouts.
- Uploaded media and documents live on the local disk of the machine that runs the API. A real deployment needs persistent storage or object storage.
- The demo is only reachable while the owner's PC is on and the demo script is running.
- Some chargeback and order/voucher states exist in the enums but no flow moves into them.

## Hosting is not included

There is no Vercel, Render or Neon setup any more (the old guide is in `docs/archive/`). The next owner chooses a server and a domain. Personal data of people in Uzbekistan must be stored in Uzbekistan, so pick a local provider or data centre. The API and the web app are plain native programs (`go build`, `next build` + `next start`) with a PostgreSQL database; the Docker files are optional and unverified.

## Running the demo vs. a real production

The demo runs in demo mode: `DEMO_MODE=true`, `APP_ENV` not `production`, `SMS_PROVIDER=log`, no Redis. Rate limits are then off, which is acceptable only because there are no real users. Start it with `scripts/demo.ps1`; share it with `scripts/share.ps1` and `docs/SHARE.md`.

A real production must:

1. Set `APP_ENV=production`. The API then refuses to start without `DATABASE_URL`, strong secrets and **`REDIS_URL`**; OTP rate limits fail closed (requests are refused when the limiter is unavailable).
2. Set `DEMO_MODE=false` and use a real SMS provider (`SMS_PROVIDER=eskiz` with its credentials). `SMS_PROVIDER=log` is refused outside demo mode.
3. Set `SITE_URL` to the exact public web address (and `ALLOWED_ORIGINS` for any extra origin).
4. Choose a server and domain in Uzbekistan, replace the mock payment with a real PSP, and add fiscal receipts.
5. Finish legal work (contracts, privacy, regulation), set up backups and monitoring.

## Next steps for a real launch

PSP and reconciliation → SMS provider → server and domain in Uzbekistan with persistent storage → legal gates → bank-change protection → ticket system → notifications by SMS/email.
