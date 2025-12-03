#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:8080"
WALLET_ADDR="0xTESTUSER"
CURRENCY="USD"
SHARES=2
PRICE=0.5

echo "[GRPC] Dev login"
TOKEN=$(curl -s -X POST "$BASE/api/auth/dev-login" -H 'Content-Type: application/json' -d "{\"wallet\":\"$WALLET_ADDR\"}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
AUTH="Authorization: Bearer $TOKEN"

echo "[GRPC] Ensure user and wallet"
USER_ID=$(curl -s "$BASE/api/users/wallet/$WALLET_ADDR" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
curl -s -X POST "$BASE/api/wallets" -H "$AUTH" -H 'Content-Type: application/json' -d "{\"user_id\":\"$USER_ID\",\"currency\":\"$CURRENCY\"}" >/dev/null || true
ACCOUNT_ID=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT id FROM wallet_accounts WHERE user_id='$USER_ID' AND currency='$CURRENCY'" | tr -d '\r')

echo "[GRPC] Deposit"
curl -s -X POST "$BASE/api/wallets/$ACCOUNT_ID/deposit" -H "$AUTH" -H 'Content-Type: application/json' -d "{\"amount\":100}" >/dev/null
BAL_BEFORE=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT balance FROM wallet_accounts WHERE id='$ACCOUNT_ID'" | tr -d '\r')
echo "Balance before: $BAL_BEFORE"

echo "[GRPC] Create market"
TITLE="GRPC E2E Market $(date +%s)"
curl -s -X POST "$BASE/api/markets" -H 'Content-Type: application/json' -d "{\"question\":\"$TITLE\",\"description\":\"e2e\",\"options\":[\"yes\",\"no\"]}" >/dev/null
MID=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT id FROM markets WHERE title='$TITLE'" | tr -d '\r')
OID=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT id FROM options WHERE market_id='$MID' ORDER BY created_at ASC LIMIT 1" | tr -d '\r')

echo "[GRPC] BUY"
curl -s -X POST "$BASE/api/transactions" -H "$AUTH" -H 'Content-Type: application/json' -d "{\"user_id\":\"$USER_ID\",\"market_id\":\"$MID\",\"option_id\":\"$OID\",\"transaction_type\":\"BUY\",\"number_of_shares\":$SHARES,\"price_per_share\":$PRICE}" >/dev/null
BAL_AFTER_BUY=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT balance FROM wallet_accounts WHERE id='$ACCOUNT_ID'" | tr -d '\r')
echo "Balance after BUY: $BAL_AFTER_BUY"

echo "[GRPC] SELL"
curl -s -X POST "$BASE/api/transactions" -H "$AUTH" -H 'Content-Type: application/json' -d "{\"user_id\":\"$USER_ID\",\"market_id\":\"$MID\",\"option_id\":\"$OID\",\"transaction_type\":\"SELL\",\"number_of_shares\":$SHARES,\"price_per_share\":$PRICE}" >/dev/null
BAL_AFTER_SELL=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT balance FROM wallet_accounts WHERE id='$ACCOUNT_ID'" | tr -d '\r')
echo "Balance after SELL: $BAL_AFTER_SELL"

EXPECTED=$(awk -v s="$SHARES" -v p="$PRICE" 'BEGIN{printf("%.6f", s*p)}')
if awk -v bb="$BAL_BEFORE" -v ba="$BAL_AFTER_BUY" -v ex="$EXPECTED" 'BEGIN{diff=(bb-ex)-ba; if (diff<0) diff=-diff; exit(diff<1e-6?0:1)}'; then
  echo "BUY decreased balance by expected amount"
else
  echo "Mismatch on BUY"; exit 1
fi
if awk -v bb="$BAL_AFTER_BUY" -v ba="$BAL_AFTER_SELL" -v ex="$EXPECTED" 'BEGIN{diff=(bb+ex)-ba; if (diff<0) diff=-diff; exit(diff<1e-6?0:1)}'; then
  echo "SELL increased balance by expected amount"
else
  echo "Mismatch on SELL"; exit 1
fi

echo "[SUCCESS] GRPC E2E flow verified"
