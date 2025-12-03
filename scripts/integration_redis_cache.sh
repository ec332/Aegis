#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:8080"

echo "[1] List markets (seed cache)"
curl -s "$BASE/api/markets?page=1&page_size=20" >/dev/null || true

echo "[2] Check Redis key for markets page"
docker compose exec -T redis redis-cli KEYS 'markets:list:page:1:20' || true
docker compose exec -T redis redis-cli TTL 'markets:list:page:1:20' || true

echo "[3] Create market and fetch summary"
TITLE="Cache Test Market $(date +%s)"
curl -s -X POST "$BASE/api/markets" -H 'Content-Type: application/json' -d "{\"question\":\"$TITLE\",\"description\":\"desc\",\"options\":[\"yes\",\"no\"]}" >/dev/null
MID=$(docker compose exec -T postgres psql -U postgres -d postgres -t -A -c "SELECT id FROM markets WHERE title='$TITLE'" | tr -d '\r')
curl -s "$BASE/api/markets/$MID" >/dev/null

echo "[4] Check Redis key for market summary"
docker compose exec -T redis redis-cli KEYS "market:$MID:summary" || true
docker compose exec -T redis redis-cli TTL "market:$MID:summary" || true

echo "[5] Update market and ensure invalidation"
curl -s -X PUT "$BASE/api/markets/$MID" -H 'Content-Type: application/json' -d '{"description":"updated-desc"}' >/dev/null
sleep 1
EXISTS_SUMMARY=$(docker compose exec -T redis redis-cli EXISTS "market:$MID:summary" | tr -d '\r')
if [ "$EXISTS_SUMMARY" != "0" ]; then
  echo "ERROR: market summary cache not invalidated"
  exit 1
else
  echo "Market summary cache invalidated"
fi
EXISTS_LIST=$(docker compose exec -T redis redis-cli EXISTS "markets:list:page:1:20" | tr -d '\r')
if [ "$EXISTS_LIST" != "0" ]; then
  echo "ERROR: markets page cache not invalidated"
  exit 1
else
  echo "Markets page cache invalidated"
fi

echo "[SUCCESS] Redis cache integration checks completed"
