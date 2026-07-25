package eventlog

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type pgLog struct {
	pool *pgxpool.Pool
}

// Open connects to Postgres and returns an event Log.
func Open(ctx context.Context, cfg Config) (Log, error) {
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "postgres://orbit:orbit@localhost:10332/event_log?sslmode=disable"
	}
	if err := migrate(ctx, cfg.DatabaseURL); err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	return &pgLog{pool: pool}, nil
}

func (s *pgLog) Append(ctx context.Context, stream string, event Event, idempotencyKey string) error {
	if stream == "" {
		return fmt.Errorf("eventlog: stream required")
	}
	if err := ValidateEvent(event); err != nil {
		return err
	}
	if idempotencyKey != "" {
		tag, err := s.pool.Exec(ctx, `
			INSERT INTO events (stream, event_type, payload, occurred_at, idempotency_key)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (stream, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
		`, stream, event.Type, event.Payload, event.OccurredAt, idempotencyKey)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO events (stream, event_type, payload, occurred_at, idempotency_key)
		VALUES ($1, $2, $3, $4, NULL)
	`, stream, event.Type, event.Payload, event.OccurredAt)
	return err
}

func (s *pgLog) Read(ctx context.Context, stream string, fromOffset int64) ([]Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, stream, event_type, payload, occurred_at, COALESCE(idempotency_key, '')
		FROM events
		WHERE stream = $1 AND id > $2
		ORDER BY id ASC
		LIMIT 1000
	`, stream, fromOffset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.Offset, &e.Stream, &e.Type, &e.Payload, &e.OccurredAt, &e.Idempotency); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *pgLog) Tail(ctx context.Context, stream string, handler TailHandler) error {
	var lastOffset int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		events, err := s.Read(ctx, stream, lastOffset)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			// ponytail: polling; upgrade to LISTEN/NOTIFY when lag matters
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, e := range events {
			if err := handler(ctx, e); err != nil {
				return err
			}
			lastOffset = e.Offset
		}
	}
}

func (s *pgLog) Close() error {
	s.pool.Close()
	return nil
}

func migrate(ctx context.Context, databaseURL string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	b, err := migrationFiles.ReadFile("migrations/001_event_log.sql")
	if err != nil {
		return err
	}
	sql := string(b)
	const marker = "-- +goose Up"
	idx := -1
	for i := 0; i+len(marker) <= len(sql); i++ {
		if sql[i:i+len(marker)] == marker {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("migration marker not found")
	}
	up := sql[idx+len(marker):]
	const down = "-- +goose Down"
	for i := 0; i+len(down) <= len(up); i++ {
		if up[i:i+len(down)] == down {
			up = up[:i]
			break
		}
	}
	_, err = pool.Exec(ctx, up)
	return err
}

var _ Log = (*pgLog)(nil)
