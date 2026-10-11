package contract

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestH1SegmentReadySerializerGolden(t *testing.T) {
	segment := testSegment(0, false)
	got, err := segment.MarshalPayload()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/vision-segment-ready-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var value, expected any
	if err := json.Unmarshal(got, &value); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &expected); err != nil {
		t.Fatal(err)
	}
	got, _ = json.Marshal(value)
	want, _ = json.Marshal(expected)
	if !bytes.Equal(got, want) {
		t.Fatalf("segment-ready serializer changed:\n got %s\nwant %s", got, want)
	}
	decoded, err := DecodeVisionSegmentReadyV1(want)
	if err != nil {
		var raw VisionSegmentReadyV1
		_ = json.Unmarshal(want, &raw)
		t.Fatalf("decode fixture %#v: %v; direct validate=%v", raw, err, raw.Validate())
	}
	if decoded.SegmentID != segment.SegmentID || decoded.CameraID != segment.CameraID || decoded.StartedAt != segment.StartedAt || decoded.EndedAt != segment.EndedAt {
		t.Fatalf("segment-ready fixture semantics changed: %#v", decoded)
	}
}
