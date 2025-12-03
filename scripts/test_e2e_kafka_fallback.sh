#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:8080"

echo "[FALLBACK] Stop market-service"
docker compose stop market-service >/dev/null
sleep 1

echo "[FALLBACK] Create market (expect 202)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/markets" -H 'Content-Type: application/json' -d '{"question":"CB Test","description":"desc","options":["yes","no"]}')
if [ "$HTTP_CODE" != "202" ]; then echo "Expected 202, got $HTTP_CODE"; exit 1; fi
echo "Received 202 Accepted"

echo "[FALLBACK] Restart market-service"
docker compose start market-service >/dev/null
sleep 2
echo "[SUCCESS] Kafka fallback path exercised"

