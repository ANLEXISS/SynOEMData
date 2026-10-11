package statestore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	foundationcontracts "synora/internal/foundation/contracts"
)

// ResidentGalleryRecord and its policy version remain available from the
// historical cognitivecore package as source-compatible aliases. Their
// authoritative definitions now live in Foundation contracts.
type ResidentGalleryRecord = foundationcontracts.ResidentGalleryRecord

const ResidentGalleryPolicyVersion = foundationcontracts.ResidentGalleryPolicyVersion

// CreateResidentGallery is called only by the Core RPC handler. Idempotency
// keys are opaque operation tokens, never names or user-supplied identities.
func (s *UniversalStore[S, S3, D, D3, A]) CreateResidentGallery(record ResidentGalleryRecord, idempotencyKey string) (ResidentGalleryRecord, bool, error) {
	if s == nil {
		return ResidentGalleryRecord{}, false, errors.New("Universal Store unavailable")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if len(idempotencyKey) < 8 || len(idempotencyKey) > 128 || strings.ContainsAny(idempotencyKey, " \t\r\n") {
		return ResidentGalleryRecord{}, false, errors.New("invalid idempotency key")
	}
	if err := record.Validate(); err != nil {
		return ResidentGalleryRecord{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idempotencyHashBytes := sha256.Sum256([]byte(idempotencyKey))
	idempotencyHash := hex.EncodeToString(idempotencyHashBytes[:])
	if existingRef := s.residentIdempotency[idempotencyHash]; existingRef != "" {
		return s.residentGalleries[existingRef], true, nil
	}
	if _, exists := s.residentGalleries[record.ResidentRef]; exists {
		return ResidentGalleryRecord{}, false, errors.New("resident reference already exists")
	}
	s.residentGalleries[record.ResidentRef] = record
	s.residentIdempotency[idempotencyHash] = record.ResidentRef
	if s.dir != "" {
		if err := s.persistStateLocked(); err != nil {
			delete(s.residentGalleries, record.ResidentRef)
			delete(s.residentIdempotency, idempotencyHash)
			return ResidentGalleryRecord{}, false, fmt.Errorf("persist resident profile: %w", err)
		}
	}
	return record, false, nil
}

func (s *UniversalStore[S, S3, D, D3, A]) DeleteResidentGallery(ref string, now time.Time) (ResidentGalleryRecord, error) {
	if s == nil || !foundationcontracts.IsOpaqueResidentRef(ref) {
		return ResidentGalleryRecord{}, errors.New("resident unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.residentGalleries[ref]
	if !ok {
		return ResidentGalleryRecord{}, errors.New("resident unavailable")
	}
	if record.Deleted {
		return record, nil
	}
	previous := record
	if now.IsZero() {
		now = time.Now().UTC()
	}
	record.Deleted, record.GalleryStatus, record.UpdatedAt = true, "disabled", now.UTC()
	s.residentGalleries[ref] = record
	if s.dir != "" {
		if err := s.persistStateLocked(); err != nil {
			s.residentGalleries[ref] = previous
			return ResidentGalleryRecord{}, fmt.Errorf("persist resident deletion: %w", err)
		}
	}
	return record, nil
}

func (s *UniversalStore[S, S3, D, D3, A]) ResidentGallery(ref string) (ResidentGalleryRecord, bool) {
	if s == nil || !foundationcontracts.IsOpaqueResidentRef(ref) {
		return ResidentGalleryRecord{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.residentGalleries[ref]
	return record, ok && !record.Deleted
}

func (s *UniversalStore[S, S3, D, D3, A]) RollbackResidentGallery(ref string, now time.Time) (ResidentGalleryRecord, error) {
	if s == nil || !foundationcontracts.IsOpaqueResidentRef(ref) {
		return ResidentGalleryRecord{}, errors.New("resident unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.residentGalleries[ref]
	if !ok || record.Deleted {
		return ResidentGalleryRecord{}, errors.New("resident unavailable")
	}
	previous := record
	if record.Generation > 0 {
		record.Generation--
	}
	record.GalleryStatus = "not_enrolled"
	record.ErrorCategory = ""
	if now.IsZero() {
		now = time.Now().UTC()
	}
	record.UpdatedAt = now.UTC()
	s.residentGalleries[ref] = record
	if s.dir != "" {
		if err := s.persistStateLocked(); err != nil {
			s.residentGalleries[ref] = previous
			return ResidentGalleryRecord{}, fmt.Errorf("persist gallery rollback: %w", err)
		}
	}
	return record, nil
}

func cloneResidentGalleryRecords(input map[string]ResidentGalleryRecord) map[string]ResidentGalleryRecord {
	result := make(map[string]ResidentGalleryRecord, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func cloneStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func validateResidentGalleryState(records map[string]ResidentGalleryRecord, idempotency map[string]string) error {
	for key, record := range records {
		if key != record.ResidentRef {
			return errors.New("resident reference key mismatch")
		}
		if err := record.Validate(); err != nil {
			return err
		}
		if record.Deleted && record.GalleryStatus != "disabled" {
			return errors.New("deleted resident must be disabled")
		}
	}
	for hash, ref := range idempotency {
		decoded, err := hex.DecodeString(hash)
		if err != nil || len(decoded) != sha256.Size {
			return errors.New("invalid resident idempotency record")
		}
		if _, ok := records[ref]; !ok {
			return errors.New("orphan resident idempotency record")
		}
	}
	return nil
}
