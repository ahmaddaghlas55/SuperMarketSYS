# Project Folder Structure — Plain-Language Guide

No problem — that's a lot of files to look at cold. Let's zoom out to just the folders, since that's really all you need to hold in your head.

The big picture: every folder maps to one of the three layers we designed way back — request comes in → gets validated → business logic runs → database gets touched. Nothing more complicated than that, just split across files so each piece has one job.

```
cmd/server/main.go     → the "on switch." Starts everything, wires all the pieces together.

internal/handlers/     → the front door. Reads the incoming web request, checks 
                          it's shaped correctly, calls the real logic, sends back JSON.

internal/services/     → the actual brain. All the business rules live here — 
                          "can this sale go through," "how much change is owed," 
                          "does this exceed stock."

internal/repository/   → the only part that talks to the database. Everyone else 
                          asks this layer for data instead of writing SQL themselves.

internal/models/       → just the shape of things — what a "Product" or "Sale" 
                          looks like as data. No logic, just definitions.

internal/middleware/   → the security guard. Checks "are you logged in" and 
                          "are you allowed to do this" before a request even 
                          reaches a handler.

internal/migrations/   → the history of your database's structure — each numbered 
                          .sql file is one step of "add this table" or "add this 
                          column," applied in order.
```

## Why each of `handlers`/`services`/`repository` has multiple files inside it

They're each split by module — `catalog.go`, `purchasing.go`, `operations.go` (sales/debt/cash), `promotions.go`, `reports.go`, `audit.go`, `auth.go`. So if you ever want to find "how does a sale get calculated," you go to `internal/services/operations.go` — same filename, same module, no matter which of the three layer-folders you're looking in.

## Tracing one request through the system

A simple example — "customer buys something":

```
handlers/operations.go   → reads the sale request, checks it's valid JSON
    ↓
services/operations.go   → does the actual math, checks stock, applies rules
    ↓
repository/operations.go → writes the sale to the database
```

## The whole mental model

Three folders you keep bumping into (`handlers`, `services`, `repository`), each holding one file per feature area, plus:
- `models` — data shapes
- `middleware` — login/permissions
- `migrations` — database history

sitting off to the side supporting all of it. `API.md` and the two root files (`0001_init_schema.sql`, `supermarket-system-requirements.md`) are just your reference documents, not code.
