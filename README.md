# JackpotBet Admin Statistics API

A Go REST API that aggregates casino transaction data from MongoDB, providing
admin-facing statistics such as Gross Gaming Revenue, daily wager volume, and
per-user wager percentile rankings.

> **Engineering Decisions** —
> [View the full write-up](https://rohitkhatri1st.github.io/Jackpotbet-admin-statistics/Engineering%20Decisions.html)
>
> **API Docs (Postman)** —
> [View the collection](https://documenter.getpostman.com/view/37222846/2sBXwyG7Uj)

---

## Tech Stack

| Layer | Choice |
| --- | --- |
| Language | Go 1.25.5 |
| Database | MongoDB (via `go.mongodb.org/mongo-driver/v2`) |
| Cache | Redis (`go-redis/v9`) |
| Router | `gorilla/mux` |
| Validation | `go-playground/validator/v10` |
| Config | `spf13/viper` (TOML) |
| Logging | `rs/zerolog` |

---

## Prerequisites

- Go 1.25+
- MongoDB running on `localhost:27017`
- Redis running on `localhost:6379`

---

## Setup

### 1. Clone the repo

```bash
git clone git@github.com:rohitkhatri1st/Jackpotbet-admin-statistics.git
cd Jackpotbet-admin-statistics
```

### 2. Create the config file

```bash
cp conf/sample.toml conf/default.toml
```

Edit `conf/default.toml` and set `auth.internal_token` to any secret string
you choose — this is the token required on every API request.

### 3. Install dependencies

```bash
go mod download
```

### 4. Seed the database

Inserts 2,000,000 game rounds (4,000,000 transactions) across 700 unique users
using a concurrent 3-stage pipeline. Expect it to take a few minutes.

```bash
make seed
```

### 5. Run the server

```bash
make run
```

The API is now available at `http://localhost:8080`.

---

## Authentication

All routes sit under `/internal` and require a static token header:

```http
X-Internal-Key: <your internal_token from config>
```

Requests without a valid token receive `401 Unauthorized`.

---

## API Reference

All endpoints accept optional `from` and `to` query parameters to narrow the
date range. Dates are accepted in RFC 3339 / ISO 8601 format
(e.g. `2024-01-01T00:00:00Z`). All timestamps in responses are UTC.

---

### GET `/internal/gross_gaming_rev`

Returns Gross Gaming Revenue (wagers − payouts) broken down by currency and in
USD over the specified period.

#### Query params

| Param | Type | Required |
| --- | --- | --- |
| `from` | ISO 8601 date-time | No |
| `to` | ISO 8601 date-time | No |

#### Example request

```bash
curl "http://localhost:8080/internal/gross_gaming_rev?from=2024-01-01T00:00:00Z&to=2024-06-01T00:00:00Z" \
  -H "X-Internal-Key: your-secret-token-here"
```

#### Example response

```json
{
  "data": [
    {
      "currency": "ETH",
      "wagers": "12345.67890000",
      "payouts": "11800.12340000",
      "ggr": "545.55550000",
      "wagersUSD": "38271956.78",
      "payoutsUSD": "36576382.54",
      "ggrUSD": "1695574.24"
    }
  ]
}
```

---

### GET `/internal/daily_wager_volume`

Returns total wager volume grouped by day, with a per-currency breakdown and a
USD total for each day.

#### Query params

| Param | Type | Required |
| --- | --- | --- |
| `from` | ISO 8601 date-time | No |
| `to` | ISO 8601 date-time | No |

#### Example request

```bash
curl "http://localhost:8080/internal/daily_wager_volume?from=2024-03-01T00:00:00Z&to=2024-03-07T00:00:00Z" \
  -H "X-Internal-Key: your-secret-token-here"
```

#### Example response

```json
{
  "data": [
    {
      "date": "2024-03-01",
      "totalVolumeUSD": "4821045.33",
      "currencies": {
        "BTC": { "volume": "42.18900000", "volumeUSD": "2531340.00" },
        "ETH": { "volume": "891.44000000", "volumeUSD": "2289705.33" }
      }
    }
  ]
}
```

---

### GET `/internal/user/{userId}/wager_percentile`

Returns a user's total USD wagered, their rank among all active users, and the
top-X percentile they fall into for the specified period.

#### Path params

| Param | Type | Required |
| --- | --- | --- |
| `userId` | Hex ObjectID | Yes |

#### Query params

| Param | Type | Required |
| --- | --- | --- |
| `from` | ISO 8601 date-time | No |
| `to` | ISO 8601 date-time | No |

#### Example request

```bash
curl "http://localhost:8080/internal/user/64a1f2b3c4d5e6f7a8b9c0d1/wager_percentile?from=2024-01-01T00:00:00Z&to=2024-12-31T00:00:00Z" \
  -H "X-Internal-Key: your-secret-token-here"
```

#### Example response

```json
{
  "data": {
    "userId": "64a1f2b3c4d5e6f7a8b9c0d1",
    "totalUSD": "98234.56",
    "rank": 10,
    "totalUsers": 500,
    "topPercentile": "2.00%"
  }
}
```

---

### GET `/internal/transactions`

Paginated list of raw transactions with optional date filtering.

#### Query params

| Param | Type | Required | Default |
| --- | --- | --- | --- |
| `from` | ISO 8601 date-time | No | — |
| `to` | ISO 8601 date-time | No | — |
| `cursor` | Hex ObjectID | No | — |
| `limit` | Integer (1–100) | No | 20 |

#### Example request

```bash
curl "http://localhost:8080/internal/transactions?limit=5" \
  -H "X-Internal-Key: your-secret-token-here"
```

#### Example response

```json
{
  "data": [ ],
  "cursor": "64a1f2b3c4d5e6f7a8b9c0d2"
}
```

Pass the returned `cursor` as the `cursor` param on the next request to fetch
the following page. A `null` cursor means there are no more results.

---

### POST `/internal/transactions`

Creates a single transaction.

#### Request body

```json
{
  "userId": "64a1f2b3c4d5e6f7a8b9c0d1",
  "roundId": "round-abc-123",
  "type": "Wager",
  "currency": "ETH",
  "amount": "1.50000000",
  "createdAt": "2024-06-15T10:30:00Z"
}
```

| Field | Type | Required | Constraints |
| --- | --- | --- | --- |
| `userId` | Hex ObjectID | Yes | — |
| `roundId` | String | Yes | — |
| `type` | String | Yes | `Wager` or `Payout` |
| `currency` | String | Yes | `ETH`, `BTC`, or `USDT` |
| `amount` | String | Yes | Valid decimal, ≥ 0 |
| `createdAt` | ISO 8601 date-time | No | Defaults to now |

---

## Caching

Stat routes (`/gross_gaming_rev`, `/daily_wager_volume`,
`/user/{userId}/wager_percentile`) are Redis-cached with a configurable TTL
(default 24 hours). Cache keys are keyed by the normalised UTC date range, so
queries covering the same calendar days share a cache entry. Redis uses
`allkeys-lru` eviction — no manual cache management needed. If Redis is
unavailable the API falls back to querying MongoDB directly.

---

## Available Make Targets

Run `make help` to see all targets. The most commonly used ones:

| Target | Description |
| --- | --- |
| `make run` | Start the server |
| `make seed` | Seed MongoDB with 2M rounds / 4M transactions |
| `make build` | Compile binary into `./bin/admin-stats` |
| `make test` | Unit tests — no infrastructure needed |
| `make test-integration` | Repository integration tests — requires MongoDB |
| `make test-all` | All tests including integration |
| `make test-cover` | Unit tests + open HTML coverage report |
| `make check` | Quick pre-commit gate: fmt + vet + lint + unit tests |
| `make pre-pr` | Full pre-PR gate: fmt + vet + lint + all tests (needs MongoDB) |

---

## Project Structure

```text
main.go                  Entry point + graceful shutdown
server/                  HTTP server wiring, middleware, logger, validator
api/                     HTTP handlers
service/                 Business logic
repository/
  mongo/                 MongoDB aggregation implementations
model/                   BSON structs
schema/                  Shared query structs (DateRangeFilter)
db/                      MongoDB + Redis connection helpers
config/                  Viper-based TOML config loader
cmd/seed/main.go         Standalone database seeder
conf/                    Config files (sample.toml, default.toml)
```
