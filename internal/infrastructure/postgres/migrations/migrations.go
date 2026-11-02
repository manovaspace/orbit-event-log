// Package migrations owns the embedded goose migration files and applies them
// through goose's Provider API. The exported API of this module does not expose
// the migrator; eventlog.Open calls Up, exactly as previous releases applied
// their schema at Open.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

const defaultTableName = "goose_db_version"

// Migrator applies the embedded migrations to one database.
type Migrator struct {
	db        *sql.DB
	provider  *goose.Provider
	tableName string
}

// Up applies every pending migration to databaseURL. Existing databases adopt
// the baseline: the first migration is idempotent, so goose records version 1
// without touching the objects the previous runner created.
func Up(ctx context.Context, databaseURL string) error {
	m, err := Open(databaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()
	if _, err := m.apply(ctx); err != nil {
		return err
	}
	return nil
}

// Open connects to databaseURL and prepares the embedded migrations. The
// returned Migrator must be closed by the caller.
func Open(databaseURL string) (*Migrator, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("migrations: open database: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, files)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrations: prepare embedded migrations: %w", err)
	}
	return &Migrator{db: db, provider: provider, tableName: defaultTableName}, nil
}

// apply runs Up after adopting a legacy empty version table, if any.
func (m *Migrator) apply(ctx context.Context) ([]*goose.MigrationResult, error) {
	if err := m.ensureBaseline(ctx); err != nil {
		return nil, err
	}
	results, err := m.provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrations: up: %w", err)
	}
	return results, nil
}

// Up applies every pending migration and returns the ones it ran.
func (m *Migrator) Up(ctx context.Context) ([]*goose.MigrationResult, error) {
	return m.apply(ctx)
}

// Down rolls back the most recently applied migration.
func (m *Migrator) Down(ctx context.Context) (*goose.MigrationResult, error) {
	if err := m.ensureBaseline(ctx); err != nil {
		return nil, err
	}
	result, err := m.provider.Down(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrations: down: %w", err)
	}
	return result, nil
}

// Status reports every embedded migration and whether it is applied.
func (m *Migrator) Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
	if err := m.ensureBaseline(ctx); err != nil {
		return nil, err
	}
	status, err := m.provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrations: status: %w", err)
	}
	return status, nil
}

// Version returns the current and latest migration versions.
func (m *Migrator) Version(ctx context.Context) (current, target int64, err error) {
	if err := m.ensureBaseline(ctx); err != nil {
		return 0, 0, err
	}
	current, target, err = m.provider.GetVersions(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("migrations: version: %w", err)
	}
	return current, target, nil
}

// ensureBaseline adopts a version table that exists but has no rows. The
// previous runner created no version table at all, so this normally does
// nothing; the guard covers a database that a first goose attempt left with an
// empty table.
func (m *Migrator) ensureBaseline(ctx context.Context) error {
	var exists bool
	if err := m.db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, m.tableName).Scan(&exists); err != nil {
		return fmt.Errorf("migrations: inspect version table: %w", err)
	}
	if !exists {
		return nil
	}
	var count int
	if err := m.db.QueryRowContext(ctx, `SELECT count(*) FROM `+m.tableName).Scan(&count); err != nil {
		return fmt.Errorf("migrations: read version table: %w", err)
	}
	if count > 0 {
		return nil
	}
	if _, err := m.db.ExecContext(ctx,
		`INSERT INTO `+m.tableName+` (version_id, is_applied)
		 SELECT 0, TRUE WHERE NOT EXISTS (SELECT 1 FROM `+m.tableName+`)`); err != nil {
		return fmt.Errorf("migrations: record baseline version: %w", err)
	}
	return nil
}

// Close closes the migration database handle.
func (m *Migrator) Close() error {
	return m.db.Close()
}
