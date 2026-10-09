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
CREATE TABLE IF NOT EXISTS facility_profiles (
  facility_id      BIGINT PRIMARY KEY,
  display_name     VARCHAR(191) NOT NULL DEFAULT '',
  location         VARCHAR(255) NULL,
  city             VARCHAR(128) NULL,
  address          TEXT NULL,
  phone            VARCHAR(32) NULL,
  whatsapp         VARCHAR(32) NULL,
  email            VARCHAR(191) NULL,
  logo_url         TEXT NULL,
  map_url          TEXT NULL,
  notes            TEXT NULL,
  open_time        VARCHAR(8) NULL,
  close_time       VARCHAR(8) NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS facility_sports (
  id               BIGSERIAL PRIMARY KEY,
  facility_id      BIGINT NOT NULL,
  name             VARCHAR(128) NOT NULL,
  sort_order       INT NOT NULL DEFAULT 0,
  status           VARCHAR(16) NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active','inactive')),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (facility_id, name)
);
CREATE INDEX IF NOT EXISTS idx_facility_sports_facility
  ON facility_sports(facility_id, status);

CREATE TABLE IF NOT EXISTS facility_courts (
  id               BIGSERIAL PRIMARY KEY,
  facility_id      BIGINT NOT NULL,
  sport_id         BIGINT NULL REFERENCES facility_sports(id) ON DELETE SET NULL,
  name             VARCHAR(128) NOT NULL,
  price_per_hour   NUMERIC(12,2) NULL,
  open_time        VARCHAR(8) NULL,
  close_time       VARCHAR(8) NULL,
  status           VARCHAR(16) NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active','inactive')),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_facility_courts_facility
  ON facility_courts(facility_id, status);

CREATE TABLE IF NOT EXISTS facility_sport_payments (
  id                     BIGSERIAL PRIMARY KEY,
  facility_id            BIGINT NOT NULL,
  sport_id               BIGINT NOT NULL DEFAULT 0,
  allow_full_payment     BOOLEAN NOT NULL DEFAULT TRUE,
  allow_advance_payment  BOOLEAN NOT NULL DEFAULT TRUE,
  allow_spot_payment     BOOLEAN NOT NULL DEFAULT FALSE,
  advance_amount         NUMERIC(12,2) NOT NULL DEFAULT 0,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (facility_id, sport_id)
);
CREATE INDEX IF NOT EXISTS idx_facility_sport_payments_facility
  ON facility_sport_payments(facility_id);

CREATE TABLE IF NOT EXISTS facility_service_flags (
  facility_id        BIGINT PRIMARY KEY,
  staff_enabled      BOOLEAN NOT NULL DEFAULT TRUE,
  customer_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hostinger sports_categories.id (PHP get_slots / bookings still use this).
ALTER TABLE facility_sports
  ADD COLUMN IF NOT EXISTS catalog_sport_id BIGINT NULL;
`)
	return err
}
