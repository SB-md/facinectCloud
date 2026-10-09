-- Booking Phase-2 extras: payment collect, coupons, user status, blocks
ALTER TABLE booking_bookings
  ADD COLUMN IF NOT EXISTS payment_status VARCHAR(32) NOT NULL DEFAULT 'unpaid';
ALTER TABLE booking_bookings
  ADD COLUMN IF NOT EXISTS pay_mode VARCHAR(64) NULL;
ALTER TABLE booking_bookings
  ADD COLUMN IF NOT EXISTS amount_paid NUMERIC(12,2) NULL;

CREATE TABLE IF NOT EXISTS booking_coupons (
  id              BIGSERIAL PRIMARY KEY,
  facility_id     BIGINT NOT NULL,
  code            VARCHAR(64) NOT NULL,
  discount        NUMERIC(12,2) NOT NULL DEFAULT 0,
  status          VARCHAR(16) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','used','disabled')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (facility_id, code)
);
CREATE INDEX IF NOT EXISTS idx_booking_coupons_facility ON booking_coupons(facility_id);

CREATE TABLE IF NOT EXISTS booking_user_status (
  facility_id     BIGINT NOT NULL,
  user_id         BIGINT NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','blocked')),
  reason          TEXT NULL,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (facility_id, user_id)
);

CREATE TABLE IF NOT EXISTS booking_blocks (
  id              BIGSERIAL PRIMARY KEY,
  facility_id     BIGINT NOT NULL,
  court_id        BIGINT NULL REFERENCES booking_courts(id) ON DELETE SET NULL,
  block_date      DATE NOT NULL,
  start_time      TIME NOT NULL,
  end_time        TIME NOT NULL,
  reason          TEXT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_booking_blocks_facility_date
  ON booking_blocks(facility_id, block_date);
