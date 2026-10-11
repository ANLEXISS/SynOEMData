package contract

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

const (
	EventCategoryAction     = "action"
	EventCategorySecurity   = "security"
	EventCategorySimulation = "simulation"
	EventCategorySystem     = "system"
	EventCategoryVision     = "vision"
)

const (
	PriorityLow      = 25
	PriorityNormal   = 50
	PriorityHigh     = 75
	PriorityCritical = 100
)

/*
EVENT TYPES
*/

const (

	// EventVisionIdentity reports that a known resident or identity was recognized by vision.
	EventVisionIdentity = "vision.identity"
	// EventVisionEnd closes one Vision activation. Retries are idempotent at Core.
	EventVisionEnd = "vision.end"
	// EventVisionUnknown reports a detected person or subject with no known identity.
	EventVisionUnknown = "vision.unknown"
	// EventVisionUncertain reports a low-confidence identity or ambiguous visual classification.
	EventVisionUncertain = "vision.uncertain"
	// EventVisionMotion reports motion detected by the vision pipeline.
	EventVisionMotion = "vision.motion"
	// EventVisionWeapon reports a potential weapon detection.
	EventVisionWeapon = "vision.weapon"
	// EventVisionFall reports a potential person fall detection.
	EventVisionFall = "vision.fall"
	// EventVisionFight reports a potential fight detection.
	EventVisionFight = "vision.fight"
	// EventVisionTamper reports camera obstruction, movement, or tampering.
	EventVisionTamper = "vision.tamper"
	// EventVisionClipSummaryV1 is the immutable per-track V1 clip observation.
	EventVisionClipSummaryV1 = "synora.vision.clip-summary/v1"
	// EventVisionPreliminaryAlertV1 is emitted only for a configured strong critical detection.
	EventVisionPreliminaryAlertV1 = "synora.vision.preliminary-alert/v1"
	EventVisionClipObservationV1  = "synora.vision.clip-observation/v1"
	EventVisionSegmentReadyV1     = "synora.vision.segment-ready/v1"
	EventVisionSegmentGapV1       = "synora.vision.segment-gap/v1"
	EventVisionContinuityResetV1  = "synora.vision.continuity-reset/v1"
	// EventVisionEnrichmentV3 carries the separate aggregate-only V3 candidate.
	EventVisionEnrichmentV3 = "synora.vision.enrichment/v3"

	// Device events
	EventDeviceTrigger = "device.trigger"
	// EventDeviceOffline reports that a device or camera is no longer reachable.
	EventDeviceOffline = "device.offline"

	// Discovery events
	EventDiscoveryCameraObserved          = "discovery.camera.observed"
	EventDiscoveryCameraOnline            = "discovery.camera.online"
	EventDiscoveryCameraOffline           = "discovery.camera.offline"
	EventDiscoveryWorkerStarted           = "discovery.worker.started"
	EventDiscoveryWorkerStopped           = "discovery.worker.stopped"
	EventDiscoveryWorkerCrashed           = "discovery.worker.crashed"
	EventDiscoveryVisionWorkerUnavailable = "discovery.vision_worker.unavailable"
	EventDiscoveryNetworkDegraded         = "discovery.network.degraded"
	EventDiscoveryVisionIngressStatus     = "discovery.vision_ingress.status"
	EventDiscoveryRuntimeStatus           = "discovery.runtime.status"
	EventClipReady                        = "clip.ready"
	EventClipProcessing                   = "clip.processing"
	EventClipProcessed                    = "clip.processed"
	EventClipFailed                       = "clip.failed"
	EventRuntimeComponentFlapping         = "runtime.component.flapping"
	EventRuntimeModelMissing              = "runtime.model.missing"

	// System events
	// EventSystemStateChanged reports a state transition published by the Core.
	EventSystemStateChanged = "system.state.changed"
	EventSystemPresence     = "system.presence.updated"
	EventSystemUnknown      = "system.unknown"

	// EventActionRequest asks the Actions service to execute an action.
	EventActionRequest = "action.request"
	// EventActionResult reports the outcome of an action request.
	EventActionResult         = "action.result"
	EventActionServiceStarted = "action.service.started"
	EventManualRisk           = "manual.risk"
	EventSystemStateReset     = "system.state.reset"
	EventSecurityModeChanged  = "security.mode.changed"

	// EventValidationTestInference is a bounded API-to-Core envelope. The
	// catalogued contract event type is carried in its redacted payload; the
	// envelope is never accepted from Discovery and never becomes a business
	// event in the Core store.
	EventValidationTestInference = "validation.test-inference"
	EventSensorNormal            = "sensor.normal"
	EventSensorAnomaly           = "sensor.anomaly"
	EventWebCommand              = "discovery.web.command"
)

/*
EVENT STRUCT
*/

type Event struct {
	ID string `json:"id,omitempty"`

	Type string `json:"type"`

	Source string `json:"source"`

	Timestamp time.Time `json:"timestamp,omitempty"`

	Payload map[string]any `json:"payload,omitempty"`

	DeviceID   string    `json:"device_id,omitempty"`
	NodeID     string    `json:"node_id,omitempty"`
	Epoch      string    `json:"epoch,omitempty"`
	Sequence   uint64    `json:"sequence,omitempty"`
	ReceivedAt time.Time `json:"received_at,omitempty"`

	Identity   string  `json:"identity,omitempty"`
	ResidentID string  `json:"resident_id,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`

	Priority     int    `json:"priority,omitempty"`
	GroupKey     string `json:"group_key,omitempty"`
	TrackID      string `json:"track_id,omitempty"`
	ClipID       string `json:"clip_id,omitempty"`
	ClipIndex    int    `json:"clip_index,omitempty"`
	ActivationID string `json:"activation_id,omitempty"`
	SequenceKey  string `json:"sequence_key,omitempty"`

	ValidationRequired bool   `json:"validation_required,omitempty"`
	ValidationReason   string `json:"validation_reason,omitempty"`
}

// Clone returns an independently owned Event, including recursively nested
// maps, slices, arrays, pointers, and exported fields inside payload structs.
// Event payloads remain typed as the existing contract; cloning does not
// round-trip values through JSON or change their concrete Go types.
func (e Event) Clone() Event {
	if e.Payload == nil {
		return e
	}
	cloned := cloneEventValue(reflect.ValueOf(e.Payload), make(map[eventCloneVisit]reflect.Value)).Interface()
	e.Payload = cloned.(map[string]any)
	return e
}

type eventCloneVisit struct {
	typeOf reflect.Type
	kind   reflect.Kind
	ptr    uintptr
}

func cloneEventValue(value reflect.Value, visited map[eventCloneVisit]reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copy := reflect.New(value.Type()).Elem()
		copy.Set(cloneEventValue(value.Elem(), visited))
		return copy
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		visit := eventCloneVisit{value.Type(), value.Kind(), value.Pointer()}
		if previous, ok := visited[visit]; ok {
			return previous
		}
		copy := reflect.New(value.Type().Elem())
		visited[visit] = copy
		copy.Elem().Set(cloneEventValue(value.Elem(), visited))
		return copy
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		visit := eventCloneVisit{value.Type(), value.Kind(), value.Pointer()}
		if previous, ok := visited[visit]; ok {
			return previous
		}
		if value.Type() == reflect.TypeOf(map[string]any{}) {
			result := make(map[string]any, value.Len())
			copy := reflect.ValueOf(result)
			visited[visit] = copy
			for key, item := range value.Interface().(map[string]any) {
				result[key] = cloneEventAny(item, visited)
			}
			return copy
		}
		if value.Type() == reflect.TypeOf(map[string]string{}) {
			result := make(map[string]string, value.Len())
			copy := reflect.ValueOf(result)
			visited[visit] = copy
			for key, item := range value.Interface().(map[string]string) {
				result[key] = item
			}
			return copy
		}
		if value.Type() == reflect.TypeOf(map[string][]string{}) {
			result := make(map[string][]string, value.Len())
			copy := reflect.ValueOf(result)
			visited[visit] = copy
			for key, item := range value.Interface().(map[string][]string) {
				result[key] = append([]string(nil), item...)
			}
			return copy
		}
		copy := reflect.MakeMapWithSize(value.Type(), value.Len())
		visited[visit] = copy
		iter := value.MapRange()
		for iter.Next() {
			copy.SetMapIndex(iter.Key(), cloneEventValue(iter.Value(), visited))
		}
		return copy
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		visit := eventCloneVisit{value.Type(), value.Kind(), value.Pointer()}
		if previous, ok := visited[visit]; ok {
			return previous
		}
		if value.Type() == reflect.TypeOf([]string{}) {
			copy := reflect.ValueOf(append([]string(nil), value.Interface().([]string)...))
			visited[visit] = copy
			return copy
		}
		if value.Type() == reflect.TypeOf([]byte{}) {
			copy := reflect.ValueOf(append([]byte(nil), value.Bytes()...))
			visited[visit] = copy
			return copy
		}
		copy := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		visited[visit] = copy
		for i := 0; i < value.Len(); i++ {
			copy.Index(i).Set(cloneEventValue(value.Index(i), visited))
		}
		return copy
	case reflect.Array:
		copy := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			copy.Index(i).Set(cloneEventValue(value.Index(i), visited))
		}
		return copy
	case reflect.Struct:
		copy := reflect.New(value.Type()).Elem()
		copy.Set(value)
		for i := 0; i < value.NumField(); i++ {
			if copy.Field(i).CanSet() && value.Field(i).CanInterface() {
				copy.Field(i).Set(cloneEventValue(value.Field(i), visited))
			}
		}
		return copy
	default:
		return value
	}
}

func cloneEventAny(value any, visited map[eventCloneVisit]reflect.Value) any {
	if value == nil {
		return nil
	}
	return cloneEventValue(reflect.ValueOf(value), visited).Interface()
}

type eventJSON struct {
	ID                 string         `json:"id,omitempty"`
	Type               string         `json:"type"`
	Source             string         `json:"source"`
	Timestamp          time.Time      `json:"timestamp,omitempty"`
	Payload            map[string]any `json:"payload,omitempty"`
	DeviceID           string         `json:"device_id,omitempty"`
	NodeID             string         `json:"node_id,omitempty"`
	Epoch              string         `json:"epoch,omitempty"`
	Sequence           uint64         `json:"sequence,omitempty"`
	ReceivedAt         time.Time      `json:"received_at,omitempty"`
	Identity           string         `json:"identity,omitempty"`
	ResidentID         string         `json:"resident_id,omitempty"`
	Confidence         float64        `json:"confidence,omitempty"`
	Priority           int            `json:"priority,omitempty"`
	GroupKey           string         `json:"group_key,omitempty"`
	TrackID            string         `json:"track_id,omitempty"`
	ClipID             string         `json:"clip_id,omitempty"`
	ClipIndex          int            `json:"clip_index,omitempty"`
	ActivationID       string         `json:"activation_id,omitempty"`
	SequenceKey        string         `json:"sequence_key,omitempty"`
	ValidationRequired bool           `json:"validation_required,omitempty"`
	ValidationReason   string         `json:"validation_reason,omitempty"`
}

func (e *Event) UnmarshalJSON(data []byte) error {
	var decoded eventJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*e = Event{
		ID:                 decoded.ID,
		Type:               decoded.Type,
		Source:             decoded.Source,
		Timestamp:          decoded.Timestamp,
		Payload:            decoded.Payload,
		DeviceID:           decoded.DeviceID,
		NodeID:             decoded.NodeID,
		Epoch:              decoded.Epoch,
		Sequence:           decoded.Sequence,
		ReceivedAt:         decoded.ReceivedAt,
		Identity:           decoded.Identity,
		ResidentID:         decoded.ResidentID,
		Confidence:         decoded.Confidence,
		Priority:           decoded.Priority,
		GroupKey:           decoded.GroupKey,
		TrackID:            decoded.TrackID,
		ClipID:             decoded.ClipID,
		ClipIndex:          decoded.ClipIndex,
		ActivationID:       decoded.ActivationID,
		SequenceKey:        decoded.SequenceKey,
		ValidationRequired: decoded.ValidationRequired,
		ValidationReason:   decoded.ValidationReason,
	}
	return nil
}

/*
EVENT WINDOW
*/

type EventWindow struct {
	NodeID string

	Events []*Event

	LastUpdate time.Time

	Score float64

	VisionKnown     int
	VisionUnknown   int
	VisionUncertain int
}

/*
TYPE HELPERS
*/

func IsVisionEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "vision.") || strings.HasPrefix(eventType, "synora.vision.")
}

func IsDeviceEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "device.")
}

func IsSystemEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "system.")
}

func EventCategory(eventType string) string {
	switch NormalizeEventType(eventType) {
	case EventDiscoveryWorkerStarted,
		EventDiscoveryWorkerStopped,
		EventDiscoveryWorkerCrashed,
		EventDiscoveryVisionWorkerUnavailable,
		EventDiscoveryNetworkDegraded,
		EventDiscoveryVisionIngressStatus,
		EventDiscoveryRuntimeStatus,
		EventRuntimeComponentFlapping,
		EventRuntimeModelMissing,
		EventClipReady,
		EventClipProcessing,
		EventClipProcessed,
		EventClipFailed,
		EventDiscoveryCameraObserved,
		EventDiscoveryCameraOnline,
		EventDiscoveryCameraOffline,
		EventDeviceOffline,
		EventSystemStateChanged,
		EventSystemPresence:
		return EventCategorySystem
	case EventVisionUnknown,
		EventVisionUncertain,
		EventVisionWeapon,
		EventVisionFall,
		EventVisionFight,
		EventVisionTamper:
		return EventCategorySecurity
	case EventManualRisk, EventSecurityModeChanged:
		return EventCategorySecurity
	case EventVisionIdentity,
		EventVisionEvidenceV1,
		EventVisionEnd,
		EventVisionMotion,
		EventVisionClipSummaryV1,
		EventVisionPreliminaryAlertV1,
		EventVisionClipObservationV1,
		EventVisionSegmentReadyV1,
		EventVisionSegmentGapV1,
		EventVisionContinuityResetV1,
		EventVisionEnrichmentV3:
		return EventCategoryVision
	case EventActionRequest,
		EventActionResult,
		EventActionServiceStarted:
		return EventCategoryAction
	default:
		if IsSystemEvent(eventType) || strings.HasPrefix(NormalizeEventType(eventType), "discovery.") {
			return EventCategorySystem
		}
		if IsVisionEvent(eventType) {
			return EventCategoryVision
		}
		return EventCategorySystem
	}
}

func IsUserValidationCandidate(eventType string) bool {
	switch NormalizeEventType(eventType) {
	case EventVisionUnknown,
		EventVisionUncertain,
		EventVisionWeapon,
		EventVisionFall,
		EventVisionFight,
		EventVisionTamper:
		return true
	default:
		return false
	}
}

/*
NORMALIZE EVENT TYPE
*/

func NormalizeEventType(raw string) string {

	raw = strings.TrimSpace(strings.ToLower(raw))

	if raw == "" {
		return EventSystemUnknown
	}

	switch raw {

	case
		EventVisionIdentity,
		EventVisionEvidenceV1,
		EventVisionUnknown,
		EventVisionUncertain,
		EventVisionEnd,
		EventVisionWeapon,
		EventVisionFall,
		EventVisionFight,
		EventVisionTamper,
		EventVisionMotion,
		EventVisionClipSummaryV1,
		EventVisionPreliminaryAlertV1,
		EventVisionClipObservationV1,
		EventVisionSegmentReadyV1,
		EventVisionSegmentGapV1,
		EventVisionContinuityResetV1,
		EventVisionEnrichmentV3,
		EventDeviceTrigger,
		EventDeviceOffline,
		EventDiscoveryCameraObserved,
		EventDiscoveryCameraOnline,
		EventDiscoveryCameraOffline,
		EventDiscoveryWorkerStarted,
		EventDiscoveryWorkerStopped,
		EventDiscoveryWorkerCrashed,
		EventDiscoveryVisionWorkerUnavailable,
		EventDiscoveryNetworkDegraded,
		EventDiscoveryVisionIngressStatus,
		EventDiscoveryRuntimeStatus,
		EventRuntimeComponentFlapping,
		EventRuntimeModelMissing,
		EventClipReady,
		EventClipProcessing,
		EventClipProcessed,
		EventClipFailed,
		EventSystemStateChanged,
		EventSystemPresence,
		EventSystemStateReset,
		EventActionRequest,
		EventActionServiceStarted,
		EventManualRisk,
		EventSecurityModeChanged:
		return raw

	case EventActionResult:

		return raw

	case "identity":
		return EventVisionIdentity

	case "unknown":
		return EventVisionUnknown

	case "motion":
		return EventVisionMotion

	case "trigger":
		return EventDeviceTrigger

	}

	if strings.Contains(raw, ".") {
		return raw
	}

	return "system." + raw
}

/*
EVENT PRIORITY
*/

func EventPriority(eventType string) int {

	eventType = NormalizeEventType(eventType)

	switch eventType {

	case EventVisionWeapon,
		EventVisionFight:
		return PriorityCritical

	case EventVisionTamper,
		EventVisionFall,
		EventDeviceOffline,
		EventDiscoveryCameraOffline,
		EventDiscoveryWorkerCrashed:
		return PriorityHigh

	case EventVisionUnknown,
		EventVisionUncertain,
		EventVisionIdentity,
		EventVisionMotion,
		EventVisionClipSummaryV1,
		EventVisionPreliminaryAlertV1,
		EventVisionClipObservationV1,
		EventVisionSegmentReadyV1,
		EventVisionSegmentGapV1,
		EventVisionContinuityResetV1,
		EventVisionEnrichmentV3,
		EventDeviceTrigger,
		EventDiscoveryCameraObserved,
		EventDiscoveryCameraOnline,
		EventDiscoveryWorkerStarted,
		EventDiscoveryWorkerStopped,
		EventActionRequest,
		EventActionResult:
		return PriorityNormal

	default:
		return PriorityLow
	}
}
