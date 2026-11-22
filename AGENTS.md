---
see_also:
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

<!-- manova:conventions v1 — generated from manovaspace/ci conventions/core.md; edit it there -->
## Shared conventions

These rules apply to every Manova repository. They are copied from `manovaspace/ci` (`conventions/core.md`); change them there, not here.

### Delivery

1. Never push to `main`. Every change goes on a topic branch (`feat/<name>`, `fix/<name>`, `refactor/<name>`, `docs/<name>`, `chore/<name>`) and reaches `main` through a pull request.
2. One pull request does one thing. An unrelated problem you find on the way becomes an issue, not part of this change.
3. Pull request titles and commit messages follow Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`, `chore:`, `test:`; `!` for a breaking change). Releases are computed from them.
4. Merge with squash once the checks are green. Do not use `--no-verify`; if you ever must, say so in the pull request.

### Checks

5. CI runs `manovaspace/ci@v1`. Run the same gate locally before you push; a red check is fixed, not skipped.
6. Tool versions (Go, Node, Bun, linters) come from `ci/versions.env` in `manovaspace/ci`. Do not pin a different version in this repository.
7. A behaviour change comes with a test that fails without it. Do not delete or weaken a test to make a change pass.

### Secrets

8. Never commit a secret, token, password, private key or `.env` file. Commit `.env.example` with placeholders instead.
9. Never print a secret in logs, test output, error messages or pull requests. Read secrets from the environment or a file at run time.

### Runtime

10. Containers talk to each other by name on a Docker network. Do not publish a port on the host unless a person or a host process needs it, and then bind `127.0.0.1`, never `0.0.0.0`.
11. Configuration comes from the environment. A service with a missing required setting refuses to start; it never falls back to a built-in secret or a development default in production.
12. Port numbers come from the workspace port registry. Do not invent a port.

### Documentation

13. This repository explains itself: `README.md` says how to set up, run, test and deploy it; `AGENTS.md` holds its binding rules; repository decisions live in `docs/decisions/`.
14. Write plain, short sentences. Keep exact names, paths, commands and versions. Do not document what the code already says.
<!-- /manova:conventions -->
