# Deployment (native runtime, no Docker)

Recommended free demo setup (verified from provider documentation, November 2026; **check the provider site before deploying**):

- **Web — Vercel Hobby:** free for personal, non-commercial use only.
- **API — Render free web service:** use a native Go build (no Docker) with `RUN_WORKER_IN_API=true`.
  It spins down after 15 minutes idle, takes about one minute to wake, and includes 750 instance hours/month.
- **Database — Neon free Postgres:** autosuspends after about five minutes idle. Do not use Render free Postgres: it expires after 30 days.

Background jobs pause while the API sleeps and catch up on wake-up; that is fine for this demo. Copy
`.env.production.example` to a private `.env.production`; do not store its real values in Git.

Build and apply the schema first:

```bash
DEMO_MODE=true DATABASE_URL='postgres://…' go run ./services/api/cmd/bootstrap
# Bash alternative: DATABASE_URL='postgres://…' scripts/bootstrap-demo.sh
go build -o bin/grupon-api ./services/api/cmd/api
RUN_WORKER_IN_API=true ./bin/grupon-api
```

## Optional Eskiz SMS on Render

Leave `SMS_PROVIDER=log` for the normal demo: sign-in codes are logged and shown only when
`DEMO_MODE=true`. To send SMS, set these **Render API service** environment variables (never Vercel):
`SMS_PROVIDER=eskiz`, `ESKIZ_EMAIL`, `ESKIZ_PASSWORD`, `ESKIZ_SENDER`,
`SMS_DAILY_CAP_PER_IP`, and `SMS_DAILY_CAP_GLOBAL`. `ESKIZ_BASE_URL` is optional and defaults to
the official Eskiz API URL. Credentials come from the Eskiz account; keep them out of Git.

## Environment variables

**The hosted demo stays in demo mode:** `DEMO_MODE=true`, `APP_ENV` unset (development behaviour), `SMS_PROVIDER=log`, no Redis. Real production is different, see the last section.

### Render (API service)

| Variable | Demo value | Notes |
|---|---|---|
| `DATABASE_URL` | Neon connection string (`sslmode=require`) | Required. |
| `JWT_SECRET`, `VOUCHER_QR_SECRET` | 32+ random bytes each | Required. |
| `DEMO_MODE` | `true` | Shows the sign-in code, enables mock payment and the header badge. |
| `PAYMENT_MOCK_ENABLED` | `true` | Only works with `DEMO_MODE=true`. |
| `PAYMENT_MOCK_WEBHOOK_SECRET` | 32+ random bytes | Needed when the mock is on outside development. |
| `SMS_PROVIDER` | `log` | `eskiz` needs `ESKIZ_EMAIL`, `ESKIZ_PASSWORD`, `ESKIZ_SENDER`, `SMS_DAILY_CAP_PER_IP`, `SMS_DAILY_CAP_GLOBAL` (`ESKIZ_BASE_URL` optional). |
| `WEB_PUBLIC_URL` | exact Vercel origin | Links, redirects and the Origin check for cookie sessions. |
| `SUPPORT_PHONE`, `SUPPORT_TELEGRAM`, `SUPPORT_EMAIL` | your contacts | Empty hides a channel. |
| `WEB_REVALIDATE_URL`, `WEB_REVALIDATE_SECRET` | Vercel revalidate endpoint and shared secret | Clears cached pages. |
| `INTERNAL_API_TOKEN` | random | Protects the cache invalidate endpoint. |
| `MEDIA_PUBLIC_BASE`, `MEDIA_LOCAL_DIR`, `DOCUMENTS_LOCAL_DIR` | API URL + `/media`, folders | Free-plan disk is erased on redeploy. |
| `RUN_WORKER_IN_API` | `true` | Runs the worker inside the API process. |
| `TRUST_PROXY_HEADERS` | `true` on Render | Reads the client address from proxy headers. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM` | optional | Without them emails are only logged. |

### Vercel (web)

| Variable | Value |
|---|---|
| `NEXT_PUBLIC_API_BASE_URL` | `https://<render-api>/api/v1` (build time) |
| `NEXT_PUBLIC_SITE_URL` | the Vercel URL (build time) |
| `WEB_REVALIDATE_SECRET` | same as on Render |
| `SUPPORT_PHONE`, `SUPPORT_TELEGRAM`, `SUPPORT_EMAIL` | same contacts (run time, no `NEXT_PUBLIC_` prefix) |
| `USE_MSW`, `NEXT_PUBLIC_USE_MSW` | `false` |

### Real production (not the demo)

Set `APP_ENV=production`. The API then refuses to start without `DATABASE_URL`, `JWT_SECRET`, `VOUCHER_QR_SECRET` and **`REDIS_URL`**, and OTP rate limits fail closed when Redis is missing or down. It also refuses `SMS_PROVIDER=log` unless `DEMO_MODE=true`, so set `DEMO_MODE=false` and a real SMS provider. Set `WEB_PUBLIC_URL` to the exact web origin: cookie-authenticated changes from any other browser origin return 403 `origin_not_allowed`.

Probe `GET /healthz` for process liveness and `GET /readyz` for database readiness. Put the API and
web behind HTTPS; the merchant scanner needs HTTPS for camera access on real phones.

Back up PostgreSQL with `DATABASE_URL=... scripts/pg-backup.sh /secure/grupon-$(date +%F).dump`.
Test restore regularly in an isolated database with `scripts/pg-restore.sh backup.dump`; it requires
an explicit `RESTORE` confirmation because it replaces database objects.

Before launch: set unique JWT/QR secrets, set `PAYMENT_MOCK_ENABLED=false`, keep
`MANUAL_PAYMENT_ENABLED=false` unless the documented pilot process is active, configure SMTP and
media storage, restrict database access, and monitor `/readyz` plus backup failures.
