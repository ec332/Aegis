## Goal
- Deduct wallet balance on BUY and increase on SELL using amount = price_per_share * number_of_shares.
- Keep current transaction validations intact and make wallet updates atomic enough to avoid double-charging.

## High-Level Flow
1. Frontend calls POST `/api/transactions` with the payload and `Authorization`.
2. API Gateway forwards the request to Transaction gRPC with the same `Authorization` metadata.
3. Transaction service validates market/option/price, computes `amount` and calls Wallet service:
   - BUY → `Withdrawal(account_id, amount, reference_id=transactionID)`
   - SELL → `Deposit(account_id, amount, reference_id=transactionID)`
4. On wallet success, persist the transaction and adjust market liquidity.
5. On wallet failure, return an error; no transaction persists. If persistence fails after wallet success, perform a compensating wallet reversal.

## Component Changes
### API Gateway
- Require `Authorization` for POST `/api/transactions`.
- Use `withAuth(ctx, r)` when invoking `transactionStub.CreateTransaction` so the token reaches Transaction service.
- References: `api-gateway/cmd/main.go:783`, `api-gateway/cmd/main.go:864`.

### Transaction Service
- Add Wallet gRPC client:
  - Env: `WALLET_GRPC_ADDR` (default `wallet-service:50052`) and `WALLET_DEFAULT_CURRENCY` (default `USD`).
  - Dial and inject `wallet.WalletServiceClient` into `TransactionGRPCServer` alongside `marketClient`.
  - References: `transaction-service/cmd/transaction-service/main.go:56-65` for existing market client pattern.
- Extend `TransactionGRPCServer` with `walletClient` field and constructor.
- In `CreateTransaction` (`transaction-service/internal/grpc/server.go:36`):
  - Keep existing validations (market status, option presence, price tolerance).
  - Compute `amount` using decimal: `decimal.NewFromFloat(req.PricePerShare) * decimal.NewFromInt(int64(req.NumberOfShares))`.
  - Determine wallet account:
    - Call `GetWalletAccountByUserID(user_id, currency=WALLET_DEFAULT_CURRENCY)` using incoming context metadata.
    - If not found → return `NotFound`.
  - Perform wallet update before DB insert:
    - BUY → `Withdrawal(AccountId, amount, ReferenceId=uuid)`.
    - SELL → `Deposit(AccountId, amount, ReferenceId=uuid)`.
    - If `FailedPrecondition` (insufficient funds) → return error; do not create transaction.
  - Persist transaction via `svc.Create` and let it adjust market liquidity as today.
  - If persistence fails after wallet success, attempt compensating wallet reversal:
    - BUY reversal → `Deposit` of same amount.
    - SELL reversal → `Withdrawal` of same amount.
    - Log critical if reversal fails.
- Map transaction type string remains as is (`mapTransactionType`).

### Wallet Service
- No changes to server logic; it already enforces `authorization` for `Deposit`/`Withdrawal` and guards against insufficient funds.
- References: `wallet/internal/grpc/auth_interceptor.go:17-35`, `wallet/internal/grpc/server.go:256-304`.

## Frontend
- Ensure `Authorization: Bearer <token>` is set when calling POST `/api/transactions` (same as wallet endpoints).
- Payload remains the same:
  - `market_id`, `option_id`, `user_id`, `transaction_type`, `number_of_shares`, `price_per_share`.

## Error Handling & Consistency
- Insufficient funds: return `FailedPrecondition` from Transaction service, surfacing HTTP 412 via gateway.
- Idempotency: use `reference_id = transactionID` for wallet transactions; recommend a future unique constraint on `wallet_transactions.reference_id` to prevent double charging on retries.
- Liquidity: since wallet update occurs before transaction persistence, no liquidity is adjusted if wallet fails. On persistence failure, attempt wallet reversal.

## Validation Plan
- Unit tests:
  - BUY with sufficient funds → wallet withdrawal succeeds, transaction persists, liquidity adjusts.
  - BUY with insufficient funds → wallet withdrawal fails, no transaction row, no liquidity change.
  - SELL → wallet deposit succeeds, transaction persists, liquidity adjusts.
  - Persistence failure after wallet success → compensating reversal executed.
- Manual via API:
  - Create wallet account and deposit funds; place BUY and check wallet balance and wallet transactions list.
  - Place SELL and verify wallet increase.

## Notes
- Default currency assumed `USD`; can be made configurable per transaction later.
- Amount calculation uses decimal internally and passes float to Wallet RPC, matching current Wallet API types.
