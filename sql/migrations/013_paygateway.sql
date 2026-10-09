-- Payment gateway abstraction (orders ledger; shared fac_identity DB for MVP)
CREATE TABLE IF NOT EXISTS gateway_orders (
  id                 BIGSERIAL PRIMARY KEY,
  provider           VARCHAR(32) NOT NULL,
  external_order_id  VARCHAR(128) NULL,
  amount_paise       BIGINT NOT NULL,
  currency           VARCHAR(8) NOT NULL DEFAULT 'INR',
  receipt            VARCHAR(128) NULL,
  status             VARCHAR(32) NOT NULL DEFAULT 'created'
                     CHECK (status IN ('created','paid','failed','refunded')),
  facility_id        BIGINT NULL,
  purpose            VARCHAR(64) NULL,
  metadata           JSONB NULL,
  checkout_json      JSONB NULL,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_gateway_orders_provider
  ON gateway_orders(provider, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gateway_orders_external
  ON gateway_orders(provider, external_order_id);
CREATE INDEX IF NOT EXISTS idx_gateway_orders_receipt
  ON gateway_orders(receipt) WHERE receipt IS NOT NULL AND receipt <> '';
