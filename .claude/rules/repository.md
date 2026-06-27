---
paths:
  - "repository/**/*.go"
---

# Repository Rules

## Interface vs implementation

- `repository/` holds only interfaces and filter/result types — no MongoDB imports.
- `repository/mongo/` holds concrete implementations. Each file maps to one logical area (e.g. `transaction.go`, `ggr.go`).
- Add the compile-time interface check at the bottom of each implementation file:
  ```go
  var _ repository.TransactionRepository = (*TransactionRepository)(nil)
  ```

## Adding a new method

1. Add the method signature to the interface in `repository/transaction.go` (or a new `repository/<entity>.go`).
2. Add filter/result types to the same file — no `bson` imports here, use plain Go types or decimal strings.
3. Implement in `repository/mongo/`. Complex aggregations go in their own file (e.g. `ggr.go`); simple CRUD stays in `transaction.go`.
4. Add required indexes in `EnsureIndexes` — it's safe to call on every startup.

## MongoDB patterns

- Build `$match` stages from `bson.D`, appending conditions only when the filter field is non-nil.
- Aggregation result structs (used for `cursor.All`) are private to the mongo package; convert them to the repository's result type before returning.
- `bson.Decimal128` amounts are returned as `.String()` decimal strings at the repository boundary — the service layer uses `shopspring/decimal` for arithmetic.
- Sort paginated queries by `_id` ascending; use `$gt: cursor` for the next-page filter.
- Fetch `limit+1` in the service layer to detect next-page existence, not in the repo — the repo just applies whatever limit it receives.

## Rank / window functions

When a query needs a single user's rank among all users, avoid returning the full list to the application. Use `$setWindowFields` + `$facet` to let MongoDB assign the rank and return only one document:

```
$match → $group (one doc per user) → $facet {
  "userRank": [ $setWindowFields { sortBy, output: { rank: $rank } }, $match targetUser ]
  "total":    [ $count "n" ]
}
```

`$facet` always returns exactly one document regardless of dataset size. The `userRank` sub-array is empty when the user had no activity in the period (handle in the service). See `buildUserWagerRankPipeline` in `repository/mongo/transaction.go` for the canonical implementation.
