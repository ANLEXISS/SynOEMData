package cognitivecore

import (
	"context"
	"testing"
	"time"

	"synora/pkg/contract"
)

func TestH1TemporalBehaviorUsesEventTimeAndArrivalOrder(t *testing.T) {
	store, err := OpenUniversalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	core := durableTestCore(store)
	logicalNow := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	core.Now = func() time.Time { return logicalNow }
	late := logicalNow.Add(10 * time.Second)
	early := logicalNow.Add(5 * time.Second)

	first := contract.Event{ID: "temporal-late", Type: "sensor.motion", Source: "fixture", Timestamp: late, ReceivedAt: logicalNow.Add(11 * time.Second), Epoch: "epoch-a", Sequence: 40, Payload: map[string]any{"movement": true}}
	if _, err := core.Process(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	// A repeated sequence is not a duplicate when the event ID differs. The
	// Store's idempotency key is the event ID; it does not enforce ordering.
	second := contract.Event{ID: "temporal-early", Type: "sensor.motion", Source: "fixture", Timestamp: early, ReceivedAt: logicalNow.Add(12 * time.Second), Epoch: "epoch-b", Sequence: 40, Payload: map[string]any{"movement": true}}
	if _, err := core.Process(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	priorRevision := store.Revision()
	priorHistory, err := store.History()
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := core.Process(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if !duplicate.Result.Duplicate || store.Revision() != priorRevision {
		t.Fatalf("event-ID duplicate behavior changed: result=%#v revision=%d", duplicate.Result, store.Revision())
	}
	history, err := store.History()
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != len(priorHistory) || len(history) < 2 || history[0].Event.ID != first.ID || history[1].Event.ID != second.ID {
		t.Fatalf("history is not commit/arrival ordered: %#v", history)
	}
	if !history[1].Event.Timestamp.Equal(early) || !history[1].Event.ReceivedAt.Equal(second.ReceivedAt) || history[1].Event.Epoch != "epoch-b" || history[1].Event.Sequence != 40 {
		t.Fatalf("event temporal metadata changed: %#v", history[1].Event)
	}
	if !history[1].Snapshot.CapturedAt.Equal(early) {
		t.Fatalf("snapshot should use event time, got %s want %s", history[1].Snapshot.CapturedAt, early)
	}
}

func TestH1MissingEventTimeUsesInjectedClockWithoutStampingReceivedAt(t *testing.T) {
	store, err := OpenUniversalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	core := durableTestCore(store)
	logicalNow := time.Date(2026, 10, 11, 12, 30, 0, 0, time.UTC)
	core.Now = func() time.Time { return logicalNow }
	event := contract.Event{ID: "temporal-missing-time", Type: "sensor.motion", Source: "fixture", Epoch: "epoch-a", Sequence: 1, Payload: map[string]any{"movement": true}}
	if _, err := core.Process(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	history, err := store.History()
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || !history[0].Event.Timestamp.Equal(logicalNow) || !history[0].Snapshot.CapturedAt.Equal(logicalNow) {
		t.Fatalf("missing event timestamp fallback changed: %#v", history)
	}
	if !history[0].Event.ReceivedAt.IsZero() {
		t.Fatalf("Core unexpectedly synthesized received_at: %s", history[0].Event.ReceivedAt)
	}
}
