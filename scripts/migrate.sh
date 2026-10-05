#!/usr/bin/env bash
# Apply pending SQL files from sql/migrations/ to local Postgres.
# Usage: ./scripts/migrate.sh
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
MIGRATIONS_DIR="$ROOT/sql/migrations"

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "error: container $CONTAINER is not running (docker compose up -d postgres)" >&2
  exit 1
fi

psql_q() {
  docker exec -i -e "PGPASSWORD=$PASS" "$CONTAINER" \
    psql -v ON_ERROR_STOP=1 -U "$USER" -d "$DB" "$@"
}

echo "==> ensuring schema_migrations"
psql_q <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  version     TEXT PRIMARY KEY,
  applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL

BASELINE="001_fac_identity"
if ! psql_q -tAc "SELECT 1 FROM schema_migrations WHERE version = '$BASELINE'" | grep -q 1; then
  if psql_q -tAc "SELECT to_regclass('public.users')" | grep -q users; then
    echo "==> baseline already present — recording $BASELINE"
    psql_q -c "INSERT INTO schema_migrations (version) VALUES ('$BASELINE') ON CONFLICT DO NOTHING;"
  fi
fi

shopt -s nullglob
files=("$MIGRATIONS_DIR"/*.sql)
if [[ ${#files[@]} -eq 0 ]]; then
  echo "no migration files in $MIGRATIONS_DIR"
  exit 0
fi

applied=0
for file in "${files[@]}"; do
  base="$(basename "$file" .sql)"
  if psql_q -tAc "SELECT 1 FROM schema_migrations WHERE version = '$base'" | grep -q 1; then
    echo "skip  $base"
    continue
  fi
  echo "apply $base"
  psql_q <"$file"
  psql_q -c "INSERT INTO schema_migrations (version) VALUES ('$base');"
  applied=$((applied + 1))
done

echo "==> done ($applied new migration(s))"
psql_q -c "SELECT version, applied_at FROM schema_migrations ORDER BY version;"
