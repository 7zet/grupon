# Grupon handover

Grupon is a polished **demonstration** of a local-deals voucher marketplace for Tashkent, based on the Dealo PRD (`docs/DEALO-REQUIREMENTS.md`). It has no real users, no real money and no legal or compliance delivery. One owner can try every role: buyer, merchant owner, manager, cashier and admin (see `docs/DEMO.md`).

Stack: Go API and worker (`services/api`), Next.js web (`apps/web`: buyer site, `/merchant/*`, `/admin/*`), PostgreSQL, OpenAPI contract in `packages/api-spec/openapi.yaml`.

## Deviations from the PRD (DEV-01 … DEV-12)

| ID | Deviation | Why / what to know |
|---|---|---|
| DEV-01 | Go API + Next.js web instead of Django + DRF + aiogram + Django Admin. | The admin panel is part of the Next.js app. |
| DEV-02 | No real PSP, fiscal receipt, MXIK or package code. A mock provider (demo only), a manual-confirmation mode and a PSP port exist. | A real provider is the first launch task. |
| DEV-03 | Hosting is Vercel + Render + Neon, outside Uzbekistan. | Fine for a demo with no real personal data; a real launch must host in Uzbekistan. |
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
- **Rate limits and Origin check.** Cookie-authenticated changes from a browser origin other than `WEB_PUBLIC_URL` are refused (`origin_not_allowed`).

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
- Uploaded media and documents live on the API host's disk. On Render's free plan that disk is erased on redeploy.
- The free hosting sleeps when idle, and background jobs pause while it sleeps.
- Some chargeback and order/voucher states exist in the enums but no flow moves into them.

## Running the hosted demo vs. a real production

The hosted demo stays in demo mode: `DEMO_MODE=true`, `APP_ENV` not `production`, `SMS_PROVIDER=log`, no Redis. Rate limits are then off, which is acceptable only because there are no real users.

A real production must:

1. Set `APP_ENV=production`. The API then refuses to start without `DATABASE_URL`, strong secrets and **`REDIS_URL`**; OTP rate limits fail closed (requests are refused when the limiter is unavailable).
2. Set `DEMO_MODE=false` and use a real SMS provider (`SMS_PROVIDER=eskiz` with its credentials). `SMS_PROVIDER=log` is refused outside demo mode.
3. Set `WEB_PUBLIC_URL` to the exact web origin.
4. Replace the mock payment with a real PSP, add fiscal receipts, and host in Uzbekistan.
5. Finish legal work (contracts, privacy, regulation), set up backups and monitoring.

## Next steps for a real launch

PSP and reconciliation → SMS provider → hosting in Uzbekistan with persistent storage → legal gates → bank-change protection → ticket system → notifications by SMS/email.
