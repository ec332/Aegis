TEST TRANSACTIONS

Overview
- Walk through testing the end-to-end transaction flow via the API Gateway (HTTP → gRPC → Transaction Service).
- Covers market creation, wallet funding, placing buy/sell orders, querying transactions, and common validation errors.

Prerequisites
- Docker and Docker Compose installed
- Ports in use: `8080` (API Gateway), `50051` (Market), `50052` (Wallet & Transaction), `50053` (Settlement)

Start Services
- `docker compose up -d`
- Verify: `docker compose ps` shows API Gateway, Market, Wallet, Settlement, Transaction, Postgres, Kafka, Redis as running and healthy

Create a Market
- POST `http://localhost:8080/api/markets`
- Body:
  {
    "question": "Will Team A win the final?",
    "description": "Championship match outcome",
    "options": ["YES", "NO"],
    "end_time": "2025-12-31T23:59:59Z"
  }
- Response includes `id` (market_id)

Get Market Options
- GET `http://localhost:8080/api/markets/{market_id}/options`
- Note the `options[].id` (option_id) and `options[].current_price` for price sanity checks

Create a Wallet
- POST `http://localhost:8080/api/wallets`
- Body:
  {
    "user_id": "<your_user_uuid>",
    "currency": "USD"
  }
- Response includes `account.id` (wallet_id)

Fund the Wallet (Deposit)
- POST `http://localhost:8080/api/wallets/{wallet_id}/deposit`
- Body:
  {
    "amount": 100.0,
    "reference_id": "test-deposit-001"
  }
- Response includes updated balances

Place a Buy Transaction
- POST `http://localhost:8080/api/transactions`
- Body:
  {
    "user_id": "<your_user_uuid>",
    "market_id": "<market_id>",
    "option_id": "<option_id>",
    "transaction_type": "BUY",
    "number_of_shares": 10,
    "price_per_share": <use options.current_price or within ±2%>
  }
- Success: returns `id` and `created_at`
- Validation:
  - `price_per_share` must be 0–1 and within ±2% of current LMSR price
  - `market` must be `active`
  - `option_id` must belong to the `market_id`

Place a Sell Transaction
- Same endpoint and body as Buy, with `transaction_type": "SELL"`

Query Transactions
- GET `http://localhost:8080/api/transactions?user_id=<your_user_uuid>&market_id=<market_id>`
- Returns list with fields: `id`, `market_id`, `option_id`, `user_id`, `transaction_type`, `number_of_shares`, `price_per_share`, `created_at`

Negative Test Cases
- Market not active: returns error indicating market status precondition failure
- Option not in market: returns `400` invalid argument
- Price sanity violation (>±2%): returns `400` invalid argument
- Upstream market service unavailable: returns `500` with message `market service unavailable`

Tips
- Create realistic prices using `Get Market Options` before placing transactions
- Use consistent UUIDs for `user_id` across wallet and transactions
- If you restart services, re-create market and wallet and re-deposit funds

Cleanup
- `docker compose down -v` to stop services and remove volumes