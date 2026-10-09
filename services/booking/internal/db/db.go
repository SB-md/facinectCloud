package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(host, port, user, password, name string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, name,
	)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
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
`)
	return err
}
