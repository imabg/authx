-- Reverse of 000006: drop the rebuilt users table and recreate the intended
-- 000002 schema. Does not restore the leftover pre-000002 users table.
CREATE EXTENSION IF NOT EXISTS citext;

ALTER TABLE IF EXISTS credentials DROP CONSTRAINT IF EXISTS credentials_user_id_fkey;
ALTER TABLE IF EXISTS refresh_tokens DROP CONSTRAINT IF EXISTS refresh_tokens_user_id_fkey;
ALTER TABLE IF EXISTS auth_challenges DROP CONSTRAINT IF EXISTS auth_challenges_user_id_fkey;

DROP TABLE IF EXISTS users;

CREATE TABLE users (
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

ALTER TABLE credentials
  ADD CONSTRAINT credentials_user_id_fkey
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE refresh_tokens
  ADD CONSTRAINT refresh_tokens_user_id_fkey
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE auth_challenges
  ADD CONSTRAINT auth_challenges_user_id_fkey
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
