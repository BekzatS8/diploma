# BuhPro Backend MVP Foundation

Backend foundation + auth/profile layer for BuhPro MVP.

## Реализовано

- Foundation: config, Gin router, middleware, health/ready/metrics, pgxpool, migrations.
- Auth module:
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/logout`
  - `GET /api/v1/auth/me`
- Profile module:
  - `GET /api/v1/profile`
  - `PATCH /api/v1/profile`
- Register создает `users` + role-profile (`client_profiles` / `executor_profiles` / `coach_profiles`) в одной транзакции.
- JWT access/refresh flow с хранением hash refresh token в `refresh_tokens`, поддержкой revoke/logout и rotation при refresh.
- Operational admin bootstrap через env.

## Что отложено

- orders/responses/reviews/courses/chat/notifications/payments business logic
- email verification / reset password / oauth

## Auth env

Критичные:
- `DB_URL`
- `JWT_ACCESS_SECRET`
- `JWT_REFRESH_SECRET`

Auth/bootstrap:
- `JWT_ISSUER`
- `JWT_ACCESS_TTL`
- `JWT_REFRESH_TTL`
- `BOOTSTRAP_ADMIN_ENABLED`
- `BOOTSTRAP_ADMIN_EMAIL`
- `BOOTSTRAP_ADMIN_PASSWORD`

## Makefile

- `make run`
- `make test`
- `make build`
- `make fmt`
- `make tidy`
- `make migrate-up`
- `make migrate-down`
- `make compose-up`
- `make compose-down`
- `make bootstrap-admin`

## Запуск (Linux/macOS)

```bash
cp .env.example .env
set -a; source .env; set +a
docker compose up -d postgres
make migrate-up
make run
```

## Запуск (Windows PowerShell)

```powershell
Copy-Item .env.example .env
Get-Content .env | ForEach-Object {
  if ($_ -and -not $_.StartsWith('#')) {
    $name, $value = $_ -split '=', 2
    [Environment]::SetEnvironmentVariable($name, $value, 'Process')
  }
}
docker compose up -d postgres
make migrate-up
make run
```

## Docker compose

```bash
cp .env.example .env
docker compose up --build -d
```

Prometheus optional profile:

```bash
docker compose --profile observability up -d
```

## Auth API examples

Register (role: client/executor/coach):

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"StrongPass1","role":"executor","profile_name":"John Doe"}'
```

Login:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"StrongPass1"}'
```

Refresh:

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<token>"}'
```

Me:

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H 'Authorization: Bearer <access_token>'
```

Profile patch (role-specific fields):

```bash
curl -X PATCH http://localhost:8080/api/v1/profile \
  -H 'Authorization: Bearer <access_token>' \
  -H 'Content-Type: application/json' \
  -d '{"profile_name":"Jane Doe","bio":"Senior accountant"}'
```

## Как создать admin

В dev можно включить bootstrap:

```env
BOOTSTRAP_ADMIN_ENABLED=true
BOOTSTRAP_ADMIN_EMAIL=admin@buhpro.local
BOOTSTRAP_ADMIN_PASSWORD=Admin12345
```

При старте приложения admin будет создан (или пропущен, если email уже есть).

Либо вручную:

```bash
BOOTSTRAP_ADMIN_EMAIL=admin@buhpro.local BOOTSTRAP_ADMIN_PASSWORD=Admin12345 make bootstrap-admin
```

## Health endpoints

- `GET /healthz`
- `GET /readyz`
- `GET /metrics`
- `GET /api/v1/ping`
