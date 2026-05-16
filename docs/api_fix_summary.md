# API Fix Summary

## Completed fixes

- Added full API audit report and endpoint inventory in `docs/api_audit_report.md`.
- Production router now disables debug error traces. Development/test behavior remains debug-friendly by default, while production panics return the standard sanitized error shape.
- `GET /api/v1/files/:id` is now JWT-protected and owner/admin-scoped instead of public.
- Swagger security/description for `GET /api/v1/files/{id}` now matches the protected route.
- Course material `upload_id` references are now checked: coach can use only own uploads, admin can use any upload.
- Basic request validation was tightened for profile avatar upload id, attachment upload/target/reorder ids, and chat message text.

## Commits

- `6b0ceb6` docs: add API audit report
- `aa58674` fix: disable debug error traces in production
- `09d98fd` fix: protect upload metadata with owner checks
- `1fbdcf8` docs: update upload metadata swagger security
- `6b5bbd2` fix: validate course material upload ownership
- `92e0103` fix: normalize basic request validation

## Remaining issues

- Generic `POST /api/v1/reviews` still needs target-specific authorization/existence checks.
- Attachments still need target ownership/visibility checks; current checks are mostly upload-owner based.
- Public `GET /api/v1/attachments` should be reviewed per target visibility rules.
- Metrics endpoint is public when enabled.
- Swagger is manual and still needs a broader review after future auth/validation changes.
- Many path params still rely on service/repository validation instead of consistent handler-level UUID binding.
- Response/status DTOs are still mixed between typed structs and `gin.H`.
- `go test`, `go vet`, `gofmt`, and build verification could not run because `go` is not available in PATH.

## Commands run

- `rg --files` - OK.
- `git status --short` - OK; untracked `.idea/` existed before changes and was not touched.
- `go test ./...` - failed to start: `go` command is not recognized.
- `go vet ./...` - failed to start: `go` command is not recognized.
- `git diff` checks before commits - OK.
- `git commit` for each completed logical step - OK.
- `swag init` - not run; project uses manual OpenAPI in `internal/http/swagger/swagger.go`, and the `swag` CLI was not verified.

## Manual checks needed

- Install or expose Go toolchain in PATH, then run `gofmt`, `go test ./...`, `go vet ./...`, and `go build ./cmd/api`.
- In Swagger/Postman, verify protected upload metadata:
  - unauthenticated `GET /api/v1/files/{id}` returns 401;
  - owner receives 200;
  - another non-admin user receives 403;
  - admin receives 200.
- Verify production-style error behavior with `APP_ENV=production`: 4xx responses keep standard error JSON and panics do not expose stack traces.
- Verify course material creation/update:
  - coach can attach own upload id;
  - coach cannot attach another user's upload id;
  - admin can attach any upload id.
- Regression-check attachments and generic reviews manually before exposing them beyond trusted clients.
