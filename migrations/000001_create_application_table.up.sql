CREATE TABLE IF NOT EXISTS applications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(50) NOT NULL,
  description VARCHAR(100),
  client_id VARCHAR(64) NOT NULL UNIQUE,
  client_secret_hash TEXT NOT NULL,
  settings JSONB NOT NULL DEFAULT '{}',
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  updated_by VARCHAR(30),
  creation_timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT applications_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_applications_client_id ON applications (client_id);
