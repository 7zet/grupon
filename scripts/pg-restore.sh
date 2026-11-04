#!/usr/bin/env bash
set -euo pipefail
: "${DATABASE_URL:?DATABASE_URL is required}"
source_file=${1:?usage: scripts/pg-restore.sh /secure/backup-file.dump}
if [[ ! -f "$source_file" ]]; then echo "backup not found: $source_file" >&2; exit 1; fi
read -r -p "Restore into DATABASE_URL (this overwrites database objects)? type RESTORE: " confirmation
[[ "$confirmation" == RESTORE ]] || { echo "cancelled"; exit 1; }
pg_restore --clean --if-exists --no-owner --no-privileges --dbname "$DATABASE_URL" "$source_file"
