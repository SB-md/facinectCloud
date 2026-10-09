# StrollX ↔ Facinect Identity

StrollX facility login uses **only** the Go Identity service (`/v1/auth`).  
PHP `api/v2/auth.php` and `ConnectToApp/check_user_email.php` are **not** used for login.

## Endpoints used by the app

| Flow | Method | Path |
|------|--------|------|
| Email + password | POST | `/v1/auth/login` |
| Google Sign-In (native id_token) | POST | `/v1/auth/google` |
| Live facilities + page_access | GET | `/v1/auth/session` |
| Refresh access token | POST | `/v1/auth/token/refresh` |
| Logout | POST | `/v1/auth/logout` |
| OTP (optional) | POST | `/v1/auth/otp/request`, `/v1/auth/otp/verify` |
| Email exists? | POST | `/v1/auth/check-email` |

Web browser Google OAuth (PKCE) remains: `/v1/auth/google/start` → callback → `/handoff`.

## Gateway URLs

| Environment | `FACINECT_GATEWAY` |
|-------------|-------------------|
| Android emulator (default in app) | `http://10.0.2.2:8080` |
| iOS simulator | `http://127.0.0.1:8080` |
| Physical phone (LAN) | `http://<your-pc-ip>:8080` |
| Production | `https://cloud.facinect.com` |

```bash
# Local (emulator default already 10.0.2.2:8080)
cd ~/StudioProjects/strollx
flutter run

# Production cloud
flutter run --dart-define=FACINECT_GATEWAY=https://cloud.facinect.com
```

## Google setup

1. Identity `.env`: `GOOGLE_CLIENT_ID` = same **Web client ID** as StrollX `ApiConfig.googleServerClientId`
2. Android OAuth client: package + SHA-1 in Google Cloud Console
3. `POST /v1/auth/google` verifies `id_token` aud/azp against `GOOGLE_CLIENT_ID`

## Dashboard

See [STROLLX_DASHBOARD.md](./STROLLX_DASHBOARD.md) — admin home uses gateway members/students/booking APIs (no PHP `dashboard.php`).

## Notes

- Identity JWT authorizes `/v1/*` gateway services. Remaining PHP Hostinger APIs still need their own tokens until migrated.
- Referee Google login remains on PHP referee API (separate from facility Identity login).
- Onboarding WhatsApp OTP may still use PHP until migrated; Identity OTP is available via `AuthService.requestOtp` / `verifyOtp`.
