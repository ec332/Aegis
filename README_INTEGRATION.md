# Integration Tests

## Overview

Integration tests validate interactions between services and core infrastructure (Postgres, Redis, gRPC/HTTP). They run against real containers locally.

## Prerequisites

- Docker + Docker Compose
- Go 1.22+

## Start Infrastructure

- `docker compose up -d postgres redis kafka`
- Wait for health checks to pass.

## Run Integration Tests (Go)

- Service-to-service tests:
  - `cd tests/integration && go test -v`
- Specific test:
  - `cd tests/integration && go test -run TestHTTPToGRPCIntegration -v`
- With race detection:
  - `cd tests/integration && go test -race -v`

## API Gateway + Services

- Bring up full stack:
  - `docker compose up -d`
- Validate ports:
  - Gateway `:8080`, Market `:50051`, Wallet `:50052`, Settlement `:50053`

## Environment Variables

- Gateway:
  - `KAFKA_BROKERS`, `CORS_*`, `*_SERVICE_GRPC_ADDR`
- Services:
  - DB host/user/password, port

## Artifacts & Logs

- View logs:
  - `docker compose logs -f api-gateway`
  - `docker compose logs -f wallet-service`
- Kafka topics inspection (optional): see root README for commands.

## Troubleshooting

- Port conflicts: stop local DB/Redis if needed.
- Migration failures: verify `db-migrations` service succeeded.
- Health failures: check container logs and environment values.

