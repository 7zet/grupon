# Grupon agent instructions

Grupon is a Go + Next.js local-deals **demo** for Uzbekistan. It will be polished and handed over; it has no real users, real money, or legal/compliance scope. The code is the source of truth for routes, schemas, generated types, and UI behavior.

## Read first

1. `docs/ROADMAP.md`
2. `docs/ACCEPTANCE.md`
3. `docs/DECISIONS.md`
4. `docs/ARCHITECTURE.md` and `docs/ENVIRONMENT.md`

Do not treat archived files as active instructions. Read `../dealo-docs` only when the original requirement is needed.

## Working rules

- Do not spawn subagents. Keep context compact and use `rg` before broad scans.
- Preserve unrelated changes. Never commit `Grupon_hisobot.pdf` or generated E2E screenshots.
- Record durable product decisions in `docs/DECISIONS.md`; update `docs/ROADMAP.md` when stage work changes.
- Do not push unless the owner explicitly asks. Commits include `Co-Authored-By: Claude Haiku 5.5 <noreply@anthropic.com>`.
- Contract first: update `packages/api-spec/openapi.yaml`, regenerate with `corepack pnpm --filter @grupon/api-client run gen`, then run `go generate ./...` in `services/api`. Never hand-edit generated types.
- Go layers are handler → domain → store/sqlc. Money is `int64` tiyin, UZS only. Ledger entries are append-only and double-entry.
- Sensitive merchant/admin actions must audit in the same transaction. Stock reservation is transactional and conditional. Errors use `problem+json` with stable codes.
- Localize UI in Uzbek, Russian, and English; use design tokens and `brand.config.ts`; do not add motion to checkout, payment, redemption, merchant forms, or admin flows. Follow `apps/web/AGENTS.md` for web work.
- Authentication remains phone OTP. Voucher rights, redemption safety, and payout/dispute holds remain intact. Do not add real payment, fiscal, tax, bank-protection, legal, or launch work.

## Verification

- Local PostgreSQL uses port 5433: `scripts/dev-db.sh start|stop|status` through Git Bash.
- Go integration tests need `TEST_DATABASE_URL=postgres://app:app_dev_password@localhost:5433/app?sslmode=disable`.
- Run focused tests for each change. At stage close run Go tests, Go vet, web lint, typecheck, web unit tests, and Playwright when browser flows change.

## Current stage — Stage 3: Telegram bot and support contacts

Stage 2 code is done at `v0.4`; deployment is manual and owned by the project owner using `docs/DEPLOY.md`. Work on `feat/stage-3`; do not push unfinished work to `main`. The site must work fully without Telegram and without real SMS.

1. Support contacts from env (`SUPPORT_PHONE`, `SUPPORT_TELEGRAM`, `SUPPORT_EMAIL`): footer, `/help` page, voucher page, Uzbek/Russian/English, never hardcoded.
2. Bot basics (Go): `/start` with language choice, `/help`, `/support`; no `TELEGRAM_BOT_TOKEN` means disabled; long polling locally and webhook on hosting with Telegram secret-token-header verification.
3. Account linking: “Connect Telegram” in the buyer profile → one-time deep link (single use, 10 min) → saved `chat_id`; unlink in profile and via `/stop`; audit-logged.
4. Notifications through the existing outbox (linked users only): voucher issued; voucher redeemed with a “Bu men emasman” button that opens a dispute through the existing idempotent dispute logic; reminders 7 and 1 day before expiry without duplicates; `/vouchers` command with links to the site.
5. Login code via Telegram (Dealo M01): “Get the code in Telegram” on the login page; bot requests a contact button, verifies it belongs to the same Telegram user, then sends the OTP under the same rules as SMS.
6. SMS adapter (Eskiz) behind `SMS_PROVIDER=eskiz`, credentials only from env; default remains `SMS_PROVIDER=log`; retain 60-second phone cooldown and IP limits; add daily IP and global daily caps from env; Telegram selection sends no SMS.
7. Bot menu button “Open site” → `NEXT_PUBLIC_SITE_URL` inside Telegram (WebApp button). Full Mini App login (`initData`) and support tickets are not included and must be recorded in `docs/HANDOVER.md`.
8. Tests with fakes only (no real Telegram/SMS in CI): link-token single-use and expiry, dispute-button idempotency, reminder de-duplication, wrong contact rejection, OTP cooldown, SMS caps, and bot disabled without a token.
9. `docs/TELEGRAM.md`: create the bot in @BotFather, set the webhook on Render, and configure env vars. Never commit tokens.

After each task: run related tests, commit, push to `feat/stage-3`, and add one `docs/ROADMAP.md` line. At close: full checks, green CI, merge to `main`, tag `v0.5`, push.
