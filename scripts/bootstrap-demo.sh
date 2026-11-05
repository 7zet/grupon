#!/usr/bin/env bash
set -euo pipefail
: "${DATABASE_URL:?DATABASE_URL is required}"
DEMO_MODE=true go run ./services/api/cmd/bootstrap
