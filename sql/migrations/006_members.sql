-- Members service tables (shared fac_identity DB for MVP; can split later)
CREATE TABLE IF NOT EXISTS members (
  id              BIGSERIAL PRIMARY KEY,
  full_name       VARCHAR(191) NOT NULL,
  contact_email   VARCHAR(191) NULL,
  contact_phone   VARCHAR(32) NULL,
  whatsapp        VARCHAR(32) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','inactive')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_members_phone ON members(contact_phone);
CREATE INDEX IF NOT EXISTS idx_members_email ON members(contact_email);

CREATE TABLE IF NOT EXISTS memberships (
  id                 BIGSERIAL PRIMARY KEY,
  facility_id        BIGINT NOT NULL,
  member_id          BIGINT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  sport_id           BIGINT NULL,
  plan_name          VARCHAR(128) NULL,
  team_name          VARCHAR(128) NULL,
  start_date         DATE NULL,
  end_date           DATE NULL,
  subscription_fee   NUMERIC(12,2) NULL,
  primary_member     BOOLEAN NOT NULL DEFAULT FALSE,
  status             VARCHAR(16) NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active','inactive','expired')),
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_memberships_facility
  ON memberships(facility_id, status);
CREATE INDEX IF NOT EXISTS idx_memberships_member
  ON memberships(member_id);
