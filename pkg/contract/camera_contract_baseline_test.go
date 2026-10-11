package contract

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestH1CameraObservationFixtureRoundTrip(t *testing.T) {
	fixture, err := os.ReadFile("testdata/camera-observation-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var observation CameraObservation
	if err := json.Unmarshal(fixture, &observation); err != nil {
		t.Fatal(err)
	}
	if observation.LastSeen != time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) {
		t.Fatalf("timestamp changed: %s", observation.LastSeen)
	}
	if got, err := json.Marshal(observation.Canonical()); err != nil {
		t.Fatal(err)
	} else {
		var gotValue, wantValue any
		if err := json.Unmarshal(got, &gotValue); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(fixture, &wantValue); err != nil {
			t.Fatal(err)
		}
		got, _ = json.Marshal(gotValue)
		want, _ := json.Marshal(wantValue)
		if !bytes.Equal(got, want) {
			t.Fatalf("camera observation changed:\n got %s\nwant %s", got, want)
		}
	}
}

func TestH1MachineReadableContractIndexReferencesExistingFixtures(t *testing.T) {
	indexPath := filepath.Join("..", "..", "testdata", "contract-baseline", "index.json")
	body, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Baseline  string `json:"baseline"`
		Contracts []struct {
			Fixtures []string `json:"fixtures"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal(body, &index); err != nil {
		t.Fatal(err)
	}
	if index.Baseline != "synora-oem-h1-contract-compatibility/v1" || len(index.Contracts) < 10 {
		t.Fatalf("incomplete contract index: baseline=%q contracts=%d", index.Baseline, len(index.Contracts))
	}
	for _, item := range index.Contracts {
		if len(item.Fixtures) == 0 {
			t.Fatal("indexed contract has no fixture reference")
		}
		for _, fixture := range item.Fixtures {
			if _, err := os.Stat(filepath.Join("..", "..", filepath.FromSlash(fixture))); err != nil {
				t.Errorf("indexed fixture %q is unavailable: %v", fixture, err)
			}
		}
	}
}

func TestH1MessageTimestampAndUnknownFieldCompatibility(t *testing.T) {
	tests := []struct {
		input   string
		seconds int64
	}{
		{`{"type":"sensor.motion","source":"fixture","timestamp":"2026-10-11T12:00:00.25Z","future_field":true}`, 1791720000},
		{`{"type":"sensor.motion","source":"fixture","timestamp":1791720000}`, 1791720000},
		{`{"type":"sensor.motion","source":"fixture","timestamp":1791720000000}`, 1791720000},
	}
	for _, test := range tests {
		var message Message
		if err := json.Unmarshal([]byte(test.input), &message); err != nil {
			t.Fatal(err)
		}
		if message.Timestamp.Unix() != test.seconds {
			t.Errorf("timestamp %s decoded as %d", test.input, message.Timestamp.Unix())
		}
	}
	var invalid Message
	if err := json.Unmarshal([]byte(`{"type":"sensor.motion","source":"fixture","timestamp":{"seconds":1}}`), &invalid); err != nil {
		t.Fatal(err)
	}
	if !invalid.Timestamp.IsZero() {
		t.Fatalf("invalid timestamp should retain current zero-time fallback: %s", invalid.Timestamp)
	}
	var invalidSequence Message
	if err := json.Unmarshal([]byte(`{"type":"sensor.motion","source":"fixture","sequence":"not-a-number"}`), &invalidSequence); err == nil {
		t.Fatal("invalid numeric sequence unexpectedly accepted")
	}
}
