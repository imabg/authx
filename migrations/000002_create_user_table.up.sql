CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
  email CITEXT NOT NULL,
  first_name VARCHAR(25) NOT NULL DEFAULT '',
  last_name VARCHAR(50) NOT NULL DEFAULT '',
  signup_method VARCHAR(25),
  email_verified_at TIMESTAMPTZ,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  creation_timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT users_status_check CHECK (status IN ('active', 'disabled')),
  CONSTRAINT users_application_email_key UNIQUE (application_id, email)
);

CREATE INDEX IF NOT EXISTS idx_users_application_id ON users (application_id);
