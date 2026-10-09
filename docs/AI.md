# Shared AI service

Go microservice for reusable AI skills across Facinect.

Prefix: `/v1/ai`  
Auth: `X-Service-Key` (`AI_SERVICE_KEY`). Local may omit key.

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/v1/ai/health` | Health + whether Gemini key is set |
| POST | `/v1/ai/enquiry/analyze` | Transcript → structured `ai_data` |
| POST | `/v1/ai/reply/suggest` | Draft reply from `ai_data` |

Without `GEMINI_API_KEY`, analyze runs a **stub** heuristic (local-friendly).

## Env

| Variable | Purpose |
|----------|---------|
| `GEMINI_API_KEY` | Google Generative Language API key |
| `GEMINI_MODEL` | Default `gemini-2.0-flash` |
| `AI_SERVICE_KEY` | Service-to-service auth |

Add more routes later (`/v1/ai/offers/copy`, etc.) without new containers.
