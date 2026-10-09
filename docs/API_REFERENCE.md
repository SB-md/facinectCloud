# Facinect Cloud — API Reference

Service-wise index of every HTTP API exposed through the Traefik gateway.

| | |
|---|---|
| **Local base URL** | `http://localhost:8080` |
| **Production** | `https://cloud.facinect.com` |
| **Interactive docs** | [`/docs`](http://localhost:8080/docs) (Swagger UI) |
| **Auth** | `Authorization: Bearer <JWT>` from identity; most services also accept `X-Service-Key` |

Routes marked **Public** do not require a JWT or service key.

---

## Quick links

| # | Service | Prefix | Guide | OpenAPI | Swagger UI |
|---|---------|--------|-------|---------|------------|
| 1 | [Identity](#1-identity) | `/v1/auth` | [README § Identity API](../README.md#identity-api-go-via-gateway) | [`/v1/auth/openapi.json`](http://localhost:8080/v1/auth/openapi.json) | [`/docs?spec=identity`](http://localhost:8080/docs?spec=identity) |
| 2 | [Notifications](#2-notifications) | `/v1/notifications` | [NOTIFICATIONS.md](./NOTIFICATIONS.md) | [`/v1/notifications/openapi.json`](http://localhost:8080/v1/notifications/openapi.json) | [`/docs?spec=notifications`](http://localhost:8080/docs?spec=notifications) |
| 3 | [Booking](#3-booking) | `/v1/booking` | [BOOKING.md](./BOOKING.md) | [`/v1/booking/openapi.json`](http://localhost:8080/v1/booking/openapi.json) | [`/docs?spec=booking`](http://localhost:8080/docs?spec=booking) |
| 4 | [Students](#4-students) | `/v1/students` | [STUDENTS.md](./STUDENTS.md) | [`/v1/students/openapi.json`](http://localhost:8080/v1/students/openapi.json) | [`/docs?spec=students`](http://localhost:8080/docs?spec=students) |
| 5 | [Members](#5-members) | `/v1/members` | [MEMBERS.md](./MEMBERS.md) | [`/v1/members/openapi.json`](http://localhost:8080/v1/members/openapi.json) | [`/docs?spec=members`](http://localhost:8080/docs?spec=members) |
| 6 | [Tournaments](#6-tournaments) | `/v1/tournaments` | [TOURNAMENTS.md](./TOURNAMENTS.md) | [`/v1/tournaments/openapi.json`](http://localhost:8080/v1/tournaments/openapi.json) | [`/docs?spec=tournaments`](http://localhost:8080/docs?spec=tournaments) |
| 7 | [AI](#7-ai-shared) | `/v1/ai` | [AI.md](./AI.md) | [`/v1/ai/openapi.json`](http://localhost:8080/v1/ai/openapi.json) | [`/docs?spec=ai`](http://localhost:8080/docs?spec=ai) |
| 8 | [Enquiry](#8-enquiry) | `/v1/enquiry` | [ENQUIRY.md](./ENQUIRY.md) | [`/v1/enquiry/openapi.json`](http://localhost:8080/v1/enquiry/openapi.json) | [`/docs?spec=enquiry`](http://localhost:8080/docs?spec=enquiry) |
| 9 | [Offers](#9-offers) | `/v1/offers` | [OFFERS.md](./OFFERS.md) | [`/v1/offers/openapi.json`](http://localhost:8080/v1/offers/openapi.json) | [`/docs?spec=offers`](http://localhost:8080/docs?spec=offers) |
| 10 | [Administration](#10-administration) | `/v1/administration` | [ADMINISTRATION.md](./ADMINISTRATION.md) | [`/v1/administration/openapi.json`](http://localhost:8080/v1/administration/openapi.json) | [`/docs?spec=administration`](http://localhost:8080/docs?spec=administration) |
| 11 | [Add facility](#11-add-facility) | `/v1/add-facility` | [ADD_FACILITY.md](./ADD_FACILITY.md) | [`/v1/add-facility/openapi.json`](http://localhost:8080/v1/add-facility/openapi.json) | [`/docs?spec=add-facility`](http://localhost:8080/docs?spec=add-facility) |
| 12 | [Payments (ledger)](#12-payments-ledger) | `/v1/payments` | [PAYMENTS.md](./PAYMENTS.md) | [`/v1/payments/openapi.json`](http://localhost:8080/v1/payments/openapi.json) | [`/docs?spec=payments`](http://localhost:8080/docs?spec=payments) |
| 13 | [Pay gateway](#13-pay-gateway) | `/v1/gateway` | [GATEWAY.md](./GATEWAY.md) | [`/v1/gateway/openapi.json`](http://localhost:8080/v1/gateway/openapi.json) | [`/docs?spec=gateway`](http://localhost:8080/docs?spec=gateway) |
| 14 | [Web portal](#14-web-portal) | `/` | — | — | [`/docs`](http://localhost:8080/docs) |

**Planned (not implemented):** SaaS billing — prefix `/v1/saas`.

**Related:** [ADD_SERVICE.md](./ADD_SERVICE.md) (how to add a new service), [VPS_DEPLOY.md](./VPS_DEPLOY.md) (production deploy).

---

## 1. Identity

**Compose service:** `identity`  
**Prefix:** `/v1/auth`  
**Guide:** [README § Identity API](../README.md#identity-api-go-via-gateway)  
**OpenAPI:** [`GET /v1/auth/openapi.json`](http://localhost:8080/v1/auth/openapi.json)  
**Swagger:** [`/docs?spec=identity`](http://localhost:8080/docs?spec=identity)

Authentication, sessions, Google OAuth, OTP, JWT issuance.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/auth/health` | Public | Health check |
| GET | `/v1/auth/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/auth/jwks.json` | Public | JSON Web Key Set (RS256) |
| POST | `/v1/auth/check-email` | Public | Check whether email is registered |
| POST | `/v1/auth/login` | Public | Email + password → JWT + session |
| POST | `/v1/auth/google` | Public | Native Google Sign-In `id_token` → JWT (StrollX) |
| GET | `/v1/auth/google/start` | Public | Start Google OAuth (PKCE) |
| GET | `/v1/auth/google/callback` | Public | Google OAuth callback |
| POST | `/v1/auth/google/handoff` | Public | Exchange handoff code → JWT |
| POST | `/v1/auth/otp/request` | Public | Request OTP |
| POST | `/v1/auth/otp/verify` | Public | Verify OTP → JWT |
| POST | `/v1/auth/token/refresh` | Public | Rotate refresh token |
| POST | `/v1/auth/logout` | Auth | Revoke refresh token |
| POST | `/v1/auth/password` | Auth | Set or update password |
| GET | `/v1/auth/me` | Auth | Current user + facilities |
| GET | `/v1/auth/session` | Auth | Live session + membership refresh |
| POST | `/v1/auth/session/refresh` | Auth | Same as session refresh |
| GET | `/v1/auth/ui` | Public | Redirect to login UI |
| GET | `/v1/auth/ui/` | Public | Redirect to login UI |

---

## 2. Notifications

**Compose service:** `notifications`  
**Prefix:** `/v1/notifications`  
**Guide:** [NOTIFICATIONS.md](./NOTIFICATIONS.md)  
**OpenAPI:** [`GET /v1/notifications/openapi.json`](http://localhost:8080/v1/notifications/openapi.json)  
**Swagger:** [`/docs?spec=notifications`](http://localhost:8080/docs?spec=notifications)

WhatsApp and push notification delivery, facility config, device registration.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/notifications/health` | Public | Health check |
| GET | `/v1/notifications/openapi.json` | Public | OpenAPI 3 spec |
| POST | `/v1/notifications/send` | Auth | Send single WhatsApp/push message |
| POST | `/v1/notifications/send-bulk` | Auth | Send bulk messages |
| GET | `/v1/notifications/facilities/{facilityId}/whatsapp` | Auth | Get facility WhatsApp config |
| PUT | `/v1/notifications/facilities/{facilityId}/whatsapp` | Auth | Update facility WhatsApp config |
| GET | `/v1/notifications/whatsapp/lookup` | Auth | Lookup facility by phone |
| GET | `/v1/notifications/{id}` | Auth | Get notification job status |
| POST | `/v1/notifications/devices` | Auth | Register push device token |

---

## 3. Booking

**Compose service:** `booking`  
**Prefix:** `/v1/booking`  
**Guide:** [BOOKING.md](./BOOKING.md)  
**OpenAPI:** [`GET /v1/booking/openapi.json`](http://localhost:8080/v1/booking/openapi.json)  
**Swagger:** [`/docs?spec=booking`](http://localhost:8080/docs?spec=booking)

Courts, slot generation, availability, bookings, block/unblock.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/booking/health` | Public | Health check |
| GET | `/v1/booking/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/booking/facilities/{facilityId}/courts` | Auth | List courts |
| POST | `/v1/booking/facilities/{facilityId}/courts` | Auth | Create court |
| GET | `/v1/booking/facilities/{facilityId}/slots` | Auth | List slots (filter by date) |
| POST | `/v1/booking/facilities/{facilityId}/slots/generate` | Auth | Generate slots for a day |
| POST | `/v1/booking/slots/{slotId}/block` | Auth | Block slot (staff) |
| POST | `/v1/booking/slots/{slotId}/unblock` | Auth | Unblock slot |
| GET | `/v1/booking/facilities/{facilityId}/bookings` | Auth | List bookings |
| POST | `/v1/booking/facilities/{facilityId}/bookings` | Auth | Create booking |
| GET | `/v1/booking/bookings/{bookingId}` | Auth | Get booking |
| POST | `/v1/booking/bookings/{bookingId}/cancel` | Auth | Cancel booking |

---

## 4. Students

**Compose service:** `students`  
**Prefix:** `/v1/students`  
**Guide:** [STUDENTS.md](./STUDENTS.md)  
**OpenAPI:** [`GET /v1/students/openapi.json`](http://localhost:8080/v1/students/openapi.json)  
**Swagger:** [`/docs?spec=students`](http://localhost:8080/docs?spec=students)

Student enrollment and attendance.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/students/health` | Public | Health check |
| GET | `/v1/students/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/students/facilities/{facilityId}/students/summary` | Auth | Enrollment summary |
| GET | `/v1/students/facilities/{facilityId}/students` | Auth | List enrolled students |
| POST | `/v1/students/facilities/{facilityId}/students` | Auth | Enroll student |
| GET | `/v1/students/facilities/{facilityId}/attendance` | Auth | List attendance records |
| POST | `/v1/students/facilities/{facilityId}/attendance` | Auth | Mark attendance |

---

## 5. Members

**Compose service:** `members`  
**Prefix:** `/v1/members`  
**Guide:** [MEMBERS.md](./MEMBERS.md)  
**OpenAPI:** [`GET /v1/members/openapi.json`](http://localhost:8080/v1/members/openapi.json)  
**Swagger:** [`/docs?spec=members`](http://localhost:8080/docs?spec=members)

Facility membership registration and status.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/members/health` | Public | Health check |
| GET | `/v1/members/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/members/facilities/{facilityId}/members/summary` | Auth | Active member count |
| GET | `/v1/members/facilities/{facilityId}/members` | Auth | List memberships |
| POST | `/v1/members/facilities/{facilityId}/members` | Auth | Register member |
| POST | `/v1/members/facilities/{facilityId}/memberships/{membershipId}/status` | Auth | Update membership status |

---

## 6. Tournaments

**Compose service:** `tournaments`  
**Prefix:** `/v1/tournaments`  
**Guide:** [TOURNAMENTS.md](./TOURNAMENTS.md)  
**OpenAPI:** [`GET /v1/tournaments/openapi.json`](http://localhost:8080/v1/tournaments/openapi.json)  
**Swagger:** [`/docs?spec=tournaments`](http://localhost:8080/docs?spec=tournaments)

Tournament CRUD and status.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/tournaments/health` | Public | Health check |
| GET | `/v1/tournaments/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/tournaments/facilities/{facilityId}/tournaments/summary` | Auth | Tournament summary |
| GET | `/v1/tournaments/facilities/{facilityId}/tournaments` | Auth | List tournaments |
| POST | `/v1/tournaments/facilities/{facilityId}/tournaments` | Auth | Create tournament |
| GET | `/v1/tournaments/tournaments/{tournamentId}` | Auth | Get tournament |
| POST | `/v1/tournaments/tournaments/{tournamentId}` | Auth | Update tournament |
| POST | `/v1/tournaments/tournaments/{tournamentId}/status` | Auth | Update tournament status |

---

## 7. AI (shared)

**Compose service:** `ai`  
**Prefix:** `/v1/ai`  
**Guide:** [AI.md](./AI.md)  
**OpenAPI:** [`GET /v1/ai/openapi.json`](http://localhost:8080/v1/ai/openapi.json)  
**Swagger:** [`/docs?spec=ai`](http://localhost:8080/docs?spec=ai)

Shared AI skills used by enquiry and other services.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/ai/health` | Public | Health check |
| GET | `/v1/ai/openapi.json` | Public | OpenAPI 3 spec |
| POST | `/v1/ai/enquiry/analyze` | Auth | Analyze enquiry transcript |
| POST | `/v1/ai/reply/suggest` | Auth | Suggest reply text |

---

## 8. Enquiry

**Compose service:** `enquiry`  
**Prefix:** `/v1/enquiry`  
**Guide:** [ENQUIRY.md](./ENQUIRY.md)  
**OpenAPI:** [`GET /v1/enquiry/openapi.json`](http://localhost:8080/v1/enquiry/openapi.json)  
**Swagger:** [`/docs?spec=enquiry`](http://localhost:8080/docs?spec=enquiry)

Enquiry inbox with AI analyze, reply suggest, and WhatsApp notify.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/enquiry/health` | Public | Health check |
| GET | `/v1/enquiry/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/enquiry/facilities/{facilityId}/enquiries/summary` | Auth | Enquiry summary |
| GET | `/v1/enquiry/facilities/{facilityId}/enquiries` | Auth | List enquiries |
| POST | `/v1/enquiry/facilities/{facilityId}/enquiries` | Auth | Create enquiry |
| GET | `/v1/enquiry/enquiries/{enquiryId}` | Auth | Get enquiry |
| POST | `/v1/enquiry/enquiries/{enquiryId}/status` | Auth | Update status |
| POST | `/v1/enquiry/enquiries/{enquiryId}/analyze` | Auth | Re-run AI analysis |
| POST | `/v1/enquiry/enquiries/{enquiryId}/suggest-reply` | Auth | Suggest reply via AI |
| POST | `/v1/enquiry/enquiries/{enquiryId}/notify` | Auth | Send WhatsApp notification |

---

## 9. Offers

**Compose service:** `offers`  
**Prefix:** `/v1/offers`  
**Guide:** [OFFERS.md](./OFFERS.md)  
**OpenAPI:** [`GET /v1/offers/openapi.json`](http://localhost:8080/v1/offers/openapi.json)  
**Swagger:** [`/docs?spec=offers`](http://localhost:8080/docs?spec=offers)

Promotions and discount coupons.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/offers/health` | Public | Health check |
| GET | `/v1/offers/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/offers/facilities/{facilityId}/offers/summary` | Auth | Active offer count |
| GET | `/v1/offers/facilities/{facilityId}/offers` | Auth | List offers |
| POST | `/v1/offers/facilities/{facilityId}/offers` | Auth | Create offer |
| GET | `/v1/offers/facilities/{facilityId}/offers/validate` | Auth | Validate coupon code |
| POST | `/v1/offers/facilities/{facilityId}/offers/{offerId}/status` | Auth | Activate/deactivate offer |

---

## 10. Administration

**Compose service:** `administration`  
**Prefix:** `/v1/administration`  
**Guide:** [ADMINISTRATION.md](./ADMINISTRATION.md)  
**OpenAPI:** [`GET /v1/administration/openapi.json`](http://localhost:8080/v1/administration/openapi.json)  
**Swagger:** [`/docs?spec=administration`](http://localhost:8080/docs?spec=administration)

Facility profile, sports, courts catalogue, payment modes, service flags, staff.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/administration/health` | Public | Health check |
| GET | `/v1/administration/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/administration/pages` | Auth | Portal page catalog + role defaults |
| GET | `/v1/administration/facilities/{facilityId}/summary` | Auth | Administration summary |
| GET | `/v1/administration/facilities/{facilityId}/profile` | Auth | Get facility profile |
| PUT | `/v1/administration/facilities/{facilityId}/profile` | Auth | Update facility profile |
| GET | `/v1/administration/facilities/{facilityId}/sports` | Auth | List sports |
| POST | `/v1/administration/facilities/{facilityId}/sports` | Auth | Create sport |
| POST | `/v1/administration/facilities/{facilityId}/sports/{sportId}/status` | Auth | Update sport status |
| GET | `/v1/administration/facilities/{facilityId}/courts` | Auth | List courts (catalogue) |
| POST | `/v1/administration/facilities/{facilityId}/courts` | Auth | Create court (catalogue) |
| POST | `/v1/administration/facilities/{facilityId}/courts/{courtId}/status` | Auth | Update court status |
| GET | `/v1/administration/facilities/{facilityId}/payment-settings` | Auth | List payment mode settings |
| PUT | `/v1/administration/facilities/{facilityId}/payment-settings` | Auth | Update payment mode settings |
| GET | `/v1/administration/facilities/{facilityId}/services` | Auth | Get enabled service flags |
| PUT | `/v1/administration/facilities/{facilityId}/services` | Auth | Update service flags |
| GET | `/v1/administration/facilities/{facilityId}/staff` | Auth | List staff memberships |
| POST | `/v1/administration/facilities/{facilityId}/staff` | Auth | Upsert staff membership |
| POST | `/v1/administration/facilities/{facilityId}/staff/{membershipId}/status` | Auth | Update staff status |

---

## 11. Add facility

**Compose service:** `addfacility`  
**Prefix:** `/v1/add-facility`  
**Guide:** [ADD_FACILITY.md](./ADD_FACILITY.md)  
**OpenAPI:** [`GET /v1/add-facility/openapi.json`](http://localhost:8080/v1/add-facility/openapi.json)  
**Swagger:** [`/docs?spec=add-facility`](http://localhost:8080/docs?spec=add-facility)

Onboarding requests for new or additional facilities; approve provisions identity + admin seed.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/add-facility/health` | Public | Health check |
| GET | `/v1/add-facility/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/add-facility/summary` | Auth | Request counts for current user |
| GET | `/v1/add-facility/requests/mine` | Auth | List my requests |
| GET | `/v1/add-facility/requests/pending` | Auth | List pending requests (reviewer) |
| GET | `/v1/add-facility/requests/{requestId}` | Auth | Get request |
| POST | `/v1/add-facility/requests` | Auth | Submit facility request |
| POST | `/v1/add-facility/requests/{requestId}/approve` | Auth | Approve → create facility |
| POST | `/v1/add-facility/requests/{requestId}/reject` | Auth | Reject request |

---

## 12. Payments (ledger)

**Compose service:** `payments`  
**Prefix:** `/v1/payments`  
**Guide:** [PAYMENTS.md](./PAYMENTS.md)  
**OpenAPI:** [`GET /v1/payments/openapi.json`](http://localhost:8080/v1/payments/openapi.json)  
**Swagger:** [`/docs?spec=payments`](http://localhost:8080/docs?spec=payments)

Facility payment ledger and summary (not Razorpay checkout — see pay gateway).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/payments/health` | Public | Health check |
| GET | `/v1/payments/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/payments/facilities/{facilityId}/ledger` | Auth | List ledger entries |
| GET | `/v1/payments/facilities/{facilityId}/summary` | Auth | Payment totals summary |
| POST | `/v1/payments/facilities/{facilityId}/payments` | Auth | Record ledger entry |
| POST | `/v1/payments/facilities/{facilityId}/payments/{paymentId}/mark-paid` | Auth | Mark entry as paid |

---

## 13. Pay gateway

**Compose service:** `paygateway`  
**Prefix:** `/v1/gateway`  
**Guide:** [GATEWAY.md](./GATEWAY.md)  
**OpenAPI:** [`GET /v1/gateway/openapi.json`](http://localhost:8080/v1/gateway/openapi.json)  
**Swagger:** [`/docs?spec=gateway`](http://localhost:8080/docs?spec=gateway)

Provider abstraction for checkout (stub + Razorpay; Cashfree planned).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/v1/gateway/health` | Public | Health + configured providers |
| GET | `/v1/gateway/openapi.json` | Public | OpenAPI 3 spec |
| GET | `/v1/gateway/providers` | Auth | List payment adapters |
| POST | `/v1/gateway/orders` | Auth | Create provider order |
| POST | `/v1/gateway/verify` | Auth | Verify payment signature |
| POST | `/v1/gateway/webhooks/{provider}` | Public | Provider webhook (`razorpay`, `stub`, …) |

---

## 14. Web portal

**Compose service:** `web`  
**Stack:** Next.js 14 (`apps/web`)

UI pages and lightweight health; API traffic is routed to Go services above.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/health` | Public | Web app health |
| GET | `/docs` | Public | Swagger UI (all service specs) |
| GET | `/login` | Public | Login page |
| GET | `/home` | Auth (session) | Post-login home |
| GET | `/facility/{slug}/…` | Auth (session) | Facility-scoped portal pages |

---

## Service keys (production)

Each Go service accepts an optional `X-Service-Key` for server-to-server calls:

| Service | Env variable |
|---------|----------------|
| Notifications | `NOTIFICATIONS_SERVICE_KEY` |
| Booking | `BOOKING_SERVICE_KEY` |
| Students | `STUDENTS_SERVICE_KEY` |
| Members | `MEMBERS_SERVICE_KEY` |
| Tournaments | `TOURNAMENTS_SERVICE_KEY` |
| AI | `AI_SERVICE_KEY` |
| Enquiry | `ENQUIRY_SERVICE_KEY` |
| Offers | `OFFERS_SERVICE_KEY` |
| Administration | `ADMINISTRATION_SERVICE_KEY` |
| Add facility | `ADDFACILITY_SERVICE_KEY` |
| Payments | `PAYMENTS_SERVICE_KEY` |
| Pay gateway | `PAYGATEWAY_SERVICE_KEY` |

Local development may omit service keys when configured to allow it.

---

## OpenAPI source files (repo)

Each spec is served at runtime and also lives in the repository:

```
services/identity/internal/httpapi/openapi.json
services/notifications/internal/httpapi/openapi.json
services/booking/internal/httpapi/openapi.json
services/students/internal/httpapi/openapi.json
services/members/internal/httpapi/openapi.json
services/tournaments/internal/httpapi/openapi.json
services/ai/internal/httpapi/openapi.json
services/enquiry/internal/httpapi/openapi.json
services/offers/internal/httpapi/openapi.json
services/administration/internal/httpapi/openapi.json
services/addfacility/internal/httpapi/openapi.json
services/payments/internal/httpapi/openapi.json
services/paygateway/internal/httpapi/openapi.json
```
