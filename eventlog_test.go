package eventlog_test

import (
	"context"
	"encoding/json"
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
