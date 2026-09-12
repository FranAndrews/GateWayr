-- GateWayr Database Schema (PostgreSQL / Google Cloud SQL)

CREATE TABLE IF NOT EXISTS applications (
    id VARCHAR(64) PRIMARY KEY,
    applicant_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    income_amount NUMERIC(14, 2) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_applications_created_at ON applications (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_applications_status ON applications (status);
