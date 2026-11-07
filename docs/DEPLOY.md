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

## Environment mode (`APP_ENV`)

The live API currently logs `environment: development` because `APP_ENV` is not set on Render. Set
`APP_ENV=production` on Render once these are in place, or the API refuses to start:
`DATABASE_URL`, `JWT_SECRET`, `VOUCHER_QR_SECRET`, `REDIS_URL` (OTP rate limits fail closed without
it), and either a real `SMS_PROVIDER` or `DEMO_MODE=true` (`SMS_PROVIDER=log` is accepted only with
`DEMO_MODE=true`). Also set `WEB_PUBLIC_URL` to the exact Vercel origin: cookie-authenticated changes
from any other browser origin are refused with `origin_not_allowed`.

Probe `GET /healthz` for process liveness and `GET /readyz` for database readiness. Put the API and
web behind HTTPS; the merchant scanner needs HTTPS for camera access on real phones.

Back up PostgreSQL with `DATABASE_URL=... scripts/pg-backup.sh /secure/grupon-$(date +%F).dump`.
Test restore regularly in an isolated database with `scripts/pg-restore.sh backup.dump`; it requires
an explicit `RESTORE` confirmation because it replaces database objects.

Before launch: set unique JWT/QR secrets, set `PAYMENT_MOCK_ENABLED=false`, keep
`MANUAL_PAYMENT_ENABLED=false` unless the documented pilot process is active, configure SMTP and
media storage, restrict database access, and monitor `/readyz` plus backup failures.
