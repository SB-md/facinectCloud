# Hostinger VPS deploy (Facinect beside RoutForge)

## Public URL

- **Portal / API:** https://cloud.facinect.com/
- Traefik on host **`:8081`** (HTTP upstream for nginx/Caddy TLS)
- RoutForge stays on **`:8080`**

## Why separate ports?

On this VPS, RoutForge already binds:

- `:8080` — routeforge-gateway
- `:5432` — routeforge-postgres (public)

Facinect **prod** stack therefore uses:

- Gateway **`:8081`**
- Postgres / Redis **internal only** (no host ports)
- Traefik dashboard only on `127.0.0.1:8088`
- Image: `traefik:v3.7.13`

## Credentials vs `git pull`

| File | Tracked? | Pull overwrites? |
|------|----------|------------------|
| `.env.prod` | No (gitignore) | **No** |
| `keys/*.pem` | No | **No** |
| `.env.prod.example` | Yes | Yes (template only) |

**Never** run `cp .env.prod.example .env.prod` after the first setup — that wipes live secrets. After pull, only add any *new* variable names from the example into your existing `.env.prod`.

## One-time upload from laptop

```bash
# from laptop (facinect-microservices root)
rsync -avz --exclude node_modules --exclude .next --exclude backups \
  --exclude '.env' --exclude '.env.prod' --exclude 'keys/*.pem' \
  ./ bharathi@169.58.139.45:~/facinect-microservices/
```

Or `git clone` / `git pull` if the repo is on GitHub.

## On VPS (first time)

```bash
ssh bharathi@169.58.139.45
cd ~/facinect-microservices

cp .env.prod.example .env.prod   # ONCE only
nano .env.prod
# POSTGRES_IDENTITY_PASSWORD, BOOTSTRAP_ADMIN_PASSWORD,
# GOOGLE_*, NOTIFICATIONS_*, META_*
# JWT_ISSUER / GOOGLE_OAUTH_REDIRECT_URI already point at cloud.facinect.com

chmod +x scripts/*.sh
./scripts/generate-jwt-keys.sh

sudo ufw allow 8081/tcp || true

./scripts/deploy-vps.sh up
./scripts/deploy-vps.sh status
```

## Updates (code only)

```bash
cd ~/facinect-microservices
git pull origin main          # does not touch .env.prod
# if .env.prod.example gained new keys, merge them into .env.prod manually
./scripts/deploy-vps.sh up
```

## Domain + HTTPS

1. DNS A record: `cloud.facinect.com` → `169.58.139.45`
2. nginx/Caddy on `:443` → `http://127.0.0.1:8081` (Let’s Encrypt)
3. Google Console redirect URI:
   `https://cloud.facinect.com/v1/auth/google/callback`
4. `.env.prod` already uses:
   - `JWT_ISSUER=https://cloud.facinect.com`
   - `GOOGLE_OAUTH_REDIRECT_URI=https://cloud.facinect.com/v1/auth/google/callback`
   - `CORS_ORIGINS=https://cloud.facinect.com`

## Verify (do not stop RoutForge)

```bash
sudo docker ps --format 'table {{.Names}}\t{{.Ports}}' | head -30
curl -s http://127.0.0.1:8081/v1/auth/health
curl -s http://127.0.0.1:8081/v1/notifications/health
curl -sI https://cloud.facinect.com/
```

Browser: https://cloud.facinect.com/login

## Useful commands

```bash
./scripts/deploy-vps.sh logs
./scripts/deploy-vps.sh down    # stops Facinect only
ENV_FILE=.env.prod ./scripts/backup.sh
```
