# BuhPro Backend MVP Foundation

Backend foundation + auth/profile + orders layer for BuhPro MVP.

## Архитектура (кратко)

- HTTP: Gin + middleware (`request_id`, logging, recovery, auth).
- Service layer: бизнес-правила и access checks.
- Repository layer: SQL-first (`pgxpool`), без смешивания SQL в handlers.
- DB: PostgreSQL + migration files в `migrations/`.
- Infra: dev payment finalization endpoints (admin + env-guarded), demo seed CLI (env-guarded).

## API conventions

- Error format: `{\"error\":{\"code\",\"message\",\"request_id\"}}`.
- Pagination params: `page`, `page_size` (default page=1, page_size=20, capped in services).
- List endpoints usually return envelope: `items`, `page`, `page_size`, `total`.
- Access model:
  - owner/participant-scoped endpoints under `my/*`,
  - admin debug/read endpoints under `admin/*`.

## Naming consistency snapshot

- Payment object types: `order_posting`, `response_submission`.
- Response lifecycle statuses: `draft`, `payment_pending`, `submitted`, `accepted`, `rejected`, `cancelled`.
- Notification types use machine-readable snake_case (`order_published`, `response_selected`, `chat_message_received`, ...).
- Sanction reasons/sources: `low_rating_first`, `low_rating_repeat`; assignment sources `manual_admin`, `sanction_low_rating_first`, `sanction_low_rating_repeat`.

## Роли

- `client`: заказы, выбор исполнителя, completion, review.
- `executor`: responses, chat participant, assignments, sanctions/notifications read.
- `coach`: courses/materials management.
- `admin`: debug/read across domains + dev payment endpoints + bootstrap/seed tooling.

## Реализовано

- Foundation: config, Gin router, middleware, health/ready/metrics, pgxpool, migrations.
- MVP модули: auth/profile, orders, responses, dev-payments, selection+lifecycle, reviews, rating+sanctions, courses+assignments, notifications, REST chats.
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

## Notifications foundation (MVP)

- In-app notifications storage via `notifications` table (`channel=in_app` for current MVP delivery).
- User endpoints:
  - `GET /api/v1/my/notifications` (filters: `status`, `type`, `unread_only`, pagination)
  - `GET /api/v1/my/notifications/:id`
  - `POST /api/v1/my/notifications/:id/read`
  - `POST /api/v1/my/notifications/read-all`
- Admin read/debug endpoints:
  - `GET /api/v1/admin/notifications` (filters: `user_id`, `type`, `status`, `channel`, pagination)
  - `GET /api/v1/admin/notifications/:id`
- `mark-read` / `mark-all-read` are idempotent and set `read_at` + `status=read` (with `sent_at` backfill if needed).
- Notification events emitted in current flows:
  - `order_published` (after dev payment confirm of order posting)
  - `response_submitted` (after dev payment confirm of response submission)
  - `response_selected` (after client selects executor response)
  - `order_completed` (after client completes in-progress order)
  - `review_created` (after client leaves review)
  - `sanction_created` (when low-rating sanction is created during review recalculation)
  - `course_assigned` (manual admin assignment + auto assignment from low-rating sanction)
  - `course_completed` (admin-facing event when executor marks assignment completed and assignment has `assigned_by`)
- Payloads are machine-readable and include only related identifiers (`order_id`, `response_id`, `review_id`, `sanction_id`, `course_id`, `course_assignment_id`, `executor_id`) plus limited contextual fields (for example sanction `reason`).
- Realtime/ws, email/sms delivery, queues/workers and notification preferences are intentionally not implemented at this stage.

## Chat foundation (MVP, REST only)

- Чат создается автоматически в selection flow при `published -> in_progress` после выбора response:
  - один чат на один `order` (`chats.order_id` unique),
  - если чат уже есть, создание идемпотентно,
  - участники: owner-client заказа + selected executor.
- Доступ:
  - `my/*` endpoints только для участников чата,
  - `admin/*` endpoints только для чтения/debug.
- Endpoints:
  - user:
    - `GET /api/v1/my/chats`
    - `GET /api/v1/my/chats/:id`
    - `GET /api/v1/my/chats/:id/messages`
    - `POST /api/v1/my/chats/:id/messages`
    - `POST /api/v1/my/chats/:id/read`
  - admin:
    - `GET /api/v1/admin/chats`
    - `GET /api/v1/admin/chats/:id`
    - `GET /api/v1/admin/chats/:id/messages`
- Messages list ordering: `oldest-first` (`created_at ASC`) with pagination.
- `mark read` обновляет `chat_participants.last_read_at` для текущего участника (idempotent participant-level read state, без per-message read receipts).
- После отправки сообщения создается in-app notification второму участнику (`type=chat_message_received`, payload: `chat_id`, `order_id`, `message_id`).
- На этом этапе не реализованы websocket/realtime, typing indicators, attachments/files, media messages, edit/delete workflow, chat moderation, history search, browser push.

## Что отложено

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
- `ENABLE_DEMO_SEED` (default: `false`, required for `seed-demo`)
- `DEMO_USER_PASSWORD` (optional, default `DemoPass123`)
- `DEMO_INCLUDE_SANCTION` (optional `true|false`, default false)

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
- `make seed-demo`

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

## Demo readiness

- Пошаговый сценарий e2e demo вынесен в `DEMO.md` (4 сценария с endpoint sequence, payload и expected transitions).
- Рекомендуемый быстрый dev/demo цикл:
  1. `make compose-up`
  2. `make migrate-up`
  3. `ENABLE_DEMO_SEED=true make seed-demo`
  4. `ENABLE_DEV_PAYMENT_ENDPOINTS=true make run`

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
