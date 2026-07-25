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
    DatabaseURL: "postgres://orbit:orbit@localhost:10332/event_log?sslmode=disable",
})
_ = log.Append(ctx, "notifications", event, "idempotency-key")
```

Create a Postgres database (e.g. `event_log`) and set `DatabaseURL` before use.

## Development

```bash
go test ./...
```

Integration tests skip when Postgres is unavailable or with `-short`.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Security reports: [SECURITY.md](./SECURITY.md).

## License

MIT — see [LICENSE](./LICENSE).
