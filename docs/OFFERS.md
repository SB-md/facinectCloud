# Offers service (Phase 1)

Go microservice for **promotions** and **discount coupons**: list, create, activate/deactivate, validate code.

Prefix: `/v1/offers`  
Auth: identity JWT (`Authorization: Bearer`) and/or `X-Service-Key` (`OFFERS_SERVICE_KEY`). Local may omit key.

## Local

```bash
docker compose up -d --build offers
./scripts/migrate.sh   # applies 009_offers.sql (also EnsureSchema on boot)
```

- Health: http://localhost:8080/v1/offers/health  
- Docs: http://localhost:8080/docs?spec=offers  

## Flow

1. `POST .../offers` — publish promotion or discount (code, title, flat/percent value, validity)  
2. `GET .../offers` — list (`kind`, `status` optional)  
3. `POST .../offers/{id}/status` — active / inactive  
4. `GET .../offers/validate?code=` — booking checkout helper  
5. `GET .../offers/summary` — active non-expired count (dashboard)

## Env

| Variable | Purpose |
|----------|---------|
| `OFFERS_SERVICE_KEY` | Optional local; recommended in production |
| `JWT_PUBLIC_KEY_PATH` | RS256 public key (same as identity) |
| `JWT_ISSUER` / `JWT_AUDIENCE` | Must match identity tokens |

Phase 2 (later): redemptions ledger, app-only / lock modes, booking checkout wire-up, AI copy via `/v1/ai`.

No MySQL stored procedures — plain SQL.
