package statestore

import (
	"testing"

	"synora/pkg/contract"
)

type nestedEventPayload struct {
	Labels   []string
	Metadata map[string][]string
}

func TestStoreReadAPIsReturnIndependentRecords(t *testing.T) {
	store := NewUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](testSnapshot{Value: "initial"})
	commit := testCommit{
		Event:      contract.Event{ID: "alias-event", Type: "test", Payload: map[string]any{"nested": map[string]any{"labels": []any{"original"}}, "typed": &nestedEventPayload{Labels: []string{"original"}, Metadata: map[string][]string{"labels": {"original"}}}}},
		Snapshot:   testSnapshot{Value: "snapshot", Facts: []string{"original"}, Metadata: map[string][]string{"labels": {"original"}}},
		SnapshotV3: &testSnapshotV3{Value: "snapshot-v3", Facts: []string{"original"}, Metadata: map[string][]string{"labels": {"original"}}},
		Decision:   testDecision{Name: "decision", Facts: []string{"original"}, Metadata: map[string][]string{"labels": {"original"}}},
		DecisionV3: &testDecisionV3{Name: "decision-v3", Facts: []string{"original"}, Metadata: map[string][]string{"labels": {"original"}}},
		Action:     &testAction{RequestID: "alias-action", Steps: []string{"original"}, Metadata: map[string][]string{"labels": {"original"}}},
	}
	result, err := store.Commit(commit)
	if err != nil {
		t.Fatal(err)
	}

	commit.Snapshot.Metadata["labels"][0] = "input-mutated"
	commit.Snapshot.Facts[0] = "input-mutated"
	commit.Decision.Metadata["labels"][0] = "input-mutated"
	commit.Event.Payload["nested"].(map[string]any)["labels"].([]any)[0] = "input-mutated"
	commit.Event.Payload["typed"].(*nestedEventPayload).Metadata["labels"][0] = "input-mutated"
	if got := store.Snapshot().Metadata["labels"][0]; got != "original" {
		t.Fatalf("commit input mutated stored snapshot: %q", got)
	}
	if got := store.Journal()[0].Event.Payload["nested"].(map[string]any)["labels"].([]any)[0]; got != "original" {
		t.Fatalf("commit input mutated stored event payload: %v", got)
	}

	snapshot := store.Snapshot()
	snapshot.Metadata["labels"][0] = "getter-mutated"
	snapshot.Facts[0] = "getter-mutated"
	if got := store.Snapshot().Metadata["labels"][0]; got != "original" {
		t.Fatalf("snapshot getter exposed state: %q", got)
	}

	decisions := store.Decisions()
	decisions[0].Metadata["labels"][0] = "getter-mutated"
	decisions[0].Facts[0] = "getter-mutated"
	if got := store.Decisions()[0].Metadata["labels"][0]; got != "original" {
		t.Fatalf("decision getter exposed state: %q", got)
	}

	journal := store.Journal()
	journal[0].Event.Payload["nested"].(map[string]any)["labels"].([]any)[0] = "getter-mutated"
	journal[0].Event.Payload["typed"].(*nestedEventPayload).Labels[0] = "getter-mutated"
	journal[0].Snapshot.Metadata["labels"][0] = "getter-mutated"
	journal[0].DecisionV3.Metadata["labels"][0] = "getter-mutated"
	if got := store.Journal()[0].Event.Payload["nested"].(map[string]any)["labels"].([]any)[0]; got != "original" {
		t.Fatalf("journal getter exposed event payload: %v", got)
	}
	if got := store.Journal()[0].DecisionV3.Metadata["labels"][0]; got != "original" {
		t.Fatalf("journal getter exposed V3 decision: %q", got)
	}

	outbox := store.ActionOutbox()
	outbox[0].Metadata["labels"][0] = "getter-mutated"
	outbox[0].Steps[0] = "getter-mutated"
	if got := store.ActionOutbox()[0].Metadata["labels"][0]; got != "original" {
		t.Fatalf("outbox getter exposed state: %q", got)
	}

	if result.Action == nil {
		t.Fatal("commit result action missing")
	}
	result.Action.Metadata["labels"][0] = "result-mutated"
	if got := store.ActionOutbox()[0].Metadata["labels"][0]; got != "original" {
		t.Fatalf("commit result exposed outbox state: %q", got)
	}

	v3, ok := store.SnapshotV3()
	if !ok {
		t.Fatal("V3 snapshot missing")
	}
	v3.Metadata["labels"][0] = "getter-mutated"
	v3.Facts[0] = "getter-mutated"
	storedV3, _ := store.SnapshotV3()
	if got := storedV3.Metadata["labels"][0]; got != "original" {
		t.Fatalf("V3 getter exposed state: %q", got)
	}
}
