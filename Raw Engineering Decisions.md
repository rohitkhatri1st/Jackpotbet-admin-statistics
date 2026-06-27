# Code Process

These are my raw notes from when I was writing this project — what I built, in what order, and why I made the choices I made. Written as I went, so the language is informal and sometimes incomplete. No cleanup done intentionally.

The `Engineering Decisions.html` file was created later using this file as the primary reference. That document is more structured and detailed, but everything in it came from the thinking captured here. This file is entirely my own — no AI involvement.

---

## Getting Started

Created the assignment markdown first to understand what I was building.

Init git, init go mod. Created the initial folder structure with empty files. The goal was to build something scalable and maintainable even though the assignment itself could have been solved much more simply. I wanted to do it properly.

Wrote `main.go`.

---

## Architecture Decisions

Chose 3-layer architecture: Handler → Service → Repository.

The reasoning: handlers deal with HTTP parsing, services hold business logic, repository is where DB interaction lives. If this were a simple CRUD project I would have skipped the service layer. If it needed cross-system event propagation I would have added a domain/event layer. Neither was the case here — the operations are self-contained and read-heavy, so three layers is the right fit.

Chose the repository pattern specifically because we are writing MongoDB aggregation pipelines and those pipelines need somewhere to live that isn't a handler or a service.

Following SOLID principles throughout — mainly to keep complexity low, not for any ceremonial reason.

Created the server package. Created the logger. Built the structure with EC2 deployment in mind (no assumptions about filesystem paths, config loaded from known search paths, graceful shutdown wired from the start).

---

## Config and Infrastructure

Init DBs (MongoDB + Redis). Init empty services as a scaffold.

Chose **viper** over burntsushi/toml and other parsers because viper does multi-path config search natively — it finds the right config file whether you run from the repo root or from `cmd/seed/`. With burntsushi you'd have to build that yourself. The one trade-off: viper silently ignores unknown config keys; burntsushi errors on them. I accepted that trade-off given the config is small and straightforward.

Chose **gorilla/mux** over Gin, Iris, or other full frameworks. The reason: I wanted to use `net/http` types directly and keep full control. Gin and Echo wrap everything in their own context, which means handlers written for them aren't portable. Also, since I was using `go-playground/validator` for input validation, I wouldn't be using Gin's built-in validation anyway — so Gin would add lock-in with no benefit.

---

## Making it Swappable

Created all services and repositories as interfaces. Everything is designed to be mockable and swappable — for testing, for infra changes, for swapping out MongoDB for something else down the line. Out of scope for this assignment but the right thing to do anyway.

Verified the logger was also behind an interface so it could be swapped without touching any call site.

---

## Routes and Handlers

Started with a create-transaction route so the same code could be reused by the seeder script. Didn't want to write the insert logic twice.

Wrote a custom validator package wrapping `go-playground/validator` for better control over error formatting. Wrote `decodeJsonBody` for consistent JSON decoding and error handling across all handlers. Wrote structured `respond`/`respondError` helpers so every API returns the same shape.

Created the GET transactions list API to test the creation flow and establish the handler pattern that everything else would follow.

Created a shared `DateRangeFilter` schema so every stat endpoint could embed it and get consistent `from`/`to` query param handling for free.

---

## The Seeder

Created a bulk insert repository method first. Then built the seeder script (`cmd/seed/main.go`) as a three-stage concurrent pipeline:

- Stage 1 (Scheduler): slices 2M rounds into batches
- Stage 2 (Builder): generates the actual transaction data in parallel
- Stage 3 (Writer): fires concurrent InsertMany calls to MongoDB

Breaking it into stages made it easy to reason about and tune independently. Each stage runs in its own goroutines and communicates through channels.

---

## Stat APIs

Built `gross_gaming_rev` first. Added `EnsureIndexes` during this step and did minor refactoring to support it cleanly.

Added CLAUDE.md rules and context documentation here to make the codebase more navigable.

Built `daily_wager_volume` next.

---

## Caching

Added Redis TTL-based caching to both stat endpoints.

I am using normal Redis key-based caching for now. A better approach would be a materialized pattern — introduce a `daily_stats` MongoDB collection, pre-compute daily wagers into it, run a nightly cron to keep it fresh, and serve GGR and daily volume from that collection instead of running aggregations on 4M rows every time. For the sake of simplicity I haven't done that. The complexity of handling backdated corrections and coordinating cache invalidation with the cron made it not worth it for this assignment. May introduce it later.

---

## UTC and Wager Percentile

Made time handling UTC throughout — inputs normalized on the way in, outputs returned as UTC, seeder generates UTC timestamps.

Built the `GET /user/{userId}/wager_percentile` API. Used MongoDB `$setWindowFields` + `$facet` so the rank is assigned inside the aggregation and only one document comes back to the application, regardless of how many users are in the dataset.

Also added Redis caching to the percentile endpoint.

---

## What's Next

Plan to introduce proper materialized view caching and write mocks and tests.

## Testing

Writing integration tests for user wager api
Writing integration tests for Gross Gaming Revenue, Daily Wager Percentile

Note: All the test cases are written using Claude

## Makefile

Creating a makefile to easily run tests and the project as well.
