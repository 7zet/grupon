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

**Holat:** Done.

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
- Stage-close verification (2026-11-04): full Go tests, Go vet, web lint, web typecheck, and 6/6 web unit tests pass.
- Full Playwright stage-close rerun (2026-11-04): 17/17 passed with four workers in 31.0 seconds. Non-fatal
  Next.js image warnings were logged for localhost media SSRF protection and external Unsplash timeouts.
- Disputes enforce the full state machine and role split; every transition is audited atomically, payout holds
  remain through review, merchant resolution releases the hold, and customer resolution keeps the existing refund flow.
  Verified by `TestDisputeAllowedTransitionsAuditAndPayoutHold`, `TestDisputeCustomerResolutionKeepsRefundBehavior`,
  `TestDisputeForbiddenTransitionAndRBAC`, and HTTP 422/403 mapping tests.
- Merchant reviews compute their reply-window reference outside render; the full web lint gate is green.
- Category capabilities are enforced independently for search, deal creation/submission, publishing, checkout,
  and redemption; pilot categories are fully enabled and closed categories retain redemption only.
- Merchant redemption history is available on the Redeem page with masked buyers, reversal state, and cashier-only
  row scoping; owners and managers see the full merchant history.
- Cashier B's browser flow signs in independently, redeems by code, sees the row in scoped history, and is denied
  direct access to merchant payouts and platform finance.
- Merchant onboarding E2E uploads certificate, bank letter, and optional price proof before submission; the admin
  review queue exposes the private price-proof document to reviewers.
- The six-step deal wizard is covered end to end: validation, live preview, options, three media selections,
  localized details, branch/schedule, and browser submission ending in `in_review`.

### Open

None. Stage 1 meets its finish line.

## Bosqich 2 — Pilot-tayyor

**Maqsad:** Make the demo operable by a small real pilot without moving real money through an integrated PSP.

**Finish line:** Production hosting and observability are ready; real SMS, Scanner PWA, privileged-user 2FA,
and a documented manual-payment operating mode pass pilot checks.

**Holat:** In progress.

- 2026-11-04: Mandatory TOTP 2FA completed for admin, moderator, support, finance, and superadmin: QR enrollment, ten one-time recovery codes, audited lifecycle/recovery use, and an eight-hour session ceiling; merchant owners remain out of scope.
- 2026-11-04: Manual payment pilot completed behind `MANUAL_PAYMENT_ENABLED` (default off): configurable hold, buyer-facing instruction payload, admin confirmation with reference/note and audit, and webhook-equivalent idempotent voucher/ledger settlement.
- 2026-11-04: Scanner PWA completed for `/merchant/redeem`: install manifest, scanner icon, online-only redemption screen, and offline navigation fallback. Camera access on a real phone requires HTTPS; verify after staging deploy.
- 2026-11-04: Portable hosting preparation completed: production API/worker/migrate/web images, compose file, backup/restore scripts, health/readiness endpoints, and deployment runbook. Docker daemon is unavailable on this machine, so image builds and runtime checks are unverified.
- 2026-11-04: SMS provider selection completed with `SMS_PROVIDER=log` as the default and the existing `CodeSender` interface preserved. No real SMS provider was implemented; provider choice is required before pilot launch.

Task list:

1. SMS adapter.
2. Hosting preparation.
3. Scanner PWA.
4. TOTP 2FA for admin and finance roles.
5. Manual payment confirmation mode.

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
