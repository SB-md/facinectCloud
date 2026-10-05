# How to add another microservice

1. **Folder** — `services/<service-name>/` with `Dockerfile` + app code.
2. **Schema** — `sql/0N_fac_<service-name>.sql` (own database; no cross-service JOINs).
3. **Compose** — add service, env, `depends_on`, Traefik labels:
   - `traefik.enable=true`
   - `traefik.http.routers.<name>.rule=PathPrefix(\`/v1/<prefix>\`)`
   - `traefik.http.services.<name>.loadbalancer.server.port=<port>`
4. **Auth** — fetch RS256 public key from `GET /v1/auth/jwks.json` (or mount `keys/jwt_public.pem`).
5. **Clients** — Flutter apps call gateway only (`http://localhost:8080` locally).

## Priority after identity

| Order | Service | Prefix |
|-------|---------|--------|
| 1 | identity | `/v1/auth` (done) |
| 2 | saas-billing | `/v1/saas` |
| 3 | whatsapp | `/v1/wa` |
| 4 | booking | `/v1/booking` |
