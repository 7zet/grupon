# Deployment

The production composition is provider-neutral: supply managed PostgreSQL (and optional Redis), a
container runtime, persistent storage for `/app/data`, and HTTPS termination. Copy
`.env.production.example` to a private `.env.production`; do not store its real values in Git.

Build and apply the schema first:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml run --rm migrate
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build api worker web
```

Probe `GET /healthz` for process liveness and `GET /readyz` for database readiness. Put the API and
web behind HTTPS; the merchant scanner needs HTTPS for camera access on real phones.

Back up PostgreSQL with `DATABASE_URL=... scripts/pg-backup.sh /secure/grupon-$(date +%F).dump`.
Test restore regularly in an isolated database with `scripts/pg-restore.sh backup.dump`; it requires
an explicit `RESTORE` confirmation because it replaces database objects.

Before launch: set unique JWT/QR secrets, set `PAYMENT_MOCK_ENABLED=false`, keep
`MANUAL_PAYMENT_ENABLED=false` unless the documented pilot process is active, configure SMTP and
media storage, restrict database access, and monitor `/readyz` plus backup failures.
