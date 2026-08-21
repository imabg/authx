UPDATE applications a
SET settings = jsonb_set(
  COALESCE(a.settings, '{}'::jsonb),
  '{mail,smtp}',
  jsonb_build_object(
    'host', s.host,
    'port', s.port,
    'username', s.username_ciphertext,
    'password', s.password_ciphertext,
    'tls', s.tls,
    'skip_verify', s.skip_verify
  )
)
FROM application_smtp_configs s
WHERE s.application_id = a.id
  AND s.active;

DROP INDEX IF EXISTS application_smtp_configs_one_active;
DROP INDEX IF EXISTS idx_application_smtp_configs_application_id;
DROP TABLE IF EXISTS application_smtp_configs;
