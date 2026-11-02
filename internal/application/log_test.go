package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/manovaspace/orbit-event-log/internal/application"
	"github.com/manovaspace/orbit-event-log/internal/domain"
)

type fakeStore struct {
	appended []recordedAppend
	events   []domain.Event
	readErr  error
	closed   bool
}

type recordedAppend struct {
	stream string
	event  domain.Event
	key    string
}

func (f *fakeStore) Append(_ context.Context, stream string, event domain.Event, key string) error {
	f.appended = append(f.appended, recordedAppend{stream: stream, event: event, key: key})
	return nil
}

func (f *fakeStore) Read(_ context.Context, _ string, _ int64) ([]domain.Event, error) {
	return f.events, f.readErr
}

func (f *fakeStore) Close() error {
	f.closed = true
	return nil
}

type fakeClock struct {
	sleeps int
	fn     func(context.Context) error
}

func (f *fakeClock) Sleep(ctx context.Context, _ time.Duration) error {
	f.sleeps++
	if f.fn != nil {
		return f.fn(ctx)
	}
	return ctx.Err()
}

func validEvent() domain.Event {
	return domain.Event{
		Type:       "test.event.created",
		Payload:    json.RawMessage(`{"n":1}`),
		OccurredAt: time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC),
	}
}

func TestAppendRejectsEmptyStream(t *testing.T) {
	store := &fakeStore{}
	svc := application.New(application.Deps{Store: store, Clock: &fakeClock{}})
	if err := svc.Append(context.Background(), "", validEvent(), ""); err == nil {
		t.Fatal("expected an error for an empty stream")
	}
	if len(store.appended) != 0 {
		t.Fatal("invalid append reached the store")
	}
}

func TestAppendDelegatesValidatedEvent(t *testing.T) {
	store := &fakeStore{}
	svc := application.New(application.Deps{Store: store, Clock: &fakeClock{}})
	if err := svc.Append(context.Background(), "notifications", validEvent(), "key-1"); err != nil {
		t.Fatal(err)
	}
	if len(store.appended) != 1 {
		t.Fatalf("store saw %d appends, want 1", len(store.appended))
	}
	got := store.appended[0]
	if got.stream != "notifications" || got.key != "key-1" || got.event.Type != "test.event.created" {
		t.Fatalf("unexpected append: %+v", got)
	}
}

func TestTailStopsWhenHandlerFails(t *testing.T) {
	want := errors.New("handler failed")
	store := &fakeStore{events: []domain.Event{validEvent()}}
	svc := application.New(application.Deps{Store: store, Clock: &fakeClock{}})
	err := svc.Tail(context.Background(), "notifications", func(context.Context, domain.Event) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Tail error = %v, want %v", err, want)
	}
}

func TestTailPollsEmptyStreamUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &fakeStore{}
	clock := &fakeClock{fn: func(context.Context) error {
		cancel()
		return context.Canceled
	}}
	svc := application.New(application.Deps{Store: store, Clock: clock})
	err := svc.Tail(ctx, "notifications", func(context.Context, domain.Event) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Tail error = %v, want context.Canceled", err)
	}
	if clock.sleeps != 1 {
		t.Fatalf("clock slept %d times, want 1", clock.sleeps)
	}
}

func TestCloseClosesStore(t *testing.T) {
	store := &fakeStore{}
	svc := application.New(application.Deps{Store: store, Clock: &fakeClock{}})
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
	if !store.closed {
		t.Fatal("Close did not close the store")
	}
}
