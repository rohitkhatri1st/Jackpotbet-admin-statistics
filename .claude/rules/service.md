---
paths:
  - "service/**/*.go"
---

# Service Layer Rules

## Responsibilities

The service layer owns business logic: computing GGR, pagination cursor handling, USD conversion. It must not import `net/http` or `bson` — the only `bson` exception is `bson.ObjectID` used as a cursor identifier.

## Input/output types

- Define dedicated `*Input` and `*Result` structs per method — never pass raw request structs from the handler.
- Amounts flow in as pre-validated decimal strings; use `shopspring/decimal` for arithmetic and call `.StringFixed(n)` before returning (8 decimal places for crypto, 2 for USD).
- Guard every exported method with `if input == nil { return nil, errors.New(...) }`.

## Currency conversion

- `rateService` (set to `NewStaticRateService()` in `NewTransactionService`) handles currency → USD. Call `s.rateService.ToUSD(ctx, currency, amount)` — don't inline rates in service methods.

## Pagination

- Fetch `limit+1` from the repo, trim to `limit`, set `nextCursor` to the last item's ID when the extra item exists. This is the canonical pattern — follow it for all paginated endpoints.

## Registering a new service

1. Add the struct to `service/<name>_service.go`.
2. Add a field to `Services` in `service/services.go` and wire it in `NewServices`.
3. Add `initServices()` wiring in `server/init.go` if it needs a direct DB dependency.
   The startup `validate()` check will fatal on nil fields — it will catch missing wiring immediately.
