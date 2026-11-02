# Migrations

`orbit-event-log` is a library. It applies its schema from migrations embedded in
the binary (`internal/infrastructure/postgres/migrations`) with
[goose](https://github.com/pressly/goose). There is no hand-written migration
runner.

## Commands

`eventlog.Open` applies every pending migration before it opens the pool. That is
the same behaviour every earlier release had, so a consumer needs no change.

For an explicit run the module ships an operator command:

```bash
# show every embedded migration and the current version
DATABASE_URL='postgres://...' go run github.com/manovaspace/orbit-event-log/cmd/eventlog-migrate status

# apply every pending migration (also what Open does)
DATABASE_URL='postgres://...' go run github.com/manovaspace/orbit-event-log/cmd/eventlog-migrate up

# roll back the most recent migration
DATABASE_URL='postgres://...' go run github.com/manovaspace/orbit-event-log/cmd/eventlog-migrate down
```

There is no `MIGRATE_ON_START` switch: a library has no start-up, and `Open` is
the start of every consumer. A service that wants migrations to be a separate
deployment step calls `Open` after that step; the migration is idempotent, so a
second run is a no-op.

`migrate down` rolls back exactly one migration. Rolling back `001` drops the
`events` table and deletes its rows: that is a data-destroying rollback, not a
deployment rollback.

## Adding a migration

1. Add the next numbered file, for example
   `internal/infrastructure/postgres/migrations/002_add_stream_index.sql`.
   The file must contain `-- +goose Up` first and `-- +goose Down` second.
2. Keep the Down section reversible. If a change cannot be reversed safely
   (dropping a column that holds data), write the Down to restore the object and
   document the manual step in this file.
3. Regenerate the sqlc package when a query or the schema changes:
   `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`. Never edit
   `internal/infrastructure/postgres/sqlcgen` by hand.
4. Run the migration tests against a disposable database (below).
5. Fix a defect in a new migration; never edit a migration that has already run
   in production.

## Production adoption (existing database)

Earlier releases used a small inline runner that executed the `CREATE TABLE IF
NOT EXISTS` statements and created no version table at all. `migrations.Up` lets
goose create `goose_db_version`, then applies `001_event_log.sql`, which uses
`CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS`. On a database
that already has the objects the migration changes nothing and goose records
version 1. If a first goose attempt left an empty `goose_db_version`, the
baseline guard inserts goose's zero row before `Up`, so the same no-op path
applies.

Exact adoption on a production database:

```bash
# 1. Back up first; this also gives the rollback point.
pg_dump "$DATABASE_URL" --format=custom --file eventlog-before-goose.dump

# 2. Inspect the legacy state.
psql "$DATABASE_URL" -c '\d events'
psql "$DATABASE_URL" -c "SELECT to_regclass('goose_db_version')"

# 3. Apply. The first migration is a no-op on the existing objects.
DATABASE_URL="$DATABASE_URL" go run github.com/manovaspace/orbit-event-log/cmd/eventlog-migrate up

# 4. Verify the recorded version and that the objects are unchanged.
psql "$DATABASE_URL" -c 'SELECT version_id, is_applied FROM goose_db_version ORDER BY id'
psql "$DATABASE_URL" -c '\d events'
```

The adoption is covered by `TestAdoptionOnLegacyDatabaseChangesNothing`: it
builds the exact legacy state (including a row), snapshots the `events` table
and index `relfilenode`s, runs `migrate up`, and asserts the snapshot and the row
are unchanged while version 1 is recorded. Run it with:

```bash
docker run -d --rm --name eventlog-pg \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=test \
  -p 55440:5432 postgres:18-alpine
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:55440/test?sslmode=disable' \
  go test ./internal/infrastructure/postgres/migrations/ -race -count=1 -v
```

## Rolling back

Rollback is migration-by-migration:

```bash
# one migration back (here: DROP TABLE events; destructive)
DATABASE_URL="$DATABASE_URL" go run github.com/manovaspace/orbit-event-log/cmd/eventlog-migrate down

# then re-apply
DATABASE_URL="$DATABASE_URL" go run github.com/manovaspace/orbit-event-log/cmd/eventlog-migrate up
```

An application rollback (deploying the previous library version) does not need a
schema rollback when the change followed expand-then-contract. The previous
version ignores new nullable columns and new tables.

To restore a full database from the adoption backup:

```bash
pg_restore --clean --if-exists --dbname "$DATABASE_URL" eventlog-before-goose.dump
```

## Expand, then contract

A breaking schema change takes two releases. Both releases must run against the
same schema, and the schema must be compatible with the release before them.

1. **Expand** (release N): add the new column/table, nullable or with a
   default; write both the old and new shapes; keep the old read path.
2. **Migrate data** (release N or a job): backfill in batches. Do not hold a
   long transaction.
3. **Switch reads** (release N+1): read the new shape, still write both if the
   previous release may still run.
4. **Contract** (release N+2): drop the old column/table in a migration whose
   Down restores it.

A single release never drops an object that the previous release still reads.

## One schema per service

`orbit-event-log` owns the `events` table and its `goose_db_version` in the
`event_log` database. No other service reads these tables; consumers append and
read through this library's `Log` port. That is the "one schema per service"
rule for this library.

If the platform later gives every library a named schema inside a shared
database, the move is:

1. `CREATE SCHEMA event_log;`
2. Set `search_path=event_log,public` for the consumer's DSN.
3. `ALTER TABLE events SET SCHEMA event_log;` (indexes and the
   `events_id_seq` sequence follow; `goose_db_version` moves the same way).
4. Apply the same DSN change in every environment, then remove `public` from the
   search path in a later release.

That move is a deployment task with a maintenance window, not a migration.
