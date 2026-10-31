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
