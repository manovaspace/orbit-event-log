// Package postgres is the pgx/sqlc adapter for the event-log Store port.
package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/manovaspace/orbit-event-log/internal/domain"
	"github.com/manovaspace/orbit-event-log/internal/infrastructure/postgres/sqlcgen"
)

// Store implements domain.Store on Postgres.
type Store struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
}

// Open connects to databaseURL and returns the store.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool, q: sqlcgen.New(pool)}, nil
}

// Append writes one event. A non-empty idempotency key makes the write a no-op
// when (stream, key) already exists.
func (s *Store) Append(ctx context.Context, stream string, event domain.Event, idempotencyKey string) error {
	occurredAt := pgtype.Timestamptz{Time: event.OccurredAt, Valid: true}
	if idempotencyKey == "" {
		return s.q.AppendEvent(ctx, sqlcgen.AppendEventParams{
			Stream:     stream,
			EventType:  event.Type,
			Payload:    []byte(event.Payload),
			OccurredAt: occurredAt,
		})
	}
	return s.q.AppendEventIdempotent(ctx, sqlcgen.AppendEventIdempotentParams{
		Stream:         stream,
		EventType:      event.Type,
		Payload:        []byte(event.Payload),
		OccurredAt:     occurredAt,
		IdempotencyKey: &idempotencyKey,
	})
}

// Read returns at most 1000 events after fromOffset.
func (s *Store) Read(ctx context.Context, stream string, fromOffset int64) ([]domain.Event, error) {
	rows, err := s.q.ReadEvents(ctx, sqlcgen.ReadEventsParams{
		Stream: stream,
		ID:     fromOffset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Event{
			Offset:      row.ID,
			Stream:      row.Stream,
			Type:        row.EventType,
			Payload:     json.RawMessage(row.Payload),
			OccurredAt:  row.OccurredAt.Time,
			Idempotency: row.IdempotencyKey,
		})
	}
	return out, nil
}

// Close closes the pool.
func (s *Store) Close() error {
	s.pool.Close()
	return nil
}

var _ domain.Store = (*Store)(nil)
