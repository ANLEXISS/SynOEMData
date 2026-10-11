package cognitivecore

import (
	"encoding/json"
	"errors"
	"time"

	"synora/internal/foundation/statestore"
	"synora/pkg/contract"
)

const (
	MaxJournalEntries  = statestore.MaxJournalEntries
	MaxDecisionEntries = statestore.MaxDecisionEntries
	MaxActionOutbox    = statestore.MaxActionOutbox
	MaxProcessedIDs    = statestore.MaxProcessedIDs
)

type ActionRequest struct {
	SchemaVersion string       `json:"schema_version"`
	RequestID     string       `json:"request_id"`
	EpisodeID     string       `json:"episode_id,omitempty"`
	Action        ActionIntent `json:"action"`
	DryRun        bool         `json:"dry_run"`
}

func (a ActionRequest) WithDryRun(dryRun bool) ActionRequest { a.DryRun = dryRun; return a }
func (a ActionRequest) StableRequestID() string              { return a.RequestID }
func (a ActionRequest) Clone() ActionRequest                 { return a }

type Commit = statestore.Commit[CognitiveSnapshot, CognitiveSnapshotV3, Decision, DecisionV3, ActionRequest]
type CommitResult = statestore.CommitResult[ActionRequest]
type CompactionOptions = statestore.CompactionOptions
type StoreDiskReport = statestore.StoreDiskReport
type PersistenceHooks = statestore.PersistenceHooks

// UniversalStore remains the compatibility-facing Core type. All authoritative
// state, synchronization, WAL, replay, retention and persistence live in the
// Foundation Store referenced here.
type UniversalStore struct {
	impl *statestore.UniversalStore[CognitiveSnapshot, CognitiveSnapshotV3, Decision, DecisionV3, ActionRequest]
}

func initialSnapshot() CognitiveSnapshot {
	return CognitiveSnapshot{SchemaVersion: SnapshotSchemaVersion, CapturedAt: time.Unix(0, 0).UTC(), Topology: "unknown", Episode: EpisodeFacts{Phase: "initial"}}
}

func NewUniversalStore() *UniversalStore {
	return &UniversalStore{impl: statestore.NewUniversalStore[CognitiveSnapshot, CognitiveSnapshotV3, Decision, DecisionV3, ActionRequest](initialSnapshot())}
}

func OpenUniversalStore(dir string) (*UniversalStore, error) {
	store, err := statestore.OpenUniversalStore[CognitiveSnapshot, CognitiveSnapshotV3, Decision, DecisionV3, ActionRequest](dir, initialSnapshot())
	if err != nil {
		return nil, err
	}
	return &UniversalStore{impl: store}, nil
}

func (s *UniversalStore) Commit(value Commit) (CommitResult, error) {
	if s == nil || s.impl == nil {
		return CommitResult{}, errors.New("universal store is nil")
	}
	if value.SnapshotV3 != nil {
		snapshotV3 := value.SnapshotV3.Normalized()
		if err := snapshotV3.Validate(); err != nil {
			return CommitResult{}, err
		}
		value.SnapshotV3 = &snapshotV3
		value.Snapshot = projectV3ToV1(snapshotV3)
	}
	value.Snapshot = value.Snapshot.Normalized()
	if err := value.Snapshot.Validate(); err != nil {
		return CommitResult{}, err
	}
	return s.impl.Commit(value)
}

func (s *UniversalStore) EraseAll() error {
	if s == nil || s.impl == nil {
		return errors.New("universal store is nil")
	}
	return s.impl.EraseAll()
}
func (s *UniversalStore) SetPersistenceHooks(hooks PersistenceHooks) {
	if s != nil && s.impl != nil {
		s.impl.SetPersistenceHooks(hooks)
	}
}
func (s *UniversalStore) Snapshot() CognitiveSnapshot {
	if s == nil || s.impl == nil {
		return CognitiveSnapshot{}
	}
	return s.impl.Snapshot()
}
func (s *UniversalStore) SnapshotV3() (CognitiveSnapshotV3, bool) {
	if s == nil || s.impl == nil {
		return CognitiveSnapshotV3{}, false
	}
	return s.impl.SnapshotV3()
}
func (s *UniversalStore) SnapshotJSON() ([]byte, error) {
	if s == nil || s.impl == nil {
		return nil, errors.New("universal store is nil")
	}
	return s.impl.SnapshotJSON()
}
func (s *UniversalStore) HistoryJSON() ([]byte, error) {
	if s == nil || s.impl == nil {
		return nil, errors.New("universal store is nil")
	}
	return s.impl.HistoryJSON()
}
func (s *UniversalStore) Revision() uint64 {
	if s == nil || s.impl == nil {
		return 0
	}
	return s.impl.Revision()
}
func (s *UniversalStore) History() ([]Commit, error) {
	if s == nil || s.impl == nil {
		return nil, errors.New("universal store is nil")
	}
	return s.impl.History()
}
func (s *UniversalStore) DiskReport() (StoreDiskReport, error) {
	if s == nil || s.impl == nil {
		return StoreDiskReport{}, errors.New("universal store is not durable")
	}
	return s.impl.DiskReport()
}
func (s *UniversalStore) Compact(options CompactionOptions) error {
	if s == nil || s.impl == nil {
		return errors.New("universal store is nil")
	}
	return s.impl.Compact(options)
}
func (s *UniversalStore) Journal() []Commit {
	if s == nil || s.impl == nil {
		return nil
	}
	return s.impl.Journal()
}
func (s *UniversalStore) Decisions() []Decision {
	if s == nil || s.impl == nil {
		return nil
	}
	return s.impl.Decisions()
}
func (s *UniversalStore) ActionOutbox() []ActionRequest {
	if s == nil || s.impl == nil {
		return nil
	}
	return s.impl.ActionOutbox()
}
func (s *UniversalStore) ClaimPendingActions() ([]ActionRequest, error) {
	if s == nil || s.impl == nil {
		return nil, errors.New("universal store is nil")
	}
	return s.impl.ClaimPendingActions()
}
func (s *UniversalStore) AcknowledgeAction(requestID string) error {
	if s == nil || s.impl == nil {
		return errors.New("universal store is nil")
	}
	return s.impl.AcknowledgeAction(requestID)
}
func (s *UniversalStore) ValidateBounds() error {
	if s == nil || s.impl == nil {
		return errors.New("universal store is nil")
	}
	return s.impl.ValidateBounds()
}
func (s *UniversalStore) Replay(events []contract.Event, processor func(contract.Event) error) error {
	if s == nil || s.impl == nil {
		return errors.New("universal store is nil")
	}
	return s.impl.Replay(events, processor)
}
func (s *UniversalStore) MarshalJSON() ([]byte, error) {
	if s == nil || s.impl == nil {
		return nil, errors.New("universal store is nil")
	}
	return json.Marshal(s.impl)
}
