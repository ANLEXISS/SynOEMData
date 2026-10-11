package testkit_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"synora/testkit"
)

func loadH3Fixture(t *testing.T) testkit.Fixture {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "testdata", "device-harness", "scenarios-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := testkit.DecodeFixture(body)
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestH3SharedScenariosAreDeterministicAndIsolated(t *testing.T) {
	fixture := loadH3Fixture(t)
	if len(fixture.Scenarios) != 10 {
		t.Fatalf("scenario count=%d want=10", len(fixture.Scenarios))
	}
	seenFaults := map[string]bool{}
	for _, scenario := range fixture.Scenarios {
		t.Run(scenario.ID, func(t *testing.T) {
			first, err := testkit.Run(scenario)
			if err != nil {
				t.Fatal(err)
			}
			if err := testkit.MatchesExpected(first, scenario.Expected); err != nil {
				t.Fatal(err)
			}
			second, err := testkit.Run(scenario)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("scenario is nondeterministic:\nfirst=%#v\nsecond=%#v", first, second)
			}
			if first.PhysicalExecuted {
				t.Fatal("test-only simulator claimed physical execution")
			}
		})
		for _, step := range scenario.Steps {
			if step.Fault != "" {
				seenFaults[step.Fault] = true
			}
		}
	}
	for _, fault := range []string{"device_unavailable", "unsupported_operation", "timeout", "delayed_response", "duplicate_observation", "out_of_order_observation", "missing_observation", "malformed_payload", "adapter_error", "device_recovery", "connection_interruption", "restart", "stale_observation", "sequence_gap", "store_write_failure", "outbox_retry"} {
		if !seenFaults[fault] {
			t.Errorf("shared scenario fixtures do not exercise fault %q", fault)
		}
	}
}

func TestH3ResetClearsDeviceStateRequestsAndObservations(t *testing.T) {
	fixture := loadH3Fixture(t)
	scenario := fixture.Scenarios[6]
	clock := testkit.NewClock(scenario.LogicalClock)
	device := testkit.NewDevice(scenario.Devices[0], clock)
	step := scenario.Steps[0]
	if got := device.Execute(step); got != "simulated" {
		t.Fatalf("execute=%q", got)
	}
	device.Observe(testkit.Step{ID: "observation", Action: "observe", EventID: "event-1", Payload: []byte(`{"movement":true}`)})
	if len(device.Requests()) != 1 || len(device.Observations()) != 1 {
		t.Fatal("device did not record requests and observations")
	}
	device.Reset()
	clock.Reset()
	reset := device.Snapshot()
	if reset.State["last_action"] != "none" || !reset.Available || !reset.Connected || len(device.Requests()) != 0 || len(device.Observations()) != 0 || !clock.Now().Equal(scenario.LogicalClock) {
		t.Fatalf("reset did not restore scenario initial state: %#v clock=%s", reset, clock.Now())
	}
}

func TestH3ScenarioFixtureRejectsUnknownSteps(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "testdata", "device-harness", "scenarios-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	bad := bytes.Replace(fixture, []byte(`"action":"observe"`), []byte(`"action":"execute-shell"`), 1)
	if _, err := testkit.DecodeFixture(bad); err == nil {
		t.Fatal("unknown scenario action accepted")
	}
}
