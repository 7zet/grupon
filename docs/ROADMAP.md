# Dealo roadmap

Status as of `v1.0` (2026-11-11). This is the single source of truth for delivery status. Acceptance evidence is in
`docs/ACCEPTANCE.md`, product decisions in `docs/DECISIONS.md`, version history in `CHANGELOG.md`.

**Next:** open items are at the end of this page. The original PRD gaps are listed in `docs/HANDOVER.md`.

## Stage 0 — Repository order (`v0.1`)

**Done.** Reference material, local database tooling, isolated E2E accounts, the first baseline.

## Stage 1 — Demo-ready product (`v0.2`)

**Done.** Buyer, merchant, cashier, admin, voucher, finance, category, dispute and wizard flows. AT-08 needs a real PSP
and AT-17 needs bank-change protection; both are outside the demo.

## Stage 2 — Demo mode (`v0.3`, `v0.4`)

**Done.** `DEMO_MODE` gates the sign-in code on the page, mock payments and the header badge; native Playwright runs
in CI. The free-hosting part was dropped in November 2026: the demo runs locally (`scripts/demo.ps1`, `docs/SHARE.md`).

## Stage 3 — Support contacts and SMS (`v0.6`)

**Done.** Support contacts from configuration; Eskiz SMS adapter with limits (default stays log delivery). Telegram was
dropped by decision (archived at tag `archive/telegram-wip`).

## Stage 4 — Polish and demo quality (`v0.5`, `v0.7`, `v0.8`, `v0.9`)

**Done.**

- `v0.5`: account menu, role badges, `docs/DEMO.md`.
- `v0.7` (audit fixes): QR refresh, refund four-eyes, voucher sequence, branch check on redeem, payout receivable
  offset, demo-only SMS log, Redis in production, Origin check, append-only audit log, CI security scans, phone
  masking, deal pause/resume/archive, business suspend/reinstate, fraud and overview pages, price proof, calculator.
  Migrations `00029`–`00032`.
- `v0.8`: curated Tashkent catalog (30 businesses, 60 deals), homepage rows, mobile fixes, translation tests.
- `v0.9`: hosting removed; one-command local demo; cached public pages; search synonyms; translated subcategories.

## Stage 5 — Handover (`v1.0`)

**Done.**

- Product renamed to Dealo.
- Role workspaces (Dealo Business, Dealo Kassa, Dealo Admin), each role on its own home.
- Green design, motion, layout smoke tests.
- Search by street address, with addresses on deal cards.
- Demo story in the seed.
- Docs: README, DEMO, HANDOVER, CHANGELOG, AUDIT recheck.
- No new migrations since `v0.7`.

## Open items

- **SEO:** no `sitemap`, `robots`, `hreflang` alternates or Open Graph images yet. Page titles and descriptions exist.
- **Not built from the PRD:** see "Not yet built" in `docs/HANDOVER.md` (for example expiry reminders,
  prohibited-offer rules, bank-change protection, admin screens for categories, cities and commission).
- **Real launch:** PSP, SMS provider, server and domain in Uzbekistan, Redis and `APP_ENV=production`, legal work
  (`docs/HANDOVER.md`, "Next steps").
