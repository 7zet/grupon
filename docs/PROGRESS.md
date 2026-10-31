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
