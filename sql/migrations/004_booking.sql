-- Booking service tables (shared fac_identity DB for MVP; can split later)
CREATE TABLE IF NOT EXISTS booking_courts (
  id            BIGSERIAL PRIMARY KEY,
  facility_id   BIGINT NOT NULL,
  name          VARCHAR(128) NOT NULL,
  sport_id      BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','disabled')),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_booking_courts_facility ON booking_courts(facility_id);

CREATE TABLE IF NOT EXISTS booking_slots (
  id            BIGSERIAL PRIMARY KEY,
  facility_id   BIGINT NOT NULL,
  court_id      BIGINT NOT NULL REFERENCES booking_courts(id) ON DELETE CASCADE,
  slot_date     DATE NOT NULL,
  start_time    TIME NOT NULL,
  end_time      TIME NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'available'
                CHECK (status IN ('available','booked','blocked')),
  notes         TEXT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (court_id, slot_date, start_time)
);
CREATE INDEX IF NOT EXISTS idx_booking_slots_facility_date
  ON booking_slots(facility_id, slot_date, status);

CREATE TABLE IF NOT EXISTS booking_bookings (
  id               BIGSERIAL PRIMARY KEY,
  facility_id      BIGINT NOT NULL,
  slot_id          BIGINT NOT NULL REFERENCES booking_slots(id),
  user_id          BIGINT NULL,
  customer_name    VARCHAR(191) NULL,
  customer_phone   VARCHAR(32) NULL,
  customer_email   VARCHAR(191) NULL,
  status           VARCHAR(16) NOT NULL DEFAULT 'confirmed'
                   CHECK (status IN ('confirmed','cancelled')),
  notes            TEXT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_booking_bookings_facility
  ON booking_bookings(facility_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_booking_bookings_slot ON booking_bookings(slot_id);
