# API Audit Report

Date: 2026-05-16

## 1. Project structure summary

Backend is a Go/Gin REST API under `cmd/api` and `internal`. `internal/app/app.go` wires config, database, JWT, storage, payments, repositories, services, handlers and the Gin router. `internal/http/router/router.go` is the single route registry. Business modules live under `internal/modules/*` with mostly consistent `handler.go`, `service.go`, `repository.go`, `model.go`, and `dto.go` files. Persistence uses PostgreSQL via `pgxpool` and SQL migrations in `migrations/`. Swagger is implemented manually as an OpenAPI map in `internal/http/swagger/swagger.go`, not generated from handler annotations.

Main roles found: `client`, `executor`, `coach`, `admin`. No `employer`, `model`, or `agency` roles exist in the code or DB enum.

Important middleware:

- `RequireAuth` validates access JWT and injects `auth_user`.
- `OptionalAuth` silently ignores invalid tokens.
- `RequireRoles` checks the single JWT role.
- `DebugErrorMiddleware` wraps all 4xx/5xx responses and panics into a debug response with `error_trace` and `stack_trace`.

Current verification status: `go test ./...` and `go vet ./...` could not be executed because `go` is not available in PATH in this environment.

## 2. API inventory

| Method | Path | Handler | Auth | Role | Status | Problem |
|-------|------|---------|------|------|--------|---------|
| GET | `/healthz` | `system.Healthz` | Public | - | OK | Liveness only. |
| GET | `/readyz` | `system.Readyz` | Public | - | OK | DB readiness. |
| GET | `/metrics` | `metrics.Handler` | Public | - | NEEDS_AUTH | Metrics are public when enabled; acceptable only behind private network/proxy. |
| GET | `/swagger/doc.json` | `swagger.Spec` | Public | - | OK | Manual OpenAPI spec. |
| GET | `/swagger`, `/swagger/index.html` | `swagger.Register` | Public | - | OK | UI only. |
| GET | `/uploads/*` | `gin.Static` | Public | - | OK | Public file serving; safe only if uploaded files are intended public. |
| GET | `/api/v1/ping` | `system.Ping` | Public | - | OK | Basic versioned ping. |
| POST | `/api/v1/auth/register` | `auth.Register` | Public | - | NEEDS_VALIDATION | Allows client/coach only, but some role profile fields are weakly validated. |
| POST | `/api/v1/auth/login` | `auth.Login` | Public | - | OK | Uses DTO and generic invalid credentials. |
| POST | `/api/v1/auth/refresh` | `auth.Refresh` | Public | - | OK | Refresh token is stored hashed and rotated. |
| POST | `/api/v1/auth/logout` | `auth.Logout` | JWT | any | OK | Requires current user and matching refresh token. |
| GET | `/api/v1/auth/me` | `auth.Me` | JWT | any | OK | Loads user and role profile. |
| GET | `/api/v1/profile` | `profile.Get` | JWT | any | OK | Returns profile by current role. |
| PATCH | `/api/v1/profile` | `profile.Patch` | JWT | any | NEEDS_VALIDATION | Patch DTO has almost no binding constraints. |
| PATCH | `/api/v1/profile/avatar` | `profile.SetAvatar` | JWT | any | NEEDS_VALIDATION | `upload_id` lacks uuid binding; service checks ownership. |
| DELETE | `/api/v1/profile/avatar` | `profile.ClearAvatar` | JWT | any | OK | Owner-scoped by current user. |
| GET | `/api/v1/files/:id` | `uploads.GetByID` | Public | - | NEEDS_AUTH | Public metadata and URL lookup for any upload id. |
| POST | `/api/v1/files` | `uploads.Upload` | JWT | any | NEEDS_VALIDATION | File size/type checked, but multipart memory limit is 32 MB while service max is 50 MB. |
| DELETE | `/api/v1/files/:id` | `uploads.Delete` | JWT | owner/admin | OK | Owner/admin check in service. |
| GET | `/api/v1/my/files` | `uploads.ListMy` | JWT | any | OK | Owner-scoped. |
| GET | `/api/v1/my/wallet` | `wallets.MyWallet` | JWT | any | OK | Owner-scoped by current user. |
| GET | `/api/v1/admin/wallets/:userId` | `wallets.AdminGet` | JWT | admin | OK | Admin middleware and service check. |
| POST | `/api/v1/admin/wallets/:userId/credit` | `wallets.AdminCredit` | JWT | admin | NEEDS_VALIDATION | `userId` path uuid and amount validation are mostly service-level only. |
| GET | `/api/v1/attachments` | `attachments.List` | Public | - | NEEDS_AUTH | Lists attachments for arbitrary target id without target visibility check. |
| POST | `/api/v1/attachments` | `attachments.Attach` | JWT | upload owner/admin | NEEDS_ROLE_CHECK | Checks upload ownership, but not whether caller owns target entity. |
| PATCH | `/api/v1/attachments/reorder` | `attachments.Reorder` | JWT | upload owner/admin | NEEDS_ROLE_CHECK | Checks upload ownership only, not target ownership/group consistency. |
| DELETE | `/api/v1/attachments/:id` | `attachments.Delete` | JWT | upload owner/admin | NEEDS_ROLE_CHECK | Delete authorization is based on upload author, not attached target owner. |
| GET | `/api/v1/reviews` | `reviews.ListByTarget` | Public | - | OK | Public generic target reviews. |
| POST | `/api/v1/reviews` | `reviews.CreateEntity` | JWT | any | NEEDS_ROLE_CHECK | Can review any valid target UUID/type; no target existence or permission check. |
| GET | `/api/v1/ratings` | `reviews.GetRatingSummary` | Public | - | OK | Public rating summary. |
| POST | `/api/v1/leads/executor` | `leads.SubmitExecutor` | Public | - | NEEDS_VALIDATION | Multipart documents checked in service, but JSON path cannot provide documents and message text is mojibake. |
| GET | `/api/v1/admin/executor-leads` | `leads.List` | JWT | admin | OK | Admin group protected. |
| GET | `/api/v1/admin/executor-leads/:id` | `leads.GetByID` | JWT | admin | OK | Admin group protected. |
| PATCH | `/api/v1/admin/executor-leads/:id/status` | `leads.UpdateStatus` | JWT | admin | NEEDS_VALIDATION | Status enum checked in service, path id not uuid-bound. |
| POST | `/api/v1/admin/executor-leads/:id/approve` | `leads.Approve` | JWT | admin | NEEDS_VALIDATION | Ignores JSON bind error for optional body; acceptable but inconsistent. |
| POST | `/api/v1/admin/executor-leads/:id/reject` | `leads.Reject` | JWT | admin | OK | Reason required. |
| GET | `/api/v1/orders` | `orders.ListPublic` | Optional JWT | public | OK | Lists published only. |
| POST | `/api/v1/orders` | `orders.Create` | JWT | client | OK | Client-only in middleware and service. |
| GET | `/api/v1/orders/:id` | `orders.GetByID` | Optional JWT | public/owner/admin | OK | Published public; draft visible to owner/admin only. |
| GET | `/api/v1/orders/my` | `orders.ListMy` | JWT | client | OK | Owner-scoped. |
| GET | `/api/v1/orders/my/:id` | `orders.GetMyByID` | JWT | client | OK | Owner-scoped. |
| PATCH | `/api/v1/orders/my/:id` | `orders.UpdateMyByID` | JWT | client | NEEDS_VALIDATION | Patch DTO has weaker min/max validation than create DTO. |
| DELETE | `/api/v1/orders/my/:id` | `orders.DeleteMyByID` | JWT | client | OK | Owner/status scoped. |
| POST | `/api/v1/orders/my/:id/submit` | `orders.Submit` | JWT | client | OK | Wallet debit and status transition in transaction. |
| POST | `/api/v1/orders/my/:id/cancel` | `orders.Cancel` | JWT | client | OK | Owner/status scoped. |
| POST | `/api/v1/orders/:id/responses` | `responses.Create` | JWT | executor | OK | Executor-only; self-response blocked. |
| GET | `/api/v1/orders/:id/responses/my` | `responses.ListOrderMy` | JWT | executor | OK | Executor-scoped. |
| GET | `/api/v1/orders/:id/responses/my/:responseId` | `responses.GetOrderMyByID` | JWT | executor | OK | Executor/order-scoped. |
| PATCH | `/api/v1/orders/:id/responses/my/:responseId` | `responses.UpdateOrderMyByID` | JWT | executor | NEEDS_VALIDATION | DTO optional fields; checks only amount/currency. |
| DELETE | `/api/v1/orders/:id/responses/my/:responseId` | `responses.DeleteOrderMyByID` | JWT | executor | OK | Executor/status scoped. |
| POST | `/api/v1/orders/:id/responses/my/:responseId/submit` | `responses.Submit` | JWT | executor | OK | Requires cover letter and payment transition. |
| POST | `/api/v1/orders/:id/responses/my/:responseId/cancel` | `responses.Cancel` | JWT | executor | OK | Executor/status scoped. |
| GET | `/api/v1/my/responses` | `responses.ListMy` | JWT | executor/admin | OK | Admin gets all; executor gets own. |
| GET | `/api/v1/my/responses/:id` | `responses.GetMyByID` | JWT | executor/admin | OK | Admin any, executor own. |
| GET | `/api/v1/client/orders/:id/responses` | `responses.ListClientOrder` | JWT | client/admin | OK | Client owner check in service. |
| GET | `/api/v1/client/orders/:id/responses/:responseId` | `responses.GetClientOrderByID` | JWT | client/admin | OK | Client owner check in service. |
| POST | `/api/v1/dev/payments/:transactionId/confirm` | `devpayments.Confirm` | JWT | admin | OK | Admin-only and disabled in production/config by service. |
| POST | `/api/v1/dev/payments/:transactionId/fail` | `devpayments.Fail` | JWT | admin | OK | Admin-only and disabled in production/config by service. |
| POST | `/api/v1/client/orders/:id/select-response/:responseId` | `selection.SelectResponse` | JWT | client/admin | OK | Owner/admin check in service. |
| GET | `/api/v1/client/orders/:id/selection` | `selection.GetSelection` | JWT | client/admin | OK | Owner/admin check. |
| POST | `/api/v1/client/orders/:id/complete` | `selection.Complete` | JWT | client/admin | OK | Owner/admin check; releases wallet escrow. |
| POST | `/api/v1/client/orders/:id/reopen` | `selection.Reopen` | JWT | client/admin | OK | Owner/admin check; prevents reopen after review. |
| POST | `/api/v1/client/orders/:id/review` | `reviews.Create` | JWT | client/admin | OK | Owner/admin and completed-order checks. |
| GET | `/api/v1/client/orders/:id/review` | `reviews.GetByOrder` | JWT | client/admin | OK | Owner/admin check. |
| GET | `/api/v1/executors/:executorId/reviews` | `reviews.ListExecutor` | Public | - | OK | Public executor review list. |
| GET | `/api/v1/executors/:executorId/rating` | `ratings.GetRating` | Public | - | OK | Public rating. |
| GET | `/api/v1/my/sanctions` | `ratings.MySanctions` | JWT | executor/admin | OK | Executor own; admin returns all in service. |
| GET | `/api/v1/admin/sanctions` | `ratings.AdminSanctions` | JWT | admin | OK | Admin only. |
| GET | `/api/v1/admin/sanctions/:id` | `ratings.AdminSanctionByID` | JWT | admin | OK | Admin only. |
| POST | `/api/v1/admin/sanctions/:id/lift` | `ratings.Lift` | JWT | admin | OK | Admin only. |
| GET | `/api/v1/coach/courses` | `courses.ListCoachCourses` | JWT | coach/admin | OK | Coach owner/admin filtered. |
| POST | `/api/v1/coach/courses` | `courses.CreateCourse` | JWT | coach/admin | OK | Role checked in middleware and service. |
| GET | `/api/v1/coach/courses/:id` | `courses.GetCoachCourse` | JWT | coach/admin | OK | Coach ownership/admin check. |
| PATCH | `/api/v1/coach/courses/:id` | `courses.PatchCourse` | JWT | coach/admin | NEEDS_VALIDATION | Patch DTO lacks max length validation. |
| POST | `/api/v1/coach/courses/:id/publish` | `courses.PublishCourse` | JWT | coach/admin | OK | Owner/admin transition. |
| POST | `/api/v1/coach/courses/:id/archive` | `courses.ArchiveCourse` | JWT | coach/admin | OK | Owner/admin transition. |
| POST | `/api/v1/coach/courses/:id/materials` | `courses.CreateMaterial` | JWT | coach/admin | NEEDS_ROLE_CHECK | Course ownership checked, but `upload_id` ownership is not checked. |
| PATCH | `/api/v1/coach/courses/:id/materials/:materialId` | `courses.PatchMaterial` | JWT | coach/admin | NEEDS_ROLE_CHECK | Course ownership checked, but `upload_id` ownership is not checked. |
| DELETE | `/api/v1/coach/courses/:id/materials/:materialId` | `courses.DeleteMaterial` | JWT | coach/admin | OK | Course ownership checked. |
| GET | `/api/v1/courses` | `courses.ListCourses` | JWT | executor/coach/admin | OK | Executor forced to published courses. |
| GET | `/api/v1/courses/:id` | `courses.GetCourse` | JWT | executor/coach/admin | OK | Executor blocked from unpublished. |
| POST | `/api/v1/admin/course-assignments` | `courses.CreateAssignment` | JWT | admin | OK | Admin only; published course required. |
| GET | `/api/v1/admin/course-assignments` | `courses.ListAdminAssignments` | JWT | admin | OK | Admin only. |
| GET | `/api/v1/my/course-assignments` | `courses.ListMyAssignments` | JWT | executor | OK | Executor own assignments. |
| GET | `/api/v1/my/course-assignments/:id` | `courses.GetMyAssignment` | JWT | executor | OK | Executor own assignment. |
| POST | `/api/v1/my/course-assignments/:id/mark-completed` | `courses.MarkCompleted` | JWT | executor | OK | Executor own assignment. |
| GET | `/api/v1/my/notifications` | `notifications.ListMy` | JWT | any | OK | Owner-scoped. |
| GET | `/api/v1/my/notifications/:id` | `notifications.GetMyByID` | JWT | any | OK | Owner-scoped. |
| POST | `/api/v1/my/notifications/:id/read` | `notifications.MarkRead` | JWT | any | OK | Owner-scoped. |
| POST | `/api/v1/my/notifications/read-all` | `notifications.MarkAllRead` | JWT | any | OK | Owner-scoped. |
| GET | `/api/v1/admin/notifications` | `notifications.ListAdmin` | JWT | admin | OK | Admin only. |
| GET | `/api/v1/admin/notifications/:id` | `notifications.GetAdminByID` | JWT | admin | OK | Admin only. |
| GET | `/api/v1/my/chats` | `chats.ListMyChats` | JWT | any | OK | Participant-scoped. |
| GET | `/api/v1/my/chats/:id` | `chats.GetMyChatByID` | JWT | any | OK | Participant-scoped. |
| GET | `/api/v1/my/chats/:id/messages` | `chats.ListMyMessages` | JWT | any | OK | Participant-scoped. |
| POST | `/api/v1/my/chats/:id/messages` | `chats.SendMyMessage` | JWT | client/executor/coach | OK | Admin blocked from my-message send; repository should enforce participant on insert. |
| POST | `/api/v1/my/chats/:id/read` | `chats.MarkMyChatRead` | JWT | any | OK | Participant-scoped. |
| GET | `/api/v1/admin/chats` | `chats.ListAdminChats` | JWT | admin | OK | Admin only. |
| GET | `/api/v1/admin/chats/:id` | `chats.GetAdminChatByID` | JWT | admin | OK | Admin only. |
| GET | `/api/v1/admin/chats/:id/messages` | `chats.ListAdminMessages` | JWT | admin | OK | Admin only. |

No duplicate Gin routes were found in `internal/http/router/router.go`. All registered handlers exist. No unregistered exported handler methods were found; helper handler methods such as `handleErr` are intentionally not routed.

## 3. Critical problems

1. `DebugErrorMiddleware` leaks internals on every 4xx/5xx and panic.
   - Location: `internal/http/middleware/debug_error.go`.
   - Impact: production responses include `error_trace` and full `stack_trace`, including intercepted JSON bodies and panic values. This also changes the normal `response.JSONError` envelope into a different debug envelope, breaking API consistency.
   - Recommendation: enable debug body/stack only outside production or behind a config flag. In production, return the standard `response.ErrorResponse` shape and log details server-side.

2. Generic review creation lacks target authorization and target existence checks.
   - Location: `reviews.CreateEntity`, `reviews.Service.CreateEntity`.
   - Impact: any authenticated user can create ratings for arbitrary UUIDs and target types (`user`, `order`, `response`, `course`, etc.) without proving they interacted with or own the target.
   - Recommendation: introduce target-specific permission checks before insert, or restrict generic reviews to public/read-only use until rules are defined.

3. Attachment authorization is based on upload ownership, not target ownership.
   - Location: `attachments.Service.Attach/Delete/Reorder/ListByTarget`.
   - Impact: a user can attach their own upload to another user's order/response/review/chat/course target if they know the target UUID; public list exposes attachments by arbitrary target.
   - Recommendation: add target ownership/visibility checks per target type before attach/delete/reorder/list, or route attachments through target-specific modules.

4. Course material upload references are not ownership-checked.
   - Location: `courses.Service.AddMaterial/UpdateMaterial`.
   - Impact: coach/admin course ownership is checked, but a coach can reference any existing `upload_id` as course material if they know its UUID.
   - Recommendation: validate that non-admin actors own the upload or use an attachment-style ownership service.

5. Public file metadata endpoint can expose upload records.
   - Location: `GET /api/v1/files/:id`, `uploads.GetByID`.
   - Impact: anyone with an upload UUID can retrieve original filename, mime type, size, URL and file path.
   - Recommendation: require auth plus owner/admin, or explicitly separate public assets from private uploads.

## 4. Medium problems

1. Swagger is manual, not annotation-generated.
   - The project uses `internal/http/swagger/swagger.go` instead of handler comments/swag annotations. This can be acceptable, but it makes drift likely. The current Swagger appears broadly aligned with router paths, but route security and request schemas must be maintained manually.

2. Metrics endpoint is public when enabled.
   - `r.GET(deps.Config.Metrics.Path, metrics.Handler())` has no auth. If exposed outside a private network, it may leak operational metadata.

3. Request validation is inconsistent.
   - Stronger binding exists for create order/course/auth.
   - Weak or missing validation exists in profile patch, response create/update, attachment UUID arrays, path params, wallet credit, chat message text binding, and update DTO lengths.

4. Error responses are inconsistent because debug middleware rewrites errors.
   - Handlers call `response.JSONError`, but the middleware converts all errors into `DebugErrorResponse`.
   - Some handlers return raw `gin.H{"status": ...}` while others return typed DTOs.

5. Multipart upload limits are inconsistent.
   - Handler `maxMultipartMemory` is 32 MB, service `MaxUploadSize` is 50 MB. Large files under 50 MB can fail during multipart parsing before the service-level error is reached.

6. Executor lead response text has broken encoding.
   - `leads.SubmitExecutor` returns mojibake text instead of valid Russian.

7. Lead submission has two payload modes but Swagger documents only multipart.
   - JSON submission cannot include documents and therefore fails required-document service rules. This is confusing API surface.

8. Path parameter validation is mostly absent.
   - Invalid UUIDs are usually discovered by database casts or repository calls, which can become 500 in some paths instead of 400.

9. `OptionalAuth` silently ignores invalid bearer tokens.
   - This is acceptable for public feeds, but should be documented because `/orders/:id` behavior changes based on valid JWT.

10. Dev payment endpoints are protected by admin and disabled in production, but still present in Swagger without a config caveat in the security model.

## 5. Low priority problems

1. Naming is mostly consistent, but uses mixed path param names: `:id`, `:responseId`, `:transactionId`, `:userId`; JSON uses snake_case.
2. Pagination parsing is duplicated in several handlers and silently defaults on invalid values in some modules, while orders returns `400`.
3. Some response DTOs are `gin.H`, not typed structs (`status`, paginated chat/notification wrappers).
4. `middleware.Recovery` exists but is not registered; `DebugErrorMiddleware` currently handles panics instead.
5. Swagger includes tag `Admin Debug`, but no concrete admin debug endpoints.
6. Some SQL migrations include demo seed data and mojibake text; seed/demo data should be clearly separated for production deployments.
7. There is no WebSocket endpoint despite chat functionality; chat is REST-only.
8. No employer/model/agency role-based API exists. Existing terminology is client/executor/coach/admin.

## 6. Recommended fix plan

1. Fix production error handling first.
   - Gate `DebugErrorMiddleware` by environment/config and preserve the standard `response.JSONError` envelope in production.
   - Add tests for production vs development error envelopes.

2. Normalize route-level security for public/private assets.
   - Decide whether uploads and attachments are public resources.
   - If private by default, require JWT on `GET /files/:id` and add owner/admin checks.
   - Add target visibility checks for public attachment listing.

3. Add target authorization for generic attachments and reviews.
   - Implement a small target authorization service or module callbacks for order/response/chat/course/profile/review targets.
   - For generic reviews, either restrict allowed targets or enforce interaction/ownership rules.

4. Close course material upload ownership gap.
   - Inject uploads service into courses service or add repository check for upload author.
   - Allow admin to reference any upload, coach only own uploads.

5. Standardize validation.
   - Add `binding:"uuid"` to UUID DTO fields.
   - Validate path UUIDs centrally or in handlers.
   - Add min/max validation to update DTOs where create DTOs already have constraints.
   - Align multipart memory limit with service max upload size.

6. Standardize response/error DTOs.
   - Replace ad hoc `gin.H{"status": ...}` with typed status response helpers or shared DTOs.
   - Keep debug details in logs; expose stable error code/message/request_id to clients.

7. Update Swagger after behavior changes.
   - Keep manual OpenAPI in sync or migrate to generated annotations.
   - Ensure all protected endpoints have bearer security, role descriptions and accurate request schemas.

8. Add focused tests.
   - Middleware error envelope tests.
   - Auth/role route tests with Gin test context.
   - Service tests for IDOR-sensitive paths: attachments, generic reviews, files, course material uploads.

## 7. Proposed commits

1. `docs: add API audit report`
2. `fix: disable debug error traces in production responses`
3. `fix: protect upload metadata with owner checks`
4. `fix: enforce attachment target ownership`
5. `fix: restrict generic review targets with authorization checks`
6. `fix: validate course material upload ownership`
7. `fix: normalize request validation for uuid and patch DTOs`
8. `fix: align multipart upload limits and lead response text`
9. `fix: standardize API status responses`
10. `docs: update swagger spec for secured files and validation`
11. `test: add middleware and authorization regression tests`

## Verification notes

- `git status --short` before edits showed only untracked `.idea/`.
- `go test ./...` failed to start: `go` command is not recognized.
- `go vet ./...` failed to start: `go` command is not recognized.
- `swag init` was not run because this project uses a manual OpenAPI spec and the `swag` CLI has not been verified in PATH.
