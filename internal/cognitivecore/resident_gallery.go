package cognitivecore

import (
	"errors"
	"time"

	foundationcontracts "synora/internal/foundation/contracts"
)

// Historical Core exports remain aliases; Foundation contracts and statestore
// own the resident record and its persisted mutation behavior.
type ResidentGalleryRecord = foundationcontracts.ResidentGalleryRecord

const ResidentGalleryPolicyVersion = foundationcontracts.ResidentGalleryPolicyVersion

func (s *UniversalStore) CreateResidentGallery(record ResidentGalleryRecord, idempotencyKey string) (ResidentGalleryRecord, bool, error) {
	if s == nil || s.impl == nil {
		return ResidentGalleryRecord{}, false, errors.New("Universal Store unavailable")
	}
	return s.impl.CreateResidentGallery(record, idempotencyKey)
}
func (s *UniversalStore) DeleteResidentGallery(ref string, now time.Time) (ResidentGalleryRecord, error) {
	if s == nil || s.impl == nil {
		return ResidentGalleryRecord{}, errors.New("resident unavailable")
	}
	return s.impl.DeleteResidentGallery(ref, now)
}
func (s *UniversalStore) ResidentGallery(ref string) (ResidentGalleryRecord, bool) {
	if s == nil || s.impl == nil {
		return ResidentGalleryRecord{}, false
	}
	return s.impl.ResidentGallery(ref)
}
func (s *UniversalStore) RollbackResidentGallery(ref string, now time.Time) (ResidentGalleryRecord, error) {
	if s == nil || s.impl == nil {
		return ResidentGalleryRecord{}, errors.New("resident unavailable")
	}
	return s.impl.RollbackResidentGallery(ref, now)
}
