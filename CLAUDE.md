# Grupon — agent guide

Local deals marketplace for Uzbekistan. Customers buy vouchers, merchants redeem them by QR at their
branches, admins moderate and pay out. The product follows the Dealo PRD (business rules and screens);
the technical stack is ours (Go + Next.js). **The code is the source of truth** for schema, routes,
tokens and components. Do not redesign what exists.

## Read first (in this order)

1. `docs/STATUS.md` — what is built, what is next.
2. `docs/DECISIONS.md` — decisions and assumptions; add one line per new decision.
3. `docs/DEALO-GAP.md` — PRD gaps and what to add or remove.
4. `docs/ARCHITECTURE.md` — layers and where code lives. `docs/ENVIRONMENT.md` — settings.

## Stack

- API and worker: Go (chi, pgx, sqlc, goose, river). Telegram bot: Go.
- Web: Next.js App Router, TypeScript strict, next-intl (uz default, ru, en), Tailwind + shadcn/ui,
  TanStack Query, Zustand, react-hook-form + Zod.
- Data: PostgreSQL, Redis, S3-compatible storage (MinIO locally).
- Monorepo: pnpm + Turborepo. Commands: `pnpm dev`, `pnpm test`, `pnpm typecheck`, `go test ./...`.

## Hard rules

1. **Contract first:** edit `packages/api-spec/openapi.yaml`, run `pnpm gen` (TS) and `go generate ./...`
   in `services/api`, then write code. Never hand-write API types.
2. **Go layers:** handler (HTTP only) → domain (rules) → store (sqlc). Authorization in middleware
   and in the domain. Errors are problem+json with a stable `code`.
3. **Money:** int64 tiyin (minor units), UZS only. No floats. Display only through `formatMoney`.
4. **Ledger:** append-only, double-entry. Corrections are reversals, never updates.
5. **Audit:** every admin action and every sensitive merchant action (deal edit, redemption, staff,
   payouts) writes an `audit_logs` row.
6. **Stock:** reserve with one conditional `UPDATE ... WHERE available >= n`, inside a transaction.
   Overselling must be impossible (there is a concurrency test).
7. **UI:** no hard-coded strings (i18n uz/ru/en, Uzbek Latin with o' and g'), colors only from tokens.
   Brand name only from `brand.config.ts`. Deal red only for the discount badge, Buy button and
   urgency. Prices are always ink. No animations in checkout, payment, redemption, merchant forms or admin.
8. **Urgency text only from real data** (ends within 72 h, stock ≤ 10, sold ≥ threshold).
9. **Scope:** Toshkent pilot, four categories open (Beauty & Wellness, Food & Drink, Things to Do,
   Health & Fitness). No goods, gifts, shipping, cart or USD. Login is by phone OTP.
10. **Out of scope now:** real payment provider, SMS provider, native app, games, ML, chat, dark mode.
    They come after testing.

## Working rules

- Keep the response short. Do not re-read files you just edited. Find code with `rg` on exact names
  and read only the lines you need.
- Run only the tests related to the change; the full suite at the end of a task group.
- When a task is done, update `docs/STATUS.md` (move the item to Done) and add decisions to `docs/DECISIONS.md`.
- Commits: add the trailer `Co-Authored-By: Claude Haiku 5.5 <noreply@anthropic.com>`. Never commit
  `Grupon_hisobot.pdf` (the owner's own file).
- If something is unclear, write the assumption in `docs/DECISIONS.md` and continue.

## Current stage: 0 then 1

Only the tasks of the current stage are listed here. Stage 2 and later are not started; do not plan for them.
A stage ends with full tests, a short report, a merge and a tag. The next stage starts only after the finish line.

### Stage 0 — order (branch `feat/merchant-and-admin-flows`)
- [x] Dealo reference copy moved out of the repo (`../dealo-docs`); ignored by `.gitignore`.
- [x] `docs/ANALYSIS-DEALO-VS-GRUPON.md` removed (owner decision); `docs/ACCEPTANCE.md` replaces it.
- [x] PostgreSQL data moved to `C:devgrupon-pgdata` (see docs/ENVIRONMENT.md); start with `scripts/dev-db.sh`.
- [x] Playwright runs in parallel: each spec signs in with its own seeded accounts.
- [x] Tag `v0.1` on the branch. Merge into `main` waits for Stage 1 (AT tests passing).

### Stage 1 — demo-ready (code only)
Finish line: every AT in `docs/ACCEPTANCE.md` is ✅, or marked "PSP kerak", "SMS kerak" or "3-bosqich".
- [~] AT tests: 16 of 18 green; AT-08 and AT-17 wait for PSP / 3-bosqich. See docs/ACCEPTANCE.md.
- [~] Dispute: `awaiting_evidence`, `under_review`, `closed` still open. Fraud burst rule sets `fraud_hold` (done, AT-14).
- [x] Late payment writes a `refunds` row (AT-13).
- [ ] Price-proof upload page for merchants (backend exists).
- [ ] Category flags at category level: publishable, purchasable.
- [ ] Merchant redemption history; cashier e2e with its own login.
- [ ] Deal wizard tested end to end in the browser.
- [ ] SMS adapter interface, with `LogSender` as the only implementation until a provider is chosen.
