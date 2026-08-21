-- Multiple SMTP configurations per application. At most one row may be active.
-- New rows default to inactive. Sending mail uses the active row; if none is
-- active, the mailer fails with a clear error rather than guessing.
CREATE TABLE IF NOT EXISTS application_smtp_configs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL REFERENCES applications (id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL DEFAULT '',
  host VARCHAR(255) NOT NULL,
  port INT NOT NULL,
  username_ciphertext TEXT NOT NULL DEFAULT '',
  password_ciphertext TEXT NOT NULL DEFAULT '',
  tls BOOLEAN NOT NULL DEFAULT FALSE,
  skip_verify BOOLEAN NOT NULL DEFAULT FALSE,
  active BOOLEAN NOT NULL DEFAULT FALSE,
  updated_by VARCHAR(30),
  creation_timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT application_smtp_configs_port_check CHECK (port >= 1 AND port <= 65535)
);

CREATE INDEX IF NOT EXISTS idx_application_smtp_configs_application_id
  ON application_smtp_configs (application_id);

CREATE UNIQUE INDEX IF NOT EXISTS application_smtp_configs_one_active
  ON application_smtp_configs (application_id)
  WHERE active;

-- Move any existing nested settings.mail.smtp into the table so sending keeps
-- working for apps that already had SMTP configured. Those migrated rows are
-- marked active only when mail.provider is smtp.
INSERT INTO application_smtp_configs (
  application_id, name, host, port, username_ciphertext, password_ciphertext, tls, skip_verify, active
)
SELECT
  id,
  'default',
  settings->'mail'->'smtp'->>'host',
  COALESCE(NULLIF(settings->'mail'->'smtp'->>'port', '')::int, 587),
  COALESCE(settings->'mail'->'smtp'->>'username', ''),
  COALESCE(settings->'mail'->'smtp'->>'password', ''),
  COALESCE((settings->'mail'->'smtp'->>'tls')::boolean, FALSE),
  COALESCE((settings->'mail'->'smtp'->>'skip_verify')::boolean, FALSE),
  (settings->'mail'->>'provider' = 'smtp')
FROM applications
WHERE COALESCE(settings->'mail'->'smtp'->>'host', '') <> '';

UPDATE applications
SET settings = settings #- '{mail,smtp}'
WHERE settings #> '{mail,smtp}' IS NOT NULL;
