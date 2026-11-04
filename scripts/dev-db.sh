#!/usr/bin/env bash
# Local PostgreSQL for development and tests (port 5433).
# Data lives outside Windows Temp, which deletes empty folders and breaks the cluster.
#   scripts/dev-db.sh start | stop | status
set -eu
PGBIN="${PGBIN:-/c/Program Files/PostgreSQL/16/bin}"
PGDATA="${PGDATA:-C:/dev/grupon-pgdata}"
PGLOG="${PGLOG:-C:/dev/grupon-pgdata.log}"
PORT="${PGPORT:-5433}"

case "${1:-status}" in
  start) "$PGBIN/pg_ctl.exe" -D "$PGDATA" -l "$PGLOG" -o "-p $PORT" -w start ;;
  stop) "$PGBIN/pg_ctl.exe" -D "$PGDATA" -m fast stop ;;
  status) "$PGBIN/pg_isready.exe" -h localhost -p "$PORT" ;;
  *) echo "usage: $0 start|stop|status" >&2; exit 2 ;;
esac
