# Dealo — local deals marketplace (demo)

Dealo is a demo of a local-deals voucher marketplace for Tashkent: buyers pay (test payment), get a QR
voucher, and a business cashier redeems it. It has a buyer site, a business panel (`/merchant`) and an admin
panel (`/admin`), in Uzbek, Russian and English. There is no real money and no hosting: the demo runs on
your own Windows PC and can be shown to others through an ngrok tunnel ([`docs/SHARE.md`](./docs/SHARE.md)).

Stack: Go API and worker (`services/api`), Next.js web (`apps/web`), PostgreSQL, API contract in
`packages/api-spec/openapi.yaml`.

## Run the demo in 3 steps

You need once: PostgreSQL 16, Go, Node.js with Corepack (`corepack enable`), then `corepack pnpm install`.

1. **Postgres.** The local database lives in `C:\dev\grupon-pgdata` on port 5433
   ([`docs/ENVIRONMENT.md`](./docs/ENVIRONMENT.md) explains how to create it the first time).
   The script starts it for you if it is stopped.
2. **Start everything** (PowerShell, from the repository folder):

   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts\demo.ps1
   ```

   It checks Postgres, applies migrations, loads the demo data when the database is empty, builds the web
   app in production mode, starts the API and the web app, and prints the addresses and demo accounts.
   Use `-Reset` to wipe and re-load the demo data. Ctrl+C stops everything.
3. **Open** <http://localhost:3000/uz>. Sign in with a demo phone from [`docs/DEMO.md`](./docs/DEMO.md);
   the sign-in code is shown on the page.

Settings live in `.env.local` (created from [`.env.local.example`](./.env.local.example) on the first run;
it is git-ignored).

## Documents

- [`docs/DEMO.md`](./docs/DEMO.md) — demo accounts and a 10-minute script
- [`docs/SHARE.md`](./docs/SHARE.md) — show the demo to others with ngrok
- [`docs/HANDOVER.md`](./docs/HANDOVER.md) — what is and is not included, next steps for a real launch
- [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md), [`docs/ENVIRONMENT.md`](./docs/ENVIRONMENT.md)
- [`AGENTS.md`](./AGENTS.md) — instructions for coding agents

## Development checks

```sh
corepack pnpm lint && corepack pnpm typecheck && corepack pnpm test
go vet ./... && go test ./...        # Go tests need TEST_DATABASE_URL (see .env.local.example)
corepack pnpm exec playwright test -c e2e/playwright.config.mjs   # with the demo running
```

On a locked-down Windows Node install, Corepack may not be able to write shims beside `node.exe`:

```powershell
$shimDirectory = Join-Path $env:LOCALAPPDATA 'corepack-shims'
New-Item -ItemType Directory -Force -Path $shimDirectory | Out-Null
corepack enable --install-directory $shimDirectory
$env:PATH = "$shimDirectory;$env:PATH"
```

Docker files (`infra/`, `docker-compose.prod.yml`) are optional and not verified.
