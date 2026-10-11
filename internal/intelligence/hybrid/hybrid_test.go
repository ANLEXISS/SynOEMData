package hybrid

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"synora/internal/intelligence/stateencoder"
)

type testLifecycle struct {
	metadata      ComponentMetadata
	initializeErr error
	shutdownErr   error
	initialized   bool
	shutdown      bool
}

func (c *testLifecycle) Metadata() ComponentMetadata { return c.metadata }
func (c *testLifecycle) Initialize(context.Context) error {
	if c.initializeErr == nil {
		c.initialized = true
	}
	return c.initializeErr
}
func (c *testLifecycle) Shutdown(context.Context) error {
	c.shutdown = true
	return c.shutdownErr
}

type eventDouble struct {
	testLifecycle
	err    error
	calls  int
	output []EventRepresentation
	mutate bool
}

func (c *eventDouble) EncodeEvents(_ context.Context, input EventInput) ([]EventRepresentation, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	if c.mutate && len(input.Observations) > 0 {
		input.Observations[0].Observation.Entity.ID = "branch-mutation"
	}
	if c.output != nil {
		return c.output, nil
	}
	result := make([]EventRepresentation, 0, len(input.Observations))
	for _, available := range input.Observations {
		observation := available.Observation
		result = append(result, EventRepresentation{Position: available.Position, Entity: observation.Entity,
			Attribute: observation.Attribute, Source: observation.Source, Provenance: observation.Provenance,
			EvidenceRef: observation.EvidenceRef, ObservedAt: observation.ObservedAt, ReceivedAt: available.ReceivedAt,
			Representation: fixtureRepresentation(c.metadata, []float32{0.25, 0.75}, []bool{true, true})})
	}
	return result, nil
}

type topoDouble struct {
	testLifecycle
	err       error
	calls     int
	lastInput TopoTemporalInput
}

func (c *topoDouble) EncodeContext(_ context.Context, input TopoTemporalInput) (Representation, error) {
	c.calls++
	c.lastInput = input
	if c.err != nil {
		return Representation{}, c.err
	}
	return fixtureRepresentation(c.metadata, []float32{0.5}, []bool{true}), nil
}

type homeDouble struct {
	testLifecycle
	err    error
	calls  int
	input  HomeAdaptationInput
	mutate bool
}

func (c *homeDouble) EncodeHomeContext(_ context.Context, input HomeAdaptationInput) (Representation, error) {
	c.calls++
	c.input = input
	if c.err != nil {
		return Representation{}, c.err
	}
	if c.mutate {
		input.General.Contributions[0].Events[0].Representation.Values[0] = 77
	}
	return fixtureRepresentation(c.metadata, []float32{0.125, 0.875}, []bool{true, true}), nil
}

type headDouble struct{ testLifecycle }

func (*headDouble) Evaluate(context.Context, CognitiveHeadInput) (HeadOutput, error) {
	return HeadOutput{}, errors.New("test head must not be invoked by representation pipeline")
}

func fixtureMetadata(id string, role ComponentRole) ComponentMetadata {
	input, output, space := "", RepresentationSchemaVersion, "fixture-space"
	switch role {
	case RoleEvent:
		input = EventInputSchemaVersion
	case RoleTopoTemporal:
		input = TopoInputSchemaVersion
	case RoleHomeAdaptation:
		input = HomeInputSchemaVersion
	case RoleCognitiveHead:
		input, output, space = HeadInputSchemaVersion, HeadOutputSchemaVersion, ""
	}
	return ComponentMetadata{ID: id, Role: role, Head: HeadSituationAssessmentIf(role), ContractVersion: ArchitectureVersion,
		ComponentVersion: "1.0.0", InputSchemaVersion: input, OutputSchemaVersion: output,
		RepresentationSpace: space, ExecutionClass: ExecutionTestFixture}
}

func HeadSituationAssessmentIf(role ComponentRole) HeadKind {
	if role == RoleCognitiveHead {
		return HeadSituationAssessment
	}
	return ""
}

func fixtureRepresentation(metadata ComponentMetadata, values []float32, mask []bool) Representation {
	return Representation{SchemaVersion: RepresentationSchemaVersion, Component: referenceFromMetadata(metadata),
		Space: metadata.RepresentationSpace, Dimension: len(values), Values: values, ValidityMask: mask}
}

func fixtureEvent() EventInput {
	cutoff := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	futureObserved := cutoff.Add(time.Minute)
	return EventInput{SchemaVersion: EventInputSchemaVersion, StateEncoderSchemaVersion: stateencoder.EncoderSchemaVersion, Cutoff: cutoff, Availability: ObservationsComplete,
		Observations: []AvailableObservation{{Position: 0, ReceivedAt: cutoff.Add(-time.Second), Observation: stateencoder.EncodedObservation{
			Entity:    stateencoder.EntityRef{ID: "sensor:opaque-1", Kind: "sensor"},
			Attribute: stateencoder.AttributeRef{Namespace: "occupancy", Name: "present", ValueType: stateencoder.ValueBoolean},
			Status:    stateencoder.StatusObserved, Source: &stateencoder.SourceRef{ID: "source:opaque-1", Kind: "presence"},
			Provenance: "record:opaque-1", EvidenceRef: "evidence:opaque-1", ObservedAt: &futureObserved,
		}}}}
}

func fixtureInput() PipelineInput {
	event := fixtureEvent()
	roomA := stateencoder.EntityRef{ID: "room:a", Kind: "room"}
	roomB := stateencoder.EntityRef{ID: "room:b", Kind: "room"}
	source := &stateencoder.SourceRef{ID: "source:floorplan", Kind: "floorplan"}
	return PipelineInput{EventInput: event, Entities: []AvailableEntity{{Entity: roomA, ReceivedAt: event.Cutoff.Add(-time.Hour)}, {Entity: roomB, ReceivedAt: event.Cutoff.Add(-time.Hour)}},
		Relations: []AvailableRelation{
			{Relation: stateencoder.EncodedRelation{Subject: roomA.ID, Kind: "adjacent-to", Object: roomB.ID, Source: source}, ReceivedAt: event.Cutoff.Add(-time.Hour)},
			{Relation: stateencoder.EncodedRelation{Subject: roomB.ID, Kind: "adjacent-to", Object: roomA.ID, Source: source}, ReceivedAt: event.Cutoff.Add(-time.Hour)},
		},
		Locations:     []EventLocationAssociation{{Position: 0, Status: "unknown"}},
		TopologyState: TopologyPartial, HistoryState: HistoryInsufficient, Home: HomeAdaptationContext{State: HomeColdStart}}
}

func fixtureBranches() (*eventDouble, *topoDouble, *homeDouble) {
	event := &eventDouble{testLifecycle: testLifecycle{metadata: fixtureMetadata("event-fixture", RoleEvent)}}
	topo := &topoDouble{testLifecycle: testLifecycle{metadata: fixtureMetadata("topo-fixture", RoleTopoTemporal)}}
	home := &homeDouble{testLifecycle: testLifecycle{metadata: fixtureMetadata("home-fixture", RoleHomeAdaptation)}}
	return event, topo, home
}

func register(t *testing.T, registry *Registry, component ManagedComponent) {
	t.Helper()
	if err := registry.Register(context.Background(), component); err != nil {
		t.Fatal(err)
	}
}

func TestPipelineProducesDeterministicStructuralFusionAndNoPrediction(t *testing.T) {
	registry := NewRegistry()
	event, topo, _ := fixtureBranches()
	register(t, registry, event)
	register(t, registry, topo)
	headMeta := fixtureMetadataForHead("situation-head", HeadSituationAssessment)
	register(t, registry, &headDouble{testLifecycle: testLifecycle{metadata: headMeta}})
	pipeline := Pipeline{Registry: registry, Config: PipelineConfig{AllowTestFixtures: true}}
	input := fixtureInput()
	input.Entities[0], input.Entities[1] = input.Entities[1], input.Entities[0]
	input.Relations[0], input.Relations[1] = input.Relations[1], input.Relations[0]
	first, err := pipeline.Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := pipeline.Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("fusion was not deterministic:\n%s\n%s", firstJSON, secondJSON)
	}
	if first.GeneralFusion.Stage != FusionGeneral || first.CognitiveFusion.Stage != FusionCognitive {
		t.Fatalf("unexpected fusion stages: %+v", first)
	}
	if len(first.GeneralFusion.Contributions) != 2 || first.GeneralFusion.Contributions[0].Role != RoleEvent || first.GeneralFusion.Contributions[1].Role != RoleTopoTemporal {
		t.Fatalf("general contributions are not canonically ordered: %+v", first.GeneralFusion.Contributions)
	}
	if first.GeneralFusion.Contributions[0].Events[0].Representation.Values[0] != 0.25 || topo.lastInput.TopologyState != TopologyPartial || topo.lastInput.HistoryState != HistoryInsufficient {
		t.Fatalf("branch representations or partial context were not preserved")
	}
	if topo.lastInput.Entities[0].Entity.ID != "room:a" || topo.lastInput.Relations[0].Relation.Subject != "room:a" || topo.lastInput.Relations[0].Relation.Object != "room:b" || topo.lastInput.Events[0].ObservedAt == nil || !topo.lastInput.Events[0].ObservedAt.After(input.EventInput.Cutoff) {
		t.Fatalf("directed relation or future observed_at was altered: %+v", topo.lastInput)
	}
	if first.CognitiveFusion.Contributions[0].State != ContributionUnavailable || first.CognitiveFusion.Contributions[0].Reason != "home_adaptation_cold-start" {
		t.Fatalf("cold-start did not explicitly omit home context: %+v", first.CognitiveFusion.Contributions[0])
	}
	serialized := string(firstJSON)
	for _, forbidden := range []string{"prediction", "danger_score", "action_authorized", "physical_action"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("representation pipeline emitted cognitive/action field %q", forbidden)
		}
	}
	if event.calls != 2 || topo.calls != 2 {
		t.Fatalf("branches were not invoked once per event: event=%d topo=%d", event.calls, topo.calls)
	}
}

func TestHomeBranchIsOptionalColdStartAndRuntimeFailureIsIsolated(t *testing.T) {
	registry := NewRegistry()
	event, topo, home := fixtureBranches()
	register(t, registry, event)
	register(t, registry, topo)
	register(t, registry, home)
	pipeline := Pipeline{Registry: registry, Config: PipelineConfig{AllowTestFixtures: true}}
	input := fixtureInput()
	if _, err := pipeline.Run(context.Background(), input); err != nil {
		t.Fatalf("cold start should not require home inference: %v", err)
	}
	if home.calls != 0 {
		t.Fatal("home branch ran during cold start")
	}
	input.Home = HomeAdaptationContext{State: HomeAvailable, AdaptationRef: "home-state-ref", ModelVersion: "7.0.0", MemoryVersion: "explicit-memory-v2", TrainingCutoff: input.EventInput.Cutoff.Add(-24 * time.Hour)}
	result, err := pipeline.Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if home.calls != 1 || result.CognitiveFusion.Contributions[0].State != ContributionAvailable {
		t.Fatalf("available home branch was not included: calls=%d contribution=%+v", home.calls, result.CognitiveFusion.Contributions[0])
	}
	if home.input.Cutoff != input.EventInput.Cutoff || home.input.Home.TrainingCutoff.After(home.input.Cutoff) {
		t.Fatal("home branch did not receive its declared historical cutoff")
	}
	home.mutate = true
	result, err = pipeline.Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.GeneralFusion.Contributions[0].Events[0].Representation.Values[0] != 0.25 {
		t.Fatal("home branch mutation altered the general fusion result")
	}
	home.mutate = false

	home.err = errors.New("synthetic fixture failure")
	result, err = pipeline.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("optional home failure blocked general cognition: %v", err)
	}
	if result.GeneralFusion.Stage != FusionGeneral || result.CognitiveFusion.Contributions[0].State != ContributionFailed {
		t.Fatalf("home failure was not isolated: %+v", result)
	}
	for _, record := range registry.Records() {
		if record.Metadata.ID == home.metadata.ID && record.State != LifecycleFailed {
			t.Fatalf("home lifecycle did not record branch failure: %+v", record)
		}
	}
}

func TestBranchCannotMutateCallerObservationInput(t *testing.T) {
	registry := NewRegistry()
	event, topo, _ := fixtureBranches()
	event.mutate = true
	register(t, registry, event)
	register(t, registry, topo)
	input := fixtureInput()
	_, err := (Pipeline{Registry: registry, Config: PipelineConfig{AllowTestFixtures: true}}).Run(context.Background(), input)
	if !errors.Is(err, ErrComponentFailed) {
		t.Fatalf("mutated branch output should violate identity contract: %v", err)
	}
	if input.EventInput.Observations[0].Observation.Entity.ID != "sensor:opaque-1" {
		t.Fatal("event branch mutated caller-owned input")
	}
}

func TestRequiredBranchFailureIsExplicitAndStopsPipeline(t *testing.T) {
	registry := NewRegistry()
	event, topo, _ := fixtureBranches()
	event.err = errors.New("event adapter failure")
	register(t, registry, event)
	register(t, registry, topo)
	_, err := (Pipeline{Registry: registry, Config: PipelineConfig{AllowTestFixtures: true}}).Run(context.Background(), fixtureInput())
	if !errors.Is(err, ErrComponentFailed) {
		t.Fatalf("required branch failure was not returned: %v", err)
	}
	if topo.calls != 0 {
		t.Fatal("topo-temporal branch ran after required event branch failed")
	}
}

func TestMissingObservationsAndPartialTopologyRemainExplicit(t *testing.T) {
	registry := NewRegistry()
	event, topo, _ := fixtureBranches()
	register(t, registry, event)
	register(t, registry, topo)
	input := fixtureInput()
	input.EventInput.Availability = ObservationsMissing
	input.EventInput.Observations = nil
	input.TopologyState = TopologyUnavailable
	input.Entities, input.Relations, input.Locations = nil, nil, nil
	result, err := (Pipeline{Registry: registry, Config: PipelineConfig{AllowTestFixtures: true}}).Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.GeneralFusion.Contributions[0].State != ContributionNoData || len(result.GeneralFusion.Contributions[0].Events) != 0 {
		t.Fatalf("missing observations were not explicit: %+v", result.GeneralFusion.Contributions[0])
	}
	if topo.lastInput.TopologyState != TopologyUnavailable || len(topo.lastInput.Relations) != 0 {
		t.Fatalf("unavailable topology was changed or hidden: %+v", topo.lastInput)
	}
}

func TestReceiptTimeCutoffAndFutureObservedTime(t *testing.T) {
	input := fixtureInput()
	if err := input.Validate(); err != nil {
		t.Fatalf("received future observed_at should remain valid: %v", err)
	}
	input.EventInput.Observations[0].ReceivedAt = input.EventInput.Cutoff.Add(time.Nanosecond)
	if err := input.Validate(); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("post-cutoff receipt time was accepted: %v", err)
	}
	input = fixtureInput()
	input.Relations[0].ReceivedAt = input.EventInput.Cutoff.Add(time.Nanosecond)
	if err := input.Validate(); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("post-cutoff topology relation was accepted: %v", err)
	}
	input = fixtureInput()
	input.Entities[0].ReceivedAt = input.EventInput.Cutoff.Add(time.Nanosecond)
	if err := input.Validate(); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("post-cutoff topology entity was accepted: %v", err)
	}
	input = fixtureInput()
	input.Home = HomeAdaptationContext{State: HomeAvailable, AdaptationRef: "home-state-ref", ModelVersion: "1.0.0", TrainingCutoff: input.EventInput.Cutoff.Add(time.Second)}
	if err := input.Validate(); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("future-trained home model was accepted: %v", err)
	}
}

func TestCompleteTopologyRequiresAllDirectedEndpoints(t *testing.T) {
	input := fixtureInput()
	input.TopologyState = TopologyComplete
	if err := input.Validate(); err != nil {
		t.Fatalf("complete topology with all endpoints should validate: %v", err)
	}
	input.Entities = input.Entities[:1]
	if err := input.Validate(); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("complete topology with a missing endpoint was accepted: %v", err)
	}
}

func TestRegisteredComponentMetadataIsImmutable(t *testing.T) {
	registry := NewRegistry()
	event, topo, _ := fixtureBranches()
	register(t, registry, event)
	register(t, registry, topo)
	event.metadata.RepresentationSpace = "changed-after-registration"
	_, err := (Pipeline{Registry: registry, Config: PipelineConfig{AllowTestFixtures: true}}).Run(context.Background(), fixtureInput())
	if !errors.Is(err, ErrComponentFailed) {
		t.Fatalf("mutated component metadata was not rejected: %v", err)
	}
}

func TestUnavailableRegistrationIsRecordedAndDoesNotMasqueradeAsReady(t *testing.T) {
	registry := NewRegistry()
	event := &eventDouble{testLifecycle: testLifecycle{metadata: fixtureMetadata("event-fixture", RoleEvent), initializeErr: ErrComponentUnavailable}}
	if err := registry.Register(context.Background(), event); !errors.Is(err, ErrComponentUnavailable) {
		t.Fatalf("unavailable initialization was not returned: %v", err)
	}
	records := registry.Records()
	if len(records) != 1 || records[0].State != LifecycleUnavailable || records[0].Error == "" {
		t.Fatalf("unavailable lifecycle state was not recorded: %+v", records)
	}
}

func TestRegistryLifecycleRegistrationReplacementAndShutdown(t *testing.T) {
	registry := NewRegistry()
	old := &eventDouble{testLifecycle: testLifecycle{metadata: fixtureMetadata("event-fixture", RoleEvent)}}
	register(t, registry, old)
	replacementMeta := fixtureMetadata("event-fixture", RoleEvent)
	replacementMeta.ComponentVersion = "1.1.0"
	replacement := &eventDouble{testLifecycle: testLifecycle{metadata: replacementMeta}}
	if err := registry.Replace(context.Background(), old.metadata.ID, replacement); err != nil {
		t.Fatal(err)
	}
	if !replacement.initialized || !old.shutdown {
		t.Fatalf("replacement lifecycle incomplete: old=%+v replacement=%+v", old.testLifecycle, replacement.testLifecycle)
	}
	records := registry.Records()
	if len(records) != 1 || records[0].State != LifecycleReady || records[0].Metadata.ComponentVersion != "1.1.0" {
		t.Fatalf("replacement metadata/state not published: %+v", records)
	}
	if err := registry.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if registry.Records()[0].State != LifecycleStopped || !replacement.shutdown {
		t.Fatal("registry shutdown did not stop the active replacement")
	}
}

func TestFailedReplacementPreservesReadyComponent(t *testing.T) {
	registry := NewRegistry()
	old := &eventDouble{testLifecycle: testLifecycle{metadata: fixtureMetadata("event-fixture", RoleEvent)}}
	register(t, registry, old)
	meta := fixtureMetadata("event-fixture", RoleEvent)
	meta.ComponentVersion = "1.1.0"
	replacement := &eventDouble{testLifecycle: testLifecycle{metadata: meta, initializeErr: errors.New("initialization failed")}}
	if err := registry.Replace(context.Background(), old.metadata.ID, replacement); err == nil {
		t.Fatal("failed replacement unexpectedly succeeded")
	}
	if records := registry.Records(); len(records) != 1 || records[0].State != LifecycleReady || records[0].Metadata.ComponentVersion != "1.0.0" {
		t.Fatalf("failed replacement displaced old ready component: %+v", records)
	}
	if old.shutdown {
		t.Fatal("failed replacement shut down the active component")
	}
}

func TestComponentCompatibilityAndFixtureExecutionGates(t *testing.T) {
	registry := NewRegistry()
	event, topo, _ := fixtureBranches()
	event.metadata.OutputSchemaVersion = "unknown/v9"
	if err := registry.Register(context.Background(), event); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("incompatible branch schema registered: %v", err)
	}
	event.metadata = fixtureMetadata("event-fixture", RoleEvent)
	register(t, registry, event)
	register(t, registry, topo)
	if _, err := (Pipeline{Registry: registry}).Run(context.Background(), fixtureInput()); !errors.Is(err, ErrComponentUnavailable) {
		t.Fatalf("test fixture ran without explicit test permission: %v", err)
	}
	if _, err := (Pipeline{Registry: registry, Config: PipelineConfig{AllowTestFixtures: true}}).Run(context.Background(), fixtureInput()); err != nil {
		t.Fatalf("explicit fixture harness could not run: %v", err)
	}
}

func TestRepresentationAndHeadOutputContracts(t *testing.T) {
	meta := fixtureMetadata("event-fixture", RoleEvent)
	valid := fixtureRepresentation(meta, []float32{0, 1}, []bool{false, true})
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := fixtureRepresentation(meta, []float32{1}, []bool{false})
	if err := invalid.Validate(); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("masked nonzero value accepted: %v", err)
	}
	invalid = fixtureRepresentation(meta, []float32{float32(math.Inf(1))}, []bool{true})
	if err := invalid.Validate(); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("infinite representation accepted: %v", err)
	}

	headMeta := fixtureMetadata("situation-head", RoleCognitiveHead)
	output := HeadOutput{SchemaVersion: HeadOutputSchemaVersion, DecisionCutoff: time.Unix(600, 0).UTC(), Head: HeadSituationAssessment,
		Component: referenceFromMetadata(headMeta), Outcome: OutcomeNoAction}
	if err := output.Validate(); err != nil {
		t.Fatalf("explicit no-action output rejected: %v", err)
	}
	output = HeadOutput{SchemaVersion: HeadOutputSchemaVersion, DecisionCutoff: time.Unix(600, 0).UTC(), Head: HeadEventPrediction,
		Component: referenceFromMetadata(fixtureMetadataForHead("event-head", HeadEventPrediction)), Outcome: OutcomeNoEvent}
	if err := output.Validate(); err != nil {
		t.Fatalf("no-event output rejected: %v", err)
	}
	badRisk := float32(1.1)
	output = HeadOutput{SchemaVersion: HeadOutputSchemaVersion, DecisionCutoff: time.Unix(600, 0).UTC(), Head: HeadSituationAssessment,
		Component: referenceFromMetadata(headMeta), Outcome: OutcomeAssessment,
		Situation: &SituationAssessment{Summary: "fixture assessment", RiskEstimate: &badRisk}}
	if err := output.Validate(); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("out-of-range estimate accepted: %v", err)
	}
}

func fixtureMetadataForHead(id string, head HeadKind) ComponentMetadata {
	meta := fixtureMetadata(id, RoleCognitiveHead)
	meta.Head = head
	return meta
}

func TestFusionKeepsBranchVectorsSeparateAndOrdered(t *testing.T) {
	eventMeta := fixtureMetadata("event-fixture", RoleEvent)
	topoMeta := fixtureMetadata("topo-fixture", RoleTopoTemporal)
	event := BranchContribution{Role: RoleEvent, State: ContributionAvailable, Component: componentRefPointer(eventMeta),
		Events: []EventRepresentation{{Position: 0, Entity: stateencoder.EntityRef{ID: "sensor:a", Kind: "sensor"},
			Attribute: stateencoder.AttributeRef{Name: "motion"}, ReceivedAt: time.Unix(500, 0).UTC(),
			Representation: fixtureRepresentation(eventMeta, []float32{1}, []bool{true})}}}
	topo := BranchContribution{Role: RoleTopoTemporal, State: ContributionPartial,
		Component: componentRefPointer(topoMeta), Representations: []Representation{fixtureRepresentation(topoMeta, []float32{2, 3}, []bool{true, true})}}
	fusion, err := NewGeneralFusion(event, topo)
	if err != nil {
		t.Fatal(err)
	}
	if fusion.Contributions[0].Events[0].Representation.Dimension != 1 || fusion.Contributions[1].Representations[0].Dimension != 2 {
		t.Fatal("fusion concatenated or changed independent branch dimensions")
	}
	encoded, _ := json.Marshal(fusion)
	var decoded FusionEnvelope
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil || !reflect.DeepEqual(fusion, decoded) {
		t.Fatalf("fusion round-trip changed contract: err=%v", err)
	}
}
