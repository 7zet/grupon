# Grupon — agent instructions

Local deals marketplace for Uzbekistan. Users buy vouchers, redeem by QR at merchants. Merchants create deals (moderated). Admins run the platform.
Foundation, DB, OpenAPI, public site and catalog API already exist. **The code is the source of truth** for schema, tokens, routes and components. Do not redesign what exists.

## Token rules (strict)
- **Never spawn subagents** (Task/Agent/Explore). Do all work yourself in this session.
- Do not scan the repo. Find code with `grep -rn` / `rg` on specific names, then read only the needed lines (`sed -n 'X,Yp'`). Never read generated files, lockfiles, `node_modules`, migrations you don't need.
- Do not re-read a file you just edited.
- Run only tests related to the change. Pipe long output: `... 2>&1 | tail -40`. Full suite only at the end of a task group.
- No Lighthouse, k6 or full Playwright runs unless the task says so.
- One task per turn. When done: tick it in `docs/TASKS.md`, reply in ≤5 lines (what changed, test result). Then stop and wait.
- If blocked or unclear: write the assumption in `docs/DECISIONS.md` (1 line), continue.

## Stack (fixed)
Turborepo + pnpm. `apps/web` (Next.js App Router, TS strict: public + `/account/*` + `/merchant/*`), `apps/admin` (superadmin), `services/api` (Go: chi, pgx, sqlc, goose, river), Postgres, Redis, MinIO. Tailwind + shadcn/ui, TanStack Query, Zustand, nuqs, react-hook-form + Zod, next-intl (uz default, ru, en), Motion/GSAP.

## Hard rules
1. **Contract first**: edit `packages/api-spec/openapi.yaml` → `pnpm gen` → code. Never hand-write API types.
2. Never hardcode UI strings (i18n uz/ru/en, Uzbek Latin with o' g'), colors (tokens), brand name (`brand.config.ts`), cities/categories.
3. Money: int64 minor units, base UZS, always charged in UZS; USD display-only (`formatMoney`). No floats.
4. Colors: brand indigo; deal red only for discount badge, Buy button, urgency. Prices always ink. No orange. Errors = error token + icon.
5. Urgency texts only from real data (ends <72h, stock ≤10, sold ≥ threshold).
6. No animations in checkout, payment, redemption, merchant forms, admin.
7. Go layers: handler (HTTP only) → domain (rules) → store (sqlc). RBAC in middleware **and** domain. Errors = problem+json with stable `code`.
8. Every admin action and sensitive merchant action (deal edit, redemption, staff, payouts) writes an `audit_logs` row.
9. Out of scope: real payment providers, SMS, native app, games/coins, ML, chat, dark mode.

## Key business rules
- **Stock**: reserve with one conditional `UPDATE ... WHERE stock_total - stock_sold - stock_reserved >= n` in a transaction; reservation 10 min; worker releases expired. Overselling must be impossible.
- **Orders**: `POST /orders` with Idempotency-Key → `pending_payment`. MockProvider page has "Simulate success/failure" and calls the same webhook a real provider would. Webhook is idempotent: success → `paid`, stock sold, one voucher per unit, email; failure → order stays pending until reservation expires, failed page offers retry.
- **Login** is only required at checkout; after login return to the same deal + option.
- **Vouchers**: status `active|redeemed|expired|refunded|cancelled`. QR holds an HMAC-signed token, never a raw id; human code like `7KQ4-M9TX` as fallback. Only staff of that voucher's merchant can redeem. Voucher page cached for offline.
- **Reviews**: only for redeemed vouchers, one per voucher, stars + text. Merchant: one reply per review (editable 24h), can report a review.
- **Deals**: merchant self-registers (no approval); every deal goes `draft → pending_review → active|scheduled|rejected(reason)`. Editing price/terms/options of an active deal sends it back to review; stock-only edits don't. Prices live on deal options.
- **Staff roles**: owner (all), manager (all except payouts/staff), cashier (redemption only).

## Task list
Keep `docs/TASKS.md` as a copy of this list with ticks. First, check existing code briefly and tick what is already done. Work top to bottom.

**A. Finish purchase flow**
- [ ] A1 Auth: Google OAuth (PKCE) + email/password (argon2id), verify email, reset password, refresh-token rotation, rate limits
- [ ] A2 Account pages: coupons (active/used/expired), coupon QR page, orders, saved, reviews, profile
- [ ] A3 Cart (guest Zustand + server, merged on login) and checkout page (savings green, gift fields, shipping only for goods, UZS total + USD hint)
- [ ] A4 MockProvider + webhook + reservations + vouchers + QR + confirmation email (mailpit) + success (confetti) / failed pages
- [ ] A5 Worker jobs: release reservations, expire vouchers, deal status transitions, rating/sold recompute
- [ ] A6 Tests: Go concurrency test (100 buyers, 10 stock → exactly 10 succeed); one Playwright test: browse → login mid-flow → buy → voucher visible

**B. Merchant panel (`/merchant/*`)** — menu: Overview, My Business, My Deals, Create Deal, Orders & Sales, Voucher Redemption, Customers, Analytics & Reports, Payouts & Finance, Reviews, Media Library, Settings
- [ ] B1 Merchant register + role guards + "Switch to business" in account menu
- [ ] B2 Overview (sales +%, vouchers sold, redeemed + rate, active deals, chart, latest orders, pending deals) and My Business (profile, locations, hours, map pin)
- [ ] B3 Media Library (presigned S3 upload) + Create Deal wizard (6 steps: basics, options/pricing, media 3–8, details, locations/schedule, review) with live preview using real card/deal components; save draft; submit to review
- [ ] B4 My Deals (status filter, duplicate, pause), Orders & Sales, Customers
- [ ] B5 Voucher Redemption, mobile-first: camera scan (@zxing/browser) + manual code → result → confirm; clear states (already redeemed, expired, wrong merchant, refunded)
- [ ] B6 Analytics (charts, CSV export), Payouts (read-only calculations), Reviews (reply/report), Settings + Staff invites
- [ ] B7 Test: cashier cannot open finance pages; one Playwright test: create deal → approve via API → buy → redeem on mobile viewport

**C. Polish**
- [ ] C1 Review write flow after redemption + review reports
- [ ] C2 Gift purchase (recipient name/email/message) and goods (shipping address, fulfillment status)
- [ ] C3 PWA (Serwist): installable, offline voucher page
- [ ] C4 SEO: sitemap per locale, robots, JSON-LD on deal page, hreflang; accessibility pass on main pages

**D. Superadmin (`apps/admin`, plain shadcn, dense tables)**
- [ ] D1 Overview + Moderation Queue (approve/reject with reason, preview)
- [ ] D2 Users, Businesses & Merchants (block/suspend), Deals & Coupons, Orders & Transactions, refunds via MockProvider
- [ ] D3 Categories, Cities, Promo codes, Collections, Campaigns & Banners, Homepage Builder (sections, ordering, schedule), exchange rate
- [ ] D4 Reviews & Reports, Support & Disputes, Roles & Permissions (admin, moderator, support, finance, superadmin), Audit Logs viewer, System Settings
- [ ] D5 Test: every admin write creates an audit log; moderator cannot access finance

**E. Hardening**
- [ ] E1 Production Dockerfiles, env docs, `docs/ARCHITECTURE.md` in plain language
- [ ] E2 k6 flash-sale script, short OWASP checklist pass, full test suite green
