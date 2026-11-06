# Grupon roadmap

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

## Stage 3 — Telegram bot and support contacts

**Goal:** Give demo users an easy way to discover the product and ask for help.

**Finish line:** Telegram bot flows and visible support contacts are complete and tested.

**Status:** In progress.

**Ordered tasks:**

1. Support contacts from configuration in the footer, help page, and voucher page.
2. Telegram bot basics: localized `/start`, `/help`, `/support`, disabled without a token, long-polling locally and verified-secret webhooks for hosting.
3. Buyer account linking and unlinking through one-time, ten-minute Telegram deep-link tokens, with audit logging.
4. Linked-user outbox notifications, redemption dispute action, expiry reminders, and `/vouchers`.
5. Fake-client tests for bot disablement, link tokens, idempotent dispute actions, and reminder de-duplication.
6. `docs/TELEGRAM.md` for BotFather, environment configuration, and Render webhook setup.

## Stage 4 — Polish and demo quality

**Goal:** Make every role’s walkthrough clear, reliable, localized, and presentation-ready.

**Finish line:** Demo scripts, visual polish, edge states, and cross-role browser tests are complete.

**Status:** In progress — G0 demo usability is the release blocker.

**Current work:** G0 items 1–5 are implemented on `feat/stage-4a`; focused web checks are green. CI and browser verification are pending before the `v0.5` merge.

**Delivery order (2026-11-06):**

1. `feat/stage-4a` G0: account menu, identity badges, demo-account picker, clear seed names, and `docs/DEMO.md`; green CI, merge to `main`, tag `v0.5`.
2. Rebase `feat/stage-3b` on that `main`; finish Eskiz SMS wiring/daily caps, Telegram cleanup, and documentation; green CI, merge to `main`, tag `v0.6`.
3. Create a new branch from updated `main` for audit groups G1 through G5.

## Stage 5 — SEO, final documentation, handover, v1.0

**Goal:** Package the project for a maintainable v1.0 handover.

**Finish line:** SEO essentials, final operational documentation, a handover checklist, and `v1.0` are complete.

**Status:** Not started.
