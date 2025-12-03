## Scope
- Implement Redis-backed caching for market summaries and market listings in API Gateway.
- Do NOT cache options.
- Add pagination support for transactions at API Gateway.
- Use request coalescing to avoid stampede and set sensible TTLs and size caps.

## Configuration
- API Gateway: add `REDIS_URL` env (default `redis://redis:6379`).
- Initialize a Redis client in API Gateway and a small cache helper (JSON marshal/unmarshal).
- TTLs:
  - `market:{id}:summary` → 60s
  - `markets:list:page:{n}` → 30s
- Size caps: `page_size` max 50; only cache pages up to `page<=10`.

## API Gateway Changes
### Cache Helper
- Create a lightweight cache utility using `go-redis/v9` and `x/sync/singleflight`:
  - `GetJSON(ctx, key, dest)` → returns bool hit
  - `SetJSON(ctx, key, data, ttl)`
  - `Del(ctx, keys...)`
  - Singleflight around cache-miss rebuilds keyed by `key`.

### Market Summary Cache
- In `getMarket`:
  - Build key `market:{id}:summary`.
  - On cache hit: return cached DTO.
  - On miss: fetch via `marketStub.GetMarket`, store DTO with TTL 60s, return.

### Market Listings Cache (without options)
- In `listMarkets`:
  - Accept `page` and `page_size` query params; enforce max size 50.
  - Build key `markets:list:page:{page}:{size}`.
  - On cache hit: return cached slice.
  - On miss: call `marketStub.ListMarkets`, paginate in gateway, store page with TTL 30s, return.

### Invalidation on Updates
- In `updateMarket`:
  - After successful update, call cache `Del` for `market:{id}:summary`.
  - Optionally delete listing pages: `markets:list:page:*` (iterate known small range) or keep TTL-based natural rotation.

## Transactions Pagination (no cache)
- In `getTransactions`:
  - Accept `page` and `page_size` (max 50).
  - Fetch via `transactionStub.GetTransactions` (filter by `user_id`/`market_id` if provided).
  - Slice in gateway to page response; include `total` and `next_page` when applicable.
  - Keep future-ready `cursor` query param (optional) without implementing caching.

## Safeguards
- Singleflight dedupes concurrent cache misses per key.
- Limit cached payload size by capping `page_size` and pages cached.
- Time-based TTL avoids heavy invalidation during trading bursts.

## Validation
- Manual tests:
  - `GET /api/markets/{id}`: first call hits gRPC, subsequent within 60s hit cache; update invalidates.
  - `GET /api/markets?page=1&page_size=20`: cache page; change market then verify new page reflects update after TTL or invalidation.
  - `GET /api/transactions?user_id=...&page=1&page_size=20`: pagination slices correctly; no caching.

## Notes
- No options caching is implemented per requirement.
- Listing cache operates on market summaries only.
- Read replicas can be added later; current approach keeps cache at the gateway for simplicity.