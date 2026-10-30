# Grupon — Full Build Specification

> This file is the single source of truth for building this project.
> Read it fully before writing any code. Re-read the relevant section before starting each phase.
> If something here is unclear or contradictory, write the question and your chosen assumption into `docs/DECISIONS.md` and continue with the most reasonable reading. Never silently invent scope.

---

## 0. How you (the coding agent) must work

1. **Work strictly phase by phase** (Section 17). Do not start a phase until the previous phase's "Definition of done" checklist fully passes.
2. After every phase:
   - `pnpm build`, `pnpm lint`, `pnpm typecheck`, `pnpm test` pass with zero errors.
   - `go build ./...`, `go vet ./...`, `go test ./...` pass.
   - Update `docs/PROGRESS.md`: what was built, what is left, known issues.
3. **Never hardcode**:
   - user-facing strings → always i18n keys (uz, ru, en);
   - colors, radii, spacing → always design tokens (Section 4);
   - the brand name "Grupon" → always `brand.config.ts` (it will be renamed before production);
   - cities, categories, currencies → always config/database.
4. **Contract first**: the OpenAPI spec is the source of truth between Go and TypeScript. Change the spec first, regenerate types, then change code. Never hand-write API types on either side.
5. Prefer boring, well-maintained libraries. Use the latest stable versions at the time of building.
6. Small, focused commits with clear messages, one logical change each.
7. Do not add features that are listed as out of scope (Section 19).

---

## 1. Product summary

Grupon is a local deals marketplace (Groupon-style) for Uzbekistan.

- **Users** browse discounted deals from local businesses, buy a voucher online, and redeem it at the business by showing a QR code.
- **Merchants** (businesses) register, create deals, which go through moderation, then sell vouchers, redeem them by scanning QR codes, and track sales and payouts.
- **Superadmins** moderate deals, manage users, merchants, categories, cities, homepage content, payments and disputes.

Three interfaces:
| Interface | Where | Design priority |
|---|---|---|
| Public site + user account | `apps/web` | Highest — fast, modern, alive, conversion-focused |
| Merchant panel | `apps/web` under `/merchant/*` | High — clean, professional, mobile-friendly (scanner) |
| Superadmin panel | `apps/admin` (separate app, separate subdomain) | Clarity over beauty — plain, dense, obvious |

The site is **dynamic** (real data from the Go API). Mock data exists only as seed data and as MSW mocks during frontend development.

A native mobile app (Expo) will come later. Everything must be built **API-first** so the mobile app can reuse the same API without changes.

---

## 2. Locked decisions

| Area | Decision |
|---|---|
| Monorepo | Turborepo + pnpm workspaces |
| Frontend | Next.js (App Router), React, TypeScript (strict) |
| Styling | Tailwind CSS + CSS variables for design tokens |
| UI kit | shadcn/ui (dashboards, forms, dialogs) |
| Client data | TanStack Query |
| Local state | Zustand (cart, UI state only) |
| URL state | nuqs (filters, sort, pagination, tabs) |
| Forms | react-hook-form + Zod |
| Animation | Motion (primary), GSAP (hero + scroll effects only), React Bits (copied-in components) |
| Carousel | Embla Carousel |
| Tables / charts | TanStack Table, Recharts (shadcn charts) |
| i18n | next-intl — `uz` (Latin script, default), `ru`, `en` |
| PWA | Serwist (service worker, installable, offline shell) |
| QR | `qrcode` (generate), `@zxing/browser` (scan via camera) |
| Maps | Provider-agnostic `MapView` component; Leaflet + OpenStreetMap now, swappable later |
| Backend | Go, `chi` router, `pgx` driver, `sqlc` queries, `goose` migrations |
| Database | PostgreSQL (+ `pg_trgm`, `unaccent` extensions) |
| Cache | Redis |
| Background jobs | `river` (Postgres-backed job queue) |
| File storage | S3-compatible (MinIO locally) |
| API contract | OpenAPI 3.1 in `packages/api-spec` → `openapi-typescript` (TS types) + `oapi-codegen` (Go server interfaces) |
| Auth | Google OAuth + email/password (argon2id). JWT access token (15 min) + rotating refresh token (30 days). Web: httpOnly cookies. Mobile later: Bearer tokens. Same endpoints. |
| Payments | `PaymentProvider` interface. Phase 1: `MockProvider` only. Payme / Click / Uzum later. |
| Currency | UZS is the base and charging currency. USD is display-only, converted with a stored exchange rate. |
| Theme | One light theme only. No dark mode. No theme switcher. |
| Tests | Go: `testing` + testcontainers. Web: Vitest + Testing Library. E2E: Playwright. Load: k6. |
| Local infra | `docker-compose`: postgres, redis, minio, mailpit |

---

## 3. Repository structure

```
grupon/
├── AGENTS.md                  ← this file
├── docs/
│   ├── PROGRESS.md
│   └── DECISIONS.md
├── apps/
│   ├── web/                   ← Next.js: public site + user account + merchant panel
│   └── admin/                 ← Next.js: superadmin panel (Phase 7)
├── services/
│   └── api/                   ← Go backend
│       ├── cmd/api/           ← main HTTP server
│       ├── cmd/worker/        ← background jobs
│       ├── cmd/seed/          ← seed data loader
│       ├── internal/
│       │   ├── http/          ← handlers (implement generated interfaces), middleware
│       │   ├── domain/        ← business logic per module (deals, orders, vouchers, ...)
│       │   ├── store/         ← sqlc generated code + repositories
│       │   ├── payments/      ← PaymentProvider + MockProvider
│       │   ├── auth/
│       │   ├── jobs/
│       │   └── platform/      ← config, logging, redis, s3, clock
│       ├── migrations/
│       └── queries/           ← .sql files for sqlc
├── packages/
│   ├── api-spec/              ← openapi.yaml (source of truth)
│   ├── api-client/            ← generated TS types + typed fetch client
│   ├── ui/                    ← shared components + design tokens
│   ├── config/                ← brand.config.ts, cities, categories, locales, currencies
│   ├── i18n/                  ← message files uz.json, ru.json, en.json
│   └── tsconfig, eslint-config
├── seed/                      ← seed JSON + image URL list
├── infra/
│   └── docker-compose.yml
└── e2e/                       ← Playwright tests
```

Rules:
- The user site and merchant panel live in one Next.js app, separated by route groups `(public)`, `(account)`, `(merchant)` with role guards in middleware **and** on the server (API is the real authority).
- Frontend apps never call the database. Only the Go API does.
- Go API is stateless (no in-memory sessions) so it can scale horizontally.

---

## 4. Design system

### 4.1 Color tokens (the only colors allowed)

| Token | Hex | Use |
|---|---|---|
| `--brand` | `#26328C` | Brand indigo: logo, header accents, nav active state, links, secondary buttons, merchant panel primary |
| `--brand-hover` | `#1E2870` | |
| `--brand-tint` | `#EEF0FA` | Secondary button bg, selected chips |
| `--deal` | `#DC2626` | Deal red: discount badge, **Buy** button, countdown, low-stock text |
| `--deal-hover` | `#B91C1C` | |
| `--deal-tint` | `#FEECEC` | Rare: subtle urgency backgrounds |
| `--save` | `#15803D` | Savings text ("You save …") — only on deal page and checkout |
| `--save-tint` | `#ECFDF3` | Savings pill bg |
| `--rating` | `#F59E0B` | Star icons only (never text) |
| `--error` | `#9F1239` | Form errors, failed states — **always with an icon** so it is never confused with deal red |
| `--error-tint` | `#FFF1F2` | |
| `--ink` | `#18181B` | Primary text, **prices** |
| `--ink-muted` | `#71717A` | Secondary text |
| `--ink-faint` | `#A1A1AA` | Strikethrough old price, placeholders |
| `--border` | `#E7E5E4` | Hairlines |
| `--bg` | `#FAFAF9` | Page background |
| `--surface` | `#FFFFFF` | Cards, panels |

Color rules:
- **60-30-10**: ~60% neutral background, ~30% text/cards, ≤10% color.
- Red is an accent, never a background. Max **one solid red element per visual region** (on the deal page that element is the Buy button).
- **Prices are always `--ink`, bold.** Never red.
- No orange anywhere.
- Photos are the main source of color; keep chrome neutral so photos breathe.
- White text on `--deal` must be ≥14px and weight ≥500.

### 4.2 Typography
- One sans-serif family with Latin + Cyrillic support (e.g. Inter or Manrope via `next/font`).
- Scale: 12 / 14 / 16 / 18 / 22 / 28 / 36 / 48. Body 16px. Prices: 20–32px, weight 600–700.
- Tabular numbers (`font-variant-numeric: tabular-nums`) for all prices and counters.

### 4.3 Shape and spacing
- Radius: 8px controls, 12px cards, 16px large panels and modals.
- Spacing on a 4px grid. Generous whitespace.
- Shadows: very subtle, only on hover and floating elements.

### 4.4 Deal card (approved design — follow exactly)

```
┌──────────────────────────────┐
│ [-40%]                   [♡] │  ← image 4:3, red badge top-left, save button top-right
│          IMAGE               │
├──────────────────────────────┤
│ Spa massage, 60 minutes      │  ← title, max 2 lines, ink, weight 600
│ Lotus Spa · Tashkent · ★ 4.8 │  ← muted; star in --rating, number in muted ink
│ 150 000 so'm                 │  ← price, ink, bold, large
│ 250 000 so'm                 │  ← old price, --ink-faint, strikethrough
│ ⏱ 2 days left · 1 200+ sold  │  ← meta line, muted, small
└──────────────────────────────┘
```
- The whole card is one link. **No button on the card.**
- **No "You save X" line on the card** (badge + old price already say it). Savings appear only on the deal page and checkout.
- Meta line: show "time left" only if the deal ends within 72 hours; show "sold" only if sold ≥ 50. Show nothing rather than fake numbers.
- Hover (desktop): lift 2px + soft shadow + image scale 1.03 (Motion). Respect reduced motion.
- Skeleton loader with the same dimensions.

### 4.5 Motion rules
- Motion (framer-motion successor) for component animations; GSAP + ScrollTrigger only for the homepage hero and scroll storytelling sections; React Bits components copied into `packages/ui` and adapted to tokens.
- Allowed "alive" moments: hero, card hover, section reveal on scroll, number count-up on prices/stats, add-to-saved heart pop, confetti on successful purchase, smooth page transitions.
- **Forbidden**: animations on checkout forms, payment, merchant redemption, admin tables. Those must feel instant.
- All animation code is lazy-loaded and never blocks LCP. Honor `prefers-reduced-motion` everywhere.

### 4.6 Superadmin look
Plain shadcn defaults with brand indigo as primary. Dense tables, clear filters, no decorative animation.

---

## 5. Internationalization and currency

### 5.1 Languages
- Locales: `uz` (default, Latin script), `ru`, `en`. URL prefix strategy: `/uz/...`, `/ru/...`, `/en/...`; `/` redirects by cookie → `Accept-Language` → `uz`.
- Language switcher in header and footer; choice saved in cookie and in user profile.
- **Content translation**: deal title, description, highlights, terms, FAQ, category names have per-locale fields (`title_uz`, `title_ru`, `title_en` or a JSON `{uz, ru, en}` column). Fallback order: requested → uz → ru → en. Merchants must fill at least one language; the UI marks missing translations.
- All UI strings in `packages/i18n`. Write all three languages yourself; Uzbek must be Latin script with correct apostrophes (o', g').

### 5.2 Currency
- All money stored as `int64` in **minor units** with an explicit currency code. Base currency: `UZS`.
- Payments are always charged in UZS.
- USD is display-only: `price_usd = price_uzs / rate`, rate stored in `exchange_rates` table (Phase 1: seeded static rate; later a daily job fetching official Central Bank of Uzbekistan rates).
- Currency switcher in header (cookie). When USD is selected, show "≈ $11.90" and at checkout show a clear note that payment is in so'm.
- Formatting: UZS `150 000 so'm` (uz), `150 000 сум` (ru), `150,000 UZS` (en). USD `$11.90`. Use one shared `formatMoney()` helper.
- Never compute money with floats. Discount percent = rounded integer.

---

## 6. Navigation and categories

Main menu order (exact):

| # | Name (en) | Slug | Notes |
|---|---|---|---|
| 1 | Coupons | `coupons` | Two tabs: **Deals** (voucher deals with coupon-style offers) and **Promo codes** (brand promo codes, see 9.4) |
| 2 | Beauty & Spas | `beauty-and-spas` | |
| 3 | Things To Do | `things-to-do` | |
| 4 | Gifts | `gifts` | Giftable deals; supports "buy as a gift" |
| 5 | Auto & Home | `auto-and-home` | |
| 6 | Food & Drink | `food-and-drink` | |
| 7 | Local | `local` | Local everyday services |
| 8 | Travel | `travel` | Voucher may require booking with merchant |
| 9 | Goods | `goods` | **Physical products delivered to the buyer** (see `Deal.kind = product`) |
| 10 | Seasonal Deals | `seasonal-deals` | Not a category: lists active **collections** (Navruz, New Year, March 8, Black Friday…) managed by admin |

Each category has subcategories (e.g. Beauty & Spas → Massage, Hair, Nails, Facial, Spa day). Seed realistic subcategories for all.

Header: logo · search bar (with suggestions) · city selector · currency · language · saved · cart · account. Mega-menu on desktop, bottom-sheet menu on mobile. Mobile bottom tab bar: Home, Search, Saved, My coupons, Account.

---

## 7. Routes

All public routes are prefixed with `/[locale]`.

### 7.1 Public
| Route | Page |
|---|---|
| `/` | Homepage (city from cookie, default first active city) |
| `/[city]` | City landing (e.g. `/tashkent`) |
| `/[city]/[category]` | Category in city (e.g. `/tashkent/beauty-and-spas`) |
| `/[category]` | Redirect to `/[currentCity]/[category]` |
| `/deal/[slug]` | Deal detail |
| `/business/[slug]` | Merchant profile: info, locations, map, all active deals, reviews |
| `/search?q=` | Search results with the same filters as category pages |
| `/coupons` | Coupons (tabs: deals / promo codes) |
| `/seasonal-deals` and `/collections/[slug]` | Collections |
| `/cart`, `/checkout`, `/checkout/success/[orderId]`, `/checkout/failed/[orderId]` | Purchase |
| `/login`, `/register`, `/forgot-password`, `/reset-password` | Auth |
| `/merchant` | "Sell on Grupon" landing for businesses |
| `/merchant/register` | Merchant sign-up |
| `/help`, `/terms`, `/privacy` | Static content |
| custom 404 / 500 | Friendly, with search and popular deals |

City and category slugs live in one namespace check: the router resolves a first segment as city or category from config; slugs must never collide (validate at build/startup).

### 7.2 User account (auth required)
`/account/coupons` (tabs: active, used, expired), `/account/coupons/[voucherId]` (QR full-screen, brightness hint, works offline once loaded), `/account/orders`, `/account/orders/[id]`, `/account/saved`, `/account/reviews`, `/account/profile` (name, email, phone, language, currency, password, linked Google, delete account).

### 7.3 Merchant panel (role: merchant owner/manager/cashier)
Follow the approved menu exactly:
`/merchant/dashboard` (Overview), `/merchant/business` (My Business: profile, logo, cover, description, locations/branches with hours and map pin), `/merchant/deals` (My Deals), `/merchant/deals/new` and `/merchant/deals/[id]/edit` (Create Deal wizard), `/merchant/orders` (Orders & Sales), `/merchant/redeem` (Voucher Redemption), `/merchant/customers`, `/merchant/analytics` (Analytics & Reports), `/merchant/payouts` (Payouts & Finance), `/merchant/reviews`, `/merchant/media` (Media Library), `/merchant/settings` (incl. **Staff** management).

Overview shows: total sales (+% vs previous period), vouchers sold, redeemed (+ redemption rate), active deals; sales chart; latest orders; deals pending review.

A user who is also a merchant sees "Switch to business" in the account menu.

### 7.4 Superadmin (`apps/admin`, Phase 7)
Follow the approved module list: Platform overview (total users, businesses, active deals, orders today + charts), Users Management, Businesses & Merchants, Deals & Coupons, Moderation Queue, Orders & Transactions, Payments & Payouts, Categories & Attributes, Cities & Locations, Reviews & Reports, Campaigns & Banners, Media Management, Analytics, Support & Disputes, Roles & Permissions, Audit Logs, System Settings.
Plus: **Homepage Builder** (choose which deals/collections appear in each homepage section, ordering, scheduling) and **Exchange rate** setting.

---

## 8. Data model

Use UUIDv7 primary keys, `created_at`, `updated_at` on every table. Soft delete (`deleted_at`) for users, merchants, deals, reviews.

### Identity
- **User**: id, email (unique), password_hash (nullable), google_sub (nullable, unique), name, phone, avatar_url, locale, currency, email_verified_at, status (`active|blocked`), last_login_at.
- **Role assignment**: user_id, role (`user|merchant_owner|merchant_manager|merchant_cashier|admin|moderator|support|finance|superadmin`), merchant_id (nullable).
- **RefreshToken**: id, user_id, token_hash, family_id, expires_at, revoked_at, user_agent, ip.

### Places and taxonomy
- **City**: id, slug, name {uz,ru,en}, lat, lng, is_active, sort_order.
- **Category**: id, slug, name {uz,ru,en}, icon, sort_order, is_active.
- **Subcategory**: id, category_id, slug, name {uz,ru,en}, sort_order.

### Merchants
- **Merchant**: id, slug, legal_name, display_name, description {uz,ru,en}, logo_url, cover_url, phone, website, instagram, telegram, status (`active|suspended`), rating_avg, rating_count, commission_bps.
- **MerchantLocation**: id, merchant_id, city_id, address, lat, lng, opening_hours (JSON per weekday), phone.

### Deals
- **Deal**: id, slug, merchant_id, category_id, subcategory_id, kind (`voucher|product`), title {uz,ru,en}, short_description {…}, description {…} (rich text, sanitized), highlights {…}[] (3–5 bullets), whats_included {…}[], terms {…}, faq [{q:{…}, a:{…}}], redemption_method (`qr|code|qr_or_code`), redemption_instructions {…}, booking_required (bool), is_giftable (bool), status (`draft|pending_review|rejected|scheduled|active|paused|sold_out|expired`), rejection_reason, starts_at, ends_at, voucher_valid_days or voucher_valid_until, max_per_user, sold_count (denormalized), view_count, rating_avg, rating_count, location_ids[] (which branches accept it), featured_rank.
- **DealOption**: id, deal_id, title {…} (e.g. "60 min massage"), original_price, deal_price, currency, stock_total, stock_sold, stock_reserved, sort_order, is_active. **Every deal has ≥1 option. Prices live on options, not on the deal.** Discount % is computed.
- **DealMedia**: id, deal_id, url, alt {…}, sort_order, width, height, blurhash. 3–8 images required for publishing.
- **Collection**: id, slug, title {…}, description {…}, cover_url, starts_at, ends_at, is_active; **CollectionDeal**: collection_id, deal_id, sort_order.
- **PromoCode** (brand coupons, Coupons → Promo codes tab): id, brand_name, brand_logo_url, title {…}, code (nullable for "no code needed"), discount_label {…}, url, category_id, starts_at, ends_at, is_active, click_count, is_verified.

### Commerce
- **Cart** (server-side for logged-in users; Zustand + localStorage for guests, merged on login): items: deal_option_id, quantity, is_gift.
- **Order**: id, number (human-readable, e.g. `GR-2026-000123`), user_id, status (`pending_payment|paid|failed|cancelled|refunded|partially_refunded`), subtotal, total, currency, idempotency_key (unique), shipping_address (JSON, only if any product item), paid_at.
- **OrderItem**: id, order_id, deal_id, deal_option_id, title_snapshot, unit_price_snapshot, original_price_snapshot, quantity, gift_recipient_name, gift_recipient_email, gift_message, fulfillment_status (for products: `processing|shipped|delivered|returned`).
- **StockReservation**: id, deal_option_id, order_id, quantity, expires_at (10 minutes), released_at.
- **Payment**: id, order_id, provider (`mock|payme|click|uzum`), provider_payment_id, status (`created|pending|succeeded|failed|refunded`), amount, currency, raw_payload (JSON), created_at.
- **Voucher**: id, order_item_id, user_id, deal_id, deal_option_id, merchant_id, code (short human code, e.g. `7KQ4-M9TX`, unique), qr_token (signed, see 12.3), status (`active|redeemed|expired|refunded|cancelled`), valid_from, valid_until, redeemed_at, redeemed_by_user_id, redeemed_location_id.
- **Payout**: id, merchant_id, period_start, period_end, gross, commission, net, status (`pending|processing|paid`), paid_at.

### Social
- **Review**: id, deal_id, merchant_id, user_id, voucher_id (required — only redeemed vouchers can be reviewed, one review per voucher), rating (1–5), text (20–2000 chars), locale, status (`published|hidden|pending`), created_at.
- **ReviewReply**: id, review_id (unique — one reply per review), merchant_user_id, text, created_at.
- **ReviewReport**: id, review_id, reporter_user_id, reason, status.
- **SavedDeal**: user_id, deal_id.
- **RecentlyViewed**: client-side (localStorage) for guests; server table for logged-in users (last 30).

### Platform
- **AuditLog**: id, actor_user_id, actor_role, action, entity_type, entity_id, before (JSON), after (JSON), ip, created_at. Written for every admin action and sensitive merchant action (deal edits, redemptions, staff changes, payout settings).
- **ExchangeRate**: base (`UZS`), quote (`USD`), rate, source, effective_at.
- **HomepageSection**: id, key, title {…}, type (`deals|collection|categories|banner`), deal_ids[] or rule (JSON), sort_order, is_active, starts_at, ends_at.
- **Banner**: id, image_url per locale, link, placement, starts_at, ends_at.

Indexes: every foreign key; `deals(status, category_id, ends_at)`; `deals` full-text (per locale `tsvector`) + trigram on titles; `vouchers(code)`; `orders(user_id, created_at)`; `merchant_locations(city_id)`.

---

## 9. Public pages in detail

### 9.1 Homepage (sections in this order)
1. **Hero**: big headline + search bar + popular search chips; animated (GSAP), lightweight, LCP-friendly (text renders first, animation enhances).
2. **Featured deals** (admin-curated carousel).
3. **Trending coupons** (top promo codes + top coupon deals).
4. **Popular near you** (selected city, by sold_count in last 7 days).
5. **Category collections** (one tile per category with icon/photo).
6. **Limited-time deals** (ending within 72h, live countdown).
7. **Recommended for you** (heuristic: categories from recently viewed + saved; fallback: popular).
8. **Recently viewed** (hidden if empty).
Plus a "List your business" banner for merchants and a footer (links, languages, currency, app-coming-soon badge).

### 9.2 Category / city / search listing
- Header: title, breadcrumb, result count.
- Filters (URL state via nuqs): subcategory, price range, minimum rating, discount ≥ X%, "ending soon", city/district.
- Sort: recommended (default), popular, newest, price ↑, price ↓, biggest discount, ending soon.
- Grid: 4 columns desktop, 2 tablet, 1–2 mobile (choose 2 compact if readable).
- Pagination: "Load more" that also updates `?page=` (crawlable).
- Empty state with suggestions.

### 9.3 Deal detail page — order and first-screen rule

**UX law: price, discount and the Buy button must be visible on the first screen, on every device, without scrolling.**

Desktop layout: left ~60% content, right ~40% **sticky purchase panel**.
Mobile layout: gallery (shorter) → title → price block → Buy button inside the first viewport; after scrolling past it, a **sticky bottom bar** shows price + Buy.

Content order:
1. Large image gallery (Embla; thumbnails on desktop, swipe + dots on mobile; full-screen lightbox).
2. Deal title + merchant name (links to `/business/[slug]`).
3. Rating summary (★ avg, count) — click scrolls to reviews.
4. **Purchase panel**: option selector (radio cards, each with its own price/discount/stock), deal price (bold ink), original price (strikethrough), **"You save X" in green** (updates live with option and quantity), quantity stepper (respects `max_per_user` and stock), **red "Buy now" button** (primary), small "Add to cart" secondary, save ♡, share, "Buy as a gift" toggle (if giftable).
   Urgency line from real data only: "Ends in 05:12:33" (if <72h), "Only 7 left" (if stock ≤ 10), "120 bought today" (if ≥ 20).
5. Detailed description.
6. What's included.
7. Terms of use (validity, booking required, max per person, restrictions).
8. Opening hours and address(es) for the participating locations.
9. Map (MapView with pins of participating locations; lazy-loaded).
10. Reviews: distribution bars, list with "Verified purchase" badge, merchant replies, "Report" action, pagination.
11. FAQ (accordion).
12. Related deals (same subcategory, same city).

SEO: `generateMetadata` per locale, OpenGraph image (first gallery image), JSON-LD `Product` + `Offer` + `AggregateRating`, `hreflang` alternates, canonical URL.

States: `sold_out` → disabled Buy with "Sold out" and "notify similar deals"; `expired` → page stays (SEO) with "This deal has ended" + related deals; `scheduled` → "Starts in …".

### 9.4 Coupons page
- Tab **Deals**: listing of deals in the Coupons category (same cards).
- Tab **Promo codes**: different card: brand logo, offer label, "Verified" badge, expiry, **"Show code"** button → reveals code with copy button and opens brand URL in a new tab; tracks `click_count`. "No code needed" variant shows "Get deal".

### 9.5 Merchant profile `/business/[slug]`
Cover, logo, name, rating, description, locations with hours + map, all active deals, reviews across deals with replies.

### 9.6 Merchant landing `/merchant`
Value proposition, how it works (3 steps), commission explained simply, FAQ, CTA → `/merchant/register`.

---

## 10. User flow (must match)

```
Homepage → Search / Categories → Deal listing → Deal detail
→ Logged in?  no → Login / Register → (return to same deal + selected option)
              yes ↓
→ Checkout → Payment → Payment successful?
     no  → Failed page: retry payment / contact support (order stays pending_payment until reservation expires)
     yes → Voucher(s) generated → My Coupons → Show QR / code → Merchant redeems → Completed → Ask for review
```

- Browsing never requires login. Login is asked only at checkout.
- After login/register the user returns to exactly where they were (return URL + preserved cart/option).
- Guests' saved deals and recently viewed are merged into the account on login.

---

## 11. Checkout and mock payment

1. User clicks Buy now → `POST /orders` with an **Idempotency-Key** header → server validates option status, stock and `max_per_user`, creates `Order(pending_payment)`, creates `StockReservation` (10 min) **atomically**.
2. Checkout page (single page, no animation): order summary, savings in green, gift details (if gift), shipping address (only if a product item), payment method selector (Phase 1: "Test payment"), total in UZS (+ USD hint), terms checkbox.
3. `POST /orders/{id}/pay` → `PaymentProvider.CreatePayment`.
   **MockProvider**: shows a mock payment screen with two buttons "Simulate success" / "Simulate failure" (and a configurable artificial delay). It calls the same webhook endpoint a real provider would.
4. Webhook handler (idempotent, verifies signature even for mock): on success → order `paid`, reservation converted to sold (`stock_sold += n`), vouchers generated (one per unit), confirmation email (mailpit locally), redirect to success page with confetti and "View my coupons". On failure → payment `failed`, order stays `pending_payment` until reservation expiry; failed page offers retry and support.
5. Worker job releases expired reservations and cancels unpaid orders.

**Overselling must be impossible.** Reserve stock with a single conditional update, e.g.
`UPDATE deal_options SET stock_reserved = stock_reserved + $n WHERE id = $id AND stock_total - stock_sold - stock_reserved >= $n RETURNING ...` inside a transaction. If zero rows → "Sold out / not enough stock". Write a concurrency test (100 parallel buyers for 10 vouchers → exactly 10 succeed) and a k6 script.

`PaymentProvider` interface (Go):
```go
type PaymentProvider interface {
    Name() string
    CreatePayment(ctx context.Context, order Order) (PaymentSession, error)
    ParseWebhook(ctx context.Context, r *http.Request) (WebhookEvent, error)
    Refund(ctx context.Context, payment Payment, amount int64) error
}
```
Adding Payme/Click later must require only a new implementation + config, no changes elsewhere.

---

## 12. Vouchers and redemption

### 12.1 Lifecycle
`active` → `redeemed` | `expired` (worker job at `valid_until`) | `refunded` | `cancelled`.

### 12.2 User side
`/account/coupons/[id]`: large QR, human code below, deal title, option, merchant, address, valid until, redemption instructions, "Add to home screen" hint (PWA). Page cached by the service worker so it opens without internet.

### 12.3 QR security
QR encodes a **signed token** (HMAC with server secret: voucher id + version), never a plain id. The human code is an alternative input. Scanning only works for staff of the voucher's merchant.

### 12.4 Merchant side `/merchant/redeem` (mobile-first, no animation, fast)
- Big "Scan QR" button → camera (zxing) → result screen: deal, option, buyer first name, status.
- Or manual code entry.
- **Confirm redemption** button → `POST /merchant/vouchers/{id}/redeem` (idempotent; selects location if merchant has several) → big success/failure state (already redeemed, expired, wrong merchant, refunded).
- Recent redemptions list. Every redemption is audit-logged.

---

## 13. Merchant flows

### 13.1 Registration
`/merchant/register`: account (Google or email) → business info (name, category, city, phone, address) → merchant created `active`, owner role assigned. **No pre-approval of the merchant; every deal goes through moderation.**

### 13.2 Create Deal wizard (with live preview)
Steps: 1) Basics (category, subcategory, kind, title, short description, languages) 2) Options and pricing (1+ options, original/deal price, stock, max per user; discount % shown live) 3) Media (3–8 images, drag to reorder, from Media Library) 4) Details (description, highlights, what's included, terms, FAQ, redemption method, booking required, giftable) 5) Locations and schedule (participating branches, start/end, voucher validity) 6) Review and submit.
Right side (desktop) / toggle (mobile): **live preview** rendering the real deal-card and deal-page components with the form data.
Save as draft at any step. Submit → `pending_review`. Admin approves → `scheduled` or `active`; rejects → `rejected` with reason (merchant edits and resubmits). Editing an active deal's prices, terms or options sends it back to review; editing only stock does not.

### 13.3 Staff
Owner invites staff by email with role: manager (everything except payouts/staff) or cashier (redemption only).

### 13.4 Reviews
Merchant sees all reviews, can reply once per review (editable for 24h), can report abusive reviews.

---

## 14. Backend (Go) requirements

- **API style**: REST, JSON, versioned `/api/v1`. Errors in RFC 9457 problem+json with a stable `code` for the frontend to translate.
- **OpenAPI**: every endpoint, schema, error and auth requirement is defined in `packages/api-spec/openapi.yaml`. CI fails if generated code is out of date.
- **Layers**: handler (HTTP only) → domain service (business rules) → store (sqlc). No SQL in handlers, no HTTP in domain.
- **Auth**: Google OAuth (authorization code + PKCE), email/password with argon2id, email verification, password reset via email token, refresh token rotation with reuse detection (revoke family). Rate-limit login, register, reset (Redis).
- **RBAC**: checked in middleware + domain layer. Merchant staff can only touch their merchant's data. Superadmin endpoints under `/api/v1/admin/*` require admin roles.
- **Search**: Postgres full-text per locale + `pg_trgm` for typo tolerance + `unaccent`; suggestions endpoint for the search bar (deals, merchants, categories). Design the search module behind an interface so Meilisearch can replace it later.
- **Caching**: Redis cache for homepage sections, category trees, city list, exchange rate, hot deals (short TTL + explicit invalidation on change). Next.js side uses ISR / `revalidateTag` triggered by the API on content changes.
- **Images**: upload via presigned S3 URLs; store original; record width/height/blurhash; served through `next/image` (AVIF/WebP, responsive sizes).
- **Jobs (river)**: release expired reservations, expire vouchers, transition deals (`scheduled→active`, `active→expired`, `active→sold_out`), recompute ratings/sold counts, send emails, nightly payout calculation, (later) fetch exchange rates.
- **Security**: CORS allowlist, CSRF protection for cookie auth (double-submit token), secure/httpOnly/SameSite cookies, HTML sanitization of rich text, size limits on uploads and bodies, no personal data in URLs, request IDs, structured logging (`slog`) without secrets or PII.
- **Ops**: `/healthz`, `/readyz`, graceful shutdown, config from env (12-factor), Dockerfile for api and worker.
- **Data residency note**: production hosting must allow storing Uzbek citizens' personal data in Uzbekistan. Keep infrastructure portable (Docker, S3-compatible storage, standard Postgres) — no vendor lock-in.

---

## 15. Frontend requirements

- **Server Components by default.** Client components only for interactivity (filters, gallery, purchase panel, cart, forms, dashboards).
- Public pages: SSR/ISR with tag revalidation. Account, checkout, merchant: dynamic, no caching of personal data.
- Data access only through `packages/api-client` (generated types). During Phase 2, MSW serves responses built from the seed data with the **same OpenAPI shapes**; switching to the real API must need only an env change.
- TanStack Query for client fetching/mutations (optimistic updates for save ♡ and cart).
- Zustand only for cart (guest) and UI state; persisted carefully (try/catch around storage).
- Forms: react-hook-form + Zod schemas shared from `packages/types` where possible; inline errors in `--error` with icon.
- Accessibility: WCAG 2.2 AA, keyboard navigable, focus rings, alt texts, proper labels, color never the only signal.
- PWA: manifest (name from brand config, indigo theme color), icons, offline shell, cached voucher pages, installable prompt on mobile after 2nd visit or after first purchase.
- Analytics hooks: a tiny `track(event, props)` wrapper (no vendor yet) called on view_deal, add_to_cart, begin_checkout, purchase, redeem.

### Performance budget (must be met on mobile, Lighthouse)
- Performance ≥ 90, Accessibility ≥ 95, Best Practices ≥ 95, SEO ≥ 95 on homepage, category and deal pages.
- LCP < 2.5s, INP < 200ms, CLS < 0.1.
- Initial JS for public pages kept minimal: animation libraries are dynamically imported; maps lazy-load on scroll.

---

## 16. Seed data

- 3 cities: Tashkent, Samarkand, Bukhara (configurable; more can be added).
- 10 nav items / categories with 4–6 subcategories each.
- ~30 merchants with 1–3 locations each (realistic addresses and coordinates inside each city).
- ~100 deals spread across categories and cities; 1–3 options each; mix of statuses (mostly active, some ending within 72h, some sold out, some expired, some pending review, a few scheduled).
- ~10 goods (product) deals.
- ~25 promo codes from fictional brands.
- 3 collections (e.g. New Year, Navruz, Black Friday) with dates.
- ~400 reviews with realistic distribution, some with merchant replies, in all three languages.
- Test users: 1 normal user, 1 merchant owner, 1 merchant cashier, 1 superadmin (credentials in `seed/README.md`, dev only).
- Images: **only free-license photos** (Unsplash / Pexels URLs) listed in `seed/images.json` with source link; configure `next/image` remote patterns. Never use Groupon assets, logos or any copyrighted brand material. Fictional brand names only.

---

## 17. Phases

Each phase ends with its Definition of Done (DoD). Do not move on until DoD passes.

### Phase 0 — Foundation
Monorepo, pnpm, Turborepo, TS strict, ESLint/Prettier, Go module, docker-compose (postgres, redis, minio, mailpit), Makefile/`pnpm` scripts (`dev`, `build`, `test`, `gen`, `seed`), design tokens + fonts + base layout in `packages/ui`, brand config, i18n setup with 3 locales, CI workflow (lint, typecheck, test, build, generated-code check).
**DoD**: `pnpm dev` starts web + api + worker; empty homepage renders in uz/ru/en with correct tokens; CI green.

### Phase 1 — Contract and database
Full OpenAPI spec for all public, account, checkout, merchant endpoints (admin can be stubbed). All migrations for Section 8. sqlc queries. Seed command loading Section 16 data. Code generation for Go + TS.
**DoD**: `make seed` fills the DB; generated code compiles; spec validates; DB constraints tested (stock can't go negative, one review per voucher, unique codes).

### Phase 2 — Public site UI ("fire")
Header/nav/mega-menu/mobile tab bar, homepage (all 8 sections + animations), city/category/search listings with filters/sort, deal detail page (full Section 9.3), coupons page with promo codes, merchant profile, merchant landing, collections, help/terms/privacy, 404/500, skeletons, empty states. Running against MSW built from seed data.
**DoD**: all public pages work in 3 languages and both currencies; first-screen rule verified on 360×640, 390×844, 1440×900 via Playwright screenshots; Lighthouse budget met on homepage/category/deal.

### Phase 3 — Catalog API
Go endpoints for cities, categories, homepage sections, listings with filters/sort/pagination, deal detail, merchant profile, search + suggestions, promo codes (+ click tracking), collections, reviews (read). Redis caching + revalidation hooks. Switch web from MSW to the real API.
**DoD**: Phase 2 Playwright tests pass against the real API; p95 latency of listing and deal endpoints < 100ms locally with seed data.

### Phase 4 — Accounts, cart, checkout, vouchers
Auth (Google + email/password + verification + reset), account pages, saved deals, recently viewed sync, cart (guest + server merge), checkout, MockProvider + webhook, reservations, vouchers + QR, emails, success/failure pages, worker jobs.
**DoD**: Playwright e2e: browse → login mid-flow → return to deal → buy → success → voucher visible with QR; failure + retry path works; concurrency test proves no overselling; k6 script included.

### Phase 5 — Merchant panel
Registration, Overview, My Business (locations, hours, map pin), Media Library, Create Deal wizard with live preview, My Deals (statuses, duplicate, pause), Orders & Sales, Voucher Redemption (camera + manual, mobile-first), Customers, Analytics & Reports (charts, CSV export), Payouts & Finance (read-only calculations), Reviews (+ replies, report), Settings + Staff roles.
**DoD**: e2e: merchant creates deal → pending_review → (approve via seed/admin API) → user buys → cashier scans QR on a mobile viewport → voucher redeemed → user can review → merchant replies. Cashier cannot access finance pages (tested).

### Phase 6 — Polish and PWA
Reviews write flow, review reports, gift purchase flow (recipient email), goods (shipping address + fulfillment status), PWA (installable, offline voucher page), animations refinement, accessibility pass, SEO (sitemap per locale, robots, JSON-LD, hreflang), error monitoring hooks.
**DoD**: Lighthouse budgets met; axe accessibility checks pass in e2e; voucher page opens offline after first load.

### Phase 7 — Superadmin app
`apps/admin` with all modules from 7.4, Homepage Builder, moderation queue (approve/reject with reason, side-by-side preview), user/merchant blocking, refunds (MockProvider), exchange-rate setting, roles & permissions, audit log viewer with filters.
**DoD**: every admin write action produces an audit log entry (tested); moderator cannot access finance; homepage changes appear on the site after revalidation.

### Phase 8 — Hardening
Load test flash-sale scenario, security review checklist (OWASP top 10 items relevant here), backup/restore notes, production Dockerfiles, environment docs, `docs/ARCHITECTURE.md` explaining how the system works in plain language.
**DoD**: all tests green; docs complete.

---

## 18. Testing requirements

- Go: unit tests for domain rules (pricing, discount rounding, stock, voucher states, permissions); integration tests against real Postgres via testcontainers; concurrency test for stock.
- Web: component tests for card, purchase panel (option switching updates price + savings), formatMoney, filters.
- E2E (Playwright, mobile + desktop viewports): the full user flow, failure path, merchant redemption flow, locale switching, currency switching, first-screen rule.
- Contract: generated clients must compile; spec lint (e.g. Redocly/Spectral) in CI.

---

## 19. Out of scope (do not build)

Real payment providers, SMS login, native mobile app, games, coins/points/loyalty, ML recommendations, chat/messaging, merchant ads marketplace, affiliate payouts for promo codes, dark mode, multi-country markets (but keep currency/market configurable).

---

## 20. Verification checklist (for the reviewer)

A reviewer (human or AI) uses this list after each phase. Every item must be checked against the running app or tests, not just the code.

- [ ] All commands in Section 0.2 pass.
- [ ] No hardcoded UI strings, colors, brand name, cities or categories (grep for literals).
- [ ] Menu order exactly matches Section 6.
- [ ] Deal card matches 4.4 (no button, no "You save" line, price in ink, real-data meta only).
- [ ] Deal page: price, discount and Buy button visible without scrolling at 360×640, 390×844, 1440×900.
- [ ] Prices never red; no orange anywhere; errors use `--error` + icon.
- [ ] Money stored as integers; USD display-only; checkout says payment is in so'm.
- [ ] Urgency texts only appear when backed by real data thresholds.
- [ ] Login only required at checkout; return to same deal after login.
- [ ] Overselling concurrency test passes.
- [ ] QR contains a signed token, not a raw id; only the right merchant's staff can redeem.
- [ ] Only users with a redeemed voucher can review; one review per voucher; one merchant reply per review.
- [ ] Every admin and sensitive merchant action creates an audit log.
- [ ] Animations absent from checkout, payment, redemption, admin; reduced-motion respected.
- [ ] Lighthouse budgets (Section 15) met.
- [ ] `docs/PROGRESS.md` and `docs/DECISIONS.md` updated.
