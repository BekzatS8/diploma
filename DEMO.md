# BuhPro MVP Demo Script (REST-only)

Этот файл — практичный скрипт для e2e демо backend MVP.

## 0) Подготовка окружения

```bash
cp .env.example .env
set -a; source .env; set +a
export ENABLE_DEV_PAYMENT_ENDPOINTS=true
export ENABLE_DEMO_SEED=true
```

```bash
make compose-up
make migrate-up
make seed-demo
make run
```

> `seed-demo` доступен только вне production (`APP_ENV != production`) и только при `ENABLE_DEMO_SEED=true`.

Базовые demo users (пароль по умолчанию `DemoPass123`, можно переопределить `DEMO_USER_PASSWORD`):
- `demo.client@buhpro.local`
- `demo.executor@buhpro.local`
- `demo.coach@buhpro.local`
- `demo.admin@buhpro.local`

---

## 1) Scenario: client order posting payment flow

### 1.1 Login as client
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo.client@buhpro.local","password":"DemoPass123"}'
```

### 1.2 Create order (draft)
```bash
curl -s -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Demo order","description":"Need bookkeeping support","category_slug":"tax","budget_amount":60000}'
```

Expected transition: `draft`.

### 1.3 Submit order
```bash
curl -s -X POST http://localhost:8080/api/v1/orders/my/$ORDER_ID/submit \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

Expected transition: `draft -> payment_pending`, created payment transaction (`object_type=order_posting`, `status=pending`).

### 1.4 Confirm payment as admin (dev endpoint)
```bash
curl -s -X POST http://localhost:8080/api/v1/dev/payments/$ORDER_TX_ID/confirm \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Expected transition: `payment_pending -> published`.

---

## 2) Scenario: executor response payment flow

### 2.1 Login as executor
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo.executor@buhpro.local","password":"DemoPass123"}'
```

### 2.2 Create response draft
```bash
curl -s -X POST http://localhost:8080/api/v1/orders/$ORDER_ID/responses \
  -H "Authorization: Bearer $EXECUTOR_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"cover_letter":"Can do this work","proposed_amount":55000}'
```

Expected transition: response `draft`.

### 2.3 Submit response
```bash
curl -s -X POST http://localhost:8080/api/v1/orders/$ORDER_ID/responses/my/$RESPONSE_ID/submit \
  -H "Authorization: Bearer $EXECUTOR_TOKEN"
```

Expected transition: `draft -> payment_pending`, payment transaction created (`object_type=response_submission`).

### 2.4 Confirm response payment as admin
```bash
curl -s -X POST http://localhost:8080/api/v1/dev/payments/$RESP_TX_ID/confirm \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Expected transition: response `payment_pending -> submitted` and visible for client in `GET /api/v1/client/orders/:id/responses`.

---

## 3) Scenario: selection → chat → completion → review

### 3.1 Select response as client
```bash
curl -s -X POST http://localhost:8080/api/v1/client/orders/$ORDER_ID/select-response/$RESPONSE_ID \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

Expected transitions:
- order: `published -> in_progress`
- selected response: `submitted -> accepted`
- other submitted responses: `-> rejected`
- chat created автоматически для order (если отсутствовал).

### 3.2 Verify chat list
```bash
curl -s http://localhost:8080/api/v1/my/chats \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

### 3.3 Client sends message
```bash
curl -s -X POST http://localhost:8080/api/v1/my/chats/$CHAT_ID/messages \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"text":"Hello, let us start"}'
```

### 3.4 Executor reads/sends back
```bash
curl -s http://localhost:8080/api/v1/my/chats/$CHAT_ID/messages \
  -H "Authorization: Bearer $EXECUTOR_TOKEN"

curl -s -X POST http://localhost:8080/api/v1/my/chats/$CHAT_ID/read \
  -H "Authorization: Bearer $EXECUTOR_TOKEN"
```

### 3.5 Complete order
```bash
curl -s -X POST http://localhost:8080/api/v1/client/orders/$ORDER_ID/complete \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

Expected transition: `in_progress -> completed`.

### 3.6 Create review
```bash
curl -s -X POST http://localhost:8080/api/v1/client/orders/$ORDER_ID/review \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"rating":2,"comment":"Need improvement"}'
```

---

## 4) Scenario: rating/sanction/course/notifications

### 4.1 Check rating and sanctions (executor)
```bash
curl -s http://localhost:8080/api/v1/executors/$EXECUTOR_ID/rating
curl -s http://localhost:8080/api/v1/my/sanctions -H "Authorization: Bearer $EXECUTOR_TOKEN"
```

### 4.2 Check course assignments
```bash
curl -s http://localhost:8080/api/v1/my/course-assignments \
  -H "Authorization: Bearer $EXECUTOR_TOKEN"
```

### 4.3 Mark assignment completed
```bash
curl -s -X POST http://localhost:8080/api/v1/my/course-assignments/$ASSIGNMENT_ID/mark-completed \
  -H "Authorization: Bearer $EXECUTOR_TOKEN"
```

### 4.4 Notifications read flow
```bash
curl -s http://localhost:8080/api/v1/my/notifications?unread_only=true \
  -H "Authorization: Bearer $EXECUTOR_TOKEN"

curl -s -X POST http://localhost:8080/api/v1/my/notifications/read-all \
  -H "Authorization: Bearer $EXECUTOR_TOKEN"
```

---

## Admin/debug read checklist

Admin должен иметь read доступ к:
- `GET /api/v1/admin/sanctions`
- `GET /api/v1/admin/course-assignments`
- `GET /api/v1/admin/notifications`
- `GET /api/v1/admin/chats`
- `GET /api/v1/client/orders/:id/responses` (роль admin разрешена).

---

## MVP limitations (demo reminder)

- Нет websocket/realtime chat push.
- Нет email/sms delivery.
- Нет attachments/media messages.
- Нет production-grade billing/webhooks.
