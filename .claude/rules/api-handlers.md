---
paths:
  - "api/**/*.go"
---

# API Handler Rules

## Handler flow — always in this order

1. Decode input (`DecodeJSONBody` for POST, `DecodeQuery` for GET)
2. Apply defaults (e.g. `if query.Limit == 0 { query.Limit = defaultLimit }`)
3. Validate with `a.validator.Validate(...)` — returns a formatted error string, wrap as `NewAppError(CodeValidation, ..., http.StatusBadRequest)`
4. Call `query.DateRangeFilter.Validate()` if the query embeds `schema.DateRangeFilter`
5. Convert any raw values that need it (e.g. hex cursor string → `bson.ObjectID`)
6. Call the service
7. Respond with `a.respond(w, status, result)` or `a.respondError(w, err)`

## Error handling

- Use `NewAppError(code Code, msg string, status int)` for all client-facing errors.
- Pass `true` as the second arg to `respondError` only when you want the error force-logged (e.g. unexpected DB errors on mutation routes).
- Non-`AppError` errors from the service layer become `500 INTERNAL_ERROR` automatically — don't wrap them, just pass through.

## Request/response structs

- Query structs: define in the handler file, use `qs:"..."` tags, embed `schema.DateRangeFilter` for date-range params.
- Request body structs: `json:"..."` tags + `validate:"..."` tags. Amounts stay as `string` — never decode to `float64` (precision loss).
- Response types live in the service layer, not here.

## Route registration

- Add new routes in `api/routes.go` under the correct subrouter: `openRoutes`, `userRoutes`, or `internalRoutes`.
- Internal routes live under `/internal` prefix and require the static token middleware already applied to that subrouter.
