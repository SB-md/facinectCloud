-- Enquiry service tables (shared fac_identity DB for MVP)
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
