# Payment gateway service (Phase 1)

Integration abstraction for **third-party payment providers** (Razorpay, stub; Cashfree later).

Prefix: `/v1/gateway`  
Compose service: **`paygateway`** (distinct from Traefik edge `gateway`).

Auth: JWT and/or `X-Service-Key` (`PAYGATEWAY_SERVICE_KEY`). Webhooks use provider signatures.

## Local

```bash
docker compose up -d --build paygateway
./scripts/migrate.sh   # 013_paygateway.sql
```

- Health: http://localhost:8080/v1/gateway/health  
- Docs: http://localhost:8080/docs?spec=gateway  

## Contract

1. `GET /providers` — configured adapters  
2. `POST /orders` — create order (`provider` optional → default)  
3. `POST /verify` — signature verify  
4. `POST /webhooks/{provider}` — normalize events  

Callers (saas-billing, booking) must **not** import Razorpay/Cashfree SDKs — only this API.

## Env

| Variable | Purpose |
|----------|---------|
| `PAYGATEWAY_SERVICE_KEY` | Service-to-service auth |
| `GATEWAY_DEFAULT_PROVIDER` | `razorpay` (falls back to `stub` if keys missing) |
| `RAZORPAY_KEY_ID` / `RAZORPAY_KEY_SECRET` | Razorpay Orders API |
| `RAZORPAY_WEBHOOK_SECRET` | Optional webhook HMAC |

Empty Razorpay keys → stub used for local dry-run.

## Adding Cashfree (Phase 2)

Implement `Provider` in `internal/gateway/providers/cashfree`, register in `main.go`, add `webhooks/cashfree`. No change to `/orders` contract.

Facinect PHP scripts are **not** modified by this service.
