package eventlog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/manovaspace/orbit-event-log/internal/application"
	"github.com/manovaspace/orbit-event-log/internal/domain"
	"github.com/manovaspace/orbit-event-log/internal/infrastructure/clock"
	"github.com/manovaspace/orbit-event-log/internal/infrastructure/postgres"
	"github.com/manovaspace/orbit-event-log/internal/infrastructure/postgres/migrations"
)

// Event is an immutable WAL entry.
type Event struct {
	Offset      int64           `json:"offset"`
	Stream      string          `json:"stream"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Idempotency string          `json:"idempotency_key,omitempty"`
}

// TailHandler receives events from Tail.
type TailHandler func(ctx context.Context, event Event) error

// Log is the append-only event log port.
type Log interface {
	Append(ctx context.Context, stream string, event Event, idempotencyKey string) error
	Read(ctx context.Context, stream string, fromOffset int64) ([]Event, error)
	Tail(ctx context.Context, stream string, handler TailHandler) error
	Close() error
}

// Config holds Postgres connection options.
type Config struct {
	DatabaseURL string
}

// ValidateEvent checks required event fields before append.
func ValidateEvent(e Event) error {
	return domain.ValidateEvent(toDomainEvent(e))
}

// Open connects to Postgres and returns an event Log.
func Open(ctx context.Context, cfg Config) (Log, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("eventlog: DatabaseURL is required")
	}
	if err := migrations.Up(ctx, cfg.DatabaseURL); err != nil {
		return nil, err
	}
	store, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	return &logAdapter{svc: application.New(application.Deps{
		Store: store,
		Clock: clock.New(),
	})}, nil
}

// logAdapter keeps the exported types in this package and translates them to
// the domain types the application layer uses. The shapes are identical, so the
// mapping is a field copy.
type logAdapter struct {
	svc *application.Service
}

func (a *logAdapter) Append(ctx context.Context, stream string, event Event, idempotencyKey string) error {
	return a.svc.Append(ctx, stream, toDomainEvent(event), idempotencyKey)
}

func (a *logAdapter) Read(ctx context.Context, stream string, fromOffset int64) ([]Event, error) {
	events, err := a.svc.Read(ctx, stream, fromOffset)
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(events))
	for _, e := range events {
		out = append(out, fromDomainEvent(e))
	}
	return out, nil
}

func (a *logAdapter) Tail(ctx context.Context, stream string, handler TailHandler) error {
	return a.svc.Tail(ctx, stream, func(ctx context.Context, event domain.Event) error {
		return handler(ctx, fromDomainEvent(event))
	})
}

func (a *logAdapter) Close() error {
	return a.svc.Close()
}

func toDomainEvent(e Event) domain.Event {
	return domain.Event{
		Offset:      e.Offset,
		Stream:      e.Stream,
		Type:        e.Type,
		Payload:     e.Payload,
		OccurredAt:  e.OccurredAt,
		Idempotency: e.Idempotency,
	}
}

func fromDomainEvent(e domain.Event) Event {
	return Event{
		Offset:      e.Offset,
		Stream:      e.Stream,
		Type:        e.Type,
		Payload:     e.Payload,
		OccurredAt:  e.OccurredAt,
		Idempotency: e.Idempotency,
	}
}

var _ Log = (*logAdapter)(nil)
