package vision

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"synora/pkg/contract"
)

type baselineMessagePublisher struct{ messages []contract.Message }

func (p *baselineMessagePublisher) Send(message contract.Message) error {
	p.messages = append(p.messages, message)
	return nil
}

func TestH1SegmentGapAndContinuityResetPayloadGoldens(t *testing.T) {
	segment := contract.VisionSegmentReadyV1{
		SchemaVersion: contract.EventVisionSegmentReadyV1, CameraID: "cam-entry", NodeID: "entry", EpisodeID: "episode-1",
		SegmentIndex: 0, StartedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), EndedAt: time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
		TopologyClass: contract.VisionTopologyProtectedInterior, Trigger: "motion", MediaRef: "local://segments/segment.mp4",
		ContentSHA256: "139e7a317233dea4215e06f105db9d56f7ae3964ffb6ce7ace187dd162a2dccf",
	}
	segment.SegmentID = contract.DeterministicVisionSegmentID(segment.CameraID, segment.EpisodeID, segment.SegmentIndex, segment.ContentSHA256)
	publisher := &baselineMessagePublisher{}
	err := PublishSegmentRuntimeResult(publisher, segment, SegmentRuntimeResult{GapDetected: true, ContinuityReset: true, GapReason: "segment_gap_detected"})
	if err != nil {
		t.Fatal(err)
	}
	if len(publisher.messages) != 3 {
		t.Fatalf("messages=%d want ready/reset/gap", len(publisher.messages))
	}
	fixtures := map[string]string{contract.EventVisionContinuityResetV1: "segment-continuity-reset-v1.json", contract.EventVisionSegmentGapV1: "segment-gap-v1.json"}
	for _, message := range publisher.messages {
		name, ok := fixtures[message.Type]
		if !ok {
			continue
		}
		want, err := os.ReadFile("../../testdata/contract-baseline/vision/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var gotValue, wantValue any
		if err := json.Unmarshal(message.Payload, &gotValue); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(want, &wantValue); err != nil {
			t.Fatal(err)
		}
		got, _ := json.Marshal(gotValue)
		want, _ = json.Marshal(wantValue)
		if !bytes.Equal(got, want) {
			t.Fatalf("%s payload changed: got %s want %s", message.Type, got, want)
		}
	}
}
