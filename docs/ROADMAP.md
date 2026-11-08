# Grupon roadmap

NEXT: Stage 4b is on `feat/stage-4b`, CI green; waiting for the owner's "merge" (then tag `v0.8`) and "bootstrap done". After that start Stage 5 on `feat/stage-5` from main (see `docs/FINISH-PLAN.md` section 5).

This is the single source of truth for delivery status. Acceptance evidence lives in
`docs/ACCEPTANCE.md`; durable product decisions live in `docs/DECISIONS.md`.

## Stage 0 — Repository order

**Status:** Done. Reference material, local database tooling, isolated E2E accounts, and the `v0.1` baseline are in place.

## Stage 1 — Demo-ready product

**Status:** Done. Buyer, merchant, cashier, admin, voucher, finance, category, dispute, and wizard workflows are covered; AT-08 is deferred to the excluded real-payment work and AT-17 to the excluded bank-protection work.

## Stage 2 — Finish: demo mode, Docker verification, free hosting preparation

**Goal:** Make a polished, safe-to-demo build that one owner can run and hand over.

**Finish line:** Demo-only switches work, the full Docker smoke/backup-restore run is verified, and one documented free hosting setup can deploy the web, API/worker, and database.

**Status:** Code done (`v0.4`); deployment in progress — manual, done by the owner with `docs/DEPLOY.md`.

**Done:**

- 2026-11-04: `DEMO_MODE` now gates OTP echo, mock payment provider/actions, and the public header badge; `SMS_PROVIDER=log` remains the default.
- 2026-11-05: Native purchase/merchant Playwright runs in CI as advisory after two failed attempts; readiness succeeds, but its test failure remains recorded in CI logs.

## Stage 3 — Support contacts and SMS adapter

**Goal:** Give demo users visible support contacts and an optional real-SMS adapter.

**Finish line:** Configured contacts and the Eskiz adapter are complete and tested; the default stays local log delivery.

**Status:** Done (`v0.6`). Telegram dropped.

**Ordered tasks:**

1. Support contacts from configuration in the footer, help page, and voucher page — done.
2. Eskiz SMS adapter with cooldown, rate limits, daily caps, and fake-server tests — done.
3. Telegram bot, Telegram OTP, notifications and Mini App — dropped by decision; WIP is archived at `archive/telegram-wip`.

## Stage 4 — Polish and demo quality

**Goal:** Make every role’s walkthrough clear, reliable, localized, and presentation-ready.

**Finish line:** Demo scripts, visual polish, edge states, and cross-role browser tests are complete.

**Status:** In progress — Stage 4a: G0–G4 done, G5 docs in progress. Follow `docs/FINISH-PLAN.md`.

**Current work:** G0 was released as `v0.5` and Stage 3b as `v0.6`; G1–G4 are on `feat/stage-4a-fixes`.

**Delivery order (2026-11-06):**

1. `feat/stage-4a` G0: account menu, identity badges, demo-account picker, clear seed names, and `docs/DEMO.md`; green CI, merge to `main`, tag `v0.5`.
2. Rebase `feat/stage-3b` on that `main`; finish Eskiz SMS wiring/daily caps, Telegram cleanup, and documentation; green CI, merge to `main`, tag `v0.6`.
3. Create a new branch from updated `main` for audit groups G1 through G5.

**2026-11-07 — Stage 4a G1:** QR refresh follows server token expiry; checkout accepts one variant only; late and dispute refunds require support proposal then a different finance approver; finance denial/self-approval is audited; native Playwright is blocking and installs Chromium in CI. Migration: `00029_refund_proposals.sql`.

**2026-11-07 — Stage 4a G2:** voucher issuance is sequenced and unique per order item; redemption requires a branch attached to the deal; partial and recently expired vouchers refund only their remaining floor-split value and commission remainder goes to the final visit; a reversal after payout becomes merchant receivable and is offset from the next payout. Migrations: `00030_voucher_sequence.sql`, `00031_payout_receivable_offset.sql`.

## Stage 5 — SEO, final documentation, handover, v1.0

**Goal:** Package the project for a maintainable v1.0 handover.

**Finish line:** SEO essentials, final operational documentation, a handover checklist, and `v1.0` are complete.

**Status:** Not started.

**2026-11-07 — Stage 4a G3/G4:** log SMS only with `DEMO_MODE`; production requires Redis and fails OTP limits closed; Origin check on cookie sessions; `audit_logs` append-only (migration `00032_audit_logs_append_only.sql`); CI security job; masked phones. Merchant sees the rejection reason; deal pause/resume/archive; merchant suspend/terminate/reinstate; admin overview and fraud-signal pages; price proof required on submit (demo merchants are seeded with a placeholder); discount over 70% flagged; calculator warns on negative net income.

**2026-11-08 — Stage 4b:** Inter font loads everywhere; curated demo catalog (`seed/catalog.json`: 30 real-sounding businesses with Tashkent addresses, 32 deal templates, one deal per business and offer, distinct option names and texts, 60 deals); homepage sections show different deals in whole rows; category icons; empty ratings hidden; validity in days; weekly grouped opening hours; quantity stepper above Buy; role-aware merchant panel nav and correct role badge; mobile overflow fixes; message-key tests. No new migrations since v0.7.
