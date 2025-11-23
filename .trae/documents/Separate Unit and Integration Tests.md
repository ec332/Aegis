## Goals
- Clearly separate unit tests from integration tests
- Make unit tests run by default; run integration tests only when explicitly enabled
- Update CI to run unit tests on every push and integration tests in a dedicated job with Postgres

## Current Tests
- Unit tests: `internal/model/transaction_test.go`, `internal/service/transaction_test.go`, `internal/http/handlers_test.go`
- Integration-like test: `internal/store/postgres/repo_test.go` gated by `PG_TEST` env

## Separation Approach
- Unit tests: keep as-is and always run with `go test ./...`
- Integration tests:
  - Move Postgres repo test to `internal/store/postgres/repo_integration_test.go`
  - Add build tag `//go:build integration` (and `// +build integration`) at top of integration tests
  - Remove env gating; rely on `-tags integration` to enable them
  - Provide DSN via env (`PG_DSN`) or compose envs (`APP_DB_*`)

## CI Updates
- Modify existing workflow to:
  - Job 1 (unit): setup Go 1.22, run `go build` and `go test ./...`
  - Job 2 (integration): add Postgres service, apply schema from `transaction-service/sql/create.sql`, run `go test -tags integration ./internal/store/postgres`

## Local Developer Commands
- Unit: `go test ./...`
- Integration (assuming Postgres up and schema applied):
  - Export `PG_DSN` or `APP_DB_*`
  - Run `go test -tags integration ./internal/store/postgres -v`
- Optional: add `docker-compose.test.yml` with Postgres and schema for convenience (can be added later)

## Changes To Perform
1. Rename `internal/store/postgres/repo_test.go` → `repo_integration_test.go`
2. Add build tags at top of the file; remove `PG_TEST` gating
3. Update GitHub Actions workflow to add an integration job with Postgres service and schema apply
4. Update README with instructions to run unit vs integration tests

## Acceptance Criteria
- `go test ./...` runs fast unit suite without external dependencies
- `go test -tags integration ./internal/store/postgres` runs against a real Postgres with schema applied
- CI shows two jobs: unit and integration; both pass

## Notes
- We will not introduce new dependencies (e.g., testcontainers-go) right now; using Actions services keeps things lean
- If you prefer testcontainers for local dev and CI parity, we can add it after this separation