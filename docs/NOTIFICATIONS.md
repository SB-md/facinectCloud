# Notifications service (N1)

Go microservice for **WhatsApp** + **push** used by bookings, students, payments, etc.

## Local

```bash
cd ~/StudioProjects/facinect-microservices
docker compose up -d --build notifications
./scripts/migrate.sh   # applies 002_notifications.sql (also auto-created on boot)
```

- Health: http://localhost:8080/v1/notifications/health  
- OpenAPI: http://localhost:8080/v1/notifications/openapi.json  

Without Meta/FCM credentials, sends are **`dry_run`** (job logged, no external call).

### Send example

```bash
curl -s -X POST http://localhost:8080/v1/notifications/send \
  -H 'Content-Type: application/json' \
  -d '{
    "channel": "whatsapp",
    "facility_id": 20,
    "template": "hello_world",
    "to": { "whatsapp": "919876543210" },
    "data": { "name": "Ravi" }
  }' | jq
```

With service key (when set):

```bash
curl -s -X POST http://localhost:8080/v1/notifications/send \
  -H 'Content-Type: application/json' \
  -H "X-Service-Key: $NOTIFICATIONS_SERVICE_KEY" \
  -d '{"channel":"push","to":{"device_tokens":["test-token"]},"data":{"title":"Hi","body":"Test"}}'
```

## Env

| Variable | Purpose |
|----------|---------|
| `NOTIFICATIONS_SERVICE_KEY` | **Required in production** (`X-Service-Key`). Unverified Bearer is not accepted. Local may omit for convenience. |
| `NOTIFICATIONS_DRY_RUN` | Force stub sends |
| `META_WHATSAPP_TOKEN` | Meta Cloud API token |
| `META_PHONE_NUMBER_ID` | WhatsApp phone number id |
| `FCM_SERVER_KEY` | Legacy FCM server key |

## N2 — Facility WhatsApp (PHP hybrid)

Same as `FacilityWhatsApp` + `metaPhoneNumberId`:

| Source | Field |
|--------|--------|
| DB `facility_whatsapp_config` | `meta_phone_number_id`, optional `whatsapp_api_token`, `whatsapp_enabled` |
| ENV fallback | `META_WHATSAPP_TOKEN`, `META_PHONE_NUMBER_ID` |

```bash
# Configure facility 20 (own Meta account)
curl -s -X PUT http://localhost:8080/v1/notifications/facilities/20/whatsapp \
  -H 'Content-Type: application/json' \
  -d '{
    "meta_phone_number_id": "YOUR_PHONE_NUMBER_ID",
    "whatsapp_api_token": "YOUR_FACILITY_TOKEN",
    "whatsapp_enabled": true
  }' | jq

# Inspect (token not returned; has_token_override + token_from)
curl -s http://localhost:8080/v1/notifications/facilities/20/whatsapp | jq

# Send uses facility creds
curl -s -X POST http://localhost:8080/v1/notifications/send \
  -H 'Content-Type: application/json' \
  -d '{"channel":"whatsapp","facility_id":20,"template":"hello_world","to":{"whatsapp":"919876543210"}}' | jq

# Webhook-style lookup
curl -s 'http://localhost:8080/v1/notifications/whatsapp/lookup?phone_number_id=YOUR_PHONE_NUMBER_ID' | jq
```

Disabled facility → send fails with `whatsapp_disabled_for_facility`.

## Next (N2+)

- Redis worker queue + retries  
- JWT verification via identity JWKS  
- Encrypt facility tokens at rest  
- Wire bookings/students callers  
