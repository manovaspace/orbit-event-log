package eventlog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
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
	if e.Type == "" {
		return fmt.Errorf("eventlog: missing event type")
	}
	if e.Payload == nil {
		return fmt.Errorf("eventlog: missing payload")
	}
	if !json.Valid(e.Payload) {
		return fmt.Errorf("eventlog: payload must be valid JSON")
	}
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("eventlog: missing occurred_at")
	}
	return nil
}
