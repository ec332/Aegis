## Goals
- Remove all C++/Drogon artifacts from the repository
- Keep SQL schema and Solidity contract
- Add Go CI pipeline (build + tests, optional Docker build)
- Update README to document the Go service usage

## Current State
- Go service present under `cmd/` and `internal/`
- C++ Drogon app remains under `transaction-service/` (controllers, models, repos, services, tests, CMake, config, Dockerfile)
- Docker Compose already points to Go `Dockerfile`; SQL schema lives in `transaction-service/sql/create.sql`
- No CI configuration found; README minimal

## Changes To Perform
- Delete C++/Drogon files and folders:
  - `transaction-service/controllers/*`
  - `transaction-service/models/*`
  - `transaction-service/repositories/*`
  - `transaction-service/services/*`
  - `transaction-service/test/*`
  - `transaction-service/CMakeLists.txt`
  - `transaction-service/main.cc`
  - `transaction-service/config.json`
  - `transaction-service/Dockerfile`
- Keep and retain:
  - `transaction-service/sql/create.sql` (schema)
  - `transaction-service/contracts/CSMMBinaryMarket.sol` (Solidity)
  - `transaction-service/.gitignore` (optional keep or prune; will leave in place)
- CI: Add GitHub Actions workflow `.github/workflows/ci.yml` to:
  - Set up Go 1.22, cache modules
  - Run `go build ./cmd/transaction-service`
  - Run `go test ./...`
  - Optionally build Docker image with root `Dockerfile` (no push)
- Documentation: Update `README.md` to cover
  - Go prerequisites and quick start (env vars)
  - Local run and Docker Compose
  - REST endpoints and example requests
  - Testing with `go test`

## Acceptance Criteria
- No Drogon/C++ files remain
- CI runs Go build and tests on pushes/PRs to `main`
- README provides clear instructions to run the Go service locally and with Docker, including environment variables
- Docker Compose continues to boot Postgres and the Go service; SQL path remains correct

## Notes
- We will not remove `transaction-service/contracts` or `sql` directories
- If you want image publishing in CI, we can add registry configuration after this cleanup