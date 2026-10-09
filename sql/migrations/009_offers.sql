-- Offers service tables (shared fac_identity DB for MVP; can split later)
CREATE TABLE IF NOT EXISTS offers (
  id               BIGSERIAL PRIMARY KEY,
  facility_id      BIGINT NOT NULL,
  code             VARCHAR(64) NOT NULL,
  title            VARCHAR(191) NOT NULL,
  description      TEXT NULL,
  offer_kind       VARCHAR(32) NOT NULL DEFAULT 'discount'
                   CHECK (offer_kind IN ('promotion','discount')),
  discount_type    VARCHAR(16) NOT NULL DEFAULT 'flat'
                   CHECK (discount_type IN ('flat','percent')),
  discount_value   NUMERIC(12,2) NOT NULL DEFAULT 0,
  valid_from       DATE NULL,
  valid_to         DATE NULL,
  usage_limit      INT NULL,
  used_count       INT NOT NULL DEFAULT 0,
  sport_id         BIGINT NULL,
  status           VARCHAR(16) NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active','inactive')),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (facility_id, code)
);
CREATE INDEX IF NOT EXISTS idx_offers_facility_status
  ON offers(facility_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_offers_facility_kind
  ON offers(facility_id, offer_kind);
CREATE INDEX IF NOT EXISTS idx_offers_facility_code
  ON offers(facility_id, code);
