// Package application is the append/read/tail use case. Every dependency is a
// domain port injected through Deps; it reads no environment variable and
// imports no infrastructure package.
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/manovaspace/orbit-event-log/internal/domain"
)

// tailPollInterval keeps the previous release's polling cadence.
const tailPollInterval = 500 * time.Millisecond

// Deps are the application service's ports.
type Deps struct {
	Store domain.Store
	Clock domain.Clock
}

// Service implements the event log use cases.
type Service struct {
	store domain.Store
	clock domain.Clock
}

// New builds the service from its ports.
func New(deps Deps) *Service {
	return &Service{store: deps.Store, clock: deps.Clock}
}

// Append validates one event and writes it to the store.
func (s *Service) Append(ctx context.Context, stream string, event domain.Event, idempotencyKey string) error {
	if stream == "" {
		return fmt.Errorf("eventlog: stream required")
	}
	if err := domain.ValidateEvent(event); err != nil {
		return err
	}
	return s.store.Append(ctx, stream, event, idempotencyKey)
}

// Read returns at most 1000 events after fromOffset.
func (s *Service) Read(ctx context.Context, stream string, fromOffset int64) ([]domain.Event, error) {
	return s.store.Read(ctx, stream, fromOffset)
}

// Tail polls the store and hands every new event to handler until the context
// is cancelled.
func (s *Service) Tail(ctx context.Context, stream string, handler domain.TailHandler) error {
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
			// Polling keeps the previous behaviour; upgrade to LISTEN/NOTIFY
			// when lag matters.
			if err := s.clock.Sleep(ctx, tailPollInterval); err != nil {
				return err
			}
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

// Close releases the store.
func (s *Service) Close() error {
	return s.store.Close()
}
