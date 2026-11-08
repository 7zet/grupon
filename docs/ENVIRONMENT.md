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
| `JWT_SECRET`                                                                                  | yes                          | none                          | Signs access tokens. Use at least 32 random bytes.                               |
| `VOUCHER_QR_SECRET`                                                                           | yes                          | none                          | Signs voucher QR tokens. At least 32 bytes.                                      |
| `INTERNAL_API_TOKEN`                                                                          | yes                          | none                          | Protects `POST /internal/cache/invalidate`.                                      |
| `DEMO_MODE`                                                                                   | no                           | `false`                       | Enables demo-only OTP display, mock-payment buttons, and the header badge.        |
| `PAYMENT_MOCK_ENABLED`                                                                        | no                           | `false`                       | Enables the mock provider only when `DEMO_MODE=true`.                              |
| `PAYMENT_MOCK_WEBHOOK_SECRET`                                                                 | when mock is on              | none                          | At least 32 bytes.                                                               |
| `MANUAL_PAYMENT_ENABLED`                                                                      | no                           | `false`                       | Enables the pilot manual-payment confirmation mode.                              |
| `MANUAL_PAYMENT_HOLD`                                                                         | no                           | `24h`                         | Reservation hold used for manual-payment attempts.                               |
| `MANUAL_PAYMENT_INSTRUCTIONS`                                                                 | when manual mode is on       | support message               | Instructions returned to the buyer for a manual payment.                         |
| `SMS_PROVIDER` | no | `log` | `log` (codes in the log) is accepted only with `DEMO_MODE=true`; `eskiz` sends real SMS. |
| `SITE_URL`                                                                              | yes                          | `http://localhost:3000`       | Public address of the site (http://localhost:3000, or the ngrok https URL). Overrides `WEB_PUBLIC_URL`.                      |
| `ALLOWED_ORIGINS`                                                                              | no                           | empty       | Comma-separated extra web origins allowed to send cookie-authenticated changes.                      |
| `WEB_PUBLIC_URL`                                                                              | yes                          | `http://localhost:3000`       | Public address of the web app, used in links and redirects.                      |
| `SUPPORT_PHONE` / `SUPPORT_TELEGRAM` / `SUPPORT_EMAIL`                                        | no                           | empty                         | Public support contacts rendered in the web footer, Help page, and voucher page. Set these on the web deployment; empty channels stay hidden. |
| `WEB_REVALIDATE_URL`                                                                          | yes                          | none                          | Web endpoint that clears cached pages.                                           |
| `WEB_REVALIDATE_SECRET`                                                                       | yes                          | none                          | Shared secret sent to the web revalidation endpoint.                             |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_FROM`                                                       | yes                          | none                          | Mail server for confirmation emails.                                             |
| `MEDIA_LOCAL_DIR`                                                                             | yes                          | `.data/media`                 | Folder where merchant images are stored. Must be a persistent volume.            |
| `DOCUMENTS_LOCAL_DIR` | yes | `.data/documents` | Private folder for merchant certificates and bank letters. Never served publicly; must be a persistent volume with restricted access. |
| `MEDIA_PUBLIC_BASE`                                                                           | yes                          | `http://localhost:8080/media` | Public address that serves those images.                                         |
| `TRUST_PROXY_HEADERS`                                                                         | behind a proxy only          | `false`                       | Read the client address from proxy headers. Turn on only behind a trusted proxy. |
| `BRAND_NAME`                                                                                  | no                           | default brand                 | Name shown in emails.                                                            |
| `SHUTDOWN_TIMEOUT`                                                                            | no                           | `10s`                         | Time allowed for in-flight requests on stop.                                     |
| `HTTP_READ_HEADER_TIMEOUT` / `HTTP_READ_TIMEOUT` / `HTTP_WRITE_TIMEOUT` / `HTTP_IDLE_TIMEOUT` | no                           | `5s` / `15s` / `30s` / `60s`  | HTTP timeouts.                                                                   |
| `ACCESS_TOKEN_TTL` / `REFRESH_TOKEN_TTL`                                                      | no                           | `15m` / `30d`                 | Token lifetimes.                                                                 |

## Web (`apps/web`)

| Variable                          | Set at              | Meaning                                                           |
| --------------------------------- | ------------------- | ----------------------------------------------------------------- |
| `NEXT_PUBLIC_API_BASE_URL`        | build time          | Public API address, for example `https://api.example.com/api/v1`. |
| `NEXT_PUBLIC_SITE_URL`            | build time          | Public web address.                                               |
| `WEB_REVALIDATE_SECRET`           | run time            | Must match the API value.                                         |
| `SUPPORT_PHONE` / `SUPPORT_TELEGRAM` / `SUPPORT_EMAIL` | run time | Public support contact values; do not prefix with `NEXT_PUBLIC_`. Empty values hide that channel. |
| `USE_MSW` / `NEXT_PUBLIC_USE_MSW` | never in production | Mock API for development without the backend. Keep `false`.       |

`NEXT_PUBLIC_*` values are copied into the browser bundle at build time, so changing them needs a new image.

## Database and workers

| Variable         | Used by              | Meaning                                                        |
| ---------------- | -------------------- | -------------------------------------------------------------- |
| `DATABASE_URL`   | api, worker, migrate | Same database for all three.                                   |
| `MIGRATIONS_DIR` | migrate              | Folder with migration files. The image sets `/app/migrations`. |

Run `migrate` once per release before starting the new API and worker. `seed` is development only and refuses to run when `APP_ENV=production`.

## Secrets checklist

- Generate `JWT_SECRET`, `VOUCHER_QR_SECRET`, `PAYMENT_MOCK_WEBHOOK_SECRET`, `WEB_REVALIDATE_SECRET` and `INTERNAL_API_TOKEN` separately with a cryptographic random generator.
- Never reuse the `.env.local.example` values; they are public.
- Keep secrets in the platform's secret store, not in the image.

## Test accounts (development seed)

Sign-in is by phone code. In development the code is returned as `dev_code` by `POST /auth/phone/code`
and shown on the login page. Run `seed` to create these accounts; the seed never deletes accounts that
people registered themselves.

| Phone | Role | Use |
|---|---|---|
| +998900000001 | user | buyer: checkout, vouchers |
| +998900000002 | merchant_owner | Atlas Wellness 1: deals, staff, submit business |
| +998900000003 | merchant_cashier | Atlas Wellness 1: redemption only |
| +998900000004 | superadmin | admin: moderation, merchant review, finance |
| +998900000005 | merchant_owner | Owner B: merchant 2 (e2e merchant flow) |
| +998900000006 | merchant_owner | Owner C: merchant 3 (e2e admin UI) |
| +998900000007 | merchant_owner | Owner D: merchant 4 (e2e merchant panel) |
| +998900000008 | superadmin | Admin B: admin approval inside the e2e merchant flow |
| +998900000009 | merchant_cashier | Cashier B: redemption for merchant 2 |
| +998910000004 | user | extra buyer (customer04@local.test) |

### Development admin 2FA

The seeded superadmins (`+998900000004` and `+998900000008`) require TOTP after the normal
development phone-code sign-in. Add the development-only Base32 secret
`JBSWY3DPEHPK3PXP` to an authenticator app, then enter its six-digit code on the second screen.
This seed data is loaded only by `cmd/seed`, which refuses production; it is not a production bypass.

For a real enrollment, phone sign-in presents a QR setup screen, scan it with an authenticator, and
enter the generated code. The ten recovery codes are shown once after enablement; only hashes are
stored in `two_factor_recovery_codes`, so plaintext recovery codes cannot be retrieved later. Save
them in the operator's approved secret store when they are displayed.

## Local PostgreSQL

- Data directory: `C:devgrupon-pgdata` (port 5433, user `app`, database `app`). Never inside Windows Temp:
  Windows deletes empty folders there and the cluster stops starting.
- Start, stop, check: `scripts/dev-db.sh start | stop | status` (Git Bash).
- Tests: `TEST_DATABASE_URL=postgres://app:app_dev_password@localhost:5433/app?sslmode=disable go test ./...`.
- Move procedure used once (2026-11-03): stop the server, copy the whole folder, start with the new `-D`,
  verify (store tests pass), and only then delete the old folder.

## Playwright

- `e2e/playwright.config.mjs` starts the web app with `corepack pnpm`, so it works without `pnpm` on PATH.
- Each spec uses its own seeded accounts; workers run in parallel.
