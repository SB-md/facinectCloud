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
CREATE TABLE IF NOT EXISTS enquiries (
  id                 BIGSERIAL PRIMARY KEY,
  facility_id        BIGINT NOT NULL,
  customer_phone     VARCHAR(32) NOT NULL,
  customer_name      VARCHAR(191) NULL,
  enquiry_details    TEXT NOT NULL,
  status             VARCHAR(32) NOT NULL DEFAULT 'New'
                     CHECK (status IN ('New','Follow-up','Resolved','Closed')),
  ai_data            JSONB NULL,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_enquiries_facility_status
  ON enquiries(facility_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_enquiries_facility_phone
  ON enquiries(facility_id, customer_phone);
`)
	return err
}
