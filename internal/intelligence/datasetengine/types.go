// Package datasetengine builds deterministic, synthetic cognitive examples.
// Ground truth, source emissions, model inputs, and targets are separate
// artifacts; only an input artifact may be passed to a model pipeline.
package datasetengine

import (
	"time"

	"synora/internal/intelligence/stateencoder"
)

const (
	DatasetSchemaVersion = "synora.cognitive-dataset/v1"
	GeneratorVersion     = "1.0.0"
	DefaultHomeCount     = 12
)

type ScenarioID string

const (
	ScenarioResidentArrival    ScenarioID = "resident-arrival"
	ScenarioResidentDeparture  ScenarioID = "resident-departure"
	ScenarioRoomMovement       ScenarioID = "room-to-room-movement"
	ScenarioMultipleOccupants  ScenarioID = "multiple-occupants"
	ScenarioDoorOccupancy      ScenarioID = "door-occupancy-correlation"
	ScenarioLighting           ScenarioID = "lighting-transitions"
	ScenarioDailyRoutine       ScenarioID = "daily-routine"
	ScenarioRoutineChange      ScenarioID = "routine-change"
	ScenarioTemporaryGuest     ScenarioID = "temporary-guest"
	ScenarioUnexpectedPresence ScenarioID = "unexpected-presence"
	ScenarioSensorUnavailable  ScenarioID = "sensor-unavailable"
	ScenarioSensorRecovery     ScenarioID = "sensor-recovery"
	ScenarioDelayedObservation ScenarioID = "delayed-observations"
	ScenarioOutOfOrder         ScenarioID = "out-of-order-observations"
	ScenarioClockDrift         ScenarioID = "clock-drift"
	ScenarioConflictingSources ScenarioID = "conflicting-observations"
	ScenarioMissingEvent       ScenarioID = "missing-event"
	ScenarioNewDevice          ScenarioID = "new-device"
	ScenarioModifiedTopology   ScenarioID = "modified-topology"
	ScenarioUnseenHomeTopology ScenarioID = "unseen-home-topology"
)

type TaskID string

const (
	TaskNextState              TaskID = "A-next-state"
	TaskTemporalTransition     TaskID = "B-temporal-transition"
	TaskTopologyTransition     TaskID = "C-topology-transition"
	TaskBehaviorAnomaly        TaskID = "D-behavior-anomaly"
	TaskObservationConsistency TaskID = "E-observation-consistency"
)

type Split string

const (
	SplitTrain      Split = "train"
	SplitValidation Split = "validation"
	SplitTest       Split = "test"
)

type ScenarioSplit string

const (
	ScenarioSplitTrain   ScenarioSplit = "training-templates"
	ScenarioSplitHeldOut ScenarioSplit = "held-out-templates"
)

type Config struct {
	Seed      int64     `json:"seed"`
	HomeCount int       `json:"home_count"`
	StartTime time.Time `json:"start_time"`
}

type ScenarioDefinition struct {
	ID          ScenarioID `json:"id"`
	Name        string     `json:"name"`
	TaskIDs     []TaskID   `json:"task_ids"`
	HeldOut     bool       `json:"held_out_template"`
	Description string     `json:"description"`
}

type DatasetManifest struct {
	SchemaVersion string            `json:"schema_version"`
	Generator     string            `json:"generator_version"`
	Seed          int64             `json:"seed"`
	GeneratedAt   time.Time         `json:"generated_at"`
	Config        Config            `json:"config"`
	ScenarioIDs   []ScenarioID      `json:"scenario_ids"`
	HomeCount     int               `json:"home_count"`
	ExampleCount  int               `json:"example_count"`
	Counts        map[string]int    `json:"counts_by_split_and_scenario"`
	SeedLineage   map[string]uint64 `json:"seed_lineage"`
}

// Dataset deliberately stores ground truth, source emissions, inputs, and
// targets in separate top-level collections. WriteArtifacts persists them in
// separate directories as an additional leakage barrier.
type Dataset struct {
	Manifest        DatasetManifest        `json:"manifest"`
	Homes           []HomeArtifact         `json:"homes"`
	GroundTruth     []GroundTruthArtifact  `json:"-"`
	SourceEmissions []SourceArtifact       `json:"-"`
	Availability    []AvailabilityArtifact `json:"-"`
	Inputs          []InputArtifact        `json:"-"`
	Targets         []TargetArtifact       `json:"-"`
	Index           []ExampleIndex         `json:"example_index"`
}

type HomeArtifact struct {
	HomeGroupID      string                   `json:"home_group_id"`
	TopologyFamilyID string                   `json:"topology_family_id"`
	Split            Split                    `json:"split"`
	Entities         []stateencoder.EntityRef `json:"entities"`
	Topology         []stateencoder.Relation  `json:"topology"`
	SeedLineage      map[string]uint64        `json:"seed_lineage"`
}

type ExampleIndex struct {
	ExampleID             string            `json:"example_id"`
	HomeGroupID           string            `json:"home_group_id"`
	TopologyFamilyID      string            `json:"topology_family_id"`
	ScenarioFamily        ScenarioID        `json:"scenario_family"`
	HomeSplit             Split             `json:"home_split"`
	ScenarioTemplateSplit ScenarioSplit     `json:"scenario_template_split"`
	Cutoff                time.Time         `json:"prediction_cutoff"`
	SeedLineage           map[string]uint64 `json:"seed_lineage"`
}

type ArtifactMeta struct {
	ExampleID        string   `json:"example_id"`
	DatasetVersion   string   `json:"dataset_version"`
	GeneratorVersion string   `json:"generator_version"`
	ParentDigests    []string `json:"parent_digests,omitempty"`
	Digest           string   `json:"digest"`
}

type GroundTruthArtifact struct {
	Meta        ArtifactMeta            `json:"meta"`
	ScenarioID  ScenarioID              `json:"scenario_id"`
	Events      []TruthEvent            `json:"events"`
	FutureEdges []stateencoder.Relation `json:"future_edges,omitempty"`
	Annotations map[string]string       `json:"annotations"`
}

type TruthEvent struct {
	At        time.Time `json:"at"`
	EntityID  string    `json:"entity_id"`
	Attribute string    `json:"attribute"`
	Value     string    `json:"value"`
	Kind      string    `json:"kind"`
}

type SourceArtifact struct {
	Meta       ArtifactMeta     `json:"meta"`
	ScenarioID ScenarioID       `json:"scenario_id"`
	Emissions  []SourceEmission `json:"emissions"`
}

type SourceEmission struct {
	EmissionID string                      `json:"emission_id"`
	Entity     stateencoder.EntityRef      `json:"entity"`
	Attribute  stateencoder.AttributeRef   `json:"attribute"`
	Status     stateencoder.SemanticStatus `json:"status"`
	Value      *stateencoder.Value         `json:"value,omitempty"`
	Source     *stateencoder.SourceRef     `json:"source,omitempty"`
	Provenance string                      `json:"provenance"`
	ObservedAt time.Time                   `json:"observed_at,omitempty"`
	ReceivedAt time.Time                   `json:"received_at"`
}

type InputArtifact struct {
	Meta                 ArtifactMeta              `json:"meta"`
	AvailableEmissionIDs []string                  `json:"available_emission_ids"`
	Availability         []AvailabilityRecord      `json:"availability"`
	Batch                stateencoder.Batch        `json:"batch"`
	Encoded              stateencoder.EncodedBatch `json:"encoded"`
}

// ModelInput is the only artifact a training/evaluation adapter should read.
// It intentionally contains no split, home, scenario, ground-truth, or target
// metadata. Those fields stay in ExampleIndex and the hidden artifact stores.
type ModelInput struct {
	Batch        stateencoder.Batch        `json:"batch"`
	Encoded      stateencoder.EncodedBatch `json:"encoded"`
	Availability []AvailabilityRecord      `json:"availability"`
}

type ModelInputArtifact struct {
	Meta ArtifactMeta `json:"meta"`
	Data ModelInput   `json:"data"`
}

// AvailabilityRecord preserves per-emission receipt time alongside the
// encoder's batch-level assembly time. It contains only records selected by
// the causal cutoff.
type AvailabilityRecord struct {
	EmissionID string                    `json:"emission_id"`
	Entity     stateencoder.EntityRef    `json:"entity"`
	Attribute  stateencoder.AttributeRef `json:"attribute"`
	Source     *stateencoder.SourceRef   `json:"source,omitempty"`
	ObservedAt time.Time                 `json:"observed_at,omitempty"`
	ReceivedAt time.Time                 `json:"received_at"`
}

type AvailabilityArtifact struct {
	Meta    ArtifactMeta         `json:"meta"`
	Cutoff  time.Time            `json:"prediction_cutoff"`
	Records []AvailabilityRecord `json:"records"`
}

type TargetArtifact struct {
	Meta   ArtifactMeta `json:"meta"`
	Labels []TaskLabel  `json:"labels"`
}

type TaskLabel struct {
	TaskID         TaskID    `json:"task_id"`
	Value          string    `json:"value"`
	TargetTime     time.Time `json:"target_time"`
	AvailableAfter time.Time `json:"available_after"`
	Definition     string    `json:"definition"`
}
