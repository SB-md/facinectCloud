package db

import (
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
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	for i := 0; i < 40; i++ {
		if err = db.Ping(); err == nil {
			return db, nil
		}
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("postgres ping failed: %w", err)
}

func EnsureSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
		  id            BIGSERIAL PRIMARY KEY,
		  email         VARCHAR(255) UNIQUE,
		  phone_e164    VARCHAR(20) UNIQUE,
		  password_hash VARCHAR(255),
		  full_name     VARCHAR(191) NOT NULL DEFAULT '',
		  status        VARCHAR(16) NOT NULL DEFAULT 'active'
		                CHECK (status IN ('active','disabled')),
		  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
		  id            BIGSERIAL PRIMARY KEY,
		  user_id       BIGINT NOT NULL REFERENCES users(id),
		  jti           CHAR(36) NOT NULL UNIQUE,
		  token_hash    CHAR(64) NOT NULL,
		  expires_at    TIMESTAMPTZ NOT NULL,
		  revoked_at    TIMESTAMPTZ NULL,
		  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_user ON refresh_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_hash ON refresh_tokens(token_hash)`,
		`CREATE TABLE IF NOT EXISTS otp_challenges (
		  id            BIGSERIAL PRIMARY KEY,
		  phone_e164    VARCHAR(20) NOT NULL,
		  code_hash     CHAR(64) NOT NULL,
		  attempts      SMALLINT NOT NULL DEFAULT 0,
		  expires_at    TIMESTAMPTZ NOT NULL,
		  consumed_at   TIMESTAMPTZ NULL,
		  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_otp_phone ON otp_challenges(phone_e164, created_at)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
		  id            BIGSERIAL PRIMARY KEY,
		  user_id       BIGINT NULL,
		  event         VARCHAR(64) NOT NULL,
		  ip            VARCHAR(45) NULL,
		  meta_json     JSONB NULL,
		  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_event ON audit_log(event, created_at)`,
		`CREATE TABLE IF NOT EXISTS oauth_pkce (
		  state         CHAR(64) PRIMARY KEY,
		  code_verifier VARCHAR(128) NOT NULL,
		  redirect_uri  VARCHAR(512) NOT NULL,
		  expires_at    TIMESTAMPTZ NOT NULL,
		  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_accounts (
		  id            BIGSERIAL PRIMARY KEY,
		  user_id       BIGINT NOT NULL REFERENCES users(id),
		  provider      VARCHAR(32) NOT NULL,
		  provider_sub  VARCHAR(191) NOT NULL,
		  email         VARCHAR(255) NULL,
		  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  UNIQUE (provider, provider_sub)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth_user ON oauth_accounts(user_id)`,
		`CREATE TABLE IF NOT EXISTS oauth_handoff (
		  code_hash     CHAR(64) PRIMARY KEY,
		  payload_json  JSONB NOT NULL,
		  expires_at    TIMESTAMPTZ NOT NULL,
		  consumed_at   TIMESTAMPTZ NULL,
		  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS facilities (
		  id            BIGSERIAL PRIMARY KEY,
		  name          VARCHAR(191) NOT NULL,
		  slug          VARCHAR(191) NOT NULL UNIQUE,
		  status        VARCHAR(16) NOT NULL DEFAULT 'active'
		                CHECK (status IN ('active','disabled')),
		  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS user_facility_memberships (
		  id            BIGSERIAL PRIMARY KEY,
		  user_id       BIGINT NOT NULL REFERENCES users(id),
		  facility_id   BIGINT NOT NULL REFERENCES facilities(id),
		  role          VARCHAR(32) NOT NULL DEFAULT 'admin',
		  page_access   JSONB NULL,
		  status        VARCHAR(16) NOT NULL DEFAULT 'active'
		                CHECK (status IN ('active','disabled')),
		  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  UNIQUE (user_id, facility_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_membership_facility ON user_facility_memberships(facility_id)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("schema: %w\nstmt: %s", err, s)
		}
	}
	return nil
}
