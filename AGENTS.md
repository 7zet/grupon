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

## Current stage — Stage 2: finish

1. Demo mode: gated OTP display, test payments, and header badge.
2. Docker verification: images, compose, migrations/seed, smoke flows, and backup/restore.
3. Free hosting preparation: embedded worker mode and one current, documented free deployment setup.
