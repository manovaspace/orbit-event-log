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

`DatabaseURL` is required: `Open` returns an error when it is empty. In the Orbit stack the `event_log` database is provisioned on the shared Postgres instance (`postgres/databases.tsv` in `orbit-infra`).

## Development

```bash
go test ./...
```

Integration tests need `TEST_DATABASE_URL` and skip when it is unset or with `-short`.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Security reports: [SECURITY.md](./SECURITY.md).

## License

MIT — see [LICENSE](./LICENSE).
