-- Students service tables (shared fac_identity DB for MVP; can split later)
CREATE TABLE IF NOT EXISTS students (
  id              BIGSERIAL PRIMARY KEY,
  first_name      VARCHAR(96) NOT NULL,
  last_name       VARCHAR(96) NOT NULL DEFAULT '',
  contact_email   VARCHAR(191) NULL,
  contact_phone   VARCHAR(32) NULL,
  whatsapp        VARCHAR(32) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','inactive')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_students_phone ON students(contact_phone);
CREATE INDEX IF NOT EXISTS idx_students_email ON students(contact_email);

CREATE TABLE IF NOT EXISTS student_enrollments (
  id              BIGSERIAL PRIMARY KEY,
  facility_id     BIGINT NOT NULL,
  student_id      BIGINT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
  sport_id        BIGINT NULL,
  plan_name       VARCHAR(128) NULL,
  start_date      DATE NULL,
  end_date        DATE NULL,
  actual_fee      NUMERIC(12,2) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','inactive','completed')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_student_enrollments_facility
  ON student_enrollments(facility_id, status);
CREATE INDEX IF NOT EXISTS idx_student_enrollments_student
  ON student_enrollments(student_id);

CREATE TABLE IF NOT EXISTS student_attendance (
  id                BIGSERIAL PRIMARY KEY,
  facility_id       BIGINT NOT NULL,
  enrollment_id     BIGINT NOT NULL REFERENCES student_enrollments(id) ON DELETE CASCADE,
  attendance_date   DATE NOT NULL,
  status            VARCHAR(16) NOT NULL
                    CHECK (status IN ('present','absent','leave')),
  marked_by         BIGINT NULL,
  notes             TEXT NULL,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (enrollment_id, attendance_date)
);
CREATE INDEX IF NOT EXISTS idx_student_attendance_facility_date
  ON student_attendance(facility_id, attendance_date);
