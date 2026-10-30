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
