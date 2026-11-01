# Environment variables

Plain-language reference for every setting. The API reads its settings at start-up and refuses to start when a required value is missing or weak. Copy `.env.example` to `.env` for local development only.

## API (`services/api`)

| Variable                                                                                      | Required in production       | Default                       | Meaning                                                                          |
| --------------------------------------------------------------------------------------------- | ---------------------------- | ----------------------------- | -------------------------------------------------------------------------------- |
| `APP_ENV`                                                                                     | yes                          | `development`                 | `production` turns on strict checks and disables seeding.                        |
| `API_ADDR`                                                                                    | no                           | `:8080`                       | Address the HTTP server listens on.                                              |
| `LOG_LEVEL`                                                                                   | no                           | `info`                        | `debug`, `info`, `warn` or `error`.                                              |
| `DATABASE_URL`                                                                                | yes                          | none                          | PostgreSQL connection string.                                                    |
| `REDIS_URL`                                                                                   | recommended                  | empty                         | Redis for caching and rate limits. Without it, rate limits are off.              |
| `JWT_SECRET`                                                                                  | yes                          | none                          | Signs access tokens. Use at least 32 random bytes.                               |
| `VOUCHER_QR_SECRET`                                                                           | yes                          | none                          | Signs voucher QR tokens. At least 32 bytes.                                      |
| `INTERNAL_API_TOKEN`                                                                          | yes                          | none                          | Protects `POST /internal/cache/invalidate`.                                      |
| `PAYMENT_MOCK_ENABLED`                                                                        | yes, must be `false`         | `false`                       | Turns on the mock payment provider. Keep it off in production.                   |
| `PAYMENT_MOCK_WEBHOOK_SECRET`                                                                 | when mock is on              | none                          | At least 32 bytes.                                                               |
| `WEB_PUBLIC_URL`                                                                              | yes                          | `http://localhost:3000`       | Public address of the web app, used in links and redirects.                      |
| `WEB_REVALIDATE_URL`                                                                          | yes                          | none                          | Web endpoint that clears cached pages.                                           |
| `WEB_REVALIDATE_SECRET`                                                                       | yes                          | none                          | Shared secret sent to the web revalidation endpoint.                             |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_FROM`                                                       | yes                          | none                          | Mail server for confirmation emails.                                             |
| `MEDIA_LOCAL_DIR`                                                                             | yes                          | `.data/media`                 | Folder where merchant images are stored. Must be a persistent volume.            |
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
- Never reuse the `.env.example` values; they are public.
- Keep secrets in the platform's secret store, not in the image.
