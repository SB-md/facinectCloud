# Students service (Phase 1)

Go microservice for **list students**, **enroll**, and **attendance by date**.

Prefix: `/v1/students`  
Auth: identity JWT (`Authorization: Bearer`) and/or `X-Service-Key` (`STUDENTS_SERVICE_KEY`). Local may omit key.

## Local

```bash
docker compose up -d --build students
./scripts/migrate.sh   # applies 005_students.sql (also EnsureSchema on boot)
```

- Health: http://localhost:8080/v1/students/health  
- Docs: http://localhost:8080/docs?spec=students  

## Flow

1. `POST .../students` — register + enroll at a facility  
2. `GET .../students` — list enrollments (`sport_id`, `status` optional)  
3. `GET .../attendance?date=` — active enrollments + mark for that day  
4. `POST .../attendance` — upsert present / absent / leave  
5. `GET .../students/summary` — active count (dashboard)

## Env

| Variable | Purpose |
|----------|---------|
| `STUDENTS_SERVICE_KEY` | Optional local; recommended in production |
| `JWT_PUBLIC_KEY_PATH` | RS256 public key (same as identity) |
| `JWT_ISSUER` / `JWT_AUDIENCE` | Must match identity tokens |

Phase 2 (later): billing, plans catalogue, bulk WhatsApp notify.

No MySQL stored procedures — plain SQL + transactions.
