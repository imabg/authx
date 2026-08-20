-- Mailer credentials live in applications.settings JSONB:
--   mail.provider: log | sendgrid | smtp
--   mail.from_email / mail.from_name
--   mail.sendgrid.api_key
--   mail.smtp.{host,port,username,password,tls,skip_verify}
-- Per-application credentials win over the process-wide mail.driver fallback.
UPDATE applications
SET settings = COALESCE(settings, '{}'::jsonb) || '{"mail":{"provider":"log"}}'::jsonb
WHERE NOT (settings ? 'mail');
