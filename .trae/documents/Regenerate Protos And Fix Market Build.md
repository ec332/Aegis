## Goals
- Regenerate protobufs to match library versions used in the repo
- Align gRPC/protobuf dependency versions across services
- Fix market service compile errors caused by model/schema mismatches
- Ensure Docker builds succeed and services run

## Dependency Alignment
- Standardize Go toolchain to `go 1.22` in all module `go.mod`
- Set unified versions:
  - `google.golang.org/grpc` ≥ `v1.63.0` (supports `grpc.StaticMethod`)
  - `google.golang.org/protobuf` `v1.33.0` (or latest compatible)
- Apply in modules: `proto`, `api-gateway`, `market`, `wallet`, `settlement`, `transaction-service`

## Protobuf Regeneration
- Install generators:
  - `go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0`
  - `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0`
- Regenerate outputs (source-relative paths, preserve repo layout):
  - `protoc -I proto --go_out=paths=source_relative:proto/gen --go-grpc_out=paths=source_relative:proto/gen proto/market.proto proto/settlement.proto proto/wallet.proto`
- Rationale: current generated files reference `grpc.StaticMethod`, which fails with `grpc v1.56.3` in `market/go.mod`; regeneration with aligned deps removes symbol mismatches.

## Market Model/Service Fixes
- Current mismatches:
  - Service expects `LiquidityParameter` and `ShareQuantity` (`market/internal/service/service.go:40-55, 69-77`), but models have neither (`market/pkg/models/models.go:18-47`).
- Proposed changes:
  - Add `LiquidityParameter float64` to `models.Market`
  - Add `ShareQuantity float64` to `models.LiquidityPool` (retain `PoolValue` or migrate to one field; prefer using `PoolValue` consistently)
  - Update service code to use `PoolValue` for computations and storage to match repository schema (`market/internal/repository/repository.go:340-359`)
  - If keeping `LiquidityParameter`, no DB storage change needed; used for pricing logic only.
- Outcome: eliminates compile errors and keeps repository schema consistent.

## Market SQL Alignment
- `market/sql/create.sql` currently defines different tables/columns than code uses (e.g., `liquidity_pools`, `options.option_text`).
- Align SQL file to match `InitSchema` used by repository (`market/internal/repository/repository.go:392-448`):
  - Tables: `markets(title, description, status, resolution_datetime, winning_option_id, ...)`
  - Tables: `options(id, market_id, title, created_at)`
  - Tables: `liquidity_pool(id, market_id, option_id, pool_value, updated_at)`
- This prevents runtime DB errors when the compose-mounted init SQL runs.

## Dockerfile Context Checks
- Keep updated builder images `golang:1.22-alpine` for all services
- Ensure per-service `WORKDIR` uses module roots and `go build` targets:
  - `api-gateway/Dockerfile` builds `./cmd/main.go`
  - `market/Dockerfile` builds `./cmd/main.go`
  - `transaction-service/Dockerfile` builds `./cmd/transaction-service/main.go`

## Compose Warning Cleanup
- Remove the top-level `version` key from `docker-compose.yml` to silence the warning

## Verification
- Build: `docker compose build --no-cache`
- Run: `docker compose up -d`
- Health:
  - Postgres healthcheck passes
  - Market service logs show DB schema initialized; no missing column errors
- Smoke tests:
  - API Gateway HTTP `POST /markets` using a minimal payload
  - Verify market retrieval and liquidity pool updates

## Rollback and Safety
- Changes are confined to Go module versions, generated code in `proto/gen`, models/service for market, and SQL init files
- No secrets or credentials are touched; no logging of sensitive data

## Acceptance Criteria
- All services build successfully
- Market service compiles and runs without schema errors
- Protobuf-generated files compile across all modules
- Compose runs cleanly without warnings/errors