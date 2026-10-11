package statestore

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	foundationcontracts "synora/internal/foundation/contracts"
	"synora/internal/foundation/persistence"
	"synora/pkg/contract"
)

const (
	MaxJournalEntries  = 512
	MaxDecisionEntries = 512
	MaxActionOutbox    = 128
	MaxProcessedIDs    = 1024
)

// SnapshotRecord is the compile-time contract required for revisioned
// materialized records. The concrete record type remains supplied by its
// owning runtime package; the Foundation Store holds it without reflection or
// interface{} payloads.
type CloneRecord[T any] interface {
	Clone() T
}

type SnapshotRecord[T any] interface {
	CloneRecord[T]
	// Normalized returns a normalized, independently owned value. The Store
	// uses it for read snapshots so a second defensive clone is unnecessary.
	Normalized() T
	Validate() error
	WithRevision(uint64) T
	RevisionValue() uint64
}

// ActionRecord captures the two operations the Store performs on a typed
// outbox entry without depending on the runtime package that defines it.
type ActionRecord[T any] interface {
	CloneRecord[T]
	WithDryRun(bool) T
	StableRequestID() string
}

type Commit[S SnapshotRecord[S], S3 SnapshotRecord[S3], D CloneRecord[D], D3 CloneRecord[D3], A ActionRecord[A]] struct {
	Event       contract.Event `json:"event"`
	Snapshot    S              `json:"snapshot"`
	SnapshotV3  *S3            `json:"snapshot_v3,omitempty"`
	Decision    D              `json:"decision"`
	DecisionV3  *D3            `json:"decision_v3,omitempty"`
	Action      *A             `json:"action,omitempty"`
	CommittedAt time.Time      `json:"committed_at"`
}

type CommitResult[A CloneRecord[A]] struct {
	Revision  uint64
	Duplicate bool
	Action    *A
}

type storeDiskState[S SnapshotRecord[S], S3 SnapshotRecord[S3], D CloneRecord[D], D3 CloneRecord[D3], A ActionRecord[A]] struct {
	SchemaVersion       string                                               `json:"schema_version"`
	Revision            uint64                                               `json:"revision"`
	Snapshot            S                                                    `json:"snapshot"`
	SnapshotV3          *S3                                                  `json:"snapshot_v3,omitempty"`
	Journal             []Commit[S, S3, D, D3, A]                            `json:"journal"`
	Decisions           []D                                                  `json:"decisions"`
	ActionOutbox        []A                                                  `json:"action_outbox"`
	Processed           []string                                             `json:"processed"`
	Claimed             []string                                             `json:"claimed"`
	Residents           map[string]foundationcontracts.ResidentGalleryRecord `json:"resident_galleries,omitempty"`
	ResidentIdempotency map[string]string                                    `json:"resident_idempotency,omitempty"`
}

type storeJournalRecord[S SnapshotRecord[S], S3 SnapshotRecord[S3], D CloneRecord[D], D3 CloneRecord[D3], A ActionRecord[A]] struct {
	SchemaVersion string                  `json:"schema_version"`
	Revision      uint64                  `json:"revision"`
	Commit        Commit[S, S3, D, D3, A] `json:"commit"`
}

// CompactionOptions controls archived-segment retention. A zero value keeps
// every immutable audit segment and never drops useful captures.
type CompactionOptions struct {
	RetainSegments int
	MaxSegmentAge  time.Duration
}

type StoreDiskReport struct {
	Revision           uint64        `json:"revision"`
	ActiveJournalBytes int64         `json:"active_journal_bytes"`
	ArchivedBytes      int64         `json:"archived_bytes"`
	SegmentCount       int           `json:"segment_count"`
	OldestSegmentAge   time.Duration `json:"oldest_segment_age"`
}

// PersistenceHooks are test-only fault injection points. A production Store
// leaves them nil. The hooks make crash-before/after-commit behavior explicit
// without weakening the normal fsync+rename path.
type PersistenceHooks struct {
	BeforeWAL   func() error
	AfterWAL    func() error
	BeforeState func() error
	AfterState  func() error
}

var errStateReplacementCompleted = errors.New("state replacement completed")

type UniversalStore[S SnapshotRecord[S], S3 SnapshotRecord[S3], D CloneRecord[D], D3 CloneRecord[D3], A ActionRecord[A]] struct {
	mu                  sync.RWMutex
	revision            uint64
	initial             S
	journal             []Commit[S, S3, D, D3, A]
	decisions           []D
	actionOutbox        []A
	processed           map[string]struct{}
	processedOrder      []string
	snapshot            S
	snapshotV3          *S3
	claimed             map[string]struct{}
	residentGalleries   map[string]foundationcontracts.ResidentGalleryRecord
	residentIdempotency map[string]string
	dir                 string
	hooks               PersistenceHooks
}

func NewUniversalStore[S SnapshotRecord[S], S3 SnapshotRecord[S3], D CloneRecord[D], D3 CloneRecord[D3], A ActionRecord[A]](initial S) *UniversalStore[S, S3, D, D3, A] {
	return newUniversalStore[S, S3, D, D3, A](1, initial)
}

func newUniversalStore[S SnapshotRecord[S], S3 SnapshotRecord[S3], D CloneRecord[D], D3 CloneRecord[D3], A ActionRecord[A]](revision uint64, initial S) *UniversalStore[S, S3, D, D3, A] {
	return &UniversalStore[S, S3, D, D3, A]{revision: revision, journal: make([]Commit[S, S3, D, D3, A], 0, MaxJournalEntries), decisions: make([]D, 0, MaxDecisionEntries), actionOutbox: make([]A, 0, MaxActionOutbox), processed: make(map[string]struct{}), claimed: make(map[string]struct{}), residentGalleries: make(map[string]foundationcontracts.ResidentGalleryRecord), residentIdempotency: make(map[string]string), snapshot: initial.Clone(), initial: initial.Clone()}
}

// OpenUniversalStore opens or creates a durable local Store. The directory is
// private to Core; the append-only journal is the recovery source and the
// materialized state file makes inspection and restart cheap.
func OpenUniversalStore[S SnapshotRecord[S], S3 SnapshotRecord[S3], D CloneRecord[D], D3 CloneRecord[D3], A ActionRecord[A]](dir string, initial S) (*UniversalStore[S, S3, D, D3, A], error) {
	if dir == "" {
		return nil, errors.New("universal store directory is required")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create universal store directory: %w", err)
	}
	s := newUniversalStore[S, S3, D, D3, A](1, initial)
	s.dir = filepath.Clean(dir)
	if err := s.recoverEraseMarker(); err != nil {
		return nil, err
	}
	statePath := filepath.Join(s.dir, "state.json")
	if body, err := os.ReadFile(statePath); err == nil {
		var disk storeDiskState[S, S3, D, D3, A]
		if err := json.Unmarshal(body, &disk); err != nil {
			return nil, fmt.Errorf("universal store state is corrupt: %w", err)
		}
		if disk.SchemaVersion != "universal-store/v1" {
			return nil, fmt.Errorf("unsupported universal store state schema %q", disk.SchemaVersion)
		}
		s.revision, s.snapshot, s.snapshotV3, s.journal, s.decisions, s.actionOutbox = disk.Revision, disk.Snapshot.Normalized(), disk.SnapshotV3, disk.Journal, disk.Decisions, disk.ActionOutbox
		if disk.Residents != nil {
			s.residentGalleries = disk.Residents
		}
		if disk.ResidentIdempotency != nil {
			s.residentIdempotency = disk.ResidentIdempotency
		}
		for _, id := range disk.Processed {
			s.processed[id] = struct{}{}
			s.processedOrder = append(s.processedOrder, id)
		}
		for _, id := range disk.Claimed {
			s.claimed[id] = struct{}{}
		}
		if err := validateResidentGalleryState(s.residentGalleries, s.residentIdempotency); err != nil {
			return nil, fmt.Errorf("universal store resident gallery state is corrupt: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read universal store state: %w", err)
	}
	if err := s.replayJournal(); err != nil {
		return nil, err
	}
	if err := s.validateArchivedSegments(); err != nil {
		return nil, err
	}
	if err := s.ValidateBounds(); err != nil {
		return nil, err
	}
	return s, nil
}

// EraseAll durably removes every user-observable record from this Store. A
// marker is written before deleting the WAL and compacted segments so a crash
// cannot make deleted history reappear on the next open.
func (s *UniversalStore[S, S3, D, D3, A]) EraseAll() error {
	if s == nil {
		return errors.New("universal store is nil")
	}
	if s.dir == "" {
		return errors.New("universal store is not durable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	marker := filepath.Join(s.dir, "erase.marker")
	if err := persistence.WriteSynced(marker, []byte("universal-store/erase/v1\n"), 0o600); err != nil {
		return fmt.Errorf("write universal store erase marker: %w", err)
	}
	if err := persistence.SyncDir(s.dir); err != nil {
		return fmt.Errorf("sync universal store erase marker directory: %w", err)
	}
	if err := s.clearDurableHistoryLocked(); err != nil {
		return err
	}
	s.resetMemoryLocked()
	if err := s.persistStateLocked(); err != nil {
		return err
	}
	if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove universal store erase marker: %w", err)
	}
	return persistence.SyncDir(s.dir)
}

func (s *UniversalStore[S, S3, D, D3, A]) recoverEraseMarker() error {
	marker := filepath.Join(s.dir, "erase.marker")
	if _, err := os.Stat(marker); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect universal store erase marker: %w", err)
	}
	if err := s.clearDurableHistoryLocked(); err != nil {
		return err
	}
	s.resetMemoryLocked()
	if err := s.persistStateLocked(); err != nil {
		return err
	}
	if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove recovered universal store erase marker: %w", err)
	}
	return persistence.SyncDir(s.dir)
}

func (s *UniversalStore[S, S3, D, D3, A]) clearDurableHistoryLocked() error {
	if err := os.Remove(filepath.Join(s.dir, "journal.jsonl")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove universal store journal: %w", err)
	}
	if err := os.RemoveAll(filepath.Join(s.dir, "segments")); err != nil {
		return fmt.Errorf("remove universal store segments: %w", err)
	}
	return nil
}

func (s *UniversalStore[S, S3, D, D3, A]) resetMemoryLocked() {
	s.journal = make([]Commit[S, S3, D, D3, A], 0, MaxJournalEntries)
	s.decisions = make([]D, 0, MaxDecisionEntries)
	s.actionOutbox = make([]A, 0, MaxActionOutbox)
	s.processed = make(map[string]struct{})
	s.processedOrder = nil
	s.snapshot = s.initial
	s.snapshotV3 = nil
	s.claimed = make(map[string]struct{})
	s.residentGalleries = make(map[string]foundationcontracts.ResidentGalleryRecord)
	s.residentIdempotency = make(map[string]string)
}

func (s *UniversalStore[S, S3, D, D3, A]) SetPersistenceHooks(hooks PersistenceHooks) {
	if s != nil {
		s.hooks = hooks
	}
}

func (s *UniversalStore[S, S3, D, D3, A]) Commit(value Commit[S, S3, D, D3, A]) (CommitResult[A], error) {
	if s == nil {
		return CommitResult[A]{}, errors.New("universal store is nil")
	}
	if value.Event.ID == "" || value.Event.Type == "" {
		return CommitResult[A]{}, errors.New("commit event id and type are required")
	}
	value = cloneCommit(value)
	if value.SnapshotV3 != nil {
		snapshotV3 := (*value.SnapshotV3).Normalized()
		if err := snapshotV3.Validate(); err != nil {
			return CommitResult[A]{}, err
		}
		value.SnapshotV3 = &snapshotV3
	}
	value.Snapshot = value.Snapshot.Normalized()
	if err := value.Snapshot.Validate(); err != nil {
		return CommitResult[A]{}, err
	}
	value.CommittedAt = value.CommittedAt.UTC()
	if value.CommittedAt.IsZero() {
		value.CommittedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.processed[value.Event.ID]; exists {
		return CommitResult[A]{Revision: s.revision, Duplicate: true}, nil
	}
	nextRevision := s.revision + 1
	value.Snapshot = value.Snapshot.WithRevision(nextRevision).Normalized().WithRevision(nextRevision)
	if value.SnapshotV3 != nil {
		snapshotV3 := (*value.SnapshotV3).Normalized()
		snapshotV3 = snapshotV3.WithRevision(nextRevision)
		value.SnapshotV3 = &snapshotV3
	}
	if value.Action != nil {
		action := (*value.Action).WithDryRun(true)
		value.Action = &action
	}
	if s.dir != "" {
		if s.hooks.BeforeWAL != nil {
			if err := s.hooks.BeforeWAL(); err != nil {
				return CommitResult[A]{}, err
			}
		}
		if err := s.appendWAL(storeJournalRecord[S, S3, D, D3, A]{SchemaVersion: "universal-store/v1", Revision: nextRevision, Commit: value}); err != nil {
			return CommitResult[A]{}, err
		}
		if s.hooks.AfterWAL != nil {
			if err := s.hooks.AfterWAL(); err != nil {
				return CommitResult[A]{}, err
			}
		}
	}
	// Apply only after the durable record exists. A restart can recover a WAL
	// record left behind by a crash between WAL and materialized-state writes.
	s.revision = nextRevision
	s.applyCommitLocked(value)
	if s.dir != "" {
		if err := s.persistStateLocked(); err != nil {
			return CommitResult[A]{}, err
		}
	}
	return CommitResult[A]{Revision: s.revision, Action: cloneAction(value.Action)}, nil
}

func (s *UniversalStore[S, S3, D, D3, A]) applyCommitLocked(value Commit[S, S3, D, D3, A]) {
	s.processed[value.Event.ID] = struct{}{}
	s.processedOrder = append(s.processedOrder, value.Event.ID)
	if len(s.processedOrder) > MaxProcessedIDs {
		oldest := s.processedOrder[0]
		s.processedOrder = s.processedOrder[1:]
		delete(s.processed, oldest)
	}
	s.journal = appendBounded(s.journal, value, MaxJournalEntries)
	s.decisions = appendBounded(s.decisions, value.Decision, MaxDecisionEntries)
	if value.Action != nil {
		s.actionOutbox = appendBounded(s.actionOutbox, *value.Action, MaxActionOutbox)
	}
	s.snapshot = value.Snapshot
	if value.SnapshotV3 != nil {
		snapshotV3 := (*value.SnapshotV3).Normalized()
		snapshotV3 = snapshotV3.WithRevision(value.Snapshot.RevisionValue())
		s.snapshotV3 = &snapshotV3
	}
}

// SnapshotV3 returns the latest aggregate-only V3 candidate snapshot. The
// returned value is detached from store state.
func (s *UniversalStore[S, S3, D, D3, A]) SnapshotV3() (S3, bool) {
	if s == nil {
		var zero S3
		return zero, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.snapshotV3 == nil {
		var zero S3
		return zero, false
	}
	return (*s.snapshotV3).Clone(), true
}

func (s *UniversalStore[S, S3, D, D3, A]) appendWAL(record storeJournalRecord[S, S3, D, D3, A]) error {
	body, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("append universal store journal: %w", err)
	}
	if err := persistence.AppendSynced(filepath.Join(s.dir, "journal.jsonl"), append(body, '\n'), 0o640); err != nil {
		return fmt.Errorf("append universal store journal: %w", err)
	}
	return nil
}

// Compact rotates the active WAL into an immutable audit segment after the
// materialized state has been durably written. Reopening after the rotation
// therefore has exactly the same state while History still sees every commit.
func (s *UniversalStore[S, S3, D, D3, A]) Compact(options CompactionOptions) error {
	if s == nil {
		return errors.New("universal store is nil")
	}
	if s.dir == "" {
		return errors.New("universal store is not durable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.persistStateLocked(); err != nil {
		return err
	}
	journalPath := filepath.Join(s.dir, "journal.jsonl")
	body, err := os.ReadFile(journalPath)
	if errors.Is(err, os.ErrNotExist) {
		body = nil
	} else if err != nil {
		return fmt.Errorf("read universal store journal for compaction: %w", err)
	}
	if len(body) > 0 {
		scanner := bufio.NewScanner(bytes.NewReader(body))
		scanner.Buffer(make([]byte, 64*1024), 8<<20)
		var first, last uint64
		for scanner.Scan() {
			var record storeJournalRecord[S, S3, D, D3, A]
			if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
				return fmt.Errorf("universal store journal is corrupt: %w", err)
			}
			if first == 0 {
				first = record.Revision
			}
			last = record.Revision
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		if first != 0 {
			segmentDir := filepath.Join(s.dir, "segments")
			if err := os.MkdirAll(segmentDir, 0o750); err != nil {
				return err
			}
			if err := persistence.SyncDir(s.dir); err != nil {
				return fmt.Errorf("sync universal store segment directory entry: %w", err)
			}
			name := fmt.Sprintf("journal-%020d-%020d-%d.jsonl", first, last, time.Now().UnixNano())
			segmentPath := filepath.Join(s.dir, "segments", name)
			if err := persistence.WriteSynced(segmentPath, body, 0o640); err != nil {
				return fmt.Errorf("write universal store audit segment: %w", err)
			}
			if err := persistence.SyncDir(segmentDir); err != nil {
				return fmt.Errorf("sync universal store audit segment directory: %w", err)
			}
			if err := os.Remove(journalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("rotate universal store journal: %w", err)
			}
		}
	}
	file, err := os.OpenFile(journalPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return fmt.Errorf("create compacted universal store journal: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := s.applyRetentionLocked(options); err != nil {
		return err
	}
	return persistence.SyncDir(s.dir)
}

// History reads immutable audit segments plus the active journal in revision
// order. Journal() remains the bounded hot in-memory view.
func (s *UniversalStore[S, S3, D, D3, A]) History() ([]Commit[S, S3, D, D3, A], error) {
	if s == nil {
		return nil, errors.New("universal store is nil")
	}
	if s.dir == "" {
		return s.Journal(), nil
	}
	records, err := s.readAllJournalRecords()
	if err != nil {
		return nil, err
	}
	commits := make([]Commit[S, S3, D, D3, A], 0, len(records))
	seen := make(map[uint64]struct{}, len(records))
	for _, record := range records {
		if _, ok := seen[record.Revision]; ok {
			continue
		}
		seen[record.Revision] = struct{}{}
		record.Commit.Snapshot = record.Commit.Snapshot.WithRevision(record.Revision)
		commits = append(commits, record.Commit)
	}
	sort.Slice(commits, func(i, j int) bool { return commits[i].Snapshot.RevisionValue() < commits[j].Snapshot.RevisionValue() })
	return commits, nil
}

func (s *UniversalStore[S, S3, D, D3, A]) DiskReport() (StoreDiskReport, error) {
	if s == nil || s.dir == "" {
		return StoreDiskReport{}, errors.New("universal store is not durable")
	}
	report := StoreDiskReport{Revision: s.Revision()}
	active, err := os.Stat(filepath.Join(s.dir, "journal.jsonl"))
	if err == nil {
		report.ActiveJournalBytes = active.Size()
	} else if !errors.Is(err, os.ErrNotExist) {
		return report, err
	}
	entries, err := os.ReadDir(filepath.Join(s.dir, "segments"))
	if errors.Is(err, os.ErrNotExist) {
		return report, nil
	}
	if err != nil {
		return report, err
	}
	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			return report, statErr
		}
		report.SegmentCount++
		report.ArchivedBytes += info.Size()
		if age := now.Sub(info.ModTime()); age > report.OldestSegmentAge {
			report.OldestSegmentAge = age
		}
	}
	return report, nil
}

func (s *UniversalStore[S, S3, D, D3, A]) readAllJournalRecords() ([]storeJournalRecord[S, S3, D, D3, A], error) {
	paths := []string{}
	entries, err := os.ReadDir(filepath.Join(s.dir, "segments"))
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
				paths = append(paths, filepath.Join(s.dir, "segments", entry.Name()))
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	paths = append(paths, filepath.Join(s.dir, "journal.jsonl"))
	sort.Strings(paths)
	var records []storeJournalRecord[S, S3, D, D3, A]
	for _, path := range paths {
		body, readErr := os.ReadFile(path)
		if errors.Is(readErr, os.ErrNotExist) {
			continue
		}
		if readErr != nil {
			return nil, readErr
		}
		scanner := bufio.NewScanner(bytes.NewReader(body))
		scanner.Buffer(make([]byte, 64*1024), 8<<20)
		for scanner.Scan() {
			var record storeJournalRecord[S, S3, D, D3, A]
			if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
				return nil, fmt.Errorf("universal store audit segment %s is corrupt: %w", path, err)
			}
			if record.SchemaVersion != "universal-store/v1" || record.Revision == 0 {
				return nil, fmt.Errorf("universal store audit segment %s has invalid record", path)
			}
			records = append(records, record)
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	return records, nil
}

func (s *UniversalStore[S, S3, D, D3, A]) validateArchivedSegments() error {
	if s == nil || s.dir == "" {
		return nil
	}
	_, err := s.readAllJournalRecords()
	return err
}

func (s *UniversalStore[S, S3, D, D3, A]) applyRetentionLocked(options CompactionOptions) error {
	entries, err := os.ReadDir(filepath.Join(s.dir, "segments"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	type segment struct {
		path string
		mod  time.Time
	}
	segments := make([]segment, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		segments = append(segments, segment{filepath.Join(s.dir, "segments", entry.Name()), info.ModTime()})
	}
	sort.Slice(segments, func(i, j int) bool { return segments[i].mod.Before(segments[j].mod) })
	remove := map[string]bool{}
	if options.RetainSegments > 0 && len(segments) > options.RetainSegments {
		for _, item := range segments[:len(segments)-options.RetainSegments] {
			remove[item.path] = true
		}
	}
	if options.MaxSegmentAge > 0 {
		cutoff := time.Now().Add(-options.MaxSegmentAge)
		for _, item := range segments {
			if item.mod.Before(cutoff) {
				remove[item.path] = true
			}
		}
	}
	removed := false
	for path := range remove {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		removed = true
	}
	if removed {
		if err := persistence.SyncDir(filepath.Join(s.dir, "segments")); err != nil {
			return fmt.Errorf("sync universal store retention directory: %w", err)
		}
	}
	return nil
}

func (s *UniversalStore[S, S3, D, D3, A]) persistStateLocked() error {
	if s.hooks.BeforeState != nil {
		if err := s.hooks.BeforeState(); err != nil {
			return err
		}
	}
	state := s.diskStateLocked()
	body, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := persistence.ReplaceSynced(s.dir, "state.json", ".state-*.tmp", append(body, '\n')); err != nil {
		return fmt.Errorf("replace universal store state: %w", err)
	}
	if s.hooks.AfterState != nil {
		if err := s.hooks.AfterState(); err != nil {
			return fmt.Errorf("%w: %w", errStateReplacementCompleted, err)
		}
	}
	return nil
}

func stateReplacementMayHaveCompleted(err error) bool {
	return errors.Is(err, errStateReplacementCompleted) || errors.Is(err, persistence.ErrDirectorySyncAfterReplace)
}

func (s *UniversalStore[S, S3, D, D3, A]) diskStateLocked() storeDiskState[S, S3, D, D3, A] {
	processed := append([]string(nil), s.processedOrder...)
	if len(processed) == 0 {
		for id := range s.processed {
			processed = append(processed, id)
		}
	}
	claimed := make([]string, 0, len(s.claimed))
	for id := range s.claimed {
		claimed = append(claimed, id)
	}
	sort.Strings(processed)
	sort.Strings(claimed)
	return storeDiskState[S, S3, D, D3, A]{
		SchemaVersion:       "universal-store/v1",
		Revision:            s.revision,
		Snapshot:            s.snapshot,
		SnapshotV3:          s.snapshotV3,
		Journal:             append([]Commit[S, S3, D, D3, A](nil), s.journal...),
		Decisions:           append([]D(nil), s.decisions...),
		ActionOutbox:        append([]A(nil), s.actionOutbox...),
		Processed:           processed,
		Claimed:             claimed,
		Residents:           cloneResidentGalleryRecords(s.residentGalleries),
		ResidentIdempotency: cloneStringMap(s.residentIdempotency),
	}
}

func (s *UniversalStore[S, S3, D, D3, A]) replayJournal() error {
	file, err := os.Open(filepath.Join(s.dir, "journal.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open universal store journal: %w", err)
	}
	defer file.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8<<20)
	for scanner.Scan() {
		var record storeJournalRecord[S, S3, D, D3, A]
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return fmt.Errorf("universal store journal is corrupt: %w", err)
		}
		if record.SchemaVersion != "universal-store/v1" || record.Revision == 0 {
			return errors.New("universal store journal record is invalid")
		}
		if _, exists := s.processed[record.Commit.Event.ID]; exists {
			continue
		}
		if record.Revision <= s.revision {
			continue
		}
		s.revision = record.Revision
		record.Commit.Snapshot = record.Commit.Snapshot.WithRevision(record.Revision)
		if record.Commit.SnapshotV3 != nil {
			snapshotV3 := (*record.Commit.SnapshotV3).Normalized()
			snapshotV3 = snapshotV3.WithRevision(record.Revision)
			record.Commit.SnapshotV3 = &snapshotV3
		}
		s.applyCommitLocked(record.Commit)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read universal store journal: %w", err)
	}
	return nil
}

// ClaimPendingActions atomically marks durable outbox entries as delivered.
// The request ID is stable, so Discovery can retry safely without replaying a
// physical action. If state replacement completed but directory sync failed,
// it returns the claimed entries together with the durability error; callers
// must still handle those returned entries.
func (s *UniversalStore[S, S3, D, D3, A]) ClaimPendingActions() ([]A, error) {
	if s == nil {
		return nil, errors.New("universal store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	claimed := make([]A, 0)
	claimedIDs := make([]string, 0)
	for _, action := range s.actionOutbox {
		if _, ok := s.claimed[action.StableRequestID()]; ok {
			continue
		}
		requestID := action.StableRequestID()
		s.claimed[requestID] = struct{}{}
		claimedIDs = append(claimedIDs, requestID)
		claimed = append(claimed, action.Clone())
	}
	if len(claimedIDs) > 0 && s.dir != "" {
		if err := s.persistStateLocked(); err != nil {
			if stateReplacementMayHaveCompleted(err) {
				return claimed, err
			} else {
				for _, requestID := range claimedIDs {
					delete(s.claimed, requestID)
				}
			}
			return nil, err
		}
	}
	return claimed, nil
}

func (s *UniversalStore[S, S3, D, D3, A]) AcknowledgeAction(requestID string) error {
	if requestID == "" {
		return errors.New("request id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.claimed == nil {
		s.claimed = make(map[string]struct{})
	}
	_, previouslyClaimed := s.claimed[requestID]
	s.claimed[requestID] = struct{}{}
	if s.dir != "" {
		if err := s.persistStateLocked(); err != nil {
			if !previouslyClaimed && !stateReplacementMayHaveCompleted(err) {
				delete(s.claimed, requestID)
			}
			return err
		}
	}
	return nil
}

func (s *UniversalStore[S, S3, D, D3, A]) Snapshot() S {
	if s == nil {
		var zero S
		return zero
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot.Normalized()
}
func (s *UniversalStore[S, S3, D, D3, A]) SnapshotJSON() ([]byte, error) {
	if s == nil {
		return nil, errors.New("universal store is nil")
	}
	return json.Marshal(s.Snapshot())
}

// HistoryJSON is the read-only Discovery view of the append-only audit. The
// Store remains the writer; callers receive a serialized copy only.
func (s *UniversalStore[S, S3, D, D3, A]) HistoryJSON() ([]byte, error) {
	if s == nil {
		return nil, errors.New("universal store is nil")
	}
	history, err := s.History()
	if err != nil {
		return nil, err
	}
	return json.Marshal(history)
}
func (s *UniversalStore[S, S3, D, D3, A]) Revision() uint64 {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revision
}
func (s *UniversalStore[S, S3, D, D3, A]) Journal() []Commit[S, S3, D, D3, A] {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.journal) == 0 {
		return nil
	}
	result := make([]Commit[S, S3, D, D3, A], len(s.journal))
	for i, commit := range s.journal {
		result[i] = cloneCommit(commit)
	}
	return result
}
func (s *UniversalStore[S, S3, D, D3, A]) Decisions() []D {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.decisions) == 0 {
		return nil
	}
	result := make([]D, len(s.decisions))
	for i, decision := range s.decisions {
		result[i] = decision.Clone()
	}
	return result
}
func (s *UniversalStore[S, S3, D, D3, A]) ActionOutbox() []A {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.actionOutbox) == 0 {
		return nil
	}
	result := make([]A, len(s.actionOutbox))
	for i, action := range s.actionOutbox {
		result[i] = action.Clone()
	}
	return result
}

func (s *UniversalStore[S, S3, D, D3, A]) Replay(events []contract.Event, processor func(contract.Event) error) error {
	for _, event := range events {
		if err := processor(event); err != nil {
			return err
		}
	}
	return nil
}

func cloneCommit[S SnapshotRecord[S], S3 SnapshotRecord[S3], D CloneRecord[D], D3 CloneRecord[D3], A ActionRecord[A]](value Commit[S, S3, D, D3, A]) Commit[S, S3, D, D3, A] {
	value.Event = value.Event.Clone()
	value.Snapshot = value.Snapshot.Clone()
	if value.SnapshotV3 != nil {
		snapshot := (*value.SnapshotV3).Clone()
		value.SnapshotV3 = &snapshot
	}
	value.Decision = value.Decision.Clone()
	if value.DecisionV3 != nil {
		decision := (*value.DecisionV3).Clone()
		value.DecisionV3 = &decision
	}
	if value.Action != nil {
		action := (*value.Action).Clone()
		value.Action = &action
	}
	return value
}

func appendBounded[T any](values []T, value T, max int) []T {
	values = append(values, value)
	if len(values) > max {
		values = values[len(values)-max:]
	}
	return values
}
func cloneAction[A CloneRecord[A]](value *A) *A {
	if value == nil {
		return nil
	}
	clone := (*value).Clone()
	return &clone
}

func (s *UniversalStore[S, S3, D, D3, A]) MarshalJSON() ([]byte, error) {
	if s == nil {
		return nil, errors.New("universal store is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.diskStateLocked())
}
func (s *UniversalStore[S, S3, D, D3, A]) ValidateBounds() error {
	if s == nil {
		return errors.New("universal store is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.journal) > MaxJournalEntries || len(s.decisions) > MaxDecisionEntries || len(s.actionOutbox) > MaxActionOutbox || len(s.processedOrder) > MaxProcessedIDs {
		return fmt.Errorf("universal store bound exceeded")
	}
	return nil
}
