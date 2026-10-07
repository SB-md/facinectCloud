# Booking service

One Go microservice for **view bookings**, **create/cancel booking**, and **slot block/unblock**.

Prefix: `/v1/booking`  
Auth: identity JWT (`Authorization: Bearer`) and/or `X-Service-Key` (`BOOKING_SERVICE_KEY`). Local may omit key.

## Local

```bash
docker compose up -d --build booking
./scripts/migrate.sh   # applies 004_booking.sql (also EnsureSchema on boot)
```

- Health: http://localhost:8080/v1/booking/health  
- Docs: http://localhost:8080/docs?spec=booking  

## Flow

1. `POST .../courts` — create court(s) for a facility  
2. `POST .../slots/generate` — fill a day with available slots  
3. `GET .../slots?date=` — view availability  
4. `POST .../bookings` — book (transaction: slot available → booked)  
5. `POST .../slots/{id}/block` — staff block without booking  
6. `POST .../bookings/{id}/cancel` — cancel + free slot  

## Env

| Variable | Purpose |
|----------|---------|
| `BOOKING_SERVICE_KEY` | Optional local; recommended in production |
| `JWT_PUBLIC_KEY_PATH` | RS256 public key (same as identity) |
| `JWT_ISSUER` / `JWT_AUDIENCE` | Must match identity tokens |
| `BOOKING_OPEN_HOUR` | Default 6 |
| `BOOKING_CLOSE_HOUR` | Default 22 |
| `BOOKING_SLOT_MINUTES` | Default 60 |

No MySQL stored procedures — plain SQL + transactions (same pattern as identity/notifications).
