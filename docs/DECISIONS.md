# Decisions

## 2026-10-31 — Phase 0 assumptions

1. **Phase boundary.** Phase 0 contains only the foundation. The admin application, full API
   contract, database schema, generated clients and seed loader remain in their explicitly assigned
   later phases. Placeholder `api-spec` and `api-client` packages establish the target structure
   without hand-written API types.
2. **Go module location.** The repository uses one root Go module even though Go sources live under
   `services/api`. This makes the specification's literal root commands (`go build ./...`,
   `go vet ./...`, `go test ./...`) test this repository and prevents an unrelated parent module
   from being selected.
3. **Compatible toolchain over incompatible newest majors.** Next.js 16.4.3 and React 19.3.1 are
   pinned. TypeScript 6.0.3 is the newest release accepted by `typescript-eslint` 8.72.1; TypeScript
   7 is therefore deferred. ESLint 9.39.5 is the newest release supported by the plugins bundled
   with `eslint-config-next`; ESLint 10 is deferred. pnpm dependency build scripts are denied by
   default and only the three required native packages are allowlisted.
4. **Generation and seeding.** `pnpm gen` and `pnpm seed` are Phase 0 aggregators with no producers.
   TypeScript/Go code generation and the database seed command are Phase 1 deliverables. CI still
   runs both generation entry points and checks the working tree for drift.
5. **Development orchestration.** `pnpm dev` starts web, API and worker. Docker Compose is started
   separately; Phase 0 API and worker startup intentionally has no database, Redis or object-storage
   dependency.
6. **Next.js routing terminology.** Next.js 16's `proxy.ts` convention implements the required
   locale middleware behavior. Routes listed in the public-route tables are interpreted as relative
   to `/[locale]`; bare `/` only negotiates and redirects by cookie, then `Accept-Language`, then
   Uzbek.
7. **Layout ownership.** Shared tokens, Inter font configuration and the reusable base layout live
   in `packages/ui`. The document-level locale layout stays in `apps/web` because Next.js owns the
   `<html>`/`<body>` boundary and request locale lifecycle.
8. **Design-rule exceptions.** The mandated `--rating` value is the sole amber-like token and is
   reserved for stars. The statement that red is not a background means it is not used for large
   surfaces; explicitly approved compact deal accents remain valid in later phases.
9. **Brand rule scope.** User-visible brand text and metadata must come from `brand.config.ts`.
   Technical identifiers such as npm scopes, Go import paths and repository paths remain stable and
   are not runtime branding.
10. **Operational probes.** `/healthz` and `/readyz` are infrastructure probes outside the versioned
    product API. They do not generate client types; the Phase 1 OpenAPI contract covers `/api/v1`.
11. **MinIO image.** Upstream public MinIO container images were withdrawn. Local Compose therefore
    pins `alpine/minio:RELEASE.2025-10-15T17-29-55Z`, a multi-architecture image built from the tagged
    upstream source, while preserving the required S3-compatible behavior.
12. **Deferred shared schemas.** `packages/types` is not created until a non-API Zod schema is truly
    shared. API shapes always come from OpenAPI generation.

## 2026-10-31 — Phase 1 assumptions

1. **Localized database values.** Translated content is stored as a JSONB object with exactly the
   `uz`, `ru`, and `en` keys. Database checks guarantee the shape and require at least one non-empty
   value where the product model requires merchant content. Locale fallback remains application
   logic because it depends on the requested locale.
2. **Deal locations are normalized.** The specification's conceptual `location_ids[]` is persisted
   as `deal_locations`. This provides foreign keys, prevents duplicate branch assignments, and
   makes merchant isolation enforceable without array parsing.
3. **Portable UUIDv7.** The migration defines an RFC 9562-compatible `public.uuidv7()` backed by
   `pgcrypto`. It supplies the required UUIDv7 defaults on PostgreSQL 16 and shadows the equivalent
   built-in safely on PostgreSQL 18, which is the pinned local container version.
4. **OpenAPI 3.1 generation.** The checked-in 3.1 contract is bundled and validated by Redocly,
   generates TypeScript with `openapi-typescript`, and generates Go models plus strict chi server
   interfaces with `oapi-codegen`. The latest `openapi-typescript` release still declares a
   TypeScript 5 peer range; pnpm permits its verified CLI use with the repository's TypeScript 6
   while all generated output is typechecked in CI.
5. **Phase boundary for internal workflow storage.** Phase 1 creates every Section 8 table. API
   contracts for later-phase workflows such as presigned media uploads and pending staff invites
   are defined now, while any additional workflow-only persistence is introduced with the phase
   that implements that workflow instead of silently extending the locked Section 8 model.
6. **Deterministic development seed.** Compact JSON sources contain taxonomy, translation, review,
   and licensed-image inputs; the Go loader deterministically expands them to the required dataset
   sizes in one transaction. It is rerunnable, applies migrations first, and refuses production.

## 2026-10-31 — Phase 2 assumptions

1. **Mock-to-live API switch.** Public pages always call the typed runtime client from
   `packages/api-client`. MSW intercepts that same `/api/v1` contract in both the Next.js server and
   the browser while `USE_MSW`/`NEXT_PUBLIC_USE_MSW` are enabled. Disabling both flags switches to
   the configured Go API without changing a page or component.
2. **Phase 2 catalog data.** The MSW catalog deterministically expands the Phase 1 seed taxonomy,
   merchants, translations, images, and review text into OpenAPI-generated response shapes. Mock
   data is confined to the MSW adapter; public pages never import it directly.
3. **Money conversion.** The Phase 1 seeded UZS/USD rate is represented as integer UZS per USD.
   Display conversion rounds once to integer USD cents, so frontend money logic never uses a
   floating-point amount as its source of truth.
4. **Next.js type-check ownership.** `next build` skips its duplicate child-process type check. The
   mandatory root `pnpm typecheck` remains the authoritative strict check and runs independently in
   local development and CI before the production build. This also keeps builds viable in restricted
   Windows environments that deny child-process creation.
5. **next-intl configuration.** Next.js receives the standard `next-intl/config` alias directly in
   its Turbopack and Webpack configuration. This preserves the documented request-config behavior
   without loading next-intl's optional build-time compiler plugin.
6. **Public performance budgets.** Homepage, category, and deal routes target performance 90 and
   accessibility/best-practices/SEO 95, with LCP ≤2.5s, CLS ≤0.1, and TBT ≤200ms. These thresholds
   are checked by `apps/web/lighthouserc.cjs` against a production server.
7. **Old-price contrast.** Section 4 names `--ink-faint` for struck-through prices, while Section 15
   requires WCAG 2.2 AA. At 14px the locked faint value has only 2.56:1 contrast on white, so old
   prices use the next approved neutral token, `--ink-muted`, to satisfy the higher accessibility
   requirement without inventing a color.
8. **Narrow first screen.** At 360×640 the complete option panel extends behind the persistent mobile
   navigation. The sticky Buy control therefore appears whenever the original Buy button is not
   fully unobscured and invokes the same purchase action; this preserves both the first-screen law
   and the required bottom navigation.

## 2026-10-31 — Phase 3 web switch to the real API

1. **Real API by default.** `USE_MSW` and `NEXT_PUBLIC_USE_MSW` now default to `false`; MSW runs only when both are `true`.
2. **Contract query names.** Listing URLs keep friendly keys (`rating`, `endingSoon`, `sort`, `page`); `apps/web/src/lib/deal-query.ts` maps them to the OpenAPI `/deals` parameters (`min_rating`, `ending_soon`, sort enum values) and drops invalid values.
3. **Cache tag.** Every catalog fetch uses `next.tags: ['catalog']`. `POST /api/revalidate` (Route Handler) accepts `X-Revalidate-Secret` matching `WEB_REVALIDATE_SECRET` and revalidates the allowlisted tags.

## 2026-10-31 — Phase 3 catalog API assumptions

1. **Contract-first, no spec change.** Every endpoint implements the existing OpenAPI operation. The
   Go handlers use the generated non-strict `ServerInterface`; operations outside Phase 3 keep the
   generated 501 `Unimplemented` response.
2. **Visibility rules.** Listings, search, collections, promo-related deals and merchant profiles
   include only deals with status `active` or `sold_out`, `ends_at` in the future, and an active,
   non-deleted merchant. Deal detail also serves `scheduled` and `expired` deals; drafts, pending and
   rejected deals are never returned. Suspended merchants disappear from all public reads.
3. **Discount rounding.** Discount percent (used for `min_discount` and `biggest_discount`) is
   computed in SQL as round-half-up integer percent over active options.
4. **Homepage sections are global in Phase 3.** The `city` parameter is accepted but does not filter
   homepage sections; sections are curated by key. Banner sections are not rendered.
5. **Related deals.** Same subcategory, same city (no city filter when `city` is empty), at most 8.
6. **Caching.** Redis is optional (`REDIS_URL`). Entries are JSON under a namespace version; one
   `INCR` on invalidation makes all of them stale. Cache errors fall back to PostgreSQL. Suggestions
   are not cached.
7. **Revalidation hook.** `POST /internal/cache/invalidate` (requires `X-Internal-Token`, disabled
   when `INTERNAL_API_TOKEN` is empty) invalidates the Redis namespace and posts the `catalog` tag to
   the web `/api/revalidate` route. It is operational, not part of the product contract.
8. **Configuration.** `DATABASE_URL` defaults to the local compose database outside production and is
   required in production. The API now needs PostgreSQL to start.
9. **Seed reuse.** The seed loader moved to `internal/seed` so integration tests can seed the same
   dataset; `cmd/seed` remains the entry point.


## 2026-10-31 — Phase 4 Identity assumptions

1. **Auth tokens table (migration 00002).** Adds `auth_tokens` (id, user_id, kind `verify_email|reset_password`, unique token_hash, expires_at, used_at, created_at) for single-use verification and reset tokens. This is an addition beyond Section 8, made only for Phase 4. No other Section 8 table changed.
2. **Soft-delete tombstones.** Deleting an account rewrites `email` to `deleted+<id>@deleted.invalid` and `google_sub` to `deleted:<id>`, so both values can be reused. `password_hash` becomes the sentinel `deleted` for password accounts and NULL for Google-only accounts, which keeps `users_check` true. The sentinel is never verified, because deleted users are filtered out of every lookup.
3. **Change-password operation.** The contract defines `PUT /api/v1/me/password` (`changePassword`). The Phase 4 brief said PATCH; the contract is the source of truth, so PUT is implemented and no spec change was made.
4. **Refresh-token transport.** Web uses the httpOnly cookie `refresh_token` on path `/api/v1`, which also covers `/api/v1/me/password` so the current session can be kept. Native clients may send the refresh token as a Bearer value. The contract's `AuthSession` has no `refresh_token` field, so native clients have no delivery path yet (see PROGRESS known issues).
5. **Access tokens.** HS256 JWT with `iss=grupon-api`, `sub`, `roles`, `iat`, `exp` (15 min), `jti`. Authentication re-reads the user status and roles from PostgreSQL on each protected request, so blocking or revoking a role takes effect at once. Refresh tokens are 256-bit random values stored as SHA-256 hashes, with a 30-day expiry.
6. **Passwords.** argon2id with m=19 MiB, t=2, p=1 (OWASP minimum), parameters stored in the PHC string. Policy: 10 to 128 characters (runes). Unknown emails, password-less accounts and wrong passwords run the same hashing work and return the same `invalid_credentials` problem. A blocked account is reported only after its password has been verified.
7. **Refresh rotation and reuse.** Rotation claims the current token with a conditional UPDATE inside a transaction, so concurrent refreshes yield one winner. Presenting a revoked token (rotated or logged out) revokes the whole family and returns 401 `refresh_token_reused`.
8. **Password reset and change.** Reset revokes every session and marks the email verified, because the mailbox proved ownership. Change revokes every session except the one that presents the refresh cookie. Reset tokens expire after 1 hour and verification tokens after 24 hours; both are single-use and stored hashed.
9. **Email verification is advisory.** Unverified accounts can sign in. Gating purchases on verification is left to Phase 4 checkout.
10. **Rate-limiting trade-off.** Redis fixed window, 10 attempts per 15 minutes per key. Keys use the client IP and the SHA-256 of the email, so no raw personal data reaches Redis. Login, register and forgot-password check both keys; Google sign-in and reset check the IP; change-password checks the user id. When `REDIS_URL` is unset the limiter is disabled and the API logs a warning at startup. A Redis error during a check fails open and is logged, so a Redis outage cannot lock every user out. The cost is that brute-force protection disappears during such an outage.
11. **Client address.** `RemoteAddr` is used unless `TRUST_PROXY_HEADERS=true`, in which case `X-Real-IP`, then the first `X-Forwarded-For` value, is used. Enable it only behind a proxy that overwrites those headers.
12. **Mail.** `platform/mail.Sender` has two implementations. `SMTPSender` (net/smtp without authentication, for mailpit on port 1025) is used when `SMTP_HOST` is set. Otherwise `LogSender` logs the subject and a masked recipient, never the body, because bodies carry single-use tokens. Email copy (uz, ru, en) lives in Go (`domain/identity/emails.go`) rather than `packages/i18n`, because the API sends the mail and the web bundle is not available server-side. `BRAND_NAME` (default `Grupon`) supplies the brand in subjects and must be kept in step with `brand.config.ts` at rename time.
13. **Links in email.** Links use `WEB_PUBLIC_URL/{locale}/verify-email?token=` and `WEB_PUBLIC_URL/{locale}/reset-password?token=`. Those web routes are not in the spec and must exist in `apps/web`.
14. **Google sign-in.** Server-side PKCE exchange at `https://oauth2.googleapis.com/token` using `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` and `GOOGLE_REDIRECT_URI`, all three required together. The request's `redirect_uri` must equal the configured value. The ID token is read from the TLS response of the token endpoint and its `iss`, `aud`, `exp` and `email_verified` claims are checked. Its JWKS signature is not re-verified, which OpenID Connect Core 3.1.3.7 permits for tokens received directly from the token endpoint. Account matching: `google_sub` first; otherwise a verified email links an existing account with no Google subject; otherwise a new account is created with locale `uz` and currency `UZS`. Without configuration the endpoint returns 503 `google_not_configured`.
15. **Register and forgot-password responses.** Registering an existing email returns 409 `email_already_registered`. This reveals account existence, which the registration response shape requires. Forgot-password returns 202 for every well-formed email. A malformed email fails the contract's `format: email` check and returns 400 `invalid_request`.
16. **Error codes.** Validation failures return 422 `validation_failed` with field codes. A wrong current password returns 422 `current_password_incorrect`. Bodies over 64 KB return 413; unknown JSON fields return 400 `invalid_request`. Rate limits return 429 `rate_limited` with `Retry-After: 900`.
17. **RBAC.** `AuthAPI.RequireRole(...)` is the middleware for future admin and merchant routes. It is tested but mounted on no route yet, because Phase 4 has none.
18. **Not done in this slice.** The double-submit CSRF token from Section 14 is not implemented. The cookies use `SameSite=Lax` and every mutating endpoint takes a JSON body, which blocks the usual cross-site form attacks, but the double-submit token is still required before browser-driven mutations ship.
- 2026-10-31: CLAUDE.md is now the working instruction source; the task list is mirrored in docs/TASKS.md. Subagents are not used. One task per turn.
- 2026-10-31: Web auth uses a same-origin proxy (apps/web/src/app/api/auth/[action]) that forwards to the Go API and re-scopes Set-Cookie to Path=/, so the browser keeps the session on the web origin. Login/register pages built; redirect target is restricted to same-site paths.
- 2026-10-31: Account pages read data via server-side accountGet (access_token cookie → Bearer). Profile is read-only for now; PATCH /me and PUT /me/password are not wired yet.
- 2026-10-31: Checkout flow is guest cart (Zustand, persisted) → /checkout (requires session; login returns via next) → POST /api/bff/orders (Idempotency-Key) → /api/bff/orders/{id}/pay → /checkout/mock/{payment} → success (polls until paid) or failed (retry). Browser calls reach the Go API through the same-origin BFF proxy, which only allows the commerce operations it lists.
- 2026-10-31: Worker runs reservation release, voucher expiry and deal lifecycle/counter sweeps (idempotent, verified by a second-run-no-change test). Concurrency test: 100 buyers for 10 units → exactly 10 orders. Playwright purchase test passes against the production build.
- 2026-10-31: Purchase e2e waits up to 20s for the mock payment step; the default 5s was too short for the order + payment calls on a cold server.
- 2026-10-31: Merchant registration is one transaction (merchant + first location + merchant_owner role). The category is validated but not stored: merchants have no category column, categories belong to deals. Switch-to-business shows for accounts with a merchant_* role.
- 2026-10-31: Merchant context = first merchant_* role assignment of the signed-in account. Panel pages live under a route group with a role guard; the API re-checks every operation. Cashiers see read-only business data. Dashboard latest_orders is empty for now.
- 2026-10-31 (assumption, pending user confirmation): media uploads use a storage interface with a local-disk adapter for development; the S3/MinIO presigned adapter is implemented later and must be tested before launch.
- 2026-10-31: After re-running the seed the Redis catalog cache can serve stale city/category ids; bump catalog:version (or call the internal invalidation hook) after seeding. Seed should do this itself before launch.
- 2026-10-31: Media library backend: merchant_media_assets table (migration 00004), local storage adapter (MEDIA_LOCAL_DIR, MEDIA_PUBLIC_BASE), presigned-style upload with image format check; PUT /merchant/media/{id}/content is a contract addition for the local adapter (single application/octet-stream body).

## Phase B3 merchant deal workflow (backend)
- Review fingerprint lives in `deals.review_fingerprint` (migration 00005). It covers title, descriptions, terms, redemption, highlights, included items, FAQ and option titles/prices/active flags. Stock is excluded so stock-only edits never trigger review.
- Editing a live deal (scheduled, active, paused, sold_out) with a changed fingerprint sends it to `pending_review`. Editing a rejected deal returns it to `draft`. Submit is allowed only from `draft` or `rejected`, otherwise 409 `invalid_deal_state`.
- Options are never deleted. Options missing from an update are deactivated, so stock history stays intact.
- Gap: owned deal detail returns only active options (reuses the public summary read), so inactive options are hidden from the merchant.
- Gap: an option id that belongs to another deal is silently skipped by the upsert. Fix by checking ownership in `References`.
- Gap: media URLs are not yet checked against the merchant's ready media assets.
- Qolgan ishlar (2026-10-31): B3 (web), B5, B7, C1, D1, D4, D5, E1.
- B3 web gap: `MediaAsset` has no width/height/blurhash, but `DealMediaInput` requires them. The wizard sends 1x1 and an empty blurhash, so the stored dimensions are wrong. Fix by adding dimensions to the media asset contract or deriving them on the server.
- B3 web gap: the preview is a simple summary card, not the real DealCard component.
- D1 placement: `apps/admin` is not scaffolded yet. The moderation queue sits in apps/web at `/admin/moderation`, gated to superadmin/admin/moderator by the page and the API. Move it to apps/admin when that app exists.
- D1 gap: the reject form checks the reason length only in the browser; the API enforces the same rule (3–500 characters).
- B5 redemption: one conditional UPDATE (`status = active` and inside validity) with the audit row in one transaction, so two cashiers cannot redeem the same voucher. On failure the code reads the current state to return the exact reason.
- B5 gap: `GET /merchant/redemptions` (recent redemptions) is not implemented yet and returns 501.
- B5 gap: lookup shows only active options (reuses the public summary read), so a voucher bought on an option that was later deactivated shows a missing option name.
- B5 gap: no live device test of the camera scanner (@zxing/browser 0.1.5); manual code entry is verified by typecheck and lint only.
- B7 e2e: `e2e/merchant.spec.mjs` covers the owner flow through the API (create with media, submit, admin approve, buyer order and mock pay) and the redeem step in a 390px browser. Requires the API on 8080 with the dev seed.
- E2E flakiness: `purchase.spec.mjs` failed once in a full run (success heading timeout) and passed on the next full run and when run alone. Webhook timing is the likely cause; not yet fixed.
- C1 review: one review per redeemed voucher enforced by the unique voucher_id; repeat reports are idempotent (unique reporter per review). Report button on the deal page is now functional; the owner review list has no report button by design.
- D4 reviews and reports: review reports close as `resolved` (hide) or `dismissed`. Hiding a review is audited each time, and a repeat hide is accepted and audited again, which is harmless.
- D4 gap: support and disputes, roles management UI, and system settings are not built. There is no support/dispute table yet, and the role-assignment endpoint does not exist.
- D4 gap: the reports and audit pages live in apps/web under /admin, as with the moderation queue, until apps/admin exists.
- D5 tests: store-level tests (admin_audit_test.go) run against a reseeded dev database because the seed truncates tables; they need TEST_DATABASE_URL.
- Fix: deal JSON arrays (faq, whats_included, highlights) are stored as [] instead of null when empty.
- E1: production images are written but unbuilt (no Docker on this machine). Verify with docker build for services/api (targets api, worker, migrate) and apps/web before release.
- BFF returned 204 with a body, which crashed approve/hide/dismiss and delete actions. Fixed: 204 is now returned with no body.
- e2e purchase.spec flaked again (stays on /checkout after the pay click). Suspect repeated runs exhausting seed stock. Not fixed.
