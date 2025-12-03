#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:8080"
WALLET_ADDR="0xTESTUSER"
CURRENCY="USD"
SHARES=10
PRICE=0.5

echo "[1] Dev login"
TOKEN=$(curl -s -X POST "$BASE/api/auth/dev-login" -H 'Content-Type: application/json' -d "{\"wallet\":\"$WALLET_ADDR\"}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
if [ -z "$TOKEN" ]; then echo "Failed to get token"; exit 1; fi
AUTH="Authorization: Bearer $TOKEN"

echo "[2] Create market"
TITLE="Integration Test Market $(date +%s)"
curl -s -X POST "$BASE/api/markets" -H 'Content-Type: application/json' -d "{\"question\":\"$TITLE\",\"description\":\"test\",\"options\":[\"yes\",\"no\"]}" >/dev/null
MARKET_ID=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT id FROM markets WHERE title='$TITLE'" | tr -d '\r')
if [ -z "$MARKET_ID" ]; then echo "Failed to find market id"; exit 1; fi
OPTION_ID=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT id FROM options WHERE market_id='$MARKET_ID' ORDER BY created_at ASC LIMIT 1" | tr -d '\r')
if [ -z "$OPTION_ID" ]; then echo "Failed to find option id"; exit 1; fi

echo "[3] Ensure user exists and fetch user id"
USER_ID=$(curl -s "$BASE/api/users/wallet/$WALLET_ADDR" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
if [ -z "$USER_ID" ]; then echo "Failed to get user id"; exit 1; fi

echo "[4] Create wallet account"
curl -s -X POST "$BASE/api/wallets" -H "$AUTH" -H 'Content-Type: application/json' -d "{\"user_id\":\"$USER_ID\",\"currency\":\"$CURRENCY\"}" >/dev/null
ACCOUNT_ID=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT id FROM wallet_accounts WHERE user_id='$USER_ID' AND currency='$CURRENCY'" | tr -d '\r')
if [ -z "$ACCOUNT_ID" ]; then echo "Failed to get wallet account id"; exit 1; fi

echo "[5] Deposit funds"
curl -s -X POST "$BASE/api/wallets/$ACCOUNT_ID/deposit" -H "$AUTH" -H 'Content-Type: application/json' -d "{\"amount\":100}" >/dev/null

BAL_BEFORE=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT balance FROM wallet_accounts WHERE id='$ACCOUNT_ID'" | tr -d '\r')
echo "Balance before trade: $BAL_BEFORE"

echo "[6] Create BUY transaction"
TX_RESP=$(curl -s -X POST "$BASE/api/transactions" -H "$AUTH" -H 'Content-Type: application/json' -d "{\"user_id\":\"$USER_ID\",\"market_id\":\"$MARKET_ID\",\"option_id\":\"$OPTION_ID\",\"transaction_type\":\"BUY\",\"number_of_shares\":$SHARES,\"price_per_share\":$PRICE}")
TX_ID=$(echo "$TX_RESP" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
if [ -z "$TX_ID" ]; then echo "Failed to create transaction: $TX_RESP"; exit 1; fi
echo "Transaction id: $TX_ID"

echo "[7] Check wallet balance and liquidity"
BAL_AFTER=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT balance FROM wallet_accounts WHERE id='$ACCOUNT_ID'" | tr -d '\r')
POOL_VALUE=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT pool_value FROM liquidity_pool WHERE market_id='$MARKET_ID' AND option_id='$OPTION_ID'" | tr -d '\r')
WITHDRAW_TX=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT id, amount, description FROM wallet_transactions WHERE wallet_id='$ACCOUNT_ID' ORDER BY created_at DESC LIMIT 1" | tr -d '\r')

echo "Balance after trade: $BAL_AFTER"
echo "Liquidity pool_value: $POOL_VALUE"
echo "Latest wallet tx: $WITHDRAW_TX"

EXPECTED_DEC=$(awk -v s="$SHARES" -v p="$PRICE" 'BEGIN{printf("%.6f", s*p)}')
echo "Expected debit: $EXPECTED_DEC"

echo "[8] Validate"
if awk -v bb="$BAL_BEFORE" -v ba="$BAL_AFTER" -v ex="$EXPECTED_DEC" 'BEGIN{diff=(bb-ex)-ba; if (diff<0) diff=-diff; exit(diff<1e-6?0:1)}'
then
  echo "Wallet balance updated correctly"
else
  echo "Wallet balance mismatch"; exit 1
fi

if awk -v pool="$POOL_VALUE" -v shares="$SHARES" 'BEGIN{diff=pool-shares; if (diff<0) diff=-diff; exit(diff<1e-6?0:1)}'
then
  echo "Liquidity updated correctly"
else
  echo "Liquidity mismatch"; exit 1
fi

echo "[SUCCESS] Integration checks passed"

echo "[9] Attempt BUY exceeding balance (expect 400)"
BIG_PRICE=1000
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/transactions" -H "$AUTH" -H 'Content-Type: application/json' -d "{\"user_id\":\"$USER_ID\",\"market_id\":\"$MARKET_ID\",\"option_id\":\"$OPTION_ID\",\"transaction_type\":\"BUY\",\"number_of_shares\":1,\"price_per_share\":$BIG_PRICE}")
if [ "$HTTP_CODE" != "400" ]; then
  echo "Expected 400, got $HTTP_CODE"; exit 1
else
  echo "Received 400 for insufficient balance as expected"
fi
