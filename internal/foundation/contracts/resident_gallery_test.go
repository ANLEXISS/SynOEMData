package contracts

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResidentGalleryRecordJSONCompatibility(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 30, 0, 0, time.UTC)
	record := ResidentGalleryRecord{
		ResidentRef:   "res_0123456789abcdef0123456789abcdef",
		GalleryStatus: "prepared",
		Generation:    3,
		PolicyVersion: ResidentGalleryPolicyVersion,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	want := `{"resident_ref":"res_0123456789abcdef0123456789abcdef","gallery_status":"prepared","gallery_generation":3,"policy_version":"face-gallery-policy/v1","created_at":"2026-10-10T12:30:00Z","updated_at":"2026-10-10T12:30:00Z"}`

	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != want {
		t.Fatalf("serialized record changed:\n got %s\nwant %s", encoded, want)
	}
	var restored ResidentGalleryRecord
	if err := json.Unmarshal([]byte(want), &restored); err != nil {
		t.Fatalf("restore serialized record: %v", err)
	}
	if restored != record {
		t.Fatalf("restored record differs:\n got %#v\nwant %#v", restored, record)
	}
	if err := restored.Validate(); err != nil {
		t.Fatalf("restored record did not validate: %v", err)
	}
}

func TestResidentGalleryRecordOptionalJSONFields(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 30, 0, 0, time.UTC)
	record := ResidentGalleryRecord{
		ResidentRef:   "res_0123456789abcdef0123456789abcdef",
		GalleryStatus: "error",
		PolicyVersion: ResidentGalleryPolicyVersion,
		ErrorCategory: "worker_unavailable",
		Deleted:       true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["last_error_category"]) != `"worker_unavailable"` || string(fields["deleted"]) != "true" {
		t.Fatalf("optional fields changed representation: %s", encoded)
	}
	if _, ok := fields["gallery_generation"]; !ok || string(fields["gallery_generation"]) != "0" {
		t.Fatalf("zero generation must remain present: %s", encoded)
	}
}
