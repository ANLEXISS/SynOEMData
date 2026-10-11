package runtime

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"synora/pkg/contract"
)

func TestH1CameraObservationProducerMatchesPublicBoundaryGolden(t *testing.T) {
	publisher := &fakePublisher{}
	registry := NewRegistry(publisher)
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	observation := contract.CameraObservation{SchemaVersion: 1, ObservationID: "camera-observation-h3", CameraID: "camera-h3", HardwareID: "synthetic-hardware", Endpoint: "rtsp://127.0.0.1/synthetic", Firmware: "fixture", Capabilities: []string{"motion_detection", "vision"}, Online: true, LastSeen: at}
	if err := registry.ObserveCamera(observation); err != nil {
		t.Fatal(err)
	}
	if len(publisher.messages) != 1 {
		t.Fatalf("published messages=%d", len(publisher.messages))
	}
	got, err := json.Marshal(publisher.messages[0])
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/camera-observation-event-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatal(err)
	}
	got, _ = json.Marshal(gotValue)
	want, _ = json.Marshal(wantValue)
	if !bytes.Equal(got, want) {
		t.Fatalf("Discovery camera event changed: got %s want %s", got, want)
	}
}

func TestH3CameraUnavailableAndRecoveryUsesRegistryBoundary(t *testing.T) {
	publisher := &fakePublisher{}
	registry := NewRegistry(publisher)
	base := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	observation := contract.CameraObservation{SchemaVersion: 1, ObservationID: "camera-availability-1", CameraID: "camera-h3", Capabilities: []string{"vision"}, Online: true, LastSeen: base}
	if err := registry.ObserveCamera(observation); err != nil {
		t.Fatal(err)
	}
	if !registry.MarkCameraOffline("camera-h3", base.Add(time.Second)) {
		t.Fatal("online camera did not transition offline")
	}
	if got := registry.Snapshot(); len(got) != 1 || got[0].Online {
		t.Fatalf("offline projection=%#v", got)
	}
	observation.ObservationID = "camera-availability-2"
	observation.Online = true
	observation.LastSeen = base.Add(2 * time.Second)
	if err := registry.ObserveCamera(observation); err != nil {
		t.Fatal(err)
	}
	got := registry.Snapshot()
	if len(got) != 1 || !got[0].Online || !got[0].LastSeen.Equal(observation.LastSeen) {
		t.Fatalf("recovery projection=%#v", got)
	}
	if len(publisher.messages) != 3 || publisher.messages[0].Type != contract.EventDiscoveryCameraObserved || publisher.messages[1].Type != contract.EventDiscoveryCameraOffline || publisher.messages[2].Type != contract.EventDiscoveryCameraObserved {
		t.Fatalf("availability event sequence changed: %#v", publisher.messages)
	}
}
