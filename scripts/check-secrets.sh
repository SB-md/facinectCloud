#!/usr/bin/env bash
# Fail if secrets look committed or .env is missing required keys.
# Usage: ./scripts/check-secrets.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
ok=1

echo "==> secrets hygiene check"

if [[ ! -f .env ]]; then
  echo "FAIL: .env missing — cp .env.example .env"
  ok=0
else
  echo "ok   .env present"
fi

if [[ -d .git ]]; then
  if git check-ignore -q .env 2>/dev/null || grep -qx '.env' .gitignore 2>/dev/null; then
    echo "ok   .env is gitignored"
  else
    echo "FAIL: .env must be listed in .gitignore"
    ok=0
  fi

  if git ls-files --error-unmatch .env >/dev/null 2>&1; then
    echo "FAIL: .env is tracked by git — remove it from the index"
    ok=0
  else
    echo "ok   .env not tracked"
  fi

  for pem in keys/jwt_private.pem; do
    [[ -e "$pem" ]] || continue
    if git check-ignore -q "$pem" 2>/dev/null; then
      echo "ok   $pem gitignored"
    else
      echo "WARN: $pem should be gitignored"
    fi
    if git ls-files --error-unmatch "$pem" >/dev/null 2>&1; then
      echo "FAIL: $pem is tracked by git"
      ok=0
    fi
  done
else
  if grep -qx '.env' .gitignore 2>/dev/null; then
    echo "ok   .env listed in .gitignore (no git repo yet)"
  else
    echo "FAIL: .env must be listed in .gitignore"
    ok=0
  fi
  if grep -q 'keys/\*\.pem' .gitignore 2>/dev/null; then
    echo "ok   keys/*.pem listed in .gitignore"
  fi
fi

# .env.example must not contain a filled Google client secret
if grep -E '^GOOGLE_CLIENT_SECRET=.+' .env.example | grep -vq 'GOOGLE_CLIENT_SECRET=$'; then
  if grep -E '^GOOGLE_CLIENT_SECRET=GOCSPX-' .env.example >/dev/null 2>&1; then
    echo "FAIL: .env.example contains a real-looking GOOGLE_CLIENT_SECRET"
    ok=0
  fi
fi
echo "ok   .env.example has empty Google secret placeholder"

if [[ -f .env ]]; then
  for key in POSTGRES_IDENTITY_PASSWORD BOOTSTRAP_ADMIN_PASSWORD; do
    val="$(grep -E "^${key}=" .env | tail -n1 | cut -d= -f2- || true)"
    if [[ -z "$val" ]]; then
      echo "FAIL: .env missing $key"
      ok=0
    else
      echo "ok   $key set"
    fi
  done
fi

if [[ "$ok" -ne 1 ]]; then
  echo "==> FAILED"
  exit 1
fi
echo "==> PASSED"
