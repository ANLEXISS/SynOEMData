// Package testkit provides test-only deterministic device and fault helpers.
// It is not a production module and does not perform physical I/O.
package testkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const ScenarioSchemaVersion = "synora.oem-test-scenario/v1"

type Fixture struct {
	SchemaVersion string     `json:"schema_version"`
	Scenarios     []Scenario `json:"scenarios"`
}

type Scenario struct {
	ID            string       `json:"scenario_id"`
	SchemaVersion string       `json:"schema_version"`
	Kind          string       `json:"classification"`
	Boundary      string       `json:"boundary"`
	LogicalClock  time.Time    `json:"logical_clock"`
	Seed          int64        `json:"seed"`
	Devices       []DeviceSpec `json:"devices"`
	Steps         []Step       `json:"steps"`
	Expected      Expected     `json:"expected"`
}

type DeviceSpec struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	Capabilities []string          `json:"capabilities"`
	Available    bool              `json:"available"`
	Connected    bool              `json:"connected"`
	State        map[string]string `json:"state"`
}

type Step struct {
	ID          string            `json:"id"`
	Action      string            `json:"action"`
	DeviceID    string            `json:"device_id,omitempty"`
	Operation   string            `json:"operation,omitempty"`
	Capability  string            `json:"capability,omitempty"`
	Response    string            `json:"response,omitempty"`
	Fault       string            `json:"fault,omitempty"`
	EventID     string            `json:"event_id,omitempty"`
	Sequence    uint64            `json:"sequence,omitempty"`
	OccurredAt  time.Time         `json:"occurred_at,omitempty"`
	DelayMillis int64             `json:"delay_ms,omitempty"`
	Payload     json.RawMessage   `json:"payload,omitempty"`
	StateAfter  map[string]string `json:"state_after,omitempty"`
}

type Expected struct {
	Trace            []string          `json:"trace"`
	ObservationIDs   []string          `json:"observation_ids"`
	DeviceState      map[string]string `json:"device_state"`
	DeviceAvailable  bool              `json:"device_available"`
	DeviceConnected  bool              `json:"device_connected"`
	PhysicalExecuted bool              `json:"physical_action_executed"`
	CleanupComplete  bool              `json:"cleanup_complete"`
	FinalClock       time.Time         `json:"final_clock"`
}

type Request struct {
	ID         string
	Operation  string
	Capability string
	At         time.Time
	Payload    json.RawMessage
}

type Observation struct {
	ID         string
	Sequence   uint64
	OccurredAt time.Time
	Payload    json.RawMessage
}

type Result struct {
	Trace            []string
	Requests         []Request
	Observations     []Observation
	DeviceState      map[string]string
	DeviceAvailable  bool
	DeviceConnected  bool
	PhysicalExecuted bool
	CleanupComplete  bool
	FinalClock       time.Time
}

func DecodeFixture(data []byte) (Fixture, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var fixture Fixture
	if err := decoder.Decode(&fixture); err != nil {
		return Fixture{}, fmt.Errorf("decode scenario fixture: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Fixture{}, errors.New("scenario fixture contains trailing JSON")
	}
	if fixture.SchemaVersion != ScenarioSchemaVersion || len(fixture.Scenarios) == 0 {
		return Fixture{}, errors.New("unsupported or empty scenario fixture")
	}
	seen := make(map[string]bool)
	for _, scenario := range fixture.Scenarios {
		if err := scenario.Validate(); err != nil {
			return Fixture{}, fmt.Errorf("scenario %q: %w", scenario.ID, err)
		}
		if seen[scenario.ID] {
			return Fixture{}, fmt.Errorf("duplicate scenario id %q", scenario.ID)
		}
		seen[scenario.ID] = true
	}
	return fixture, nil
}

func (s Scenario) Validate() error {
	if s.ID == "" || s.SchemaVersion != ScenarioSchemaVersion || s.Kind == "" || s.Boundary == "" || s.LogicalClock.IsZero() || len(s.Devices) != 1 || len(s.Steps) == 0 {
		return errors.New("scenario metadata, clock, devices, and steps are required")
	}
	devices := map[string]bool{}
	for _, device := range s.Devices {
		if device.ID == "" || !validDeviceKind(device.Kind) {
			return fmt.Errorf("invalid device %q", device.ID)
		}
		if devices[device.ID] {
			return fmt.Errorf("duplicate device %q", device.ID)
		}
		devices[device.ID] = true
	}
	for _, step := range s.Steps {
		if step.ID == "" || !validAction(step.Action) {
			return fmt.Errorf("invalid step %q", step.ID)
		}
		if step.DeviceID == "" {
			return fmt.Errorf("step %q has no device", step.ID)
		}
		if step.DeviceID != "" && !devices[step.DeviceID] {
			return fmt.Errorf("step %q references missing device", step.ID)
		}
		if step.Response != "" && step.Response != "success" && step.Response != "unsupported" && step.Response != "unavailable" && step.Response != "error" {
			return fmt.Errorf("step %q has unknown scripted response %q", step.ID, step.Response)
		}
		if step.Fault != "" && !validFault(step.Fault) {
			return fmt.Errorf("step %q has unknown fault %q", step.ID, step.Fault)
		}
		if step.Action == "observe" && step.EventID == "" {
			return fmt.Errorf("observation step %q has no event id", step.ID)
		}
	}
	if !s.Expected.CleanupComplete {
		return errors.New("scenario must require cleanup")
	}
	return nil
}

func validDeviceKind(value string) bool {
	return value == "camera" || value == "sensor" || value == "network" || value == "abstract"
}
func validAction(value string) bool {
	return value == "operate" || value == "observe" || value == "recover" || value == "restart"
}
func validFault(value string) bool {
	switch value {
	case "device_unavailable", "unsupported_operation", "timeout", "delayed_response", "duplicate_observation", "out_of_order_observation", "missing_observation", "malformed_payload", "adapter_error", "device_recovery", "connection_interruption", "restart", "stale_observation", "sequence_gap", "store_write_failure", "outbox_retry":
		return true
	default:
		return false
	}
}

type Device struct {
	initial      DeviceSpec
	spec         DeviceSpec
	clock        *Clock
	requests     []Request
	observations []Observation
}

type Clock struct{ initial, current time.Time }

func NewClock(at time.Time) *Clock { at = at.UTC(); return &Clock{initial: at, current: at} }
func (c *Clock) Now() time.Time    { return c.current }
func (c *Clock) Advance(d time.Duration) {
	if d > 0 {
		c.current = c.current.Add(d)
	}
}
func (c *Clock) Reset() { c.current = c.initial }

func NewDevice(spec DeviceSpec, clock *Clock) *Device {
	initial := cloneDeviceSpec(spec)
	return &Device{initial: initial, spec: cloneDeviceSpec(spec), clock: clock}
}

func (d *Device) Execute(step Step) string {
	d.requests = append(d.requests, Request{ID: step.ID, Operation: step.Operation, Capability: step.Capability, At: d.clock.Now(), Payload: append(json.RawMessage(nil), step.Payload...)})
	if step.Fault == "device_unavailable" {
		d.spec.Available = false
		return "unavailable"
	}
	if step.Fault == "device_recovery" {
		d.Recover()
		return "recovered"
	}
	if step.Fault == "restart" {
		d.Restart()
		return "restarted"
	}
	if step.Fault == "connection_interruption" {
		d.spec.Connected = false
		return "connection_interrupted"
	}
	if !d.spec.Connected {
		return "connection_interrupted"
	}
	if !d.spec.Available {
		return "unavailable"
	}
	if step.Fault == "unsupported_operation" || !contains(d.spec.Capabilities, step.Capability) {
		return "unsupported"
	}
	switch step.Response {
	case "unsupported":
		return "unsupported"
	case "unavailable":
		return "unavailable"
	case "error":
		return "adapter_error"
	}
	switch step.Fault {
	case "timeout":
		return "timeout"
	case "delayed_response":
		d.clock.Advance(time.Duration(step.DelayMillis) * time.Millisecond)
		return "delayed"
	case "adapter_error", "store_write_failure":
		return "adapter_error"
	case "outbox_retry":
		return "retry_scheduled"
	case "malformed_payload":
		return "malformed_payload"
	}
	for key, value := range step.StateAfter {
		d.spec.State[key] = value
	}
	return "simulated"
}

func (d *Device) Recover() { d.spec.Available, d.spec.Connected = true, true }
func (d *Device) Reset() {
	d.spec = cloneDeviceSpec(d.initial)
	d.requests = nil
	d.observations = nil
}
func (d *Device) Snapshot() DeviceSpec { return cloneDeviceSpec(d.spec) }
func (d *Device) Requests() []Request {
	out := append([]Request(nil), d.requests...)
	for i := range out {
		out[i].Payload = append(json.RawMessage(nil), out[i].Payload...)
	}
	return out
}
func (d *Device) Observations() []Observation {
	out := append([]Observation(nil), d.observations...)
	for i := range out {
		out[i].Payload = append(json.RawMessage(nil), out[i].Payload...)
	}
	return out
}

func (d *Device) Observe(step Step) string {
	if step.Fault == "missing_observation" {
		return "missing"
	}
	if step.Fault == "malformed_payload" || !isObject(step.Payload) {
		return "malformed_payload"
	}
	observation := Observation{ID: step.EventID, Sequence: step.Sequence, OccurredAt: step.OccurredAt.UTC(), Payload: append(json.RawMessage(nil), step.Payload...)}
	if step.Fault == "duplicate_observation" {
		d.observations = append(d.observations, observation, observation)
		return "duplicate_observation"
	}
	d.observations = append(d.observations, observation)
	switch step.Fault {
	case "out_of_order_observation":
		return "out_of_order_observation"
	case "stale_observation":
		return "stale_observation"
	case "sequence_gap":
		return "sequence_gap"
	default:
		return "observed"
	}
}

func (d *Device) Restart() { d.Reset() }

func Run(s Scenario) (Result, error) {
	if err := s.Validate(); err != nil {
		return Result{}, err
	}
	clock := NewClock(s.LogicalClock)
	devices := make(map[string]*Device, len(s.Devices))
	for _, spec := range s.Devices {
		devices[spec.ID] = NewDevice(spec, clock)
	}
	result := Result{PhysicalExecuted: false, CleanupComplete: true}
	for _, step := range s.Steps {
		var status string
		device := devices[step.DeviceID]
		switch step.Action {
		case "operate":
			status = device.Execute(step)
		case "observe":
			status = device.Observe(step)
		case "recover":
			device.Recover()
			status = "recovered"
		case "restart":
			device.Restart()
			status = "restarted"
		}
		result.Trace = append(result.Trace, step.ID+":"+status)
	}
	ids := make([]string, 0, len(devices))
	for id := range devices {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		device := devices[id]
		result.Requests = append(result.Requests, device.requests...)
		result.Observations = append(result.Observations, device.observations...)
		result.DeviceState = device.Snapshot().State
		result.DeviceAvailable = device.spec.Available
		result.DeviceConnected = device.spec.Connected
	}
	result.FinalClock = clock.Now()
	return result, nil
}

func MatchesExpected(got Result, want Expected) error {
	if strings.Join(got.Trace, "\x00") != strings.Join(want.Trace, "\x00") {
		return fmt.Errorf("trace=%v want=%v", got.Trace, want.Trace)
	}
	ids := make([]string, 0, len(got.Observations))
	for _, observation := range got.Observations {
		ids = append(ids, observation.ID)
	}
	if strings.Join(ids, "\x00") != strings.Join(want.ObservationIDs, "\x00") {
		return fmt.Errorf("observation ids=%v want=%v", ids, want.ObservationIDs)
	}
	if !equalStringMap(got.DeviceState, want.DeviceState) || got.DeviceAvailable != want.DeviceAvailable || got.DeviceConnected != want.DeviceConnected || got.PhysicalExecuted != want.PhysicalExecuted || got.CleanupComplete != want.CleanupComplete || !got.FinalClock.Equal(want.FinalClock) {
		return fmt.Errorf("state/clock/result mismatch: state=%v available=%v connected=%v physical=%v cleanup=%v clock=%s", got.DeviceState, got.DeviceAvailable, got.DeviceConnected, got.PhysicalExecuted, got.CleanupComplete, got.FinalClock)
	}
	return nil
}

func cloneDeviceSpec(spec DeviceSpec) DeviceSpec {
	spec.Capabilities = append([]string(nil), spec.Capabilities...)
	copyState := make(map[string]string, len(spec.State))
	for key, value := range spec.State {
		copyState[key] = value
	}
	spec.State = copyState
	return spec
}
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func isObject(data []byte) bool {
	var value map[string]json.RawMessage
	return len(data) > 0 && json.Unmarshal(data, &value) == nil && value != nil
}
func equalStringMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
