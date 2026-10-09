# Payments service (Phase 1)

Go microservice for **payment ledger** and **summary** (booking / membership / coaching / tournament).

Prefix: `/v1/payments`  
Auth: identity JWT and/or `X-Service-Key` (`PAYMENTS_SERVICE_KEY`). Local may omit key.

## Local

```bash
docker compose up -d --build payments
./scripts/migrate.sh   # applies 012_payments.sql (also EnsureSchema on boot)
```

- Health: http://localhost:8080/v1/payments/health  
- Docs: http://localhost:8080/docs?spec=payments  

## Flow

1. `POST .../payments` — record ledger entry (usually pending)  
2. `GET .../ledger` — filter by `category`, `status`, `from`, `to`  
3. `GET .../summary` — totals + pending counts  
4. `POST .../payments/{id}/mark-paid` — collect cash/UPI and close  

Payment **modes** (full/advance/spot) live in **administration**, not here.

## Env

| Variable | Purpose |
|----------|---------|
| `PAYMENTS_SERVICE_KEY` | Optional local; recommended in production |
| `JWT_*` | Same as other facility services |

Phase 2: Razorpay orders/webhooks, auto-ingest from booking/members/students.
