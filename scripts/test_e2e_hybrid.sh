#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:8080"
WALLET_ADDR="0xTESTUSER"

echo "[HYBRID] Dev login"
TOKEN=$(curl -s -X POST "$BASE/api/auth/dev-login" -H 'Content-Type: application/json' -d "{\"wallet\":\"$WALLET_ADDR\"}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
AUTH="Authorization: Bearer $TOKEN"

echo "[HYBRID] Create market (should succeed)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/markets" -H 'Content-Type: application/json' -d '{"question":"Hybrid Market","description":"desc","options":["yes","no"]}')
if [ "$HTTP_CODE" != "201" ]; then echo "Expected 201, got $HTTP_CODE"; exit 1; fi

echo "[HYBRID] Stop wallet-service"
docker compose stop wallet-service >/dev/null
sleep 1

echo "[HYBRID] Create wallet account (expect 202)"
USER_ID=$(curl -s "$BASE/api/users/wallet/$WALLET_ADDR" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/wallets" -H "$AUTH" -H 'Content-Type: application/json' -d "{\"user_id\":\"$USER_ID\",\"currency\":\"USD\"}")
if [ "$HTTP_CODE" != "202" ]; then echo "Expected 202, got $HTTP_CODE"; exit 1; fi
echo "Received 202 for wallet create with service down"

echo "[HYBRID] Restart wallet-service"
docker compose start wallet-service >/dev/null
sleep 2
echo "[SUCCESS] Hybrid flow verified"

