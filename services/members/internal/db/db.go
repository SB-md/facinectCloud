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
CREATE TABLE IF NOT EXISTS members (
  id              BIGSERIAL PRIMARY KEY,
  full_name       VARCHAR(191) NOT NULL,
  contact_email   VARCHAR(191) NULL,
  contact_phone   VARCHAR(32) NULL,
  whatsapp        VARCHAR(32) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','inactive')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_members_phone ON members(contact_phone);
CREATE INDEX IF NOT EXISTS idx_members_email ON members(contact_email);

CREATE TABLE IF NOT EXISTS memberships (
  id                 BIGSERIAL PRIMARY KEY,
  facility_id        BIGINT NOT NULL,
  member_id          BIGINT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  sport_id           BIGINT NULL,
  plan_name          VARCHAR(128) NULL,
  team_name          VARCHAR(128) NULL,
  start_date         DATE NULL,
  end_date           DATE NULL,
  subscription_fee   NUMERIC(12,2) NULL,
  primary_member     BOOLEAN NOT NULL DEFAULT FALSE,
  status             VARCHAR(16) NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active','inactive','expired')),
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_memberships_facility
  ON memberships(facility_id, status);
CREATE INDEX IF NOT EXISTS idx_memberships_member
  ON memberships(member_id);
`)
	return err
}
