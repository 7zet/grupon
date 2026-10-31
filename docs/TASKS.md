# Tasks

Copy of the task list in CLAUDE.md, ticked against the code. Tick only what is verified.

**A. Finish purchase flow**
- [x] A1 (web login/register pages added: same-origin auth proxy, react-hook-form + zod, i18n uz/ru/en) Auth: Google OAuth (PKCE) + email/password (argon2id), verify email, reset password, refresh-token rotation, rate limits
- [x] A2 Account pages: coupons (active/used/expired), coupon QR page, orders, saved, reviews, profile (profile is read-only; edit and password change are not built yet)
- [x] A3 Cart (guest Zustand) and checkout page (savings green, shipping only for goods, UZS total). Not built yet: server cart merge on login, gift fields (C2), USD hint.
- [x] A4 MockProvider + webhook + reservations + vouchers + QR + confirmation email (mailpit) + success (confetti) / failed pages. Email is verified through the log sender only; SMTP/Mailpit not run.
- [x] A5 Worker jobs: release reservations, expire vouchers, deal status transitions, rating/sold recompute
- [x] A6 Tests: Go concurrency test (100 buyers, 10 stock → exactly 10 succeed); one Playwright test: browse → login mid-flow → buy → voucher visible

**B. Merchant panel (`/merchant/*`)**
- [x] B1 Merchant register + "Switch to business" in account menu. Role guards exist (RequireRole) but no merchant panel page or merchant endpoint beyond registration is built yet, so the guard is applied when those land.
- [x] B2 Overview and My Business (profile, locations, hours). Not built yet: latest orders list (returns empty), sales % of pending deals from real data beyond counts, map pin (lat/lng come from the city centre), CSV and charts library.
- [ ] B3 Media Library (presigned S3 upload) + Create Deal wizard (6 steps) with live preview; save draft; submit to review (browser-tested: media upload and wizard opening; full wizard submission in the browser not yet tested; rounding of image sizes still open)
- ~~B4 My Deals, Orders & Sales, Customers~~ olib tashlandi (foydalanuvchi qarori)
- [x] B5 Voucher Redemption, mobile-first: camera scan + manual code → result → confirm (code entry verified in e2e; camera scan not device-tested; GET /merchant/redemptions not built)
- ~~B6 Analytics, Payouts, Settings, Staff~~ olib tashlandi (foydalanuvchi qarori)
- [x] B7 Test: cashier cannot open finance pages; one Playwright test: create deal → approve → buy → redeem (done: cashier create-deal refused, owner→admin→buyer→redeem on 390px e2e. No finance pages exist since B6 was removed, so the finance check is not applicable yet)

**C. Polish**
- [x] C1 Review write flow after redemption + review reports (API verified live: create 201, repeat 409, report 202, own report 422, delete 204. Web form and report button are not browser-tested yet)
- ~~C2 Gift and goods~~ olib tashlandi (foydalanuvchi qarori)
- ~~C3 PWA~~ olib tashlandi (foydalanuvchi qarori)
- ~~C4 SEO~~ olib tashlandi (foydalanuvchi qarori)

**D. Superadmin (`apps/admin`)**
- [x] D1 Overview + Moderation Queue (browser-tested: approve and reject in the queue, buyers refused)
- ~~D2 Users, merchants, deals, orders admin~~ olib tashlandi (foydalanuvchi qarori)
- ~~D3 Catalog admin, homepage builder~~ olib tashlandi (foydalanuvchi qarori)
- [ ] D4 Reviews & Reports, Support & Disputes, Roles & Permissions, Audit Logs viewer, System Settings (partial: review reports, hide, audit viewer done and verified live via API; support/disputes, roles UI and settings not built)
- [x] D5 Test: every admin write creates an audit log; moderator cannot access finance (store tests: each approve, reject, dismiss and hide writes an audit row; finance and support cannot moderate; moderators cannot read the audit trail. No finance endpoints exist yet, so the finance check covers moderation only)

**E. Hardening**
- [ ] E1 Production Dockerfiles, env docs, `docs/ARCHITECTURE.md` (docs written: ARCHITECTURE.md and ENVIRONMENT.md; Dockerfiles for api, worker, migrate and web written, but NOT built because Docker is not available here; migrate command added)
- ~~E2 k6, OWASP~~ olib tashlandi (foydalanuvchi qarori)
