---
paths:
  - "service/**/*.go"
---

# Service Layer Rules

## Responsibilities

The service layer owns business logic: computing GGR, daily wager volume grouping, pagination cursor handling, USD conversion. It must not import `net/http` or `bson` — the only `bson` exception is `bson.ObjectID` used as a cursor identifier.

## Input/output types

- Define dedicated `*Input` and `*Result` structs per method — never pass raw request structs from the handler.
- Amounts flow in as pre-validated decimal strings; use `shopspring/decimal` for arithmetic and call `.StringFixed(n)` before returning (8 decimal places for crypto, 2 for USD).
- Guard every exported method with `if input == nil { return nil, errors.New(...) }`.

## Currency conversion

- `rateService` (set to `NewStaticRateService()` in `NewTransactionService`) handles currency → USD. Call `s.rateService.ToUSD(ctx, currency, amount)` — don't inline rates in service methods.

## Grouping / reshaping repo results

When a repo returns flat rows that need to be grouped for the response (e.g. `(date, currency)` rows → one entry per date), extract the grouping into a **pure helper function** — not inside the service method. Pattern from `groupDailyWagersByDate`:

- Use a `[]string` slice (`dateOrder`) alongside a `map[string]T` (`buckets`) to accumulate data while preserving the sort order returned by the repository.
- Per-currency results use `map[string]CurrencyVolume` — never a fixed struct with named currency fields. The currency set is extensible.
- The service method itself stays thin: nil guard → repo call → delegate to helper.

## Pagination

- Fetch `limit+1` from the repo, trim to `limit`, set `nextCursor` to the last item's ID when the extra item exists. This is the canonical pattern — follow it for all paginated endpoints.

## Redis caching for stat endpoints

`GetGGR`, `GetDailyWagerVolume`, and `GetWagerPercentile` all wrap their repo calls with a Redis cache. The pattern is the same in every method:

1. Build the cache key:
   - Global stats: `statsCacheKey(prefix, from, to)` — e.g. `ggr:2024-01-01:2024-01-31`
   - User-scoped stats: `userStatsCacheKey(prefix, userID, from, to)` — e.g. `{userId}:wager_percentile:2024-01-01:2024-01-31`
   Both helpers normalise `*time.Time` pointers to UTC day strings (`"2006-01-02"`) so queries over the same calendar days share one entry.
2. Attempt `redis.Get`; on hit, unmarshal JSON and return early.
3. On miss, call the repo, build the result, then `redis.Set` with `s.cacheTTL`.
4. Both the `redis` client and `cacheTTL` are injected via `TransactionServiceOptions` — nil `redis` silently skips caching (useful in tests).

`ServicesOptions` accepts `Redis *redis.Client` and `CacheTTL time.Duration`; `server/init.go` reads `CacheTTLHours` from config and passes `s.Redis.Client`.

## Time / UTC

All `time.Time` values created or stored in the service must be UTC:

- Use `time.Now().UTC()` — never bare `time.Now()` at a persistence boundary.
- Normalise caller-supplied times: `input.CreatedAt.UTC()` before storing.
- `statsCacheKey` already calls `.UTC()` on its inputs — no extra normalisation needed there.

## Registering a new service

1. Add the struct to `service/<name>_service.go`.
2. Add a field to `Services` in `service/services.go` and wire it in `NewServices`.
3. Add `initServices()` wiring in `server/init.go` if it needs a direct DB dependency.
   The startup `validate()` check will fatal on nil fields — it will catch missing wiring immediately.
