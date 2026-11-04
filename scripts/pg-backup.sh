#!/usr/bin/env bash
set -euo pipefail
: "${DATABASE_URL:?DATABASE_URL is required}"
destination=${1:?usage: scripts/pg-backup.sh /secure/backup-file.dump}
umask 077
pg_dump --format=custom --no-owner --no-privileges --file "$destination" "$DATABASE_URL"
echo "backup written: $destination"
