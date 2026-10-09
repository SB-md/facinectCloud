# Add facility service (Phase 1)

Go microservice for **facility onboarding / additional facility requests**: submit, list mine, approve/reject.

Prefix: `/v1/add-facility`  
Auth: identity JWT and/or `X-Service-Key` (`ADDFACILITY_SERVICE_KEY`). Local may omit key.

## Local

```bash
docker compose up -d --build addfacility
./scripts/migrate.sh   # applies 011_add_facility.sql
```

- Health: http://localhost:8080/v1/add-facility/health  
- Portal: http://localhost:8080/add-facility  
- Docs: http://localhost:8080/docs?spec=add-facility  

## Flow

1. `POST .../requests` — submit (name, location, optional maps/WhatsApp/sports)  
2. `GET .../requests/mine` — requester history  
3. `GET .../requests/pending` — review queue  
4. `POST .../requests/{id}/approve` — creates `facilities` row + admin membership; seeds administration profile/sports/courts when those tables exist  
5. `POST .../requests/{id}/reject`  

## Env

| Variable | Purpose |
|----------|---------|
| `ADDFACILITY_SERVICE_KEY` | Optional local; recommended in production for approve/reject |
| `JWT_*` | Same as other services |

Phase 2: logo upload, platform-only review UI, WhatsApp notify on approve, booking court sync.
