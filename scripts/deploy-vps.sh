#!/usr/bin/env bash
# Deploy Facinect on VPS beside existing stacks (RoutForge on :8080).
# Usage:
#   ./scripts/deploy-vps.sh          # up -d --build
#   ./scripts/deploy-vps.sh down
#   ./scripts/deploy-vps.sh logs
#   ./scripts/deploy-vps.sh status
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${ENV_FILE:-.env.prod}"
COMPOSE=(docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE")

if [[ ! -f "$ENV_FILE" ]]; then
  echo "error: $ENV_FILE missing — cp .env.prod.example .env.prod && edit secrets (once only)" >&2
  exit 1
fi
# Never auto-overwrite $ENV_FILE from .env.prod.example — git pull must not wipe secrets.

if [[ ! -f keys/jwt_private.pem || ! -f keys/jwt_public.pem ]]; then
  echo "==> generating JWT keys"
  chmod +x scripts/generate-jwt-keys.sh
  ./scripts/generate-jwt-keys.sh
fi

cmd="${1:-up}"

case "$cmd" in
  up)
    echo "==> building & starting Facinect (gateway :8081)"
    "${COMPOSE[@]}" up -d --build
    echo "==> waiting for identity healthy..."
    for i in $(seq 1 30); do
      if "${COMPOSE[@]}" ps identity 2>/dev/null | grep -qi healthy; then
        break
      fi
      sleep 2
    done
    echo "==> migrate"
    ENV_FILE="$ENV_FILE" POSTGRES_CONTAINER=facinect-postgres ./scripts/migrate.sh || true
    echo "==> status"
    "${COMPOSE[@]}" ps
    echo
    echo "Open: http://$(hostname -I 2>/dev/null | awk '{print $1}'):8081/"
    echo "  or: http://169.58.139.45:8081/login"
    echo "Health: curl -s http://127.0.0.1:8081/v1/auth/health"
    ;;
  down)
    "${COMPOSE[@]}" down
    ;;
  logs)
    "${COMPOSE[@]}" logs -f --tail=100
    ;;
  status)
    "${COMPOSE[@]}" ps
    curl -sS -o /dev/null -w "gateway_health:%{http_code}\n" http://127.0.0.1:8081/v1/auth/health || true
    curl -sS -o /dev/null -w "web_health:%{http_code}\n" http://127.0.0.1:8081/api/health || true
    ;;
  *)
    echo "usage: $0 [up|down|logs|status]" >&2
    exit 1
    ;;
esac
