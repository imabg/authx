CREATE TABLE IF NOT EXISTS auth_challenges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  type VARCHAR(20) NOT NULL,
  target VARCHAR(255) NOT NULL,
  secret_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  attempt_count INT NOT NULL DEFAULT 0,
  creation_timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT auth_challenges_type_check CHECK (type IN ('otp', 'magic_link'))
);

CREATE INDEX IF NOT EXISTS idx_auth_challenges_lookup
  ON auth_challenges (application_id, target, type, expires_at);
