# Hostinger VPS deploy (Facinect beside RoutForge)

## Why separate ports?

On this VPS, RoutForge already binds:

- `:8080` — routeforge-gateway
- `:5432` — routeforge-postgres (public)

Facinect **prod** stack therefore uses:

- Gateway **`:8081`**
- Postgres / Redis **internal only** (no host ports)
- Traefik dashboard only on `127.0.0.1:8088`

## One-time upload from laptop

```bash
# from laptop (facinect-microservices root)
rsync -avz --exclude node_modules --exclude .next --exclude backups \
  --exclude '.env' --exclude 'keys/*.pem' \
  ./ bharathi@169.58.139.45:~/facinect-microservices/
```

Or `git clone` if the repo is on GitHub/GitLab.

## On VPS

```bash
ssh bharathi@169.58.139.45
cd ~/facinect-microservices

cp .env.prod.example .env.prod
nano .env.prod
# set strong POSTGRES_IDENTITY_PASSWORD + BOOTSTRAP_ADMIN_PASSWORD
# set GOOGLE_* and redirect:
#   http://169.58.139.45:8081/v1/auth/google/callback

chmod +x scripts/*.sh
./scripts/generate-jwt-keys.sh

# open firewall for Facinect gateway
sudo ufw allow 8081/tcp || true

./scripts/deploy-vps.sh up
./scripts/deploy-vps.sh status
```

## Verify (do not stop RoutForge)

```bash
sudo docker ps --format 'table {{.Names}}\t{{.Ports}}' | head -30
curl -s http://127.0.0.1:8081/v1/auth/health
curl -s http://127.0.0.1:8081/api/health
```

Browser: http://169.58.139.45:8081/login

## Useful commands

```bash
./scripts/deploy-vps.sh logs
./scripts/deploy-vps.sh down    # stops Facinect only
ENV_FILE=.env.prod ./scripts/backup.sh
```

## Later: domain + HTTPS

Point DNS A record to `169.58.139.45`, then put nginx/Caddy on `:443` proxying to `127.0.0.1:8081`, and update:

- `JWT_ISSUER`
- `GOOGLE_OAUTH_REDIRECT_URI`
- Google Console redirect URI
