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
CREATE TABLE IF NOT EXISTS facility_requests (
  id                   BIGSERIAL PRIMARY KEY,
  requester_user_id    BIGINT NOT NULL,
  requester_email      VARCHAR(255) NOT NULL,
  facility_name        VARCHAR(191) NOT NULL,
  location             VARCHAR(255) NOT NULL,
  contact_name         VARCHAR(191) NULL,
  whatsapp             VARCHAR(32) NULL,
  alternate_number     VARCHAR(32) NULL,
  google_map_url       TEXT NULL,
  notes                TEXT NULL,
  sports_json          JSONB NULL,
  status               VARCHAR(16) NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending','approved','rejected')),
  facility_id          BIGINT NULL,
  reviewed_by          BIGINT NULL,
  reviewed_at          TIMESTAMPTZ NULL,
  review_note          TEXT NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_facility_requests_requester
  ON facility_requests(requester_user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_facility_requests_status
  ON facility_requests(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_facility_requests_email
  ON facility_requests(requester_email, status);
`)
	return err
}
