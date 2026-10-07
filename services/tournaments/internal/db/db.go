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
`)
	return err
}
