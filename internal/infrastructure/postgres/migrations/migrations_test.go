package migrations

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/manovaspace/orbit-event-log/internal/testutil"
)

// legacyStatements reproduce the state the previous custom runner left behind:
// the events table and both indexes, and no goose_db_version at all.
var legacyStatements = []string{
	`CREATE TABLE IF NOT EXISTS events (
		id BIGSERIAL PRIMARY KEY,
		stream TEXT NOT NULL,
		event_type TEXT NOT NULL,
		payload JSONB NOT NULL,
		occurred_at TIMESTAMPTZ NOT NULL,
		idempotency_key TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_events_stream_idempotency
		ON events (stream, idempotency_key)
		WHERE idempotency_key IS NOT NULL`,
	`CREATE INDEX IF NOT EXISTS idx_events_stream_id ON events (stream, id)`,
}

func TestFreshDatabaseAppliesEmbeddedMigrations(t *testing.T) {
	db := testutil.OpenPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := Open(db.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()

	results, err := m.Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Source.Version != 1 {
		t.Fatalf("applied %d migrations, want version 1 only", len(results))
	}
	if !tableExists(t, ctx, db.Pool, db.Schema, "events") {
		t.Fatal("events was not created")
	}
	if current, target, err := m.Version(ctx); err != nil || current != 1 || target != 1 {
		t.Fatalf("version = %d/%d, err=%v, want 1/1", current, target, err)
	}
	assertGooseRow(t, ctx, db.Pool, db.Schema, 1)
}

func TestAdoptionOnLegacyDatabaseChangesNothing(t *testing.T) {
	db := testutil.OpenPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, statement := range legacyStatements {
		if _, err := db.Pool.Exec(ctx, statement); err != nil {
			t.Fatalf("apply legacy statement: %v", err)
		}
	}
	// A row proves the adoption does not drop or recreate the table.
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO events (stream, event_type, payload, occurred_at, idempotency_key)
		VALUES ('legacy', 'legacy.created', '{"legacy":true}'::jsonb, NOW(), 'legacy-key')
	`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	before := schemaSnapshot(t, ctx, db.Pool, db.Schema)

	m, err := Open(db.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()

	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("adopt legacy database: %v", err)
	}
	after := schemaSnapshot(t, ctx, db.Pool, db.Schema)

	if before != after {
		t.Fatalf("adoption changed the schema:\nbefore: %s\nafter:  %s", before, after)
	}
	var idempotencyKey string
	if err := db.Pool.QueryRow(ctx, `
		SELECT idempotency_key FROM events WHERE stream = 'legacy'
	`).Scan(&idempotencyKey); err != nil {
		t.Fatalf("read legacy row after adoption: %v", err)
	}
	if idempotencyKey != "legacy-key" {
		t.Fatalf("legacy row idempotency_key = %q, want legacy-key", idempotencyKey)
	}
	if current, target, err := m.Version(ctx); err != nil || current != 1 || target != 1 {
		t.Fatalf("version = %d/%d, err=%v, want 1/1", current, target, err)
	}
	assertGooseRow(t, ctx, db.Pool, db.Schema, 1)
}

func TestDownRemovesTheMigrationAndUpReappliesIt(t *testing.T) {
	db := testutil.OpenPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := Open(db.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()

	if _, err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	result, err := m.Down(ctx)
	if err != nil {
		t.Fatalf("down: %v", err)
	}
	if result.Source.Version != 1 {
		t.Fatalf("rolled back version %d, want 1", result.Source.Version)
	}
	if tableExists(t, ctx, db.Pool, db.Schema, "events") {
		t.Fatal("down left events in place")
	}
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("reapply after down: %v", err)
	}
	if !tableExists(t, ctx, db.Pool, db.Schema, "events") {
		t.Fatal("up after down did not recreate events")
	}
}

// schemaSnapshot covers the objects the previous runner created. A relfilenode
// changes when a table or index is dropped and recreated, so equal snapshots
// prove IF NOT EXISTS was a no-op.
func schemaSnapshot(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schema string) string {
	t.Helper()
	const query = `
		SELECT c.relname || ':' || c.relfilenode::text || ':' || c.relkind::text
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
		  AND c.relname IN ('events', 'idx_events_stream_idempotency', 'idx_events_stream_id')
		ORDER BY c.relname`
	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		t.Fatalf("read schema snapshot: %v", err)
	}
	defer rows.Close()
	var out string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		out += line + "\n"
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	const columnsQuery = `
		SELECT c.relname || '.' || a.attname || ':' || format_type(a.atttypid, a.atttypmod)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid
		WHERE n.nspname = $1 AND c.relname = 'events' AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`
	columnRows, err := pool.Query(ctx, columnsQuery, schema)
	if err != nil {
		t.Fatalf("read column snapshot: %v", err)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		var line string
		if err := columnRows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		out += line + "\n"
	}
	if err := columnRows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func tableExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schema, name string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1 AND c.relname = $2 AND c.relkind IN ('r', 'p')
		)`, schema, name).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}

func assertGooseRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schema string, wantVersion int64) {
	t.Helper()
	var version int64
	var applied bool
	if err := pool.QueryRow(ctx, `
		SELECT version_id, is_applied FROM "`+schema+`".goose_db_version ORDER BY id DESC LIMIT 1
	`).Scan(&version, &applied); err != nil {
		t.Fatalf("read goose_db_version: %v", err)
	}
	if version != wantVersion || !applied {
		t.Fatalf("goose row = version %d applied %v, want %d/true", version, applied, wantVersion)
	}
}
