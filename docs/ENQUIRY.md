# Enquiry service

Go microservice for the facility enquiry inbox. Uses shared **ai** for analyze/suggest and **notifications** for WhatsApp.

Prefix: `/v1/enquiry`  
Auth: JWT and/or `ENQUIRY_SERVICE_KEY`.

## Local

```bash
docker compose up -d --build ai enquiry
./scripts/migrate.sh   # 008_enquiries.sql
```

- Health: http://localhost:8080/v1/enquiry/health  
- AI health: http://localhost:8080/v1/ai/health  
- Docs: http://localhost:8080/docs?spec=enquiry  

## Flow

1. `POST .../enquiries` with phone + transcript (`run_ai` default true) → calls `ai` → stores `ai_data`  
2. `GET .../enquiries` list / filter  
3. `POST .../enquiries/{id}/analyze` re-run AI  
4. `POST .../enquiries/{id}/suggest-reply` draft message  
5. `POST .../enquiries/{id}/notify` send via notifications  

## Env

| Variable | Purpose |
|----------|---------|
| `ENQUIRY_SERVICE_KEY` | Optional local |
| `AI_BASE_URL` | Default `http://ai:8080` |
| `AI_SERVICE_KEY` | Passed to AI as `X-Service-Key` |
| `NOTIFICATIONS_BASE_URL` | Default `http://notifications:8080` |
| `NOTIFICATIONS_SERVICE_KEY` | For WhatsApp send |
