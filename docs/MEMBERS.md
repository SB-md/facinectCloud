# Members service (Phase 1)

Go microservice for **list memberships**, **register members**, and **status updates**.

Prefix: `/v1/members`  
Auth: identity JWT (`Authorization: Bearer`) and/or `X-Service-Key` (`MEMBERS_SERVICE_KEY`). Local may omit key.

## Local

```bash
docker compose up -d --build members
./scripts/migrate.sh   # applies 006_members.sql (also EnsureSchema on boot)
```

- Health: http://localhost:8080/v1/members/health  
- Docs: http://localhost:8080/docs?spec=members  

## Flow

1. `POST .../members` — create member + active membership  
2. `GET .../members` — list (`sport_id`, `status` optional)  
3. `POST .../memberships/{id}/status` — active / inactive / expired  
4. `GET .../members/summary` — active count (dashboard)

## Env

| Variable | Purpose |
|----------|---------|
| `MEMBERS_SERVICE_KEY` | Optional local; recommended in production |
| `JWT_PUBLIC_KEY_PATH` | RS256 public key (same as identity) |
| `JWT_ISSUER` / `JWT_AUDIENCE` | Must match identity tokens |

Phase 2 (later): membership plans catalogue, billing, member requests / notify.

No MySQL stored procedures — plain SQL + transactions.
