-- Per-facility WhatsApp config (PHP Facinect hybrid model)
-- Token/phone override in DB; empty → env META_WHATSAPP_TOKEN / META_PHONE_NUMBER_ID

CREATE TABLE IF NOT EXISTS facility_whatsapp_config (
  facility_id            BIGINT PRIMARY KEY,
  meta_phone_number_id   VARCHAR(64) NULL,
  whatsapp_api_token     VARCHAR(512) NULL,
  waba_id                VARCHAR(64) NULL,
  whatsapp_enabled       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_facility_wa_phone
  ON facility_whatsapp_config(meta_phone_number_id)
  WHERE meta_phone_number_id IS NOT NULL AND meta_phone_number_id <> '';
