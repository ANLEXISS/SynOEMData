// Package contracts contains Foundation-owned records shared across runtime
// modules. Persistence operations and store policy remain with their owners.
package contracts

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

const ResidentGalleryPolicyVersion = "face-gallery-policy/v1"

var opaqueResidentRef = regexp.MustCompile(`^res_[a-f0-9]{32}$`)

// ResidentGalleryRecord is an identity-free projection of resident gallery
// state. Names, images, embeddings, scores, paths, and track identifiers are
// intentionally excluded.
type ResidentGalleryRecord struct {
	ResidentRef   string    `json:"resident_ref"`
	GalleryStatus string    `json:"gallery_status"`
	Generation    uint64    `json:"gallery_generation"`
	PolicyVersion string    `json:"policy_version"`
	ErrorCategory string    `json:"last_error_category,omitempty"`
	Deleted       bool      `json:"deleted,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Validate checks the stable, identity-free resident gallery record contract.
func (r ResidentGalleryRecord) Validate() error {
	if !IsOpaqueResidentRef(r.ResidentRef) {
		return errors.New("invalid opaque resident reference")
	}
	switch r.GalleryStatus {
	case "not_enrolled", "prepared", "unavailable", "disabled", "error":
	default:
		return errors.New("invalid face gallery status")
	}
	if r.PolicyVersion != ResidentGalleryPolicyVersion {
		return errors.New("invalid face gallery policy version")
	}
	if len(r.ErrorCategory) > 64 || strings.ContainsAny(r.ErrorCategory, "/\\\n\r") {
		return errors.New("invalid redacted error category")
	}
	if r.CreatedAt.IsZero() || r.UpdatedAt.IsZero() || r.UpdatedAt.Before(r.CreatedAt) {
		return errors.New("invalid resident record timestamps")
	}
	return nil
}

// IsOpaqueResidentRef reports whether ref has the stable opaque reference
// representation used by ResidentGalleryRecord.
func IsOpaqueResidentRef(ref string) bool { return opaqueResidentRef.MatchString(ref) }
