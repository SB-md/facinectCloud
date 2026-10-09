# Administration service (Phase 1)

Go microservice for **facility profile**, **sports/courts catalogue**, **booking payment modes**, **maintenance flags**, and **staff memberships** (via identity tables).

Prefix: `/v1/administration`  
Auth: identity JWT and/or `X-Service-Key` (`ADMINISTRATION_SERVICE_KEY`). Local may omit key.

## Local

```bash
docker compose up -d --build administration
./scripts/migrate.sh   # applies 010_administration.sql (also EnsureSchema on boot)
```

- Health: http://localhost:8080/v1/administration/health  
- Docs: http://localhost:8080/docs?spec=administration  

## Flow

1. `GET/PUT .../profile` — facility display details + hours  
2. `GET/POST .../sports` + status — sport catalogue  
3. `GET/POST .../courts` + status — court catalogue (not booking slots)  
4. `GET/PUT .../payment-settings` — full / advance / spot  
5. `GET/PUT .../services` — staff + customer enabled flags  
6. `GET/POST .../staff` — upsert membership for an **existing** identity user email  
7. `GET .../summary` — dashboard counts  

## Env

| Variable | Purpose |
|----------|---------|
| `ADMINISTRATION_SERVICE_KEY` | Optional local; recommended in production |
| `JWT_*` | Same as other facility services |

Phase 2: sync courts → booking, payment time windows, invite/create users, WhatsApp Meta credentials, additional-facility requests.

Staff passwords stay in **identity** only.
