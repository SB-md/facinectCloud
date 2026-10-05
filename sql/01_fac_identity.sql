-- Bootstrap for empty volumes (docker-entrypoint-initdb.d).
-- Canonical incremental changes: sql/migrations/ + ./scripts/migrate.sh
-- Keep in sync with sql/migrations/001_fac_identity.sql

-- fac_identity schema (PostgreSQL)
-- Runs in POSTGRES_DB context on first container boot.

CREATE TABLE IF NOT EXISTS users (
  id            BIGSERIAL PRIMARY KEY,
  email         VARCHAR(255) UNIQUE,
  phone_e164    VARCHAR(20) UNIQUE,
  password_hash VARCHAR(255),
  full_name     VARCHAR(191) NOT NULL DEFAULT '',
  avatar_url    TEXT,
  status        VARCHAR(16) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','disabled')),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT NOT NULL REFERENCES users(id),
  jti           CHAR(36) NOT NULL UNIQUE,
  token_hash    CHAR(64) NOT NULL,
  expires_at    TIMESTAMPTZ NOT NULL,
  revoked_at    TIMESTAMPTZ NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_refresh_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_hash ON refresh_tokens(token_hash);

CREATE TABLE IF NOT EXISTS otp_challenges (
  id            BIGSERIAL PRIMARY KEY,
  phone_e164    VARCHAR(20) NOT NULL,
  code_hash     CHAR(64) NOT NULL,
  attempts      SMALLINT NOT NULL DEFAULT 0,
  expires_at    TIMESTAMPTZ NOT NULL,
  consumed_at   TIMESTAMPTZ NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_otp_phone ON otp_challenges(phone_e164, created_at);

CREATE TABLE IF NOT EXISTS audit_log (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT NULL,
  event         VARCHAR(64) NOT NULL,
  ip            VARCHAR(45) NULL,
  meta_json     JSONB NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_event ON audit_log(event, created_at);

CREATE TABLE IF NOT EXISTS facilities (
  id            BIGSERIAL PRIMARY KEY,
  name          VARCHAR(191) NOT NULL,
  slug          VARCHAR(191) NOT NULL UNIQUE,
  status        VARCHAR(16) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','disabled')),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_facility_memberships (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT NOT NULL REFERENCES users(id),
  facility_id   BIGINT NOT NULL REFERENCES facilities(id),
  role          VARCHAR(32) NOT NULL DEFAULT 'admin',
  page_access   JSONB NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','disabled')),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, facility_id)
);
CREATE INDEX IF NOT EXISTS idx_membership_facility ON user_facility_memberships(facility_id);
