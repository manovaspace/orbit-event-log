package eventlog_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/manovaspace/orbit-event-log"
)

func TestValidateEvent(t *testing.T) {
	err := eventlog.ValidateEvent(eventlog.Event{
		Type:       "message_created",
		Payload:    json.RawMessage(`{}`),
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateEventRejectsEmptyType(t *testing.T) {
	err := eventlog.ValidateEvent(eventlog.Event{
		Payload:    json.RawMessage(`{}`),
		OccurredAt: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenRequiresDatabaseURL(t *testing.T) {
	if _, err := eventlog.Open(context.Background(), eventlog.Config{}); err == nil {
		t.Fatal("expected an error when DatabaseURL is empty")
	}
}

func TestAppendIdempotencyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	log, err := eventlog.Open(ctx, eventlog.Config{DatabaseURL: dsn})
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	defer func() { _ = log.Close() }()

	stream := "test-" + time.Now().Format("150405")
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
}
