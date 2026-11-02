# Local deals marketplace

This repository contains the web application, Go API and worker for the marketplace described in
[`CLAUDE.md`](./CLAUDE.md).

## Prerequisites

- Node.js 26.13.0 (the minimum supported local line is 24.15.0)
- pnpm 12.16.1 through Corepack
- Go 1.25.6
- Docker with Compose

## Local setup

```sh
corepack enable
pnpm install
docker compose --env-file .env.example -f infra/docker-compose.yml up -d
pnpm dev
```

`pnpm dev` starts the web app on port 3000, the API on port 8080 and the worker. Infrastructure is
started separately so application processes can be tested without Docker.

On a locked-down Windows Node installation, Corepack may not be able to write shims beside
`node.exe`. Install the shim into a user-writable directory and add it to `PATH`:

```powershell
$shimDirectory = Join-Path $env:LOCALAPPDATA 'corepack-shims'
New-Item -ItemType Directory -Force -Path $shimDirectory | Out-Null
corepack enable --install-directory $shimDirectory
$env:PATH = "$shimDirectory;$env:PATH"
```

## Quality gates

```sh
pnpm format:check
pnpm lint
pnpm typecheck
pnpm test
pnpm build
go build ./...
go vet ./...
go test ./...
```

Generation and seed commands already exist as Phase 0 aggregators. Their implementations are added
with the API contract and database in Phase 1.
