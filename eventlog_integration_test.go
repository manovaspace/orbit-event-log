package eventlog_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/manovaspace/orbit-event-log"
	"github.com/manovaspace/orbit-event-log/internal/testutil"
)

func TestAppendIdempotencyIntegration(t *testing.T) {
	db := testutil.OpenPostgres(t)
	ctx := context.Background()
	log, err := eventlog.Open(ctx, eventlog.Config{DatabaseURL: db.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = log.Close() }()

	stream := "test-stream"
	ev := eventlog.Event{
		Type:       "test.event.created",
		Payload:    json.RawMessage(`{"n":1}`),
		OccurredAt: time.Now().UTC(),
	}
	if err := log.Append(ctx, stream, ev, "idem-1"); err != nil {
		t.Fatal(err)
	}
	if err := log.Append(ctx, stream, ev, "idem-1"); err != nil {
		t.Fatal(err)
	}
	events, err := log.Read(ctx, stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Idempotency != "idem-1" {
		t.Fatalf("idempotency = %q, want idem-1", events[0].Idempotency)
	}
}
