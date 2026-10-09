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
CREATE TABLE IF NOT EXISTS payment_ledger (
  id               BIGSERIAL PRIMARY KEY,
  facility_id      BIGINT NOT NULL,
  category         VARCHAR(32) NOT NULL DEFAULT 'BOOKING'
                   CHECK (category IN ('BOOKING','MEMBERSHIP','COACHING','TOURNAMENT','OTHER')),
  title            VARCHAR(191) NOT NULL DEFAULT '',
  customer_name    VARCHAR(191) NULL,
  customer_phone   VARCHAR(32) NULL,
  amount           NUMERIC(12,2) NOT NULL DEFAULT 0,
  paid_amount      NUMERIC(12,2) NOT NULL DEFAULT 0,
  status           VARCHAR(16) NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending','paid','partial','failed','refunded')),
  payment_method   VARCHAR(32) NULL,
  reference_type   VARCHAR(32) NULL,
  reference_id     VARCHAR(64) NULL,
  notes            TEXT NULL,
  paid_at          TIMESTAMPTZ NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payment_ledger_facility_status
  ON payment_ledger(facility_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_ledger_facility_category
  ON payment_ledger(facility_id, category, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_ledger_facility_dates
  ON payment_ledger(facility_id, created_at DESC);
`)
	return err
}
