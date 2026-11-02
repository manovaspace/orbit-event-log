# orbit-event-log

[![CI](https://github.com/manovaspace/orbit-event-log/actions/workflows/ci.yml/badge.svg)](https://github.com/manovaspace/orbit-event-log/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

Go library: append-only event log (WAL) for Orbit event sourcing — Postgres backend.

Part of the [Manova / Orbit](https://github.com/manovaspace) open toolkit.

## Install

```bash
go get github.com/manovaspace/orbit-event-log@latest
```

## Usage

```go
import eventlog "github.com/manovaspace/orbit-event-log"

log, err := eventlog.Open(ctx, eventlog.Config{
    DatabaseURL: os.Getenv("DATABASE_URL"), // required
})
_ = log.Append(ctx, "notifications", event, "idempotency-key")
```

`DatabaseURL` is required: `Open` returns an error when it is empty. In the Orbit
stack the `event_log` database is provisioned on the shared Postgres instance
(`postgres/databases.tsv` in `orbit-infra`).

`Open` applies the embedded goose migrations before it opens the pool. The first
migration is idempotent, so a database created by an earlier release adopts
goose without changing its objects. For an explicit run use the operator
command:

```bash
DATABASE_URL='postgres://...' go run github.com/manovaspace/orbit-event-log/cmd/eventlog-migrate status
DATABASE_URL='postgres://...' go run github.com/manovaspace/orbit-event-log/cmd/eventlog-migrate up
```

See [docs/migrations.md](./docs/migrations.md) for adoption, rollback and the
expand-then-contract rule.

## Layout

```
eventlog.go                                  public facade (unchanged API)
cmd/eventlog-migrate/                        explicit up/down/status command
internal/domain/                             Event, validation, Store and Clock ports
internal/application/                        append/read/tail use case
internal/infrastructure/postgres/            Store adapter, sqlc queries, generated code
internal/infrastructure/postgres/migrations/ embedded goose migrations
internal/infrastructure/clock/               polling clock
internal/testutil/                           disposable-schema integration-test helper
```

The domain imports no infrastructure package. The application layer receives
every dependency through `application.Deps`, and the exported facade only wires
adapters.

## Development

```bash
go test ./...
```

Integration tests run against a disposable database and skip without it:

```bash
docker run -d --rm --name eventlog-pg \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=test \
  -p 55440:5432 postgres:18-alpine

TEST_DATABASE_URL='postgres://postgres:postgres@localhost:55440/test?sslmode=disable' \
  go test ./... -race -count=1

docker stop eventlog-pg
```

`TEST_DATABASE_URL` is accepted only when the connected database name starts
with `test`; each test creates and drops its own schema.

Regenerate the sqlc package after changing SQL:

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
```

Generated code in `internal/infrastructure/postgres/sqlcgen` is committed and
must never be edited by hand.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Security reports: [SECURITY.md](./SECURITY.md).

## License

MIT — see [LICENSE](./LICENSE).
