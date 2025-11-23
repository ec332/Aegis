## Smoke Tests Via API Gateway
- Health: GET `/health`
- Wallet: POST `/api/wallets` with `{"user_id":"u-1","currency":"USD"}`; GET `/api/wallets/{id}`; POST deposit/withdraw
- Markets: GET `/api/markets`; POST `/api/markets` with question/description/category/end_time; GET `/api/markets/{id}`; GET `/api/markets/{id}/options`
- Settlements: POST `/api/settlements`; GET `/api/settlements/{id}`; PUT `/api/settlements/{id}/complete`

## Data Validation
- Confirm DB rows created for wallets/transactions/markets with service logs
- Verify Market schema init succeeded and Redis pub/sub works (creation publishes liquidity update)

## Proto Generation Pipeline Cleanup
- Replace ad‑hoc `proto/google/protobuf/timestamp.proto` with proper well‑known types resolution
- Document and script protoc generation using pinned plugins (go 1.33.0, go‑grpc 1.3.0)
- Optional: adopt Buf with a local Docker image that works on Apple Silicon, check in `buf.yaml`/`buf.gen.yaml`

## Compose And Config Hygiene
- Remove `version:` from `docker-compose.yml`
- Keep `market-service` using `DATABASE_URL`, `REDIS_URL`, `PORT`; ensure other services don’t require env mismatches

## Observability
- Ensure API Gateway logs include service/method metadata
- Check gRPC health services report SERVING for all services

## Acceptance Criteria
- All endpoints return 2xx with correct JSON payloads
- No protobuf marshal/unmarshal errors in Gateway
- All services healthy; DB schemas present and no runtime SQL errors
- Regeneration instructions are repeatable across environments