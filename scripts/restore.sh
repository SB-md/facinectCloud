#!/usr/bin/env bash
# Restore a gzipped pg_dump into local fac_identity (DESTRUCTIVE for that DB).
# Usage: ./scripts/restore.sh backups/fac_identity_YYYYMMDDThhmmssZ.sql.gz
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

DUMP="${1:-}"
if [[ -z "$DUMP" || ! -f "$DUMP" ]]; then
  echo "usage: $0 backups/<file>.sql.gz" >&2
  exit 1
fi

CONTAINER="${POSTGRES_CONTAINER:-facinect-postgres}"
DB="$(env_get POSTGRES_IDENTITY_DB fac_identity)"
USER="$(env_get POSTGRES_IDENTITY_USER identity)"
PASS="$(env_get POSTGRES_IDENTITY_PASSWORD identity_local_change_me)"

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "error: container $CONTAINER is not running" >&2
  exit 1
fi

echo "==> restoring $DUMP into $DB (drops public schema objects)"
docker exec -i -e "PGPASSWORD=$PASS" "$CONTAINER" \
  psql -v ON_ERROR_STOP=1 -U "$USER" -d "$DB" <<'SQL'
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
GRANT ALL ON SCHEMA public TO PUBLIC;
SQL

gunzip -c "$DUMP" | docker exec -i -e "PGPASSWORD=$PASS" "$CONTAINER" \
  psql -v ON_ERROR_STOP=1 -U "$USER" -d "$DB"

echo "==> restore done — restart identity to refresh connections:"
echo "    docker compose restart identity"
