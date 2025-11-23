## Goals
- Replace the C++ Drogon service with a Go service providing identical REST endpoints and behavior
- Keep PostgreSQL schema and data intact; manage DB via Go migrations
- Preserve Docker-based local deployment and integration test coverage
- Integrate Solidity contract usage via Go bindings; leave on-chain logic in Solidity

## Current Architecture Snapshot
- Web API: C++ service using Drogon, entrypoint `transaction-service/main.cc:2-6`
- HTTP routes: `TransactionController.h:18-26` defines CRUD on `/transactions`
- Controllers: request handling and JSON responses in `TransactionController.cc:10-168`
- Business logic: validation and defaults in `services/TransactionService.cc:19-57`
- Persistence: ORM CRUD via CoroMapper in `repositories/TransactionRepository.cc:7-29`
- Data model: generated model with JSON validators `models/Transactions.h:92-105`
- DB schema: `transaction-service/sql/create.sql:4-18`
- Config: server/db settings `transaction-service/config.json`
- Tests: HTTP-level CRUD and list `transaction-service/test/test_main.cc:12-46,48-63`
- Deployment: `transaction-service/Dockerfile` and root `docker-compose.yml`

## Target Go Architecture
- Module: `go.mod` with Go 1.22
- Binaries: `cmd/transaction-service/main.go` starts HTTP server and wires dependencies
- Packages:
  - `internal/http`: handlers for `/transactions`
  - `internal/service`: business logic and validation
  - `internal/store/postgres`: repository layer using `pgx` (optionally `sqlc`)
  - `internal/model`: transaction structs and JSON validation
  - `internal/config`: config loading (env + file)
  - `internal/log`: structured logging
- Frameworks/Libraries:
  - HTTP router: `chi` (lightweight, idiomatic) or `gin` (popular); choose `chi`
  - DB driver: `pgx` + `pgxpool`; optional `sqlc` for typed queries
  - Config: `viper` (env + file); Logging: `zap`
  - Testing: `stretchr/testify`, `net/http/httptest`

## Endpoint Parity
- Implement handlers matching C++ routes and semantics:
  - `GET /transactions` → list all; 200 with JSON array
  - `GET /transactions/{id}` → 200 with JSON or 404 on missing
  - `POST /transactions` → 201 with created entity; 400 on validation errors
  - `PUT /transactions/{id}` → 200 with updated entity; 400 on validation errors; preserve/ensure `created_at`
  - `DELETE /transactions/{id}` → 204 if deleted, 404 if missing
- Mirror error shapes and fields from `TransactionController.cc`

## Data Model and Validation
- Go struct `Transaction` with fields: `id`, `user_id`, `market_id`, `option_id`, `transaction_type`, `number_of_shares` (decimal), `price_per_share` (decimal), `created_at`
- JSON validation functions replicating `models/Transactions.h:92-105`
- UUID handling with `google/uuid`; decimals with `shopspring/decimal`
- Set `created_at` default on create; preserve on update (as in `TransactionService.cc:41-57`)

## Persistence Layer
- Use `pgxpool` for connection pooling
- SQL queries equivalent to Drogon ORM operations:
  - `findAll`: `SELECT * FROM transactions ORDER BY created_at DESC`
  - `findById`: `SELECT * FROM transactions WHERE id=$1`
  - `insert`: explicit columns matching schema; return `RETURNING *`
  - `update`: update mutable fields by `id`; return `RETURNING *`
  - `deleteById`: `DELETE FROM transactions WHERE id=$1`
- Optional: adopt `sqlc` generating typed methods from `sql/create.sql`

## Configuration
- Replace `config.json` with env-first config (`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASS`, `HTTP_PORT`)
- Optional `config.yaml` support via `viper`
- Default HTTP port `5555` to match `config.json:5`

## Logging and Errors
- Structured logs via `zap` with request logging middleware
- Uniform error responses with `{"error": "..."}` to match current behavior

## Testing Strategy
- Unit tests for validation and repository
- Integration tests using `httptest` and a real Postgres (via docker-compose or `testcontainers-go`)
- Test cases ported from `transaction-service/test/test_main.cc:12-63`:
  - Create returns 201 and includes `id`, `created_at`
  - List returns 200 with array
  - Update preserves/sets `created_at` appropriately
  - Delete returns 204, missing returns 404

## Docker and Deployment
- Multi-stage Go Dockerfile: build static binary; minimal runtime image
- Update `docker-compose.yml`:
  - Keep `postgres` service and schema init (`sql/create.sql`)
  - Replace app service to run Go binary; pass DB env
- Healthchecks for app and DB

## Solidity Integration
- Keep `contracts/CSMMBinaryMarket.sol` unchanged
- Generate Go bindings via `abigen` and place under `internal/eth`
- Provide optional service methods to call contract functions if/when needed

## Migration Steps
1. Scaffold Go module and directory layout
2. Implement config, logging, and server bootstrap
3. Implement models and JSON validation
4. Implement repository with `pgxpool` (or `sqlc`)
5. Implement service layer logic
6. Implement HTTP handlers and routing
7. Port tests and add integration tests
8. Create Go Dockerfile and update compose
9. Generate optional Solidity Go bindings
10. Run full test suite and adjust as needed

## Acceptance Criteria
- All endpoints behave identically to C++ service
- Existing schema `sql/create.sql` remains the source of truth; data preserved
- Docker compose brings up Postgres and Go service; `GET /transactions` and `POST /transactions` work
- Tests pass; parity confirmed for CRUD

## Risks and Mitigations
- Decimal precision in `NUMERIC(10,4)`: use `decimal` library and explicit scanning
- Timezone handling for `TIMESTAMP WITH TIME ZONE`: test serialization consistency
- UUID defaults: ensure insert behavior matches `uuid_generate_v4()` or generate client-side UUIDs

## Deliverables
- Go source tree under `cmd/` and `internal/`
- Updated Dockerfile and docker-compose
- Test suite demonstrating parity
- Short migration notes embedded in code comments where necessary