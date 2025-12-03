# Backend E2E

## Overview

End-to-end tests simulate full prediction market flows across API Gateway and microservices using real infrastructure.

## Prerequisites

- Docker + Docker Compose
- Go 1.22+

## Start Full Stack

- `docker compose up -d`
- This starts Postgres, Redis, Kafka, migrations, and all services.

## Run E2E Tests

- `cd tests/e2e && go test -v`
- Specific flow:
  - `cd tests/e2e && go test -run TestCompletePredictionMarketFlow -v`
- With timeout:
  - `cd tests/e2e && go test -timeout 5m -v`

## What’s Covered

- Market lifecycle (create → trade → settle)
- Wallet balance updates and transaction listings
- Error recovery and health checks

## Logs & Artifacts

- Service logs via `docker compose logs -f <service>`
- Use Gateway health: `curl http://localhost:8080/health`

## Troubleshooting

- Ensure migrations completed (`db-migrations` service success).
- If Kafka/Redis connectivity fails, restart containers and check environment.

