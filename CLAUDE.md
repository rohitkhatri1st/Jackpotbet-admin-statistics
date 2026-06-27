# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the server (reads conf/default.toml)
go run main.go

# Seed the database with 2M rounds (4M transactions)
go run cmd/seed/main.go

# Build
go build ./...

# Run all tests
go test ./...

# Run a single package's tests
go test ./service/...
```

Config is loaded from `conf/default.toml`. Copy `conf/sample.toml` to `conf/default.toml` and set `auth.internal_token` before running. MongoDB and Redis must be running locally.

New Redis config keys (optional):

- `redis.max_memory` — e.g. `"256mb"`; empty string leaves memory limits to the operator.
- `redis.cache_ttl_hours` — stat cache TTL in hours; defaults to 24h if zero or omitted.

## Architecture

Three-layer: **Handler → Service → Repository**. No domain/event layers — intentional given the bounded, read-heavy scope.

```text
main.go              entry point + graceful shutdown
server/              HTTP server wiring
  server.go          Server struct + Start/Stop
  init.go            InitLoggers / InitDBs / InitServices wiring
  middleware/        CORS, auth (static token), request logging
  logger/            zerolog-backed Logger interface
  validator/         wraps go-playground/validator
config/              viper-based TOML config (conf/*.toml)
api/                 HTTP handlers
  api.go             API struct + constructor
  routes.go          route registration (open / user / internal subrouters)
  decode.go          JSON body decoder helper
  respond.go         structured JSON response helpers
  stats_handler.go   GGR + daily wager volume handlers
  transaction_handler.go
  user_handler.go    user-scoped handlers (wager percentile)
schema/              shared query structs (DateRangeFilter)
service/             business logic
  services.go        Services registry struct
  transaction_service.go
  rate_service.go    static currency → USD conversion
repository/
  repository.go      Repos struct + TransactionRepository interface
  transaction.go     filter/result types for the interface
  mongo/             concrete MongoDB implementations
    transaction.go
    ggr.go
    daily_wager_volume.go
model/               raw BSON structs (Transaction)
db/                  db connection wrappers (MongoDB, Redis)
cmd/seed/main.go     standalone seeder — 3-stage concurrent pipeline (scheduler → builder → writer)
conf/                TOML config files (default.toml, test.toml, sample.toml)
```

## Key Conventions

**Adding a new repository:** Add the interface method to `repository/transaction.go` (or a new `repository/<entity>.go`), implement it in `repository/mongo/`, add the field to `repository/Repos`, and wire it in `server/init.go`. The startup `validate()` call in `server/server.go` will fatal if any `Repos` or `Services` field is left nil.

**Adding a new service:** Add the struct to `service/`, add a field to `service/Services`, wire in `service/services.go` and `server/init.go`.

**Adding a new route:** Register in `api/routes.go` under the correct subrouter (`openRoutes`, `userRoutes`, or `internalRoutes`). Internal routes use the static `Authorization` token middleware. Add the handler in the relevant `api/*_handler.go` file.

**Date filters:** All stat routes accept optional `from` / `to` query params. Embed `schema.DateRangeFilter` in the query struct and call `.Validate()` after decoding.

**Amounts:** Stored and returned as `bson.Decimal128`. Use `shopspring/decimal` in the service layer for arithmetic; convert to/from `bson.ParseDecimal128` at the repository boundary.

**Currency:** Currently seeded with `ETH`, `BTC`, and `USDT`. Static USD rates live in `service/rate_service.go`. All transactions in a round share the same currency. The currency set is treated as open/extensible — do not hardcode the list in business logic; use `map[string]T` for per-currency results.

**Pagination:** `GetTransactions` uses cursor-based pagination (ObjectID cursor). Fetch `limit+1`, trim to `limit`, return the last ID as the next cursor.

**Caching:** `GetGGR`, `GetDailyWagerVolume`, and `GetWagerPercentile` use simple Redis TTL-based caching. Global stat keys use `statsCacheKey(prefix, from, to)`; user-scoped keys use `userStatsCacheKey(prefix, userID, from, to)` which prefixes with the userId hex (e.g. `{userId}:wager_percentile:{from}:{to}`). Both normalise dates to UTC day strings so queries over the same calendar days share an entry. `db/redis.go` applies `allkeys-lru` eviction policy on connect and optionally sets `maxmemory`. A materialized view (`daily_stats` collection + nightly cron) would scale better, but is intentionally omitted due to the complexity of coordinating DB corrections and cache invalidation.

**Time / UTC:** All timestamps are stored and returned in UTC. `time.Now()` must always be called as `time.Now().UTC()` at any persistence boundary. `schema.DateRangeFilter.Validate()` normalises client-supplied `from`/`to` to UTC before use — call it on every handler that embeds `DateRangeFilter`.

**Config search paths:** Viper searches `../conf/`, `../../conf/`, `./`, `./conf/` — works whether running from the repo root or from `cmd/seed/`.

## API Endpoints

All routes are prefixed with `/internal` and require `Authorization: <internal_token>` header.

| Method | Path | Description |
| -------- | ------ | ------------- |
| GET | `/internal/transactions` | Paginated transaction list with optional date filter |
| POST | `/internal/transactions` | Create a single transaction |
| GET | `/internal/gross_gaming_rev` | GGR per currency + USD over a date range |
| GET | `/internal/daily_wager_volume` | Daily wager volume grouped by date, with per-currency breakdown and total USD |
| GET | `/internal/user/{userId}/wager_percentile` | User's wager rank and top-X% position among all users in the period |
