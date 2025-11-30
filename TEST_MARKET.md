# TEST_MARKET

This guide shows how to spin up the services and test the end‑to‑end workflow: market creation, trading with LMSR validation, state guards after resolution, and settlement completion. It uses Docker Compose and HTTP/gRPC calls.

## Prerequisites
- Docker and Docker Compose
- curl
- Optional: `grpcurl` for gRPC calls (`brew install grpcurl` on macOS)

## Start Services

```sh
docker compose up --build
```

Services exposed:
- API Gateway: `http://localhost:8080`
- Transaction Service (HTTP): `http://localhost:5555`
- Market gRPC: `localhost:50051`
- Wallet gRPC: `localhost:50052`
- Settlement gRPC: `localhost:50053`

## Create a Market (HTTP via API Gateway)

```sh
curl -s -X POST http://localhost:8080/api/markets \
  -H 'Content-Type: application/json' \
  -d '{
    "question": "Will BTC exceed $100k in 2025?",
    "description": "Resolves YES if BTC >= 100k at any point in 2025",
    "options": ["Yes", "No"],
    "end_time": "2025-12-31T23:59:59Z"
  }' | jq
```

Copy the `id` from the response as `MARKET_ID`.

List options to get their IDs and current LMSR price:

```sh
curl -s http://localhost:8080/api/markets/$MARKET_ID/options | jq
```

Note the `id` for the chosen option (e.g., the `Yes` option) as `OPTION_ID`, and `current_price` as `PRICE`.

## Create a Wallet and Fund It (HTTP via API Gateway)

Create a wallet for a user (replace `USER_ID` with a UUID):

```sh
USER_ID=$(uuidgen)
curl -s -X POST http://localhost:8080/api/wallets \
  -H 'Content-Type: application/json' \
  -d '{"user_id": "'$USER_ID'", "currency": "USD"}' | jq
```

Copy the returned wallet `id` as `WALLET_ID`. Deposit funds:

```sh
curl -s -X POST http://localhost:8080/api/wallets/$WALLET_ID/deposit \
  -H 'Content-Type: application/json' \
  -d '{"amount": 1000.00, "reference_id": "seed-001"}' | jq
```

## Place a Trade (HTTP via Transaction Service)

Buy shares for the market option. The transaction service enforces:
- Market must be `active`
- `option_id` must belong to the market
- `price_per_share` must be close to current LMSR price (±2%)

```sh
curl -s -X POST http://localhost:5555/transactions \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id": "'$USER_ID'",
    "market_id": "'$MARKET_ID'",
    "option_id": "'$OPTION_ID'",
    "transaction_type": "BUY",
    "number_of_shares": "10",
    "price_per_share": "'$PRICE'"
  }' | jq
```

Sanity checks:
- Try the same request with `price_per_share` far from `PRICE` → should return `400` with a deviation error.
- Try an `option_id` not in the market → should return `400` with an option membership error.

## Block Trading After Resolution

Resolve the market via API Gateway. Allowed status transitions are enforced in the market service.

Mark as resolving, then set outcome and resolved:

```sh
curl -s -X PUT http://localhost:8080/api/markets/$MARKET_ID \
  -H 'Content-Type: application/json' \
  -d '{"status": "resolving"}' | jq

WINNING_OPTION_ID=$OPTION_ID # e.g., choose the Yes option
curl -s -X PUT http://localhost:8080/api/markets/$MARKET_ID \
  -H 'Content-Type: application/json' \
  -d '{"status": "resolved", "outcome": "'$WINNING_OPTION_ID'"}' | jq
```

Attempt to buy again:

```sh
curl -s -X POST http://localhost:5555/transactions \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id": "'$USER_ID'",
    "market_id": "'$MARKET_ID'",
    "option_id": "'$OPTION_ID'",
    "transaction_type": "BUY",
    "number_of_shares": "1",
    "price_per_share": "'$PRICE'"
  }' | jq
```

Expected: `400` with `market is not active`.

## Settlement and Payout (gRPC)

Create and complete the settlement:

```sh
grpcurl -d '{"market_id": "'$MARKET_ID'", "winning_option_id": "'$WINNING_OPTION_ID'"}' \
  -plaintext localhost:50053 settlement.SettlementService/CreateSettlement | jq

SETTLEMENT_ID=<copy from response>

grpcurl -d '{"id": "'$SETTLEMENT_ID'"}' -plaintext \
  localhost:50053 settlement.SettlementService/CompleteSettlement | jq
```

Process payouts:

```sh
# If distributions are present, this will credit wallets and be idempotent
grpcurl -d '{"settlement_id":"'$SETTLEMENT_ID'"}' -plaintext \
  localhost:50053 settlement.SettlementService/ProcessPayout | jq

# Idempotency check: re-run; should return success with an idempotent message
grpcurl -d '{"settlement_id":"'$SETTLEMENT_ID'"}' -plaintext \
  localhost:50053 settlement.SettlementService/ProcessPayout | jq
```

Notes:
- In this build, distributions are managed in-memory inside the settlement service; if none are present, `ProcessPayout` returns `NotFound`. The payout path demonstrates idempotency when distributions are available.

## Environment Variables

- Transaction Service → Market gRPC:
  - `APP_MARKET_GRPC_ADDR` (default `localhost:50051` locally; `market-service:50051` in Compose)
- Settlement Service → Wallet gRPC:
  - `WALLET_GRPC_ADDR` (set to `wallet-service:50052` in Compose)

## Troubleshooting

- `503 Service Unavailable` from transaction service on create/update:
  - Market gRPC is unreachable; check `market-service` logs.
- Price sanity errors:
  - Refresh current option prices via `GET /api/markets/$MARKET_ID/options` and keep `price_per_share` within 2% of `current_price`.
- Trading blocked:
  - Ensure market status is `active` before placing trades.