# BuhPro Backend MVP Foundation

Backend foundation + auth/profile + orders layer for BuhPro MVP.

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
- Orders module:
  - `POST /api/v1/orders` (client only, create draft)
  - `GET /api/v1/orders/my`
  - `GET /api/v1/orders/my/:id`
  - `PATCH /api/v1/orders/my/:id` (draft only)
  - `DELETE /api/v1/orders/my/:id` (soft delete draft/cancelled)
  - `POST /api/v1/orders/my/:id/submit` (draft -> payment_pending + payment transaction)
  - `POST /api/v1/orders/my/:id/cancel` (allowed statuses only)
  - `GET /api/v1/orders` (public published orders feed)
  - `GET /api/v1/orders/:id` (public for published; owner/admin for non-published)
- Selection/Lifecycle module:
  - `POST /api/v1/client/orders/:id/select-response/:responseId`
  - `GET /api/v1/client/orders/:id/selection`
  - `POST /api/v1/client/orders/:id/complete`
  - `POST /api/v1/client/orders/:id/reopen` (MVP simplification)
- Reviews module:
  - `POST /api/v1/client/orders/:id/review`
  - `GET /api/v1/client/orders/:id/review`
  - `GET /api/v1/executors/:executorId/reviews`
- Responses module:
  - `POST /api/v1/orders/:id/responses` (executor creates response draft)
  - `GET /api/v1/orders/:id/responses/my`
  - `GET /api/v1/orders/:id/responses/my/:responseId`
  - `PATCH /api/v1/orders/:id/responses/my/:responseId`
  - `DELETE /api/v1/orders/:id/responses/my/:responseId`
  - `POST /api/v1/orders/:id/responses/my/:responseId/submit`
  - `POST /api/v1/orders/:id/responses/my/:responseId/cancel`
  - `GET /api/v1/my/responses`
  - `GET /api/v1/my/responses/:id`
  - `GET /api/v1/client/orders/:id/responses`
  - `GET /api/v1/client/orders/:id/responses/:responseId`
- Register создает `users` + role-profile (`client_profiles` / `executor_profiles` / `coach_profiles`) в одной транзакции.
- JWT access/refresh flow с хранением hash refresh token в `refresh_tokens`, поддержкой revoke/logout и rotation при refresh.
- Operational admin bootstrap через env.

## Orders/Responses access model

- Создание/редактирование/submit/cancel/delete заказа: только роль `client`.
- Создание/редактирование/submit/cancel/delete отклика: только роль `executor`.
- `client` не может создавать responses, `coach` не может создавать responses.
- `admin` может читать заказы/отклики, но не участвует в user-flow мутаций.
- Клиент видит отклики только по своим заказам и только подтвержденные к показу (`submitted` + `is_paid=true`).
- Исполнитель видит только свои отклики.

## Orders flow (MVP)

1. `POST /api/v1/orders` создает заказ в `draft`.
2. `PATCH /api/v1/orders/my/:id` редактирует только `draft`.
3. `POST /api/v1/orders/my/:id/submit`:
   - валидирует обязательные поля,
   - переводит заказ в `payment_pending`,
   - пишет `order_status_history`,
   - создает `payment_transactions` (`object_type=order_posting`, `status=pending`),
   - возвращает foundation payment payload (`checkout_url`, provider reference).
4. `POST /api/v1/orders/my/:id/cancel` переводит в `cancelled` (по допустимым переходам) и пишет `order_status_history`.
5. Публичный список `GET /api/v1/orders` возвращает только `published` с фильтрами `category`, `budget_min`, `budget_max`, `q` + пагинацией.

> Примечание: в текущей SQL-схеме у `orders` нет `region/work_format/deadline_at`, поэтому эти поля и связанные фильтры не реализованы на этом этапе.

## Responses flow (MVP)

1. `POST /api/v1/orders/:id/responses` создает отклик исполнителя в статусе `draft`.
2. `PATCH /api/v1/orders/:id/responses/my/:responseId` доступен только для `draft`.
3. `POST /api/v1/orders/:id/responses/my/:responseId/submit` переводит `draft -> payment_pending`, создает foundation payment transaction (`object_type=response_submission`, `status=pending`) и пишет `response_status_history`.
4. `POST /api/v1/orders/:id/responses/my/:responseId/cancel` переводит отклик в `cancelled` (разрешено из `draft` и `payment_pending`).
5. Dev confirm payment переводит `payment_pending -> submitted` и выставляет `is_paid=true`.
6. Клиент в `GET /api/v1/client/orders/:id/responses` видит только `submitted` + `is_paid=true` (draft/payment pending/cancelled скрыты).
7. Flow выбора исполнителя сознательно не реализован на этом этапе.

## Selection flow (MVP)

- Owner-client (или admin) выбирает response по `POST /api/v1/client/orders/:id/select-response/:responseId`.
- Preconditions: order=`published`, response принадлежит order, response=`submitted`, `is_paid=true`, у order еще нет выбранного response.
- Транзакционно: `orders.selected_response_id`, `orders.selected_executor_id`, `orders.status: published -> in_progress`, выбранный response `submitted -> accepted`, остальные `submitted -> rejected`, плюс history записи.
- `GET /api/v1/client/orders/:id/selection` возвращает текущий selection snapshot.

## Order lifecycle completion (MVP)

- `POST /api/v1/client/orders/:id/complete`: `in_progress -> completed` (только если есть selected response).
- `POST /api/v1/client/orders/:id/reopen`: `completed -> in_progress` (сознательное MVP/dev business simplification).

## Reviews foundation (MVP)

- Review создается только для `completed` order и только один review на order.
- Executor для review берется из `orders.selected_response_id -> responses.executor_id`, не из client payload.
- Публичный endpoint `GET /api/v1/executors/:executorId/reviews` с пагинацией и сортировкой newest-first.

## Dev payment finalization (MVP/demo)

- Dev endpoints: 
  - `POST /api/v1/dev/payments/:transactionId/confirm`
  - `POST /api/v1/dev/payments/:transactionId/fail`
- Ограничения доступа:
  - только `admin` + authenticated
  - endpoint работает только при `APP_ENV != production` и `ENABLE_DEV_PAYMENT_ENDPOINTS=true`
- Confirm rules:
  - payment `pending -> succeeded` (+ `paid_at`)
  - `order_posting`: `order payment_pending -> published` + `order_status_history`
  - `response_submission`: `response payment_pending -> submitted`, `is_paid=true` + `response_status_history`
- Fail rules:
  - payment `pending -> failed`
  - `order_posting`: `order payment_pending -> draft`
  - `response_submission`: `response payment_pending -> draft`

## Courses + sanction follow-up foundation (MVP)

- Coach/Admin courses:
  - `POST /api/v1/coach/courses`
  - `PATCH /api/v1/coach/courses/:id`
  - `GET /api/v1/coach/courses/:id`
  - `GET /api/v1/coach/courses`
  - `POST /api/v1/coach/courses/:id/publish` (`draft -> published`)
  - `POST /api/v1/coach/courses/:id/archive` (`published -> archived`)
- Course materials:
  - `POST /api/v1/coach/courses/:id/materials`
  - `PATCH /api/v1/coach/courses/:id/materials/:materialId`
  - `DELETE /api/v1/coach/courses/:id/materials/:materialId`
  - supported material `type`: `video|pdf|link|text`
  - `video/pdf/link` expect `url`; `text` expects `content`
- Authenticated catalog (no public catalog):
  - `GET /api/v1/courses`
  - `GET /api/v1/courses/:id`
  - roles: `executor|coach|admin` only
- Course assignments:
  - admin:
    - `POST /api/v1/admin/course-assignments`
    - `GET /api/v1/admin/course-assignments` (filters: `executor_id`, `course_id`, `status`, `source`, plus pagination)
  - executor:
    - `GET /api/v1/my/course-assignments`
    - `GET /api/v1/my/course-assignments/:id`
    - `POST /api/v1/my/course-assignments/:id/mark-completed`
- Assignment source values:
  - `manual_admin`
  - `sanction_low_rating_first`
  - `sanction_low_rating_repeat`
- Low-rating sanctions integration:
  - sanctions still created as before
  - metadata stores machine-readable follow-up intent
  - optional auto-assignment when enabled by env flags (see below)
  - assignment is created only for published default course and deduplicated by active `(course_id, executor_id)` assignment.

## Что отложено

- chat/notifications business logic
- реальные payment callbacks/webhooks
- automatic publish after successful payment
- automatic response submit after payment callback
- admin moderation flows beyond read access
- email verification / reset password / oauth
- LMS-advanced features: quizzes/tests, certificates, recommendation engine, realtime progress updates

## Auth + Orders env

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

Payments/orders foundation:
- `PAYMENTS_PROVIDER` (default: `mock`)
- `ORDER_POSTING_FEE` (default: `1000`)
- `RESPONSE_SUBMISSION_FEE` (default: `500`)
- `DEFAULT_CURRENCY` (default: `KZT`)
- `ENABLE_DEV_PAYMENT_ENDPOINTS` (default: `false`)
- `AUTO_ASSIGN_COURSE_ON_LOW_RATING` (default: `false`)
- `DEFAULT_LOW_RATING_COURSE_ID` (optional UUID course id)

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

Create order draft (client):

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H 'Authorization: Bearer <access_token>' \
  -H 'Content-Type: application/json' \
  -d '{"title":"Нужен бухгалтер для ИП","description":"Подготовка налоговой отчетности","category_slug":"tax","budget_amount":50000}'
```

Submit order:

```bash
curl -X POST http://localhost:8080/api/v1/orders/my/<order_id>/submit \
  -H 'Authorization: Bearer <access_token>'
```

Public feed:

```bash
curl 'http://localhost:8080/api/v1/orders?page=1&page_size=20&category=tax&q=отчетность'
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
