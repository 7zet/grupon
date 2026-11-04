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

**Status:** In progress.

**Done:**

- 2026-11-04: `DEMO_MODE` now gates OTP echo, mock payment provider/actions, and the public header badge; `SMS_PROVIDER=log` remains the default.

**Open, in order:**

1. Build and verify Docker images, compose runtime, migrations/seed, browser smoke flows, and backup/restore.
2. Add one-service API/worker mode and a verified-current free-hosting deployment guide.

## Stage 3 — Telegram bot and support contacts

**Goal:** Give demo users an easy way to discover the product and ask for help.

**Finish line:** Telegram bot flows and visible support contacts are complete and tested.

**Status:** Not started.

## Stage 4 — Polish and demo quality

**Goal:** Make every role’s walkthrough clear, reliable, localized, and presentation-ready.

**Finish line:** Demo scripts, visual polish, edge states, and cross-role browser tests are complete.

**Status:** Not started.

## Stage 5 — SEO, final documentation, handover, v1.0

**Goal:** Package the project for a maintainable v1.0 handover.

**Finish line:** SEO essentials, final operational documentation, a handover checklist, and `v1.0` are complete.

**Status:** Not started.
