#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
KEYS="$ROOT/keys"
mkdir -p "$KEYS"

if [[ -f "$KEYS/jwt_private.pem" && -f "$KEYS/jwt_public.pem" ]]; then
  echo "JWT keys already exist in $KEYS"
  exit 0
fi

openssl genrsa -out "$KEYS/jwt_private.pem" 2048
openssl rsa -in "$KEYS/jwt_private.pem" -pubout -out "$KEYS/jwt_public.pem"
# Local Docker identity runs as non-owner; keep readable inside the container mount.
chmod 644 "$KEYS/jwt_private.pem" "$KEYS/jwt_public.pem"
echo "Wrote RS256 key pair to $KEYS"
