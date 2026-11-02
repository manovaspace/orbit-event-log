---
read_with:
  - handbook/docs/orbit/architecture/data-patterns.md
  - handbook/docs/orbit/architecture/messaging.md
---

# orbit-event-log

Append-only event log (WAL) for Orbit event sourcing — Postgres backend. Status: `beta`. MIT — `github.com/manovaspace/orbit-event-log`.

## Commands

```bash
go test ./...
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:55440/test?sslmode=disable' go test ./... -race -count=1
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
DATABASE_URL='postgres://...' go run ./cmd/eventlog-migrate status   # up, down, status
```

## Architecture

- `internal/domain` — `Event`, `TailHandler`, validation and the `Store`/`Clock` ports. It imports no driver, SQL engine, HTTP server or adapter.
- `internal/application` — append/read/tail use case, built from `application.Deps`.
- `internal/infrastructure/postgres` — sqlc-generated queries and the `domain.Store` adapter, with embedded goose migrations under `postgres/migrations`.
- `internal/infrastructure/clock` — polling clock used by `Tail`.
- `internal/testutil` — disposable-schema integration-test helper.
- `eventlog.go` — the public facade. `Open`, `ValidateEvent`, `Event`, `Log`, `TailHandler` and `Config` keep their previous names and signatures.
- `.golangci.yml` `depguard` rules enforce both boundaries. Do not weaken them to land a change.

## Migrations and SQL

- Embedded goose migrations live in `internal/infrastructure/postgres/migrations/*.sql`. There is no runtime `migrations/` directory.
- `Open` always applies pending migrations before it opens the pool, exactly as previous releases did. The first migration is idempotent, so an existing database adopts goose as a no-op.
- `cmd/eventlog-migrate` runs `up`, `down` or `status` explicitly; `migrate down` is destructive for `001`.
- Adoption, rollback, expand-then-contract and the one-schema rule: [docs/migrations.md](./docs/migrations.md).
- Queries: `internal/infrastructure/postgres/queries/*.sql`; generated package `internal/infrastructure/postgres/sqlcgen` is committed and never edited by hand.

## Events

This library writes an event log. It publishes nothing to NATS and consumes nothing; it does not wire `messagebus.Bus`. The WAL is the durable record a consumer reads.

## Docs

- Public README in this repo
- [docs/migrations.md](./docs/migrations.md)
