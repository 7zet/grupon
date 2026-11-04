# Grupon agent instructions

Grupon is a Go + Next.js local-deals marketplace for Uzbekistan. Buyers purchase vouchers,
merchants redeem them at branches, and admins moderate deals and finance operations. The code is
the source of truth for schemas, routes, generated types, and UI behavior.

## Read first

1. `docs/ROADMAP.md` — current stage, completed work, and ordered open work.
2. `docs/ACCEPTANCE.md` — AT-01…AT-18 evidence.
3. `docs/DECISIONS.md` — chronological decisions; history may describe superseded states.
4. `docs/ARCHITECTURE.md` and `docs/ENVIRONMENT.md` — structure and local configuration.

Do not use archived documents as current instructions. The Dealo reference copy is outside this
repository at `../dealo-docs`; read it only when a task needs the original product requirement.

## Token and working rules

- Do not spawn subagents. Work in the current agent only.
- Keep responses and context compact. Use `rg` and read only relevant ranges after the initial audit.
- Do not rescan the repository when `docs/ROADMAP.md` and the touched area are enough.
- Make reasonable, reversible assumptions; record a new product decision in `docs/DECISIONS.md`.
- Preserve unrelated working-tree changes. Never commit `Grupon_hisobot.pdf` or generated E2E screenshots.
- Do not push unless the owner explicitly asks. Push only `feat/merchant-and-admin-flows`.
- Commits use: `Co-Authored-By: Claude Haiku 5.5 <noreply@anthropic.com>`.
- Keep `docs/ROADMAP.md` current when stage work changes. Keep evidence in `docs/ACCEPTANCE.md`.

## Hard engineering rules

1. Contract first: edit `packages/api-spec/openapi.yaml`, regenerate the TypeScript client with
   `corepack pnpm --filter @grupon/api-client run gen`, and run `go generate ./...` in `services/api`.
   Never hand-edit generated API types.
2. Go layers are HTTP handler → domain → store/sqlc. HTTP maps requests and errors; domain owns rules
   and authorization; store owns persistence.
3. Money is `int64` tiyin, UZS only. No floats. Display through `formatMoney`.
4. Ledger entries are append-only and double-entry. Corrections are reversals, never updates.
5. Admin actions and sensitive merchant actions must write an audit row in the same transaction.
6. Stock reservation is one conditional update inside a transaction. Overselling must remain impossible.
7. Errors use problem+json and stable machine-readable codes.
8. UI text is localized in Uzbek, Russian, and English. No hard-coded UI strings; use design tokens and
   `brand.config.ts`. No motion in checkout, payment, redemption, merchant forms, or admin flows.
9. Urgency copy must come from real data only.
10. On Next.js work, also follow the generated rules in `apps/web/AGENTS.md`.

## Business rules

- Pilot scope: Toshkent; Beauty & Wellness, Food & Drink, Things to Do, Health & Fitness.
- No cart, goods, gifts, shipping, USD, password login, or Google login. Authentication is phone OTP.
- A purchased order item snapshots title, price, and terms; later deal edits cannot alter the agreement.
- Voucher expiry uses Asia/Tashkent; multi-use vouchers remain usable until the final visit.
- Redemption is merchant-scoped, concurrency-safe, audited, and blocked for invalid status or holds.
- Existing voucher redemption rights survive later category closure.
- Payouts use redemption-based payable amounts, disputes hold the affected redemption, and approval
  follows four-eyes separation.
- Real PSP, fiscalization, MXIK/VAT, and bank-account-change protection belong to Stage 3.

## Verification

- Database: PostgreSQL on port 5433; helper `scripts/dev-db.sh start|stop|status` via Git Bash.
- Go integration tests require
  `TEST_DATABASE_URL=postgres://app:app_dev_password@localhost:5433/app?sslmode=disable`.
- Focused tests first; at the end of a task group run `go test ./... -count=1`, `go vet ./...`, web
  typecheck, and web unit tests. Run Playwright when browser flows change.
- E2E rewrites `docs/screenshots/*.png`; do not commit those incidental changes.

## Current stage — Stage 1: Demo-tayyor

Goal and finish line are defined in `docs/ROADMAP.md`. Work only on these ordered open items:

1. Complete dispute transitions: `open → awaiting_evidence → under_review → resolved_* → closed`,
   with authorization, audit/state history, and tests.
2. Add category-level `publishable` and `purchasable` flags; keep existing vouchers redeemable.
3. Implement `GET /merchant/redemptions` (the contract exists but the handler/store do not), then add
   a cashier E2E that signs in as Cashier B and performs redemption.
4. Add full browser coverage for all six deal-wizard steps, validation, live preview, and submission.
5. Add E2E coverage for the existing merchant `price_proof` upload UI and private-document flow.
6. Run the full four-worker Playwright suite after these changes; keep screenshots unstaged.

Do not start Stage 2 work until Stage 1 meets its finish line.
