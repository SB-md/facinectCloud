-- Tournaments service tables (shared fac_identity DB for MVP; can split later)
CREATE TABLE IF NOT EXISTS tournaments (
  id                 BIGSERIAL PRIMARY KEY,
  facility_id        BIGINT NOT NULL,
  sport_id           BIGINT NULL,
  name               VARCHAR(191) NOT NULL,
  start_date         DATE NULL,
  end_date           DATE NULL,
  venue_address      TEXT NULL,
  location_url       TEXT NULL,
  contact_numbers    VARCHAR(255) NULL,
  rules              TEXT NULL,
  slug               VARCHAR(191) NULL,
  status             VARCHAR(16) NOT NULL DEFAULT 'draft'
                     CHECK (status IN ('draft','upcoming','live','completed','cancelled')),
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tournaments_facility
  ON tournaments(facility_id, status);
CREATE INDEX IF NOT EXISTS idx_tournaments_facility_dates
  ON tournaments(facility_id, start_date DESC NULLS LAST);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tournaments_facility_slug
  ON tournaments(facility_id, slug) WHERE slug IS NOT NULL AND slug <> '';
