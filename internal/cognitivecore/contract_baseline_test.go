package cognitivecore

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"synora/pkg/contract"
)

const h1PersistenceFixtures = "../../testdata/contract-baseline/persistence"

func h1BaselineCommit() Commit {
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	return Commit{
		Event:       contract.Event{ID: "h1-store-event-001", Type: "sensor.motion", Source: "contract-test", Timestamp: at, Payload: map[string]any{"movement": true}},
		Snapshot:    CognitiveSnapshot{SchemaVersion: SnapshotSchemaVersion, CapturedAt: at, Topology: contract.VisionTopologyPrivatePerimeter, Episode: EpisodeFacts{Phase: VisionPhaseCandidate, SegmentCount: 1}},
		SnapshotV3:  &CognitiveSnapshotV3{SchemaVersion: SnapshotSchemaVersionV3, CapturedAt: at, BaseV2: CognitiveSnapshotV2{SchemaVersion: SnapshotSchemaVersionV2, CapturedAt: at, Topology: contract.VisionTopologyPrivatePerimeter}},
		Decision:    Decision{SchemaVersion: DecisionSchemaVersion, Status: "available", Mode: "active_dry_run", Source: "contract-test", HeadOrder: append([]string(nil), HeadOrder[:]...), InputDimension: CognitiveVectorSize, Action: ActionAssessment{Proposed: ActionIntent{Action: "notify"}, Status: "allowed_dry_run", PhysicalActionExecuted: false}, GeneratedAt: at},
		DecisionV3:  &DecisionV3{SchemaVersion: "cognitive-decision/v3", Status: "available", Mode: "active_dry_run", Source: "contract-test", HeadOrder: append([]string(nil), HeadOrderV3[:]...), InputDimension: CognitiveVectorSizeV3, Action: ActionAssessment{Proposed: ActionIntent{Action: "notify"}, Status: "allowed_dry_run", PhysicalActionExecuted: false}, GeneratedAt: at},
		Action:      &ActionRequest{SchemaVersion: "cognitive-action-request/v1", RequestID: "h1-action-001", EpisodeID: "episode-h1", Action: ActionIntent{Action: "notify"}},
		CommittedAt: at,
	}
}

func TestH1UniversalStoreV1GoldenAndJournalReplay(t *testing.T) {
	// Compare the current production serializer with the checked-in immutable
	// output, then separately open that output from an isolated temporary path.
	dir := t.TempDir()
	store, err := OpenUniversalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Commit(h1BaselineCommit()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"state.json", "journal.jsonl"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(h1PersistenceFixtures, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s serializer differs from H1 golden\n got: %s\nwant: %s", name, got, want)
		}
	}

	replayDir := t.TempDir()
	for _, name := range []string{"state.json", "journal.jsonl"} {
		body, err := os.ReadFile(filepath.Join(h1PersistenceFixtures, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(replayDir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	replayed, err := OpenUniversalStore(replayDir)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Revision() != 2 || len(replayed.Journal()) != 1 || len(replayed.ActionOutbox()) != 1 {
		t.Fatalf("V1 persistence replay lost records: revision=%d journal=%d outbox=%d", replayed.Revision(), len(replayed.Journal()), len(replayed.ActionOutbox()))
	}
	if snapshotV3, ok := replayed.SnapshotV3(); !ok || snapshotV3.SchemaVersion != SnapshotSchemaVersionV3 || snapshotV3.BaseV2.SchemaVersion != SnapshotSchemaVersionV2 {
		t.Fatalf("versioned V3/V2 snapshot compatibility changed: ok=%v value=%#v", ok, snapshotV3)
	}
	if replayed.ActionOutbox()[0].RequestID != "h1-action-001" || !replayed.ActionOutbox()[0].DryRun {
		t.Fatalf("action outbox compatibility changed: %#v", replayed.ActionOutbox())
	}
	var state map[string]json.RawMessage
	if err := json.Unmarshal(mustRead(t, filepath.Join(h1PersistenceFixtures, "state.json")), &state); err != nil {
		t.Fatal(err)
	}
	var schema string
	if err := json.Unmarshal(state["schema_version"], &schema); err != nil {
		t.Fatal(err)
	}
	if schema != "universal-store/v1" {
		t.Fatalf("store schema=%q", schema)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
