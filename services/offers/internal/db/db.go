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
`)
	return err
}
