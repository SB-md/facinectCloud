-- Members Phase-2 extras: plans, settings, requests, coupons, open slots
ALTER TABLE memberships
  ADD COLUMN IF NOT EXISTS payment_status VARCHAR(32) NOT NULL DEFAULT 'unpaid';
ALTER TABLE memberships
  ADD COLUMN IF NOT EXISTS plan_id BIGINT NULL;

CREATE TABLE IF NOT EXISTS member_plans (
  id              BIGSERIAL PRIMARY KEY,
  facility_id     BIGINT NOT NULL,
  name            VARCHAR(128) NOT NULL,
  price           NUMERIC(12,2) NULL,
  duration_days   INT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','inactive')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_member_plans_facility ON member_plans(facility_id, status);

CREATE TABLE IF NOT EXISTS member_plan_settings (
  plan_id         BIGINT PRIMARY KEY REFERENCES member_plans(id) ON DELETE CASCADE,
  facility_id     BIGINT NOT NULL,
  settings_json   JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS member_requests (
  id              BIGSERIAL PRIMARY KEY,
  facility_id     BIGINT NOT NULL,
  member_id       BIGINT NULL,
  membership_id   BIGINT NULL,
  full_name       VARCHAR(191) NULL,
  contact_phone   VARCHAR(32) NULL,
  request_box     TEXT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','approved','rejected')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_member_requests_facility
  ON member_requests(facility_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS member_coupons (
  id              BIGSERIAL PRIMARY KEY,
  facility_id     BIGINT NOT NULL,
  plan_id         BIGINT NULL,
  code            VARCHAR(64) NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','used','disabled')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (facility_id, code)
);
CREATE INDEX IF NOT EXISTS idx_member_coupons_facility ON member_coupons(facility_id);

CREATE TABLE IF NOT EXISTS member_open_slots (
  id              BIGSERIAL PRIMARY KEY,
  facility_id     BIGINT NOT NULL,
  sport_id        BIGINT NULL,
  sub_facility    VARCHAR(128) NULL,
  slot_time       VARCHAR(64) NULL,
  from_date       DATE NULL,
  to_date         DATE NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open','closed')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_member_open_slots_facility ON member_open_slots(facility_id);
