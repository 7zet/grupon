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
