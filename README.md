# Authx

Authx is a standalone auth service other applications call over HTTP. Signup and login share one application-bound endpoint: `POST /api/v1/auth`. The consuming app’s config chooses `password`, `otp`, or `magic_link`.

## Tech stack

- Go
- Postgres
- JWT access tokens and hashed refresh tokens

## Run locally

1. Create a Postgres database named `authx`.
2. Copy config and start the server:

```bash
cp config.example.yaml config.yaml
go run ./cmd/authx
```

Migrations run on startup. With `app.env` set to `development` or `dev`, unmapped 500s are logged with caller and stack to stdout and included in the JSON body; other environments return a generic `internal_error`. Authx does not create an application for you — create one via the admin API (see [Create an application](#create-an-application)) and use the returned `client_id` and `client_secret` on auth routes.

Health check: `GET http://localhost:8080/api/health`

## API documentation

OpenAPI 3 spec: [`docs/openapi.yaml`](docs/openapi.yaml). With the server running, open [http://localhost:8080/api/docs](http://localhost:8080/api/docs) (Swagger UI) or fetch the YAML at `/api/docs/openapi.yaml`. `/swagger` redirects to the UI.

## Docker

```bash
docker build -t authx .
docker run --rm -p 8080:8080 \
  -v "$PWD/config.yaml:/app/config.yaml" \
  authx
```

Point `database.host` in `config.yaml` at a reachable Postgres instance (use `host.docker.internal` from Docker Desktop).

## Authenticate (password)

Signup or login with an application created via the admin API (`auth_method: password`):

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth \
  -H 'Content-Type: application/json' \
  -H "X-Authx-Client-Id: $CLIENT_ID" \
  -H "X-Authx-Client-Secret: $CLIENT_SECRET" \
  -d '{"email":"user@example.com","password":"ValidPass1","first_name":"Ada"}'
```

Success returns `status: authenticated` plus `access_token`, `refresh_token`, and `user`.

Current user:

```bash
curl -sS http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "X-Authx-Client-Id: $CLIENT_ID" \
  -H "X-Authx-Client-Secret: $CLIENT_SECRET"
```

Update your profile (`first_name`, `last_name`):

```bash
curl -sS -X PATCH http://localhost:8080/api/v1/me \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "X-Authx-Client-Id: $CLIENT_ID" \
  -H "X-Authx-Client-Secret: $CLIENT_SECRET" \
  -d '{"first_name":"Grace","last_name":"Hopper"}'
```

Refresh and logout:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -H "X-Authx-Client-Id: $CLIENT_ID" \
  -H "X-Authx-Client-Secret: $CLIENT_SECRET" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"

curl -sS -X POST http://localhost:8080/api/v1/auth/logout \
  -H 'Content-Type: application/json' \
  -H "X-Authx-Client-Id: $CLIENT_ID" \
  -H "X-Authx-Client-Secret: $CLIENT_SECRET" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"
```

## OTP and magic link

Create an application with `auth_method` set to `otp` or `magic_link` (admin API below). The client still calls the same `POST /api/v1/auth`.

Request a challenge (no `code` / `token`):

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth \
  -H 'Content-Type: application/json' \
  -H "X-Authx-Client-Id: $CLIENT_ID" \
  -H "X-Authx-Client-Secret: $CLIENT_SECRET" \
  -d '{"email":"user@example.com"}'
```

Response: `{"status":"challenge_sent","challenge_type":"otp"|"magic_link","expires_in":...}`. Emails are sent with the application's mail settings (`sendgrid` or `smtp`). If `mail.provider` is `log` or unset, the OTP or link is printed in server logs (`mail.driver` in `config.yaml` is only this process-wide fallback).

Complete OTP:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth \
  -H 'Content-Type: application/json' \
  -H "X-Authx-Client-Id: $CLIENT_ID" \
  -H "X-Authx-Client-Secret: $CLIENT_SECRET" \
  -d '{"email":"user@example.com","code":"123456"}'
```

Complete magic link (the consuming app hosts `/auth/callback` and posts the token back):

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth \
  -H 'Content-Type: application/json' \
  -H "X-Authx-Client-Id: $CLIENT_ID" \
  -H "X-Authx-Client-Secret: $CLIENT_SECRET" \
  -d '{"email":"user@example.com","token":"OPAQUE_TOKEN_FROM_LINK"}'
```

Wrong fields for the configured method (for example `password` on an OTP app) return `400` with `invalid_payload_for_auth_method`.

## Create an application

Create an application via the admin API before calling auth routes. Use `auth_method` `password`, `otp`, or `magic_link`.

```bash
curl -sS -X POST http://localhost:8080/api/v1/admin/applications \
  -H 'Content-Type: application/json' \
  -H 'X-Authx-Admin-Key: dev-admin-key' \
  -d '{"name":"Acme","settings":{"auth_method":"password","signup_enabled":true}}'
```

The response includes `client_id` and a one-time `client_secret`. Export them and send them as `X-Authx-Client-Id` and `X-Authx-Client-Secret` on auth routes:

```bash
export CLIENT_ID=...
export CLIENT_SECRET=...
```

Update OTP length, expiry, and mail credentials (SendGrid or SMTP) on an existing application. Stored SendGrid API keys are returned as `********`; posting that masked value leaves the secret unchanged. SMTP username and password are never returned. Send the SMTP password as standard base64; Authx decodes it and encrypts username and password at rest (`encryption.key` in config).

An application can have multiple SMTP configurations. Create them with `POST /api/v1/admin/applications/{id}/smtp-configs` (new configs are **inactive**). Activate one with `POST /api/v1/admin/applications/{id}/smtp-configs/{sid}/activate` — that call deactivates any other active config for the application. Sending mail uses the active configuration; if none is active, auth returns `mail_not_configured`. Nested `settings.mail.smtp` on application create still seeds one inactive default config.

```bash
curl -sS -X PATCH http://localhost:8080/api/v1/admin/applications/$APPLICATION_ID \
  -H 'Content-Type: application/json' \
  -H 'X-Authx-Admin-Key: dev-admin-key' \
  -d '{"settings":{"otp":{"length":8,"ttl_seconds":120},"mail":{"provider":"sendgrid","from_email":"noreply@acme.example","from_name":"Acme","sendgrid":{"api_key":"SG.xxxxxxxx"}}}}'
```

Admins can also update any user's name:

```bash
curl -sS -X PATCH http://localhost:8080/api/v1/admin/users/$USER_ID \
  -H 'Content-Type: application/json' \
  -H 'X-Authx-Admin-Key: dev-admin-key' \
  -d '{"first_name":"Grace","last_name":"Hopper"}'
```

OpenAPI: [`docs/openapi.yaml`](docs/openapi.yaml). Postman: [`docs/Authx.postman_collection.json`](docs/Authx.postman_collection.json).
