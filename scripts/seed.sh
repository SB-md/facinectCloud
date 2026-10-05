#!/usr/bin/env bash
# Apply optional seed SQL (not tracked as migrations).
# Usage: ./scripts/seed.sh [seed_name]
# Default: facilities_20_28
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

SEED_NAME="${1:-facilities_20_28}"
SEED_FILE="$ROOT/sql/seeds/${SEED_NAME}.sql"
CONTAINER="${POSTGRES_CONTAINER:-facinect-postgres}"
DB="$(env_get POSTGRES_IDENTITY_DB fac_identity)"
USER="$(env_get POSTGRES_IDENTITY_USER identity)"
PASS="$(env_get POSTGRES_IDENTITY_PASSWORD identity_local_change_me)"

if [[ ! -f "$SEED_FILE" ]]; then
  echo "error: seed not found: $SEED_FILE" >&2
  exit 1
fi

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "error: container $CONTAINER is not running" >&2
  exit 1
fi

echo "==> seeding $SEED_NAME"
docker exec -i -e "PGPASSWORD=$PASS" "$CONTAINER" \
  psql -v ON_ERROR_STOP=1 -U "$USER" -d "$DB" <"$SEED_FILE"
echo "==> seed done"
