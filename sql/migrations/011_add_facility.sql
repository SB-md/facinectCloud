-- Add-facility / onboarding requests (shared fac_identity DB for MVP)
CREATE TABLE IF NOT EXISTS facility_requests (
  id                   BIGSERIAL PRIMARY KEY,
  requester_user_id    BIGINT NOT NULL,
  requester_email      VARCHAR(255) NOT NULL,
  facility_name        VARCHAR(191) NOT NULL,
  location             VARCHAR(255) NOT NULL,
  contact_name         VARCHAR(191) NULL,
  whatsapp             VARCHAR(32) NULL,
  alternate_number     VARCHAR(32) NULL,
  google_map_url       TEXT NULL,
  notes                TEXT NULL,
  sports_json          JSONB NULL,
  status               VARCHAR(16) NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending','approved','rejected')),
  facility_id          BIGINT NULL,
  reviewed_by          BIGINT NULL,
  reviewed_at          TIMESTAMPTZ NULL,
  review_note          TEXT NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_facility_requests_requester
  ON facility_requests(requester_user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_facility_requests_status
  ON facility_requests(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_facility_requests_email
  ON facility_requests(requester_email, status);
