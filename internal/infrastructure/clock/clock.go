// Package clock provides the real clock adapter.
package clock

import (
	"context"
	"time"
)

// Clock is the system clock.
type Clock struct{}

// New returns a system clock.
func New() *Clock { return &Clock{} }

// Sleep waits for d or returns early when ctx is cancelled.
func (*Clock) Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
