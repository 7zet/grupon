# Progress

## Phase 0 — Foundation

Status: complete on 2026-10-31.

### Built

- pnpm workspace and Turborepo orchestration with strict TypeScript, shared ESLint and Prettier
  configuration, deterministic lockfile and pnpm build-script policy.
- Next.js App Router web shell with Inter, Tailwind CSS, the approved design tokens, a reusable base
  layout and localized routing for Uzbek, Russian and English.
- Brand configuration and three message catalogs with locale/message parity tests.
- Root Go module with API and worker entry points, validated environment configuration, structured
  logging, request IDs, health/readiness probes, panic recovery and graceful shutdown tests.
- Local Docker Compose services for PostgreSQL, Redis, MinIO and Mailpit with pinned images, health
  checks and named volumes.
- CI jobs for formatting, lint, typecheck, tests, builds, generated-code drift and Compose validation.
- Phase 0 package placeholders for the API contract and generated client; implementation remains
  contract-first in Phase 1.

### Verified

- `pnpm format:check`, `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm build`.
- `go build ./...`, `go vet ./...`, `go test ./...` from the repository root.
- `pnpm dev` started web, API and worker together; all processes stopped cleanly.
- `/healthz` and `/readyz` returned 200.
- `/uz`, `/ru` and `/en` returned 200 with matching `<html lang>` values.
- Bare `/` respected locale cookie first, then `Accept-Language`, with Uzbek as the fallback.
- Docker Compose configuration and GitHub Actions syntax validated.

### Left for Phase 1

- Full OpenAPI contract and deterministic Go/TypeScript generation.
- PostgreSQL migrations, sqlc configuration and queries.
- Complete Section 8 schema, seed loader/data and database constraint tests.

### Known limitations

- The local Docker daemon was unavailable during Phase 0 verification, so Compose was validated
  statically but the four containers were not started in this environment.
- `pnpm gen` and `pnpm seed` intentionally have no task producers until Phase 1.
- `apps/admin` is intentionally deferred to Phase 7.

## Phase 1 — Contract, database, and seed

Status: complete on 2026-10-31.

### Built

- OpenAPI 3.1 contract for public catalog/search, authentication, customer account, cart, checkout,
  payment webhooks, vouchers, reviews, merchant registration and every merchant-panel module; the
  admin API remains an explicit Phase 1 stub.
- Deterministic contract generation: strict chi server interfaces and Go models, generated
  TypeScript paths/operations/components, Redocly validation, and generated-code drift support.
- Complete Section 8 PostgreSQL schema with UUIDv7 keys, timestamps/update triggers, soft deletes,
  enums, translation checks, money and stock constraints, all foreign-key indexes, locale full-text
  search, trigram indexes, and normalized deal locations.
- sqlc configuration, generated store types/interfaces, catalog/identity/commerce queries, and the
  atomic conditional stock-reservation query.
- Production-guarded, transactional seed loader with documented test accounts and free-license
  image sources. It creates 3 cities, 10 categories/45 subcategories, 30 merchants/60 branches,
  100 deals/199 options, 25 promo codes, 3 collections, and 400 verified-voucher reviews.
- Testcontainers database tests for stock bounds, one review per voucher, and unique voucher codes;
  tests can also target an explicit isolated PostgreSQL through `TEST_DATABASE_URL`.

### Verified

- OpenAPI validation and both Go/TypeScript generators; generated packages compile and typecheck.
- `make seed` equivalent through the workspace seed target against an isolated PostgreSQL 16
  cluster; resulting counts were `3/10/45/30/60/100/199/25/3/400` for the primary seed entities.
- All three required database constraint tests passed against that real database.
- `pnpm build`, `pnpm lint`, `pnpm typecheck`, `pnpm test`.
- `go build ./...`, `go vet ./...`, `go test ./...` from the repository root.

### Left for Phase 2

- Public/user frontend implementation and MSW handlers generated from the Phase 1 seed shapes.
- Live API handlers remain in their assigned backend phases; Phase 1 generated strict interfaces
  and persistence primitives but did not implement later-phase business behavior.

### Known limitations

- Docker Desktop is not installed in this environment. Container-backed tests skip locally when no
  provider exists; the same migrations, seed, and constraint suite were instead run successfully
  against an isolated temporary PostgreSQL cluster. CI runners execute the Testcontainers path.

## Phase 2 — Public marketplace frontend

Status: complete on 2026-10-31.

### Built

- Responsive public shell with search suggestions, exact category navigation, desktop mega-menu,
  mobile menu/tab bar, city/currency/language controls, footer, and locale-aware routing.
- All eight homepage sections, GSAP-enhanced hero, Motion deal-card interactions, Embla galleries,
  skeleton/empty/error states, and token-only responsive styling.
- City, category, and search listings with nuqs URL filters, sorting, pagination, and shared deal
  cards; coupons with voucher/promo tabs and reveal/copy behavior.
- Full deal detail with the first-screen purchase layout, option/quantity/gift state, integer money
  formatting, gallery, content/terms/locations, provider-agnostic Leaflet map, reviews, FAQ, related
  deals, localized metadata, hreflang, OpenGraph, and Product/Offer JSON-LD.
- Merchant profiles and merchant landing, seasonal collections, help/terms/privacy, and localized
  404/500 routes.
- Typed OpenAPI public client plus server/browser MSW handlers generated from Phase 1 seed inputs;
  environment-only switching to the live Go API.
- Unit/component coverage for money, route namespaces, deal cards, purchase controls, and filters;
  authored Playwright public-route/viewport specs and Lighthouse budgets.

### Verified

- `pnpm build`, `pnpm lint`, `pnpm typecheck`, and `pnpm test` pass.
- `go build ./...`, `go vet ./...`, and `go test ./...` pass from the repository root.
- `pnpm gen` completed without generated contract drift.
- Homepage, city, category, search, deal, coupons, merchant, collection, and static routes returned
  HTTP 200 from a production Next.js server; homepage returned 200 in Uzbek, Russian, and English.
- Playwright passed all 7 public-route, locale, URL-filter, and first-screen tests against the
  production build. The 360×640, 390×844, and 1440×900 screenshots were visually reviewed; the
  compact viewport uses a functional sticky Buy action above the mobile navigation.
- Lighthouse CI ran homepage, category, and deal three times each. Scores were performance 98–100,
  accessibility 100, best practices 96, and SEO 100; worst LCP was 1,045 ms, CLS 0, and TBT 0 ms.

### Known limitations

- Phase 2 intentionally uses deterministic MSW catalog data. Phase 3 replaces that adapter with the
  real Go catalog API without changing page-level response shapes.

## Phase 3 — Catalog API

Status: complete on 2026-10-31.

### Built

- Go catalog endpoints for config, cities, categories, homepage, deal listing (filters, sort,
  pagination), deal detail, related deals, reviews, merchant profile, search suggestions, promo codes
  with click tracking, collections and collection detail. Errors are RFC 9457 problems with stable codes.
- Domain service (`internal/domain/catalog`) with pure validation and visibility rules; PostgreSQL
  read model (`internal/store/catalog_repository.go`) on sqlc queries with batched option/media loads.
- Optional Redis read-through cache with namespace-version invalidation, Redis-backed readiness,
  and an internal token-protected invalidation hook that revalidates the web `catalog` tag.
- Web switched to the real API by default; MSW is opt-in. Listing parameters map to the contract
  (`price_asc`, `biggest_discount`, `min_rating`, `ending_soon`). Revalidation route added.
- Seed loader extracted to `internal/seed` for reuse by tests.

### Verified

- `go test ./...`, `go vet ./...`, `go build ./...`, `go generate` (no drift).
- Catalog integration tests against a freshly seeded PostgreSQL 16 (15 subtests: listing, sort,
  paging, city filter, search, detail, expired visibility, reviews, suspension, promo clicks,
  collections, homepage, suggestions). Redis round-trip and invalidation test against Memurai.
- HTTP contract tests for problems, binding errors, 501 fallback, promo click and internal token.
- Latency (300 sequential requests, client-side): p95 between 9 and 18 ms across listing,
  search, detail, reviews, related, merchant, homepage, config, suggestions and promo endpoints.
- Web workspace typecheck, lint, vitest, and `next build`; production server returned 200 for
  homepage, category, search, coupons, collection, deal and business routes; Playwright public suite 7/7.
- Revalidation route: correct secret 202, missing or wrong secret 401, unknown tag 400.

### Left

- Phases 4–8 remain; the admin app, cart/checkout, auth and merchant panels are not started.
- Docker Desktop was not running, so the Testcontainers-backed catalog and constraint tests skip
  locally; they run in CI, and the same suite was run against a local PostgreSQL 16 instead.
- `pnpm dev` now needs PostgreSQL for the API; start `make seed` / compose first.

### Known issues

- Unknown deal or merchant slugs return HTTP 200 with a not-found UI instead of HTTP 404 (the
  `notFound()` flow in the deal and business pages). This hurts SEO and should be fixed before launch.
- `turbo run` fails on this Windows machine with "Unable to find package manager binary"; the
  same tasks pass when run through `pnpm -r`.


## Phase 4 — Identity (in progress)

Status: identity part only. Cart, checkout, payments, vouchers, order emails and worker jobs are not started, so Phase 4 is not complete.

### Built

- Migration `00002_auth_tokens.sql` (single-use verification and reset tokens) and identity sqlc queries, including tombstoning on account deletion.
- `internal/domain/identity`: argon2id hashing, opaque tokens (hashed at rest), HS256 access tokens, the service (register, login, Google sign-in, refresh rotation with reuse detection, logout, verify email, forgot and reset password, change password, profile, account deletion, `Authenticate`), the repository, limiter and Google ports, and uz/ru/en email copy.
- `internal/domain/identity/identitytest`: in-memory repository, mailer and limiter for tests.
- `internal/platform/mail` (SMTP and log senders), `internal/platform/ratelimit` (Redis fixed window and a disabled limiter), `internal/platform/google` (PKCE code exchange and ID token claim checks).
- `internal/store/identity_repository.go`: PostgreSQL adapter; user creation and refresh rotation run in transactions.
- `internal/http/auth_handlers.go`: all twelve auth and `/me` operations, cookies (httpOnly, SameSite=Lax, Secure outside development), Bearer support, 64 KB body limit, problem codes, `RequireRole`. Operations are served through one combined generated server in `router.go`.
- Configuration: `JWT_SECRET`, `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL`, `WEB_PUBLIC_URL`, `BRAND_NAME`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `TRUST_PROXY_HEADERS`.

### Verified

- `go build ./...`, `go vet ./services/...`, `gofmt -l services` (clean).
- `go test ./services/... -count=1` with `TEST_DATABASE_URL` (PostgreSQL 16, migrated to version 2) and `TEST_REDIS_URL` (local Redis) passes.
- Unit tests: password hashing and policy, token hashing, JWT round trip, expiry and forgery rejection; service flows for uniform login errors, refresh rotation and family revocation, expiry, single-use verification and reset, change-password session handling, deletion, rate limits, limiter outage fail-open, and Google linking.
- Integration tests on PostgreSQL: register, duplicate email, verify, login, rotation, reuse, reset, old-password rejection, deletion; eight concurrent refreshes yield exactly one rotation.
- HTTP tests: cookie flags (secure and non-secure), 429 with `Retry-After`, uniform login problem bodies, refresh reuse 401, logout clears cookies, Bearer access, `RequireRole` 403 and 204, 413 body limit, unknown-field rejection, forgot-password 202, password change error code.
- Google client tests against an httptest token endpoint: claim validation, redirect mismatch, Google error responses.
- Config tests for the secret rule, the Google all-or-nothing rule, SMTP port, and defaults.

### Left

- CSRF double-submit token (Section 14).
- Google flow against real Google (not tested with live credentials).
- SMTP delivery through mailpit (not run; the SMTP sender is covered only by render tests).
- Web pages for `/{locale}/verify-email` and `/{locale}/reset-password`, and the apps/web auth UI.
- Worker jobs that expire `auth_tokens` and `refresh_tokens` rows.
- A resend-verification endpoint (none in the contract).
- Admin and merchant routes that use `RequireRole`.

### Known issues

- `AuthSession` has no `refresh_token` field, so native clients cannot receive a refresh token from the contract. A spec decision is needed before mobile work. `refreshSession` also declares `bearerAuth` with `bearerFormat: JWT`, which does not match the opaque refresh token.
- `logout` requires a valid access token (its `cookieAuth` requirement). A client whose access token has expired must refresh first.
- The rate limiter fails open on Redis errors (logged), as recorded in DECISIONS.
