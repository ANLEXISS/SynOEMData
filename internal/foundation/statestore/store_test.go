package statestore

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"synora/pkg/contract"
)

type testSnapshot struct {
	Revision uint64              `json:"revision"`
	Value    string              `json:"value"`
	Facts    []string            `json:"facts,omitempty"`
	Metadata map[string][]string `json:"metadata,omitempty"`
}

func (s testSnapshot) Clone() testSnapshot {
	s.Facts = append([]string(nil), s.Facts...)
	s.Metadata = cloneTestMetadata(s.Metadata)
	return s
}
func (s testSnapshot) Normalized() testSnapshot                  { return s.Clone() }
func (s testSnapshot) Validate() error                           { return nil }
func (s testSnapshot) WithRevision(revision uint64) testSnapshot { s.Revision = revision; return s }
func (s testSnapshot) RevisionValue() uint64                     { return s.Revision }

type testSnapshotV3 struct {
	Revision uint64              `json:"revision"`
	Value    string              `json:"value"`
	Facts    []string            `json:"facts,omitempty"`
	Metadata map[string][]string `json:"metadata,omitempty"`
}

func (s testSnapshotV3) Clone() testSnapshotV3 {
	s.Facts = append([]string(nil), s.Facts...)
	s.Metadata = cloneTestMetadata(s.Metadata)
	return s
}
func (s testSnapshotV3) Normalized() testSnapshotV3                  { return s.Clone() }
func (s testSnapshotV3) Validate() error                             { return nil }
func (s testSnapshotV3) WithRevision(revision uint64) testSnapshotV3 { s.Revision = revision; return s }
func (s testSnapshotV3) RevisionValue() uint64                       { return s.Revision }

type testDecision struct {
	Name     string              `json:"name"`
	Facts    []string            `json:"facts,omitempty"`
	Metadata map[string][]string `json:"metadata,omitempty"`
}
type testDecisionV3 struct {
	Name     string              `json:"name"`
	Facts    []string            `json:"facts,omitempty"`
	Metadata map[string][]string `json:"metadata,omitempty"`
}

func (d testDecision) Clone() testDecision {
	d.Facts = append([]string(nil), d.Facts...)
	d.Metadata = cloneTestMetadata(d.Metadata)
	return d
}
func (d testDecisionV3) Clone() testDecisionV3 {
	d.Facts = append([]string(nil), d.Facts...)
	d.Metadata = cloneTestMetadata(d.Metadata)
	return d
}

type testAction struct {
	RequestID string              `json:"request_id"`
	DryRun    bool                `json:"dry_run"`
	Steps     []string            `json:"steps,omitempty"`
	Metadata  map[string][]string `json:"metadata,omitempty"`
}

func (a testAction) WithDryRun(dryRun bool) testAction { a.DryRun = dryRun; return a }
func (a testAction) StableRequestID() string           { return a.RequestID }
func (a testAction) Clone() testAction {
	a.Steps = append([]string(nil), a.Steps...)
	a.Metadata = cloneTestMetadata(a.Metadata)
	return a
}

func cloneTestMetadata(input map[string][]string) map[string][]string {
	if input == nil {
		return nil
	}
	result := make(map[string][]string, len(input))
	for key, values := range input {
		result[key] = append([]string(nil), values...)
	}
	return result
}

type testStore = UniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction]
type testCommit = Commit[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction]

func TestUniversalStorePersistsAndReplaysTypedRecords(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	store, err := OpenUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](dir, testSnapshot{Value: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	commit := testCommit{
		Event:       contract.Event{ID: "event-1", Type: "test"},
		Snapshot:    testSnapshot{Value: "committed"},
		SnapshotV3:  &testSnapshotV3{Value: "v3"},
		Decision:    testDecision{Name: "decision"},
		DecisionV3:  &testDecisionV3{Name: "decision-v3"},
		Action:      &testAction{RequestID: "action-1"},
		CommittedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("test", 3600)),
	}
	result, err := store.Commit(commit)
	if err != nil {
		t.Fatal(err)
	}
	if result.Revision != 2 || result.Action == nil || !result.Action.DryRun {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := store.Snapshot(); got.Revision != 2 || got.Value != "committed" {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
	if got := store.Journal(); len(got) != 1 || got[0].CommittedAt.Location() != time.UTC {
		t.Fatalf("unexpected journal: %+v", got)
	}

	reopened, err := OpenUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](dir, testSnapshot{Value: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Snapshot(); got.Revision != 2 || got.Value != "committed" {
		t.Fatalf("unexpected restored snapshot: %+v", got)
	}
	if got, ok := reopened.SnapshotV3(); !ok || got.Revision != 2 || got.Value != "v3" {
		t.Fatalf("unexpected restored v3 snapshot: %+v, %v", got, ok)
	}
	if got := reopened.ActionOutbox(); len(got) != 1 || got[0].RequestID != "action-1" || !got[0].DryRun {
		t.Fatalf("unexpected restored outbox: %+v", got)
	}
	duplicate, err := reopened.Commit(commit)
	if err != nil {
		t.Fatal(err)
	}
	if !duplicate.Duplicate || duplicate.Revision != 2 {
		t.Fatalf("duplicate was not idempotent: %+v", duplicate)
	}
	history, err := reopened.History()
	if err != nil || len(history) != 1 || history[0].Snapshot.Revision != 2 {
		t.Fatalf("unexpected history: %+v, %v", history, err)
	}
}

func TestUniversalStoreReplayRecoversWALAfterStateWriteFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	store, err := OpenUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](dir, testSnapshot{Value: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	store.SetPersistenceHooks(PersistenceHooks{BeforeState: func() error { return errInjected{} }})
	commit := testCommit{Event: contract.Event{ID: "event-2", Type: "test"}, Snapshot: testSnapshot{Value: "wal"}}
	if _, err := store.Commit(commit); err == nil {
		t.Fatal("expected state write failure")
	}

	reopened, err := OpenUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](dir, testSnapshot{Value: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Snapshot(); got.Revision != 2 || got.Value != "wal" {
		t.Fatalf("WAL commit was not recovered: %+v", got)
	}
}

type errInjected struct{}

func (errInjected) Error() string { return "injected state write failure" }

func TestUniversalStoreClaimAndAcknowledgeRollbackOnPersistenceFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	store, err := OpenUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](dir, testSnapshot{Value: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Commit(testCommit{Event: contract.Event{ID: "outbox-event", Type: "test"}, Snapshot: testSnapshot{Value: "with-action"}, Action: &testAction{RequestID: "outbox-1"}}); err != nil {
		t.Fatal(err)
	}

	store.SetPersistenceHooks(PersistenceHooks{BeforeState: func() error { return errInjected{} }})
	if _, err := store.ClaimPendingActions(); err == nil {
		t.Fatal("expected claim persistence failure")
	}
	if err := store.AcknowledgeAction("outbox-1"); err == nil {
		t.Fatal("expected acknowledge persistence failure")
	}
	store.SetPersistenceHooks(PersistenceHooks{})
	claimed, err := store.ClaimPendingActions()
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].RequestID != "outbox-1" {
		t.Fatalf("failed persistence consumed the outbox entry: %+v", claimed)
	}
}

func TestUniversalStoreOutboxRetainsClaimWhenStateReplacementCompleted(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	store, err := OpenUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](dir, testSnapshot{Value: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Commit(testCommit{Event: contract.Event{ID: "outbox-event", Type: "test"}, Snapshot: testSnapshot{Value: "with-action"}, Action: &testAction{RequestID: "outbox-1"}}); err != nil {
		t.Fatal(err)
	}
	store.SetPersistenceHooks(PersistenceHooks{AfterState: func() error { return errInjected{} }})
	claimed, err := store.ClaimPendingActions()
	if err == nil {
		t.Fatal("expected post-replacement hook error")
	}
	if len(claimed) != 1 || claimed[0].RequestID != "outbox-1" {
		t.Fatalf("completed claim did not return its entries: %+v", claimed)
	}
	store.SetPersistenceHooks(PersistenceHooks{})
	if claimed, err := store.ClaimPendingActions(); err != nil || len(claimed) != 0 {
		t.Fatalf("claim diverged after completed replacement: claimed=%+v err=%v", claimed, err)
	}

	reopened, err := OpenUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](dir, testSnapshot{Value: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	if claimed, err := reopened.ClaimPendingActions(); err != nil || len(claimed) != 0 {
		t.Fatalf("reopen lost completed claim: claimed=%+v err=%v", claimed, err)
	}
}

func TestConcurrentStoreReadsAndWritesRemainCoherent(t *testing.T) {
	store := NewUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](testSnapshot{Value: "initial"})
	const writers, commitsPerWriter = 4, 32
	done := make(chan struct{})
	readerDone := make(chan struct{})
	errors := make(chan error, writers)
	readerErrors := make(chan error, 1)
	for writer := 0; writer < writers; writer++ {
		writer := writer
		go func() {
			for i := 0; i < commitsPerWriter; i++ {
				value := fmt.Sprintf("%d-%d", writer, i)
				_, err := store.Commit(testCommit{
					Event:      contract.Event{ID: fmt.Sprintf("concurrent-%d-%d", writer, i), Type: "test"},
					Snapshot:   testSnapshot{Value: value, Facts: []string{value}, Metadata: map[string][]string{"value": {value}}},
					SnapshotV3: &testSnapshotV3{Value: value, Facts: []string{value}, Metadata: map[string][]string{"value": {value}}},
					Decision:   testDecision{Name: value, Facts: []string{value}, Metadata: map[string][]string{"value": {value}}},
					Action:     &testAction{RequestID: "action-" + value, Steps: []string{value}, Metadata: map[string][]string{"value": {value}}},
				})
				if err != nil {
					errors <- err
					return
				}
			}
			errors <- nil
		}()
	}
	go func() {
		defer close(readerDone)
		for {
			select {
			case <-done:
				return
			default:
				snapshot := store.Snapshot()
				if snapshot.Value != "initial" && (len(snapshot.Metadata["value"]) != 1 || snapshot.Metadata["value"][0] != snapshot.Value) {
					readerErrors <- fmt.Errorf("snapshot was inconsistent: %+v", snapshot)
					return
				}
				if v3, ok := store.SnapshotV3(); ok && v3.Value != "initial" && (len(v3.Metadata["value"]) != 1 || v3.Metadata["value"][0] != v3.Value) {
					readerErrors <- fmt.Errorf("V3 snapshot was inconsistent: %+v", v3)
					return
				}
				journal := store.Journal()
				if len(journal) > 0 {
					journal[0].Event.Payload = map[string]any{"caller": "mutation"}
					journal[0].Snapshot.Metadata["value"][0] = "caller mutation"
				}
				decisions := store.Decisions()
				if len(decisions) > 0 {
					decisions[0].Metadata["value"][0] = "caller mutation"
				}
				outbox := store.ActionOutbox()
				if len(outbox) > 0 {
					outbox[0].Metadata["value"][0] = "caller mutation"
				}
			}
		}
	}()
	for i := 0; i < writers; i++ {
		if err := <-errors; err != nil {
			close(done)
			t.Fatal(err)
		}
	}
	close(done)
	<-readerDone
	select {
	case err := <-readerErrors:
		t.Fatal(err)
	default:
	}
	history := store.Journal()
	if len(history) != writers*commitsPerWriter {
		t.Fatalf("journal length=%d", len(history))
	}
	for i := 1; i < len(history); i++ {
		if history[i].Snapshot.Revision != history[i-1].Snapshot.Revision+1 {
			t.Fatalf("journal revisions out of order at %d: %d then %d", i, history[i-1].Snapshot.Revision, history[i].Snapshot.Revision)
		}
	}
}

func TestConcurrentDurableCommitsRestoreInJournalOrder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	store, err := OpenUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](dir, testSnapshot{Value: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	const writers, perWriter = 4, 8
	errCh := make(chan error, writers)
	for writer := 0; writer < writers; writer++ {
		writer := writer
		go func() {
			for i := 0; i < perWriter; i++ {
				value := fmt.Sprintf("%d-%d", writer, i)
				if _, err := store.Commit(testCommit{Event: contract.Event{ID: fmt.Sprintf("durable-%d-%d", writer, i), Type: "test"}, Snapshot: testSnapshot{Value: value}}); err != nil {
					errCh <- err
					return
				}
			}
			errCh <- nil
		}()
	}
	for i := 0; i < writers; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}

	reopened, err := OpenUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](dir, testSnapshot{Value: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	history, err := reopened.History()
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != writers*perWriter {
		t.Fatalf("restored history length=%d", len(history))
	}
	for i := 1; i < len(history); i++ {
		if history[i].Snapshot.Revision != history[i-1].Snapshot.Revision+1 {
			t.Fatalf("restored journal out of order at %d", i)
		}
	}
	if got := reopened.Snapshot().Revision; got != 1+writers*perWriter {
		t.Fatalf("restored revision=%d", got)
	}
}
