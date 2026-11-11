# Changelog

Dealo is a demo of a local-deals voucher marketplace for Tashkent. Each version below is a git tag.

## v1.0 — handover

- Product renamed to **Dealo** everywhere (brand, packages, order numbers `DL-…`).
- Role workspaces: buyer marketplace, **Dealo Business** (owner, manager), **Dealo Kassa** (cashier), **Dealo Admin**
  (moderator, support, finance, superadmin); each role lands on its own home and sees only its own menu.
- Green design: new colours, Manrope headings, ticket logo, hero motion, scroll reveal, "How Dealo works", dark
  business sidebar with groups, kassa scan button, success and error screens.
- Search by street address, with address suggestions; deal cards show the address.
- Business dashboard chart with values and dates; empty comparisons hidden.
- Demo story in the seed: deals ending soon, moderation queue with a new business, an open dispute, payout history,
  cashier redemptions today. Docs: README, DEMO, HANDOVER, AUDIT recheck.
- Fixes: profile icon no longer sends signed-in users to login; account reviews page; search suggestions
  (server error and CORS); staff two-factor sign-in in the demo; layout tokens restored; layout smoke tests.

## v0.9 — local demo, no hosting

- Hosting dropped; one-command local demo (`scripts/demo.ps1`) and sharing with ngrok (`docs/SHARE.md`).
- `SITE_URL` and `ALLOWED_ORIGINS` from env; cached public pages; search synonyms; translated subcategories;
  404 page for unknown URLs.

## v0.8 — demo catalog and mobile polish

- 30 realistic Tashkent businesses and 60 deals; varied reviews; homepage rows without repeats; Inter font;
  rating and opening-hours fixes; mobile layout fixes; copy test for all three languages.

## v0.7 — audit fixes

- Security: demo-only SMS log, Redis required in production, Origin check, append-only audit log, CI scans.
- Business pause/resume/archive, business suspend/reinstate, fraud signals and admin overview, campaign calculator,
  price proof for deal submission, phone masking.

## v0.6 — SMS and Telegram scope

- Eskiz SMS adapter with daily limits; Telegram scope closed (work archived at tag `archive/telegram-wip`).

## v0.5 — demo roles

- Easier demo roles, deal wizard linked to media upload, configured API images.

## v0.4 — native CI demo

- Native (no Docker) demo bootstrap and Playwright run in CI.

## v0.3 — stage 2 complete

- Production configuration template, installable cashier scanner, SMS sender selection, admin two-factor sign-in.

## v0.2 — stage 1 demo-ready

- Catalog with category capabilities, deal wizard, checkout with term snapshots, vouchers, cashier redemption and
  history, disputes, reviews; end-to-end tests for the main flows.

## v0.1 — foundation

- Go API and worker, localized Next.js web, PostgreSQL, OpenAPI contract, local infrastructure and CI.
