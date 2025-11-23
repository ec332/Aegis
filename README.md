# Aegis

- Language: Go 1.22
- Service: `transaction-service` HTTP API
- Database: PostgreSQL (see `transaction-service/sql/create.sql`)
- Contracts: Solidity `transaction-service/contracts/CSMMBinaryMarket.sol`

## Quick Start (Local)
- Prerequisites: Go 1.22+, Postgres running with database `transaction`
- Env vars:
  - `APP_DB_HOST` (default `localhost`)
  - `APP_DB_PORT` (default `5432`)
  - `APP_DB_NAME` (default `transaction`)
  - `APP_DB_USER` (default `postgres`)
  - `APP_DB_PASSWORD` (default `postgres`)
  - `APP_HTTP_PORT` (default `5555`)

- Build: `go build ./cmd/transaction-service`
- Run: `./transaction-service`

## Docker Compose
- Start Postgres and the service: `docker compose up --build`
- Service: `http://localhost:5555`

## REST Endpoints
- `GET /transactions` → list
- `GET /transactions/{id}` → get one
- `POST /transactions` → create
- `PUT /transactions/{id}` → update
- `DELETE /transactions/{id}` → delete

### Example Create
- Request JSON:
  - `{ "user_id": "<uuid>", "market_id": "<uuid>", "option_id": "<uuid>", "transaction_type": "BUY", "number_of_shares": "10", "price_per_share": "1.23" }`
- Response: `201` with created object including `id` and `created_at`

## Testing
- Unit tests: `go test ./...`
- Integration tests (Postgres):
  - Ensure Postgres is up and schema applied (Compose does this automatically)
  - Provide `PG_DSN` or `APP_DB_*` env vars
  - Run: `go test -tags integration ./internal/store/postgres -v`

## Notes
- C++/Drogon implementation has been removed; the Go service provides identical endpoint semantics.