<?php

declare(strict_types=1);

namespace Facinect\Identity\Db;

use PDO;

/** Ensures oauth tables exist (safe for already-initialized MySQL volumes). */
final class Schema
{
    public static function ensure(PDO $pdo): void
    {
        $pdo->exec(
            'CREATE TABLE IF NOT EXISTS oauth_pkce (
              state         CHAR(64) NOT NULL,
              code_verifier VARCHAR(128) NOT NULL,
              redirect_uri  VARCHAR(512) NOT NULL,
              expires_at    DATETIME(3) NOT NULL,
              created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
              PRIMARY KEY (state)
            ) ENGINE=InnoDB'
        );
        $pdo->exec(
            'CREATE TABLE IF NOT EXISTS oauth_accounts (
              id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
              user_id       BIGINT UNSIGNED NOT NULL,
              provider      VARCHAR(32) NOT NULL,
              provider_sub  VARCHAR(191) NOT NULL,
              email         VARCHAR(255) NULL,
              created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
              PRIMARY KEY (id),
              UNIQUE KEY uq_oauth_provider_sub (provider, provider_sub),
              KEY idx_oauth_user (user_id),
              CONSTRAINT fk_oauth_user FOREIGN KEY (user_id) REFERENCES users(id)
            ) ENGINE=InnoDB'
        );
        $pdo->exec(
            'CREATE TABLE IF NOT EXISTS oauth_handoff (
              code_hash     CHAR(64) NOT NULL,
              payload_json  JSON NOT NULL,
              expires_at    DATETIME(3) NOT NULL,
              consumed_at   DATETIME(3) NULL,
              created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
              PRIMARY KEY (code_hash)
            ) ENGINE=InnoDB'
        );
    }
}
