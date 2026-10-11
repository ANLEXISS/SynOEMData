// Package hybrid defines model-independent contracts and orchestration for the
// experimental Triple-MLP cognitive architecture. It performs no inference.
package hybrid

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	"synora/internal/intelligence/stateencoder"
)

const (
	ArchitectureVersion         = "1.0.0"
	EventInputSchemaVersion     = "synora.hybrid.event-input/v1"
	TopoInputSchemaVersion      = "synora.hybrid.topo-temporal-input/v1"
	HomeInputSchemaVersion      = "synora.hybrid.home-input/v1"
	HeadInputSchemaVersion      = "synora.hybrid.head-input/v1"
	RepresentationSchemaVersion = "synora.cognitive-representation/v1"
	FusionSchemaVersion         = "synora.cognitive-fusion/v1"
	HeadOutputSchemaVersion     = "synora.cognitive-head-output/v1"
)

var (
	ErrComponentUnavailable = errors.New("cognitive component unavailable")
	ErrComponentFailed      = errors.New("cognitive component failed")
	ErrIncompatible         = errors.New("cognitive component or representation incompatible")
	ErrNotRegistered        = errors.New("cognitive component not registered")
	ErrExecutionDisabled    = errors.New("component execution class is disabled")
)

var semanticVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type ComponentRole string

const (
	RoleEvent          ComponentRole = "event"
	RoleTopoTemporal   ComponentRole = "topo-temporal"
	RoleHomeAdaptation ComponentRole = "home-adaptation"
	RoleCognitiveHead  ComponentRole = "cognitive-head"
)

type ExecutionClass string

const (
	ExecutionTestFixture ExecutionClass = "test-fixture"
	ExecutionUntrained   ExecutionClass = "untrained-adapter"
	ExecutionTrained     ExecutionClass = "trained-model"
)

type HeadKind string

const (
	HeadSituationAssessment HeadKind = "situation-assessment"
	HeadEventPrediction     HeadKind = "event-risk-prediction"
	HeadDomesticAssistance  HeadKind = "domestic-assistance"
)

// ComponentMetadata versions the adapter contract independently from any
// learned model weights. Representation dimensions are deliberately runtime
// values, not architecture constants.
type ComponentMetadata struct {
	ID                  string         `json:"id"`
	Role                ComponentRole  `json:"role"`
	Head                HeadKind       `json:"head,omitempty"`
	ContractVersion     string         `json:"contract_version"`
	ComponentVersion    string         `json:"component_version"`
	InputSchemaVersion  string         `json:"input_schema_version"`
	OutputSchemaVersion string         `json:"output_schema_version"`
	RepresentationSpace string         `json:"representation_space,omitempty"`
	ExecutionClass      ExecutionClass `json:"execution_class"`
	ModelVersion        string         `json:"model_version,omitempty"`
}

func (m ComponentMetadata) Validate() error {
	if m.ID == "" || m.ContractVersion != ArchitectureVersion || !semanticVersionPattern.MatchString(m.ComponentVersion) {
		return fmt.Errorf("%w: missing identity or unsupported component contract/version", ErrIncompatible)
	}
	if m.ExecutionClass != ExecutionTestFixture && m.ExecutionClass != ExecutionUntrained && m.ExecutionClass != ExecutionTrained {
		return fmt.Errorf("%w: unknown execution class %q", ErrIncompatible, m.ExecutionClass)
	}
	if m.ExecutionClass == ExecutionTrained && m.ModelVersion == "" {
		return fmt.Errorf("%w: trained component %q has no model version", ErrIncompatible, m.ID)
	}
	if m.ExecutionClass == ExecutionTrained && !semanticVersionPattern.MatchString(m.ModelVersion) {
		return fmt.Errorf("%w: trained component %q has invalid model version", ErrIncompatible, m.ID)
	}
	if m.ExecutionClass != ExecutionTrained && m.ModelVersion != "" {
		return fmt.Errorf("%w: non-trained component %q declares a model version", ErrIncompatible, m.ID)
	}
	expectedInput, expectedOutput := "", RepresentationSchemaVersion
	switch m.Role {
	case RoleEvent:
		expectedInput = EventInputSchemaVersion
	case RoleTopoTemporal:
		expectedInput = TopoInputSchemaVersion
	case RoleHomeAdaptation:
		expectedInput = HomeInputSchemaVersion
	case RoleCognitiveHead:
		expectedInput, expectedOutput = HeadInputSchemaVersion, HeadOutputSchemaVersion
		if !validHead(m.Head) {
			return fmt.Errorf("%w: cognitive head %q has invalid head kind", ErrIncompatible, m.ID)
		}
	default:
		return fmt.Errorf("%w: unknown component role %q", ErrIncompatible, m.Role)
	}
	if m.Role != RoleCognitiveHead && m.Head != "" {
		return fmt.Errorf("%w: non-head component %q declares a head kind", ErrIncompatible, m.ID)
	}
	if m.InputSchemaVersion != expectedInput || m.OutputSchemaVersion != expectedOutput {
		return fmt.Errorf("%w: schema versions do not match role %q", ErrIncompatible, m.Role)
	}
	if m.Role != RoleCognitiveHead && m.RepresentationSpace == "" {
		return fmt.Errorf("%w: component %q has no representation-space identifier", ErrIncompatible, m.ID)
	}
	return nil
}

func validHead(head HeadKind) bool {
	switch head {
	case HeadSituationAssessment, HeadEventPrediction, HeadDomesticAssistance:
		return true
	default:
		return false
	}
}

type LifecycleState string

const (
	LifecycleInitializing LifecycleState = "initializing"
	LifecycleReady        LifecycleState = "ready"
	LifecycleUnavailable  LifecycleState = "unavailable"
	LifecycleFailed       LifecycleState = "failed"
	LifecycleStopped      LifecycleState = "stopped"
)

type ManagedComponent interface {
	Metadata() ComponentMetadata
	Initialize(context.Context) error
	Shutdown(context.Context) error
}

type EventBranch interface {
	ManagedComponent
	EncodeEvents(context.Context, EventInput) ([]EventRepresentation, error)
}

type TopoTemporalBranch interface {
	ManagedComponent
	EncodeContext(context.Context, TopoTemporalInput) (Representation, error)
}

type HomeAdaptationBranch interface {
	ManagedComponent
	EncodeHomeContext(context.Context, HomeAdaptationInput) (Representation, error)
}

type CognitiveHead interface {
	ManagedComponent
	Evaluate(context.Context, CognitiveHeadInput) (HeadOutput, error)
}

type ObservationAvailability string

const (
	ObservationsComplete ObservationAvailability = "complete"
	ObservationsPartial  ObservationAvailability = "partial"
	ObservationsMissing  ObservationAvailability = "missing"
)

// AvailableObservation pairs an I2 encoded record with its receipt time. The
// event adapter enforces receipt-time eligibility; observed_at may be ahead of
// the cutoff and remains untouched.
type AvailableObservation struct {
	Position    int                             `json:"position"`
	Observation stateencoder.EncodedObservation `json:"observation"`
	ReceivedAt  time.Time                       `json:"received_at"`
}

type EventInput struct {
	SchemaVersion             string                  `json:"schema_version"`
	StateEncoderSchemaVersion string                  `json:"state_encoder_schema_version"`
	Cutoff                    time.Time               `json:"decision_cutoff"`
	Availability              ObservationAvailability `json:"availability"`
	Observations              []AvailableObservation  `json:"observations"`
}

func (i EventInput) Validate() error {
	if i.SchemaVersion != EventInputSchemaVersion || i.StateEncoderSchemaVersion != stateencoder.EncoderSchemaVersion || i.Cutoff.IsZero() {
		return fmt.Errorf("%w: invalid event input schema or cutoff", ErrIncompatible)
	}
	switch i.Availability {
	case ObservationsComplete, ObservationsPartial:
		if len(i.Observations) == 0 {
			return fmt.Errorf("%w: %s observations require records", ErrIncompatible, i.Availability)
		}
	case ObservationsMissing:
		if len(i.Observations) != 0 {
			return fmt.Errorf("%w: missing observations must be empty", ErrIncompatible)
		}
	default:
		return fmt.Errorf("%w: unknown observation availability %q", ErrIncompatible, i.Availability)
	}
	lastPosition := -1
	for _, item := range i.Observations {
		if item.Position < 0 || item.Position <= lastPosition || item.ReceivedAt.IsZero() || item.ReceivedAt.After(i.Cutoff) {
			return fmt.Errorf("%w: invalid order or post-cutoff observation at position %d", ErrIncompatible, item.Position)
		}
		if item.Observation.Entity.ID == "" || item.Observation.Attribute.Name == "" || item.Observation.Status == "" {
			return fmt.Errorf("%w: incomplete encoded observation at position %d", ErrIncompatible, item.Position)
		}
		lastPosition = item.Position
	}
	return nil
}

type TopologyAvailability string

const (
	TopologyComplete    TopologyAvailability = "complete"
	TopologyPartial     TopologyAvailability = "partial"
	TopologyUnknown     TopologyAvailability = "unknown"
	TopologyUnavailable TopologyAvailability = "unavailable"
)

type HistoryAvailability string

const (
	HistoryAvailable    HistoryAvailability = "available"
	HistoryPartial      HistoryAvailability = "partial"
	HistoryInsufficient HistoryAvailability = "insufficient"
	HistoryUnavailable  HistoryAvailability = "unavailable"
)

type EventLocationAssociation struct {
	Position   int                     `json:"position"`
	Entity     *stateencoder.EntityRef `json:"entity,omitempty"`
	Status     string                  `json:"status"` // known, unknown, or unavailable; no room semantics are inferred.
	ReceivedAt time.Time               `json:"received_at,omitempty"`
}

type AvailableEntity struct {
	Entity     stateencoder.EntityRef `json:"entity"`
	ReceivedAt time.Time              `json:"received_at"`
}

type AvailableRelation struct {
	Relation   stateencoder.EncodedRelation `json:"relation"`
	ReceivedAt time.Time                    `json:"received_at"`
}

type TopoTemporalInput struct {
	SchemaVersion             string                     `json:"schema_version"`
	StateEncoderSchemaVersion string                     `json:"state_encoder_schema_version"`
	Cutoff                    time.Time                  `json:"decision_cutoff"`
	Events                    []EventRepresentation      `json:"ordered_events"`
	Entities                  []AvailableEntity          `json:"entities,omitempty"`
	Relations                 []AvailableRelation        `json:"directed_relations,omitempty"`
	Locations                 []EventLocationAssociation `json:"event_locations,omitempty"`
	TopologyState             TopologyAvailability       `json:"topology_availability"`
	HistoryState              HistoryAvailability        `json:"history_availability"`
	HistoryStart              *time.Time                 `json:"history_start,omitempty"`
	HistoryEnd                *time.Time                 `json:"history_end,omitempty"`
}

func (i TopoTemporalInput) Validate() error {
	if i.SchemaVersion != TopoInputSchemaVersion || i.StateEncoderSchemaVersion != stateencoder.EncoderSchemaVersion || i.Cutoff.IsZero() {
		return fmt.Errorf("%w: invalid topo-temporal input schema or cutoff", ErrIncompatible)
	}
	switch i.TopologyState {
	case TopologyComplete, TopologyPartial, TopologyUnknown, TopologyUnavailable:
	default:
		return fmt.Errorf("%w: invalid topology availability %q", ErrIncompatible, i.TopologyState)
	}
	switch i.HistoryState {
	case HistoryAvailable, HistoryPartial, HistoryInsufficient, HistoryUnavailable:
	default:
		return fmt.Errorf("%w: invalid history availability %q", ErrIncompatible, i.HistoryState)
	}
	if (i.HistoryStart == nil) != (i.HistoryEnd == nil) || (i.HistoryStart != nil && i.HistoryEnd.Before(*i.HistoryStart)) {
		return fmt.Errorf("%w: invalid history interval", ErrIncompatible)
	}
	last := -1
	for _, event := range i.Events {
		if err := event.Validate(); err != nil {
			return err
		}
		if event.Position <= last || event.ReceivedAt.After(i.Cutoff) {
			return fmt.Errorf("%w: events must be ordered and received by cutoff", ErrIncompatible)
		}
		last = event.Position
	}
	seenLocations := map[int]bool{}
	for _, location := range i.Locations {
		if location.Position < 0 || seenLocations[location.Position] {
			return fmt.Errorf("%w: invalid or duplicate event-location position", ErrIncompatible)
		}
		seenLocations[location.Position] = true
		switch location.Status {
		case "known":
			if location.Entity == nil || location.Entity.ID == "" || location.ReceivedAt.IsZero() || location.ReceivedAt.After(i.Cutoff) {
				return fmt.Errorf("%w: known event location has no entity", ErrIncompatible)
			}
		case "unknown", "unavailable":
			if location.Entity != nil || (location.ReceivedAt.After(i.Cutoff)) {
				return fmt.Errorf("%w: unknown event location carries an entity", ErrIncompatible)
			}
		default:
			return fmt.Errorf("%w: invalid event-location status %q", ErrIncompatible, location.Status)
		}
	}
	entityIDs := map[string]bool{}
	for _, item := range i.Entities {
		if item.Entity.ID == "" || item.Entity.Kind == "" || item.ReceivedAt.IsZero() || item.ReceivedAt.After(i.Cutoff) || entityIDs[item.Entity.ID] {
			return fmt.Errorf("%w: invalid, duplicate, or post-cutoff topology entity", ErrIncompatible)
		}
		entityIDs[item.Entity.ID] = true
	}
	for _, item := range i.Relations {
		relation := item.Relation
		if relation.Subject == "" || relation.Kind == "" || relation.Object == "" || item.ReceivedAt.IsZero() || item.ReceivedAt.After(i.Cutoff) {
			return fmt.Errorf("%w: invalid or post-cutoff topology relation", ErrIncompatible)
		}
		if i.TopologyState == TopologyComplete && (!entityIDs[relation.Subject] || !entityIDs[relation.Object]) {
			return fmt.Errorf("%w: complete topology relation has an unknown endpoint", ErrIncompatible)
		}
	}
	if i.TopologyState == TopologyUnavailable && (len(i.Entities) > 0 || len(i.Relations) > 0) {
		return fmt.Errorf("%w: unavailable topology cannot carry graph data", ErrIncompatible)
	}
	if i.TopologyState == TopologyComplete && len(i.Entities) == 0 {
		return fmt.Errorf("%w: complete topology has no entities", ErrIncompatible)
	}
	if i.TopologyState == TopologyComplete {
		for _, location := range i.Locations {
			if location.Status == "known" && (location.Entity == nil || !entityIDs[location.Entity.ID]) {
				return fmt.Errorf("%w: complete topology omits a known event location", ErrIncompatible)
			}
		}
	}
	return nil
}

type HomeState string

const (
	HomeColdStart   HomeState = "cold-start"
	HomeAvailable   HomeState = "available"
	HomeUnavailable HomeState = "unavailable"
	HomeReset       HomeState = "reset"
)

type HomeAdaptationContext struct {
	State          HomeState `json:"state"`
	AdaptationRef  string    `json:"adaptation_ref,omitempty"`
	ModelVersion   string    `json:"model_version,omitempty"`
	MemoryVersion  string    `json:"explicit_memory_version,omitempty"`
	TrainingCutoff time.Time `json:"training_cutoff,omitempty"`
}

func (h HomeAdaptationContext) Validate() error {
	switch h.State {
	case HomeColdStart, HomeAvailable, HomeUnavailable, HomeReset:
	default:
		return fmt.Errorf("%w: invalid home adaptation state %q", ErrIncompatible, h.State)
	}
	if h.State == HomeAvailable && h.AdaptationRef == "" {
		return fmt.Errorf("%w: available home adaptation needs an opaque adaptation reference", ErrIncompatible)
	}
	if h.State == HomeAvailable && h.ModelVersion == "" {
		return fmt.Errorf("%w: available home adaptation needs a model version", ErrIncompatible)
	}
	if h.State == HomeAvailable && !semanticVersionPattern.MatchString(h.ModelVersion) {
		return fmt.Errorf("%w: available home adaptation has invalid model version", ErrIncompatible)
	}
	if h.State == HomeAvailable && h.TrainingCutoff.IsZero() {
		return fmt.Errorf("%w: available home adaptation needs a training cutoff", ErrIncompatible)
	}
	if h.State != HomeAvailable && (h.AdaptationRef != "" || h.ModelVersion != "" || !h.TrainingCutoff.IsZero()) {
		return fmt.Errorf("%w: inactive home adaptation carries an active model reference", ErrIncompatible)
	}
	return nil
}

type HomeAdaptationInput struct {
	SchemaVersion string                `json:"schema_version"`
	Cutoff        time.Time             `json:"decision_cutoff"`
	General       FusionEnvelope        `json:"general_fusion"`
	Home          HomeAdaptationContext `json:"home_context"`
}

type ComponentReference struct {
	ID                  string         `json:"id"`
	Role                ComponentRole  `json:"role"`
	Head                HeadKind       `json:"head,omitempty"`
	ContractVersion     string         `json:"contract_version"`
	ComponentVersion    string         `json:"component_version"`
	OutputSchemaVersion string         `json:"output_schema_version"`
	ExecutionClass      ExecutionClass `json:"execution_class"`
	ModelVersion        string         `json:"model_version,omitempty"`
}

func (r ComponentReference) Validate(expectedRole ComponentRole) error {
	if r.ID == "" || r.Role != expectedRole || r.ContractVersion != ArchitectureVersion || !semanticVersionPattern.MatchString(r.ComponentVersion) {
		return fmt.Errorf("%w: invalid component reference", ErrIncompatible)
	}
	if r.ExecutionClass != ExecutionTestFixture && r.ExecutionClass != ExecutionUntrained && r.ExecutionClass != ExecutionTrained {
		return fmt.Errorf("%w: invalid component execution class", ErrIncompatible)
	}
	if r.ExecutionClass == ExecutionTrained && !semanticVersionPattern.MatchString(r.ModelVersion) {
		return fmt.Errorf("%w: invalid referenced model version", ErrIncompatible)
	}
	expectedOutput := RepresentationSchemaVersion
	if expectedRole == RoleCognitiveHead {
		expectedOutput = HeadOutputSchemaVersion
		if !validHead(r.Head) {
			return fmt.Errorf("%w: invalid referenced head kind", ErrIncompatible)
		}
	} else if r.Head != "" {
		return fmt.Errorf("%w: non-head component reference declares head kind", ErrIncompatible)
	}
	if r.OutputSchemaVersion != expectedOutput {
		return fmt.Errorf("%w: referenced output schema mismatch", ErrIncompatible)
	}
	return nil
}

func referenceFromMetadata(m ComponentMetadata) ComponentReference {
	return ComponentReference{ID: m.ID, Role: m.Role, Head: m.Head, ContractVersion: m.ContractVersion,
		ComponentVersion: m.ComponentVersion, OutputSchemaVersion: m.OutputSchemaVersion,
		ExecutionClass: m.ExecutionClass, ModelVersion: m.ModelVersion}
}

type Representation struct {
	SchemaVersion string             `json:"schema_version"`
	Component     ComponentReference `json:"component"`
	Space         string             `json:"space"`
	Dimension     int                `json:"dimension"`
	Values        []float32          `json:"values"`
	ValidityMask  []bool             `json:"validity_mask"`
	EvidenceRefs  []string           `json:"evidence_refs,omitempty"`
}

func (r Representation) Validate() error {
	if r.SchemaVersion != RepresentationSchemaVersion || r.Component.ID == "" || r.Component.Role == RoleCognitiveHead || r.Space == "" {
		return fmt.Errorf("%w: invalid representation metadata", ErrIncompatible)
	}
	if err := r.Component.Validate(r.Component.Role); err != nil {
		return err
	}
	if r.Dimension <= 0 || r.Dimension != len(r.Values) || r.Dimension != len(r.ValidityMask) {
		return fmt.Errorf("%w: representation dimension/mask mismatch", ErrIncompatible)
	}
	for i, value := range r.Values {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) || (!r.ValidityMask[i] && value != 0) {
			return fmt.Errorf("%w: invalid value or masked nonzero at dimension %d", ErrIncompatible, i)
		}
	}
	for _, ref := range r.EvidenceRefs {
		if ref == "" {
			return fmt.Errorf("%w: empty evidence reference", ErrIncompatible)
		}
	}
	return nil
}

func validateRepresentationFor(r Representation, m ComponentMetadata) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Component != referenceFromMetadata(m) || r.Space != m.RepresentationSpace {
		return fmt.Errorf("%w: representation metadata does not match component %q", ErrIncompatible, m.ID)
	}
	return nil
}

type EventRepresentation struct {
	Position       int                       `json:"position"`
	Entity         stateencoder.EntityRef    `json:"entity"`
	Attribute      stateencoder.AttributeRef `json:"attribute"`
	Source         *stateencoder.SourceRef   `json:"source,omitempty"`
	Provenance     string                    `json:"provenance,omitempty"`
	EvidenceRef    string                    `json:"evidence_ref,omitempty"`
	ObservedAt     *time.Time                `json:"observed_at,omitempty"`
	ReceivedAt     time.Time                 `json:"received_at"`
	Representation Representation            `json:"representation"`
}

func (e EventRepresentation) Validate() error {
	if e.Position < 0 || e.Entity.ID == "" || e.Attribute.Name == "" || e.ReceivedAt.IsZero() {
		return fmt.Errorf("%w: incomplete event representation metadata", ErrIncompatible)
	}
	if err := e.Representation.Validate(); err != nil {
		return err
	}
	if e.Representation.Component.Role != RoleEvent {
		return fmt.Errorf("%w: event representation came from role %q", ErrIncompatible, e.Representation.Component.Role)
	}
	return nil
}

type ContributionState string

const (
	ContributionAvailable   ContributionState = "available"
	ContributionPartial     ContributionState = "partial"
	ContributionNoData      ContributionState = "no-data"
	ContributionUnavailable ContributionState = "unavailable"
	ContributionFailed      ContributionState = "failed"
)

type BranchContribution struct {
	Role            ComponentRole         `json:"role"`
	State           ContributionState     `json:"state"`
	Component       *ComponentReference   `json:"component,omitempty"`
	Events          []EventRepresentation `json:"events,omitempty"`
	Representations []Representation      `json:"representations,omitempty"`
	Reason          string                `json:"reason,omitempty"`
}

type FusionStage string

const (
	FusionGeneral   FusionStage = "general"
	FusionCognitive FusionStage = "cognitive"
)

// FusionEnvelope is structural, not a learned or concatenated fusion. Each
// branch representation and its validity mask remain independently addressable.
type FusionEnvelope struct {
	SchemaVersion string               `json:"schema_version"`
	Stage         FusionStage          `json:"stage"`
	General       *FusionEnvelope      `json:"general,omitempty"`
	Contributions []BranchContribution `json:"contributions"`
}

func (f FusionEnvelope) Validate() error {
	if f.SchemaVersion != FusionSchemaVersion {
		return fmt.Errorf("%w: unsupported fusion schema", ErrIncompatible)
	}
	switch f.Stage {
	case FusionGeneral:
		if f.General != nil || len(f.Contributions) != 2 || f.Contributions[0].Role != RoleEvent || f.Contributions[1].Role != RoleTopoTemporal {
			return fmt.Errorf("%w: general fusion must contain event then topo-temporal contribution", ErrIncompatible)
		}
		if f.Contributions[0].State == ContributionUnavailable || f.Contributions[0].State == ContributionFailed ||
			f.Contributions[1].State == ContributionUnavailable || f.Contributions[1].State == ContributionFailed || f.Contributions[1].State == ContributionNoData {
			return fmt.Errorf("%w: required general branch contribution is unavailable", ErrComponentUnavailable)
		}
	case FusionCognitive:
		if f.General == nil || len(f.Contributions) != 1 || f.Contributions[0].Role != RoleHomeAdaptation {
			return fmt.Errorf("%w: cognitive fusion must nest general fusion and one home contribution", ErrIncompatible)
		}
		if err := f.General.Validate(); err != nil || f.General.Stage != FusionGeneral {
			return fmt.Errorf("%w: invalid nested general fusion", ErrIncompatible)
		}
	default:
		return fmt.Errorf("%w: unknown fusion stage %q", ErrIncompatible, f.Stage)
	}
	for _, contribution := range f.Contributions {
		if err := contribution.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c BranchContribution) Validate() error {
	switch c.Role {
	case RoleEvent, RoleTopoTemporal, RoleHomeAdaptation:
	default:
		return fmt.Errorf("%w: invalid fusion contribution role %q", ErrIncompatible, c.Role)
	}
	switch c.State {
	case ContributionAvailable, ContributionPartial, ContributionNoData:
		if c.Component == nil {
			return fmt.Errorf("%w: active contribution has no component metadata", ErrIncompatible)
		}
	case ContributionUnavailable, ContributionFailed:
		if c.Reason == "" {
			return fmt.Errorf("%w: unavailable contribution needs a reason", ErrIncompatible)
		}
	default:
		return fmt.Errorf("%w: invalid contribution state %q", ErrIncompatible, c.State)
	}
	if c.Component != nil {
		if err := c.Component.Validate(c.Role); err != nil {
			return err
		}
	}
	for _, representation := range c.Representations {
		if err := representation.Validate(); err != nil {
			return err
		}
		if c.Component == nil || representation.Component != *c.Component || representation.Component.Role != c.Role {
			return fmt.Errorf("%w: contribution representation has mismatched component", ErrIncompatible)
		}
	}
	for _, event := range c.Events {
		if err := event.Validate(); err != nil {
			return err
		}
		if c.Component == nil || event.Representation.Component != *c.Component || c.Role != RoleEvent {
			return fmt.Errorf("%w: event contribution has mismatched component", ErrIncompatible)
		}
	}
	if c.Role == RoleEvent {
		if len(c.Representations) != 0 {
			return fmt.Errorf("%w: event contributions must retain per-event metadata", ErrIncompatible)
		}
		if (c.State == ContributionAvailable || c.State == ContributionPartial) && len(c.Events) == 0 {
			return fmt.Errorf("%w: available event contribution has no event records", ErrIncompatible)
		}
		if c.State == ContributionNoData && len(c.Events) != 0 {
			return fmt.Errorf("%w: no-data event contribution contains events", ErrIncompatible)
		}
	} else {
		if len(c.Events) != 0 {
			return fmt.Errorf("%w: non-event contribution contains event metadata", ErrIncompatible)
		}
		if (c.State == ContributionUnavailable || c.State == ContributionFailed || c.State == ContributionNoData) && len(c.Representations) != 0 {
			return fmt.Errorf("%w: empty/unavailable contribution contains representations", ErrIncompatible)
		}
		if (c.State == ContributionAvailable || c.State == ContributionPartial) && len(c.Representations) == 0 {
			return fmt.Errorf("%w: available contribution has no representations", ErrIncompatible)
		}
	}
	return nil
}
