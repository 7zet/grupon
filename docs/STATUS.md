# Status

Last updated: 2026-11-02. Progress is an estimate: about half of the Dealo PRD is covered.
Earlier history is in git (`git log -- docs/`).

## Done

- **Foundation:** monorepo, Go API and worker, Next.js web, contract-first OpenAPI, PostgreSQL schema
  with migrations 00001–00009, seed data (Toshkent open, four pilot categories).
- **Public site:** home, categories, deal page (option, price, savings, red Buy button), search,
  business page, coupons listing. Lighthouse on the production build: performance 98–100, accessibility 100.
- **Purchase:** phone OTP sign-in (login is required at checkout only), orders with idempotency key,
  mock payment, vouchers with 30-second QR rotation, backup code, multi-use per option.
- **Money:** double-entry ledger, redemption and commission, payout with four-eyes approval,
  refunds (automatic within 14 days, support approval after), disputes (customer or merchant
  resolution), breakage, reconciliation, outbox relay, fraud signals (burst, off-hours, distance).
- **Removed (DEALO-ALIGN step 2):** cart, saved deals, promo codes, collections, banners, exchange
  rate, gifts, goods, shipping, email/password login, Google OAuth.

- **Merchant onboarding and review (FR-01):** STIR, legal form, tax regime, bank MFO and account,
  responsible person, offer acceptance (time and IP). Certificate and bank letter uploads (private storage).
  Reviewers move `submitted` → `in_review` → `approved` or `rejected`, with an audit row per decision.
  Deal creation requires `approved`. Admin queue page `/admin/merchants`; merchant status page `/merchant/review`.

## Done since the review

- PRD state machines aligned (deal, order, payment, voucher, dispute, merchant). Categories set to the PRD list (4 open pilot categories). Public /coupons page removed. Remaining: dispute transitions beyond open and resolve, chargeback flow, fraud_hold rule, the late-payment refund record.

- Redemption reversal rule and support reversal endpoint; 15-minute hold; 10-character backup codes; buyer notifications with "not me" dispute (tested end to end on localhost).
- Merchant panel sections built and checked in the browser: staff (owner invites by phone, role change, removal), payouts with CSV statement, orders, customers, analytics (date range), reviews with one reply (editable 24 h), settings, deal calculator. Price proof is a third optional document kind.
- Cleanup: `users.password_hash` and `google_sub` dropped (login is phone code only); `currency_code` is UZS only (migration 00018 refuses to run while any USD row exists).
- Playwright suite: 16 of 16 pass when run one at a time (`workers: 1`). Cashier redemption is covered only inside the owner's redemption step, not as a separate cashier login.

## In progress

- **Merchant panel** (`/merchant/*`): deal wizard exists but its six steps and the live preview are not
  verified; redemption page not verified on mobile. Full target in `docs/DEALO-GAP.md` section 3.4.

## Next (in order)

1. Show the rejection reason to the merchant on the status page (API change needed).
2. Merchant panel: Deal Studio (quotas, blackout, terms template), reports. Price proof upload and the
   campaign calculator page are done; staff invites are done (browser test covers the page, not the invite flow).
3. Redemption page for cashiers on a mobile viewport with clear states (used, expired, wrong merchant, refunded).
4. Admin panel: moderation queue, merchants, orders, finance pages, audit log viewer.
5. Run the Playwright suite again (not run since the finance and removal changes).
6. Channels: Telegram Mini App and bot (Go), scanner PWA.
7. Hardening: k6 flash-sale script, OWASP checklist, Dockerfiles, environment docs.

## Full comparison with Dealo

See `docs/ANALYSIS-DEALO-VS-GRUPON.md`: every Dealo module, rule, state, endpoint and test compared with the code.

## Known gaps against the PRD

- Hold period is 10 minutes; PRD says 15.
- Backup code is 8 characters (`XXXX-XXXX`); PRD says 10 characters, Crockford base32.
- Search has no Cyrillic/Latin transliteration.
- No payment provider, no SMS provider (development logs codes), no legal gates (G1–G9).
