-- Notification history / inbox log
CREATE TABLE IF NOT EXISTS notification_log (
  id              BIGSERIAL PRIMARY KEY,
  job_id          BIGINT NULL,
  facility_id     BIGINT NULL,
  user_id         BIGINT NULL,
  channel         VARCHAR(16) NOT NULL DEFAULT 'push',
  template_key    VARCHAR(64) NULL,
  title           VARCHAR(191) NULL,
  body            TEXT NULL,
  to_whatsapp     VARCHAR(32) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'sent',
  meta_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  seen_at         TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_notification_log_facility
  ON notification_log(facility_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_log_user
  ON notification_log(user_id, created_at DESC);
