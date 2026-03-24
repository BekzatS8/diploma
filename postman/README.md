# BuhPro Postman Notes

## Источники истины
- `internal/http/router/router.go` (маршруты + RBAC).
- соответствующие `handler.go`/`dto.go` (payload/response shape).
- `DEMO.md` и `cmd/seed-demo/main.go` (demo users/passwords, flow).

## Перед запуском
1. Запусти backend локально (`http://localhost:8080`).
2. Убедись, что включены demo/dev flags для нужных сценариев:
   - `ENABLE_DEMO_SEED=true`
   - `ENABLE_DEV_PAYMENT_ENDPOINTS=true`
3. Импортируй environment `BuhPro Local` и collection `BuhPro MVP`.
4. Запусти папки в порядке из description collection.

## Примечания
- `Coach` вынесен отдельным логином, т.к. `coach/courses` требуют роль `coach|admin`.
- `Admin Lift Sanction` может возвращать conflict/not_found в зависимости от текущего состояния санкции.
- Для `Admin Create Assignment` нужен `executorUserId`; он автоматически проставляется после `Login Executor`/`Me (Executor)`.
