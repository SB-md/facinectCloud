# StrollX ↔ Dashboard (gateway)

Admin dashboard in StrollX uses **Facinect gateway** summaries — not PHP `api/v2/dashboard.php`.

## App

`lib/services/dashboard_service.dart` → `ApiConfig.gatewayHost` + Bearer JWT.

## Calls (parallel)

| UI field | Method | Path |
|----------|--------|------|
| Members active/total | GET | `/v1/members/facilities/{facilityId}/members/summary` |
| Students active/total | GET | `/v1/students/facilities/{facilityId}/students/summary` |
| Bookings month + weekly bars | GET | `/v1/booking/facilities/{facilityId}/bookings?year=&month=` |

Booking weekly / % are **aggregated in the app** from the bookings list (no booking summary microservice yet).

## Not migrated

| Feature | Status |
|---------|--------|
| Member-request notifications | Stub empty |
| Approve/reject request | Stub `false` |

## Gateway URL

Same as Identity: `FACINECT_GATEWAY` (see [STROLLX_IDENTITY.md](./STROLLX_IDENTITY.md)).
