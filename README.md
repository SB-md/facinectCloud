# Facinect Microservices (local)

Local Docker Compose: **Traefik** + **Go identity** + **Next.js login** + PostgreSQL `fac_identity` + Redis.

PHP identity kept as reference only: `services/identity-php/`.

## Requirements

- Docker + Docker Compose v2
- OpenSSL (for JWT key generation)

## Quick start

```bash
cd ~/StudioProjects/facinect-microservices
cp .env.example .env
chmod +x scripts/*.sh
./scripts/generate-jwt-keys.sh
./scripts/check-secrets.sh
docker compose up -d --build
./scripts/migrate.sh
```

Optional local DB UI (Adminer, **dev profile only**):

```bash
docker compose --profile dev up -d adminer
```

- **Home:** http://localhost:8080/
- **Login UI:** http://localhost:8080/login
- **API gateway:** http://localhost:8080
- **Traefik dashboard:** http://localhost:8088
- **Adminer (dev only):** http://localhost:8089
- Postgres: localhost:5433
- Redis: localhost:6380

## Local ops scripts

| Script | Purpose |
|--------|---------|
| `./scripts/migrate.sh` | Apply `sql/migrations/*.sql` |
| `./scripts/seed.sh` | Optional seed (`sql/seeds/`) |
| `./scripts/check-secrets.sh` | Ensure `.env` / keys not leaked |
| `./scripts/backup.sh` | `pg_dump` → `backups/*.sql.gz` |
| `./scripts/restore.sh backups/<file>.sql.gz` | Restore dump (destructive) |
| `./scripts/deploy-vps.sh` | VPS deploy with `docker-compose.prod.yml` |

## Hostinger VPS (beside RoutForge)

RoutForge already uses **host `:8080`** and **`:5432`**. Facinect prod compose uses:

| Service | Host binding |
|---------|----------------|
| Gateway | **`:8081`** → Traefik |
| Traefik dashboard | `127.0.0.1:8088` only |
| Postgres / Redis | **no public ports** (internal network) |
| Adminer | not included |

### Deploy on VPS

Public URL: **https://cloud.facinect.com/** (Traefik upstream on host `:8081`, image `traefik:v3.7.13`).

`.env.prod` is gitignored — `git pull` does **not** overwrite it. Copy from `.env.prod.example` **once** only; never re-copy after updates.

```bash
# 1) SSH
ssh bharathi@169.58.139.45

# 2) Upload / clone project (example path)
cd ~
# scp -r from laptop, or git clone
cd facinect-microservices

# 3) Env + keys (first time only for cp)
cp .env.prod.example .env.prod
nano .env.prod   # strong DB + admin passwords, Google OAuth, Meta
chmod +x scripts/*.sh
./scripts/generate-jwt-keys.sh

# 4) Firewall: allow 8081 (and keep 22)
# sudo ufw allow 8081/tcp

# 5) Start (does not touch RoutForge)
./scripts/deploy-vps.sh up

# 6) Verify
curl -s http://127.0.0.1:8081/v1/auth/health
# Browser: https://cloud.facinect.com/login
```

Google Console redirect URI:
`https://cloud.facinect.com/v1/auth/google/callback`

Put nginx/Caddy on `:443` → `127.0.0.1:8081` for TLS. Keep RoutForge on `:8080` unless you unify under one reverse proxy. See `docs/VPS_DEPLOY.md`.

## Identity API (Go, via gateway)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/auth/health` | Health (`runtime: go`) |
| GET | `/v1/auth/openapi.json` | OpenAPI 3 spec |
| GET | `/v1/notifications/health` | Notifications health |
| POST | `/v1/notifications/send` | Send WhatsApp/push (dry-run local) |
| GET | `/v1/booking/health` | Booking health |
| GET | `/v1/booking/facilities/{id}/slots` | View slots |
| POST | `/v1/booking/facilities/{id}/bookings` | Create booking |
| GET | `/v1/students/health` | Students health |
| GET | `/v1/students/facilities/{id}/students` | List enrollments |
| POST | `/v1/students/facilities/{id}/attendance` | Mark attendance |
| GET | `/v1/members/health` | Members health |
| GET | `/v1/members/facilities/{id}/members` | List memberships |
| POST | `/v1/members/facilities/{id}/members` | Register member |
| GET | `/v1/tournaments/health` | Tournaments health |
| GET | `/v1/tournaments/facilities/{id}/tournaments` | List tournaments |
| POST | `/v1/tournaments/facilities/{id}/tournaments` | Create tournament |
| GET | `/v1/ai/health` | Shared AI health |
| POST | `/v1/ai/enquiry/analyze` | Analyze enquiry transcript |
| GET | `/v1/enquiry/health` | Enquiry health |
| GET | `/v1/enquiry/facilities/{id}/enquiries` | List enquiries |
| GET | `/v1/offers/health` | Offers health |
| GET | `/v1/offers/facilities/{id}/offers` | List promotions / discounts |
| POST | `/v1/offers/facilities/{id}/offers` | Create offer |
| GET | `/v1/administration/health` | Administration health |
| GET | `/v1/administration/facilities/{id}/profile` | Facility profile |
| PUT | `/v1/administration/facilities/{id}/profile` | Update facility profile |
| GET | `/v1/add-facility/health` | Add-facility health |
| POST | `/v1/add-facility/requests` | Submit facility request |
| POST | `/v1/add-facility/requests/{id}/approve` | Approve + provision facility |
| GET | `/v1/payments/health` | Payments health |
| GET | `/v1/payments/facilities/{id}/ledger` | Payment ledger |
| POST | `/v1/payments/facilities/{id}/payments` | Record ledger entry |
| GET | `/docs` | Swagger UI (Identity / Notifications / Booking) |
| GET | `/api/health` | Web health |
| POST | `/v1/auth/check-email` | Email exists? |
| POST | `/v1/auth/login` | Email + password → JWT + facilities session |
| GET | `/v1/auth/google/start` | Google OAuth PKCE start |
| GET | `/v1/auth/google/callback` | Google callback → `/login?handoff=` |
| POST | `/v1/auth/google/handoff` | Handoff → JWT |
| POST | `/v1/auth/otp/request` | Request OTP (local `dev_otp`) |
| POST | `/v1/auth/otp/verify` | Verify OTP → JWT |
| POST | `/v1/auth/token/refresh` | Rotate refresh token |
| POST | `/v1/auth/logout` | Revoke refresh |
| POST | `/v1/auth/password` | Set/update password (session required) |
| GET | `/v1/auth/me` | Bearer access token → user + facilities |
| GET | `/v1/auth/session` | Live membership refresh |
| GET | `/v1/auth/jwks.json` | Public JWKS |

### Examples

```bash
curl -s http://localhost:8080/v1/auth/health | jq
curl -s http://localhost:8080/api/health | jq

curl -s -X POST http://localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@facinect.local","password":"ChangeMe_Admin_123!"}' | jq
```

## Stack

| Piece | Tech |
|-------|------|
| UI | Next.js 14 (`apps/web`) |
| Identity | Go 1.22 (`services/identity`) |
| Gateway | Traefik v3 |
| DB | PostgreSQL 16 (`fac_identity`) |

## Security defaults

- Passwords: **bcrypt**
- Access JWT: **RS256**, short TTL (default 15m)
- Refresh tokens: opaque, hashed, rotated
- Google: OAuth 2.0 + **PKCE**
- JWT private key mounted read-only; not in git
- Adminer only with `--profile dev` (not default)

## Layout

```
facinect-microservices/
  apps/web/
  services/identity/
  gateway/
  sql/migrations/     # schema versions
  sql/seeds/          # optional data
  scripts/            # migrate, backup, secrets
  backups/            # local dumps (gitignored)
  keys/
```
