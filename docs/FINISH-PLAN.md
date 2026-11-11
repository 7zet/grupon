# FINISH PLAN — from the current point to v1.0

> **Status: completed at `v1.0` (2026-11-11).** Kept as the record of how the project was finished. Current status is
> in `docs/ROADMAP.md`; later decisions (Dealo name, role workspaces with "Switch workspace", local demo) are in
> `docs/DECISIONS.md` and win over this file. Still open from this plan: SEO (section 5).

You (Claude Code) take over this project from another agent (Codex) and finish it to v1.0.
This file is your task list. Re-read it at the start of every session.

## 0. What this project is

A **demo** of a local-deals voucher marketplace for Tashkent, based on the Dealo PRD
(`docs/DEALO-REQUIREMENTS.md`). No real users, no real money, no legal work.
Stack: Go API + worker (`services/api`), Next.js web (`apps/web`, buyer + `/merchant/*` + `/admin/*`),
PostgreSQL, OpenAPI contract in `packages/api-spec/openapi.yaml`.
**No hosting (decision of November 2026).** The demo runs on the owner's Windows PC (localhost) and is shared
when needed through one ngrok HTTPS tunnel to the web app. Anything in this file that mentions Vercel, Render or
Neon is superseded by section 6.

Decisions already made (do NOT reopen them):
- Telegram bot is dropped (WIP archived at tag `archive/telegram-wip`). Only `SUPPORT_TELEGRAM`
  contact and the merchant Telegram profile link remain.
- No one-click demo login, no role switcher. The login page no longer lists demo accounts (docs/DEMO.md does).
- Real PSP, fiscal, MXIK/VAT, bank-change protection, legal gates: not included (interfaces only).
- Docker files are optional and unverified.
- SMS: `SMS_PROVIDER=log` by default, Eskiz adapter exists.

## 1. Start of EVERY session (do this first)

1. Read: this file, `AGENTS.md`, `docs/ROADMAP.md`, `docs/AUDIT.md` (sections A–E),
   `docs/DECISIONS.md` (only the last ~40 lines).
2. Run: `git status`, `git branch -a`, `git log --oneline -15`, `git tag`.
3. Find the first task below that is not done (check the code and ROADMAP, not only checkboxes).
4. Continue from there. Do not redo finished work.

## 2. Working rules

- **No subagents.** Do all work yourself in this session.
- Read code with `rg` + only the lines you need. Never read node_modules, generated code,
  lockfiles, `.next`. Do not re-read a file you just edited.
- Contract first: change `openapi.yaml` → regenerate (`go generate ./...` in services/api,
  `pnpm --filter @dealo/api-client run gen`) → code. Never hand-write API types.
- Never hardcode UI strings (uz/ru/en, Uzbek Latin with o' g'), colors (tokens), brand name, cities, categories.
- One branch per stage. Commit after each task with a clear message. Push the branch.
- After each task: run the related tests. After each group: full checks
  (`go test ./...` with TEST_DATABASE_URL, `go vet ./...`, web lint, typecheck, unit tests, Playwright)
  and wait for CI to be green (`gh run watch`).
- Merge to `main` ONLY at the stage close points below, ONLY with CI green
  (main is the release line).
- Forbidden: force-push, rewriting history, deleting branches or tags, touching any remote
  database, using real secrets. You MAY run migrations and bootstrap against the LOCAL database
  (`C:\dev\grupon-pgdata`, see `docs/ENVIRONMENT.md`).
- Every new migration: list it in the stage report.
- If a task needs a decision from the owner (accounts, money, legal, design taste) —
  write it in `docs/DECISIONS.md` under "Open questions", skip it, continue.
- Stop only if: the same error repeats twice, or a forbidden action would be needed.
- **Before your context runs out:** commit, push, and write the exact next step at the top of
  `docs/ROADMAP.md` ("NEXT: ..."). The owner will start a new session and say "resume".
- Reports: max 10 lines per stage close.

### UI rules (apply to every page you touch)
- The current user and role are always visible (badge from G0).
- Every list/page has empty, loading and error states.
- Works at 390 px and 1440 px. No horizontal scroll on mobile.
- No placeholder or seed-looking text ("Special local offer 91", "Seed ko'chasi").
- Red (`--deal`) only for discount badge, Buy button and urgency. Prices in ink. No orange.

## 3. Stage 4a — audit fixes (branch `feat/stage-4a-fixes`, already exists)

Source of findings: `docs/AUDIT.md`. For each finding, verify it in the code first; if the audit
is wrong, write why in DECISIONS.md and skip.

- [x] G0 — account menu, role badges, demo accounts list, clear demo names, docs/DEMO.md (v0.5)
- [x] G1 — QR refresh, one merchant per order, refund 4-eyes, denial audits, blocking E2E
- [x] G2 — voucher seq unique, deal-branch check, reversal offset, partial/expired refunds
- [ ] **G3 — security** (was in progress; check what is already done)
  - C-1: LogSender only when DEMO_MODE=true; otherwise refuse to start with SMS_PROVIDER=log
  - C-2: OTP rate limits fail closed in production when Redis is missing
  - C-5: untrack `.data/documents` from git
  - C-6: Origin check on cookie-authenticated mutations
  - A-12: `audit_logs` append-only (trigger like the ledger)
  - SEC-04: mask phone numbers in the UI
  - C-9: CI: govulncheck, pnpm audit, gitleaks
  - production environment setting (the live API logs "environment: development"); document it
- [ ] **G4 — visible features**
  - A-8: merchant sees the rejection reason
  - deal pause/archive for merchants; merchant suspend/terminate for admins
  - A-11: admin page listing fraud signals
  - FR-02.3: price proof required on submit; discount > 70% flagged for the moderator
  - FR-02.6: calculator warns when net income is negative
  - admin overview: total users, businesses, active deals, orders today
- [ ] **G5 — honest docs**
  - HANDOVER.md: every deviation DEV-01…12, QR token design (FR-04.2), page pagination (API-04),
    409 vs 422 (S-10), Telegram dropped, every ➖ item
  - ACCEPTANCE.md: re-check each AT against the audit; mark partials honestly
  - fix stale lines in DECISIONS.md / ROADMAP.md listed in AUDIT section B
  - DEPLOY.md: final env var table for Render and Vercel (incl. SUPPORT_*, SMS_*)
- [ ] **Stage close 4a:** full checks, CI green → merge into main, tag `v0.7`, push.
  Report: list ALL migrations since v0.6 (00029, 00030, 00031, …) — the owner runs bootstrap on Neon.

## 4. Stage 4b — design and demo content (new branch `feat/stage-4b` from main)

Findings from a live review of the site:

- [ ] **A. Font:** the site renders in a serif fallback (Times-like). Load one sans font with
  Latin + Cyrillic via `next/font` (Inter or Manrope) and apply it everywhere.
- [ ] **B. Realistic demo data** (bootstrap must stay re-runnable and keep demo accounts from G0):
  - real-sounding deal titles per category (e.g. "60-minute relaxing massage"),
    merchant names without numbers, real Tashkent street addresses
  - options with distinct names (e.g. 60 min / 90 min), never equal to the deal title
  - distinct texts for description, highlights and "what's included"
  - category-matched free-license images (Unsplash/Pexels), 3–5 per deal, a large varied pool,
    no mismatches (no spa photo in "Things to Do"); update next.config remotePatterns if needed
  - each deal's category consistent with its merchant and photos
- [ ] **C. Bugs:**
  - the category nav row shows a stray scrollbar/arrows on the right
  - hide the rating when there are 0 reviews (now "0.0 (0)")
  - "Voucher validity: 30" → "30 days" (all locales)
  - opening hours "09:00,20:00" → "09:00–20:00", per weekday
  - uz money format "137 500 so'm" (ru "137 500 сум", en "137,500 UZS")
- [ ] **D. Homepage:**
  - each section shows different deals and fills the row (4 or 8 cards); "Featured" and
    "Popular near you" must not show the same deals
  - category tiles get an icon or photo instead of faded "01, 02"
  - hero search button uses brand indigo (red only for discount badge and Buy)
  - hero side card must not show a placeholder title
- [ ] **E. Deal page:** quantity stepper above the Buy button; option names from data;
  gallery shows only this deal's images.
- [ ] **F. Screenshots + mobile pass:** Playwright screenshots at 390 px and 1440 px of:
  home, category, search, deal, checkout, voucher page, account menu, merchant dashboard,
  merchant redeem, admin moderation, admin finance. Save OUTSIDE the repo
  (`../grupon-artifacts/<date>/`). Fix every layout problem you see; list what you fixed.
- [ ] **G. Copy check:** no untranslated keys, no English text on uz/ru pages, no dead
  strings (e.g. "shipping"). Add a test that fails on missing keys in any locale.
- [ ] **Stage close 4b:** full checks, CI green → merge into main, tag `v0.8`, push.
  Report migrations (if any). (Done, v0.8.)

## 5. Stage 5 — handover (new branch `feat/stage-5` from main)

- [ ] **SEO:** `sitemap.ts` and `robots.ts` per locale, `hreflang` alternates, page metadata
  and Open Graph images for home, category and deal pages; JSON-LD Offer stays valid.
- [ ] **Docs (plain language, short):**
  - `README.md`: what it is, how to run locally (Windows-friendly), demo link
  - `docs/ARCHITECTURE.md`: how the parts work together, main flows
  - (hosting guide dropped: see section 6; `docs/SHARE.md` explains ngrok sharing)
  - `docs/DEMO.md`: every demo account (role, business, phone), admin 2FA note,
    a 10-minute demo script (buyer → cashier redeem → "Siz emasmisiz?" → admin)
  - `docs/HANDOVER.md`: what is included, what is not (and why), known limitations,
    what to do next for a real launch (PSP, SMS provider, hosting in Uzbekistan, legal)
- [ ] **Self-check:** update the "Holat" column of `docs/AUDIT.md` for every item that changed
  since the audit, with evidence. Add a short "v1.0 recheck" summary with new counts.
- [ ] **Final checks:** full test suite, CI green, Playwright green, no secrets in the repo
  (gitleaks), `git status` clean.
- [ ] **Stage close 5:** merge into main, tag `v1.0`, push. Final report (max 15 lines):
  what was delivered, final counts, migrations to run on Neon, env vars to set,
  and the list of "Open questions" for the owner.

## 6. Local demo, no hosting (branch `feat/local-demo` from main) — replaces all hosting parts

Runs on localhost; `scripts/demo.ps1` starts everything; `scripts/share.ps1` helps with ngrok.
Site URL and allowed origins come from env (`SITE_URL`, `ALLOWED_ORIGINS`), default localhost.
Old hosting docs live in `docs/archive/`. Stop before merging to main and report.

