# Environment variables

Plain-language reference for every setting. The API reads its settings at start-up and refuses to start when a required value is missing or weak. Copy `.env.local.example` to `.env.local` (git-ignored); `scripts/demo.ps1` does this on its first run and passes the values to the API and the web app.

## API (`services/api`)

| Variable                                                                                      | Required in production       | Default                       | Meaning                                                                          |
| --------------------------------------------------------------------------------------------- | ---------------------------- | ----------------------------- | -------------------------------------------------------------------------------- |
| `APP_ENV`                                                                                     | yes                          | `development`                 | `production` turns on strict checks and disables seeding.                        |
| `API_ADDR`                                                                                    | no                           | `:8080`                       | Address the HTTP server listens on.                                              |
| `LOG_LEVEL`                                                                                   | no                           | `info`                        | `debug`, `info`, `warn` or `error`.                                              |
| `DATABASE_URL`                                                                                | yes                          | none                          | PostgreSQL connection string.                                                    |
| `REDIS_URL` | yes in production | empty | Redis for caching and OTP rate limits. `APP_ENV=production` refuses to start without it and rate limits fail closed. In the demo it may be empty (limits are then off). |
| `JWT_SECRET` | yes | dev value in `development` | Signs access tokens. At least 32 random bytes; required outside development. |
| `VOUCHER_QR_SECRET`                                                                           | yes                          | none                          | Signs voucher QR tokens. At least 32 bytes.                                      |
| `INTERNAL_API_TOKEN` | no | empty | Protects `POST /internal/cache/invalidate` (header `X-Internal-Token`); the endpoint is off when empty. |
| `RUN_WORKER_IN_API` | no | `false` | Runs the background jobs inside the API process (the local demo sets `true`). |
| `DEMO_MODE`                                                                                   | no                           | `false`                       | Enables demo-only OTP display, mock-payment buttons, and the header badge.        |
| `PAYMENT_MOCK_ENABLED`                                                                        | no                           | `false`                       | Enables the mock provider only when `DEMO_MODE=true`.                              |
| `PAYMENT_MOCK_WEBHOOK_SECRET`                                                                 | when mock is on              | none                          | At least 32 bytes.                                                               |
| `MANUAL_PAYMENT_ENABLED`                                                                      | no                           | `false`                       | Enables the pilot manual-payment confirmation mode.                              |
| `MANUAL_PAYMENT_HOLD`                                                                         | no                           | `24h`                         | Reservation hold used for manual-payment attempts.                               |
| `MANUAL_PAYMENT_INSTRUCTIONS`                                                                 | when manual mode is on       | support message               | Instructions returned to the buyer for a manual payment.                         |
| `SMS_PROVIDER` | no | `log` | `log` (codes in the log) is accepted only with `DEMO_MODE=true`; `eskiz` sends real SMS. |
| `ESKIZ_EMAIL` / `ESKIZ_PASSWORD` / `ESKIZ_SENDER` / `ESKIZ_BASE_URL` | with `eskiz` | empty / official URL | Eskiz account for real SMS. |
| `SMS_DAILY_CAP_PER_IP` / `SMS_DAILY_CAP_GLOBAL` | no | `50` / `1000` | Daily limits on sign-in codes, against SMS cost abuse. |
| `SITE_URL` | yes | `http://localhost:3000` | Public address of the site (localhost, or the ngrok https URL). Used for links and the Origin check; overrides `WEB_PUBLIC_URL`. |
| `ALLOWED_ORIGINS`                                                                              | no                           | empty       | Comma-separated extra web origins allowed to send cookie-authenticated changes.                      |
| `WEB_PUBLIC_URL` | no | `http://localhost:3000` | Older name for `SITE_URL`, read only when `SITE_URL` is empty. |
| `WEB_REVALIDATE_URL` | no | empty | Web endpoint that clears cached pages (`http://localhost:3000/api/revalidate`). |
| `WEB_REVALIDATE_SECRET` | with the URL | empty | Shared secret sent to the web revalidation endpoint. |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_FROM` | no | empty | Mail server. An empty `SMTP_HOST` means emails are only logged (the demo). |
| `MEDIA_LOCAL_DIR` | no | `.data/media` | Folder where business photos are stored. Must survive restarts. |
| `DOCUMENTS_LOCAL_DIR` | no | `.data/documents` | Private folder for business certificates and bank letters. Never served publicly. |
| `MEDIA_PUBLIC_BASE` | no | `http://localhost:8080/media` | Address that serves those images; the web app fetches them server-side. |
| `TRUST_PROXY_HEADERS`                                                                         | behind a proxy only          | `false`                       | Read the client address from proxy headers. Turn on only behind a trusted proxy. |
| `BRAND_NAME`                                                                                  | no                           | default brand                 | Name shown in emails.                                                            |
| `SHUTDOWN_TIMEOUT`                                                                            | no                           | `10s`                         | Time allowed for in-flight requests on stop.                                     |
| `HTTP_READ_HEADER_TIMEOUT` / `HTTP_READ_TIMEOUT` / `HTTP_WRITE_TIMEOUT` / `HTTP_IDLE_TIMEOUT` | no                           | `5s` / `15s` / `30s` / `60s`  | HTTP timeouts.                                                                   |
| `ACCESS_TOKEN_TTL` / `REFRESH_TOKEN_TTL`                                                      | no                           | `15m` / `30d`                 | Token lifetimes.                                                                 |

## Web (`apps/web`)

| Variable                          | Set at              | Meaning                                                           |
| --------------------------------- | ------------------- | ----------------------------------------------------------------- |
| `NEXT_PUBLIC_API_BASE_URL` | build and run time | API address the web server calls, for example `http://localhost:8080/api/v1`. Browsers never call it. |
| `SITE_URL` (or `NEXT_PUBLIC_SITE_URL`) | run time | Public web address, used for page metadata. |
| `APP_ENV` | build and run time | Not `production`: the image optimizer may load photos from `localhost` (the local demo). |
| `WEB_REVALIDATE_SECRET`           | run time            | Must match the API value.                                         |
| `SUPPORT_PHONE` / `SUPPORT_TELEGRAM` / `SUPPORT_EMAIL` | run time | Public support contact values; do not prefix with `NEXT_PUBLIC_`. Empty values hide that channel. |
| `USE_MSW` / `NEXT_PUBLIC_USE_MSW` | never in production | Mock API for development without the backend. Keep `false`.       |

`NEXT_PUBLIC_*` values are copied into the build, so changing them needs a new `next build` (`scripts/demo.ps1` rebuilds on every start unless `-SkipBuild`).

## Database and workers

| Variable         | Used by              | Meaning                                                        |
| ---------------- | -------------------- | -------------------------------------------------------------- |
| `DATABASE_URL`   | api, worker, migrate | Same database for all three.                                   |
| `MIGRATIONS_DIR` | migrate | Folder with migration files (`services/api/migrations`; `scripts/demo.ps1` sets it). |

Run `migrate` once per release before starting the new API and worker. `seed` and `bootstrap` rebuild the demo data: they **empty the tables first** (all users, orders and vouchers are deleted). `seed` refuses to run when `APP_ENV=production` unless `DEMO_MODE=true`.

## Secrets checklist

- Generate `JWT_SECRET`, `VOUCHER_QR_SECRET`, `PAYMENT_MOCK_WEBHOOK_SECRET`, `WEB_REVALIDATE_SECRET` and `INTERNAL_API_TOKEN` separately with a cryptographic random generator.
- Never reuse the `.env.local.example` values; they are public.
- Keep secrets in the platform's secret store, not in the image.

## Test accounts (development seed)

Sign-in is by phone code. In demo mode the code is returned as `dev_code` by `POST /auth/phone/code` and shown on
the login page. The full list of demo accounts (buyers, every business's owner, manager and cashier, and the staff
roles) is in `docs/DEMO.md`. Automated tests also use their own accounts (`e2e/*.spec.mjs`): owners B, C and D
(`+998900000005`, `+998900000006`, `+998900000007`), admin B (`+998900000008`) and cashier B (`+998900000009`).

### Development two-factor code

Staff accounts (superadmins `+998900000004` and `+998900000008`, moderator, support and finance) ask for an
authenticator code after the phone code. In the seed they all share the development Base32 key `JBSWY3DPEHPK3PXP`:
add it to an authenticator app and enter its six-digit code on the second screen. This key exists only in the demo
seed; it is not a production bypass.

For a real enrollment, phone sign-in presents a QR setup screen, scan it with an authenticator, and
enter the generated code. The ten recovery codes are shown once after enablement; only hashes are
stored in `two_factor_recovery_codes`, so plaintext recovery codes cannot be retrieved later. Save
them in the operator's approved secret store when they are displayed.

## Local PostgreSQL

- Data directory: `C:\dev\grupon-pgdata` (port 5433, user `app`, database `app`; the folder keeps its old name). Never inside Windows Temp:
  Windows deletes empty folders there and the cluster stops starting.
- Start, stop, check: `scripts/dev-db.sh start | stop | status` (Git Bash). `scripts/demo.ps1` starts it when needed.
- Tests: `TEST_DATABASE_URL=postgres://app:app_dev_password@localhost:5433/app?sslmode=disable go test ./...`.
  The store tests reseed this database, so run `scripts\demo.ps1 -Reset` afterwards to restore the demo data.
- Move procedure used once (2026-11-03): stop the server, copy the whole folder, start with the new `-D`,
  verify (store tests pass), and only then delete the old folder.

## Playwright

- `e2e/playwright.config.mjs` starts the web app with `corepack pnpm`, so it works without `pnpm` on PATH.
- Each spec uses its own seeded accounts; workers run in parallel.
- Locally, start the demo first; the config reuses the running web server.
- `e2e/layout.spec.mjs` fails when page styles are missing; `e2e/roles.spec.mjs` checks every role at 390 and 1440 px.
