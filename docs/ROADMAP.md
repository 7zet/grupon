# Grupon roadmap

This file is the single source of truth for where the project is. Detailed acceptance evidence is in
`docs/ACCEPTANCE.md`; durable decisions are in `docs/DECISIONS.md`.

## Bosqich 0 — Tartib

**Maqsad:** Repository, reference material, local database, test accounts, and parallel E2E are orderly.

**Finish line:** Reference docs are outside the repo, local tooling is documented, E2E accounts do not collide,
and baseline `v0.1` exists.

**Holat:** Done.

- Dealo reference moved to `../dealo-docs`; obsolete analysis removed from the active docs.
- PostgreSQL test data moved to `C:\dev\grupon-pgdata`; helper script documented.
- Playwright uses separate seeded accounts and four workers.
- Tag `v0.1` exists at the Stage 0 baseline.

## Bosqich 1 — Demo-tayyor

**Maqsad:** The mock-payment product demo covers the agreed buyer, merchant, admin, voucher, and finance rules.

**Finish line:** Each AT is green, or deferred with a named stage (`needs PSP` / `Stage 3`); all Stage 1 browser and workflow gaps below are closed.

**Holat:** In progress.

### Done — verified in code and tests

- AT tracker: 16/18 green. AT-08 is deferred as `needs PSP`; AT-17 is deferred to Stage 3.
- Phone OTP, merchant onboarding/review, deal moderation, mock purchase, voucher issuance/redemption,
  refund, payout four-eyes, fraud hold, dispute payout hold, late-payment refund, and terms/price snapshots.
- Oversell and concurrent redemption protections have integration tests.
- Four pilot categories are open; closed categories are refused by the deal gate (AT-10).
- Merchant panel includes staff, payouts/CSV, orders, customers, analytics, reviews, settings, calculator,
  media, redeem, and a six-step deal wizard with live preview.
- `price_proof` upload is present in the merchant review form and private-document backend.
- SMS delivery has the `identity.CodeSender` port; `sms.LogSender` is the only runtime implementation.
- Latest verified non-browser gates (2026-11-04): full Go tests, Go vet, web typecheck, and 6/6 web unit tests pass.

### Open — ordered

1. Complete dispute state transitions `awaiting_evidence`, `under_review`, and `closed`, with audit/history tests.
2. Add category-level `publishable` and `purchasable` flags; do not block existing voucher redemption.
3. Implement merchant redemption history. `GET /merchant/redemptions` is in OpenAPI but has no handler/domain/store implementation.
4. Add a cashier redemption E2E using Cashier B's own login; current E2E only checks cashier deal denial and owner redemption.
5. Test the complete six-step deal wizard, live preview, validation, and submission in the browser; current E2E only opens step one.
6. Add browser coverage for the existing `price_proof` upload UI.
7. Rerun the full four-worker Playwright suite after the recent AT commits; the last verified 16/16 run predates them.

## Bosqich 2 — Pilot-tayyor

**Maqsad:** Make the demo operable by a small real pilot without moving real money through an integrated PSP.

**Finish line:** Production hosting and observability are ready; real SMS, Scanner PWA, privileged-user 2FA,
and a documented manual-payment operating mode pass pilot checks.

**Holat:** Not started.

Scope: SMS provider, Uzbekistan hosting/data residency, shared object storage, Scanner PWA, 2FA for privileged
roles, manual payment mode, operational monitoring, backups, and release/security checklists.

## Bosqich 3 — Real pul

**Maqsad:** Process and reconcile real customer money compliantly.

**Finish line:** A selected PSP, fiscal receipts, MXIK/VAT data, reconciliation, refund/chargeback operations,
and 72-hour bank-account-change protection pass sandbox and production-readiness tests.

**Holat:** Not started.

Scope: PSP integration, fiscalization, MXIK/VAT, bank-account history/change hold (AT-17), real reconciliation
(AT-08), chargebacks, and finance/legal controls.

## Bosqich 4 — O'sish

**Maqsad:** Expand acquisition, support, discovery, and channels after the pilot is stable.

**Finish line:** Prioritized growth capabilities have owners, metrics, production monitoring, and acceptance tests.

**Holat:** Not started.

Scope: Telegram Mini App and bot, support/ticketing, SEO and structured data, transliteration and richer search,
additional cities/categories, marketing consent, and later product experiments.
