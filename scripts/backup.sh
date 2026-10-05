#!/usr/bin/env bash
# Dump local fac_identity Postgres to backups/
# Usage: ./scripts/backup.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${ENV_FILE:-}"
if [[ -z "$ENV_FILE" ]]; then
  if [[ -f .env ]]; then
    ENV_FILE=.env
  elif [[ -f .env.prod ]]; then
    ENV_FILE=.env.prod
  else
    ENV_FILE=.env
  fi
fi

env_get() {
  local key="$1" default="${2:-}"
  local line val
  if [[ -f "$ENV_FILE" ]]; then
    line="$(grep -E "^${key}=" "$ENV_FILE" | tail -n1 || true)"
    if [[ -n "$line" ]]; then
      val="${line#*=}"
      val="${val%$'\r'}"
      val="${val#\"}"
      val="${val%\"}"
      val="${val#\'}"
      val="${val%\'}"
      printf '%s' "$val"
      return
    fi
  fi
  printf '%s' "$default"
}

CONTAINER="${POSTGRES_CONTAINER:-facinect-postgres}"
DB="$(env_get POSTGRES_IDENTITY_DB fac_identity)"
USER="$(env_get POSTGRES_IDENTITY_USER identity)"
PASS="$(env_get POSTGRES_IDENTITY_PASSWORD identity_local_change_me)"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="$ROOT/backups"
OUT_FILE="$OUT_DIR/${DB}_${STAMP}.sql.gz"

mkdir -p "$OUT_DIR"

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "error: container $CONTAINER is not running" >&2
  exit 1
fi

echo "==> backing up $DB -> $OUT_FILE"
docker exec -e "PGPASSWORD=$PASS" "$CONTAINER" \
  pg_dump -U "$USER" -d "$DB" --no-owner --no-acl \
  | gzip -c >"$OUT_FILE"

ls -lh "$OUT_FILE"
echo "==> backup done"
echo "$OUT_FILE"
