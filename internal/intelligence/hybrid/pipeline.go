package hybrid

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"synora/internal/intelligence/stateencoder"
)

type PipelineInput struct {
	EventInput    EventInput                 `json:"event_input"`
	Entities      []AvailableEntity          `json:"entities,omitempty"`
	Relations     []AvailableRelation        `json:"directed_relations,omitempty"`
	Locations     []EventLocationAssociation `json:"event_locations,omitempty"`
	TopologyState TopologyAvailability       `json:"topology_availability"`
	HistoryState  HistoryAvailability        `json:"history_availability"`
	HistoryStart  *time.Time                 `json:"history_start,omitempty"`
	HistoryEnd    *time.Time                 `json:"history_end,omitempty"`
	Home          HomeAdaptationContext      `json:"home_context"`
}

func (i PipelineInput) Validate() error {
	if err := i.EventInput.Validate(); err != nil {
		return err
	}
	if err := i.Home.Validate(); err != nil {
		return err
	}
	if i.Home.State == HomeAvailable && i.Home.TrainingCutoff.After(i.EventInput.Cutoff) {
		return fmt.Errorf("%w: home adaptation was trained beyond the decision cutoff", ErrIncompatible)
	}
	return (TopoTemporalInput{SchemaVersion: TopoInputSchemaVersion, StateEncoderSchemaVersion: i.EventInput.StateEncoderSchemaVersion, Cutoff: i.EventInput.Cutoff,
		Entities: i.Entities, Relations: i.Relations, Locations: i.Locations,
		TopologyState: i.TopologyState, HistoryState: i.HistoryState,
		HistoryStart: i.HistoryStart, HistoryEnd: i.HistoryEnd}).Validate()
}

type PipelineConfig struct {
	// Test fixtures are rejected unless this is explicitly set. Production
	// defaults never run deterministic test representations as model output.
	AllowTestFixtures bool
}

type Pipeline struct {
	Registry *Registry
	Config   PipelineConfig
}

type PipelineResult struct {
	GeneralFusion   FusionEnvelope `json:"general_fusion"`
	CognitiveFusion FusionEnvelope `json:"cognitive_fusion"`
}

// Run invokes only representation branches and structural fusion. It does not
// evaluate cognitive heads, emit predictions, or call Cognitive Core.
func (p Pipeline) Run(ctx context.Context, input PipelineInput) (PipelineResult, error) {
	if err := ctx.Err(); err != nil {
		return PipelineResult{}, err
	}
	if p.Registry == nil {
		return PipelineResult{}, fmt.Errorf("%w: nil registry", ErrNotRegistered)
	}
	if err := input.Validate(); err != nil {
		return PipelineResult{}, err
	}
	var eventRepresentations []EventRepresentation
	eventMetadata, _, err := p.Registry.invoke(RoleEvent, "", func(component ManagedComponent) error {
		branch, ok := component.(EventBranch)
		if !ok {
			return fmt.Errorf("%w: registered event component does not implement EventBranch", ErrIncompatible)
		}
		if !p.executionAllowed(component.Metadata()) {
			return ErrExecutionDisabled
		}
		var callErr error
		eventRepresentations, callErr = branch.EncodeEvents(ctx, cloneEventInput(input.EventInput))
		if callErr != nil {
			return callErr
		}
		if len(eventRepresentations) != len(input.EventInput.Observations) {
			return fmt.Errorf("%w: event branch returned %d representations for %d observations", ErrIncompatible, len(eventRepresentations), len(input.EventInput.Observations))
		}
		for index, event := range eventRepresentations {
			available := input.EventInput.Observations[index]
			if !eventMatchesObservation(event, available) {
				return fmt.Errorf("%w: event branch changed identity/provenance at position %d", ErrIncompatible, available.Position)
			}
			if err := validateRepresentationFor(event.Representation, component.Metadata()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return PipelineResult{}, fmt.Errorf("event branch: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return PipelineResult{}, err
	}
	eventState := ContributionAvailable
	if input.EventInput.Availability == ObservationsPartial {
		eventState = ContributionPartial
	} else if input.EventInput.Availability == ObservationsMissing {
		eventState = ContributionNoData
	}
	eventContribution := BranchContribution{Role: RoleEvent, State: eventState,
		Component: componentRefPointer(eventMetadata), Events: cloneEventRepresentations(eventRepresentations)}

	topoInput := TopoTemporalInput{SchemaVersion: TopoInputSchemaVersion, StateEncoderSchemaVersion: input.EventInput.StateEncoderSchemaVersion, Cutoff: input.EventInput.Cutoff,
		Events: cloneEventRepresentations(eventRepresentations), Entities: input.Entities, Relations: input.Relations, Locations: input.Locations,
		TopologyState: input.TopologyState, HistoryState: input.HistoryState,
		HistoryStart: input.HistoryStart, HistoryEnd: input.HistoryEnd}
	topoInput = canonicalizeTopoInput(topoInput)
	var topoRepresentation Representation
	topoMetadata, _, err := p.Registry.invoke(RoleTopoTemporal, "", func(component ManagedComponent) error {
		branch, ok := component.(TopoTemporalBranch)
		if !ok {
			return fmt.Errorf("%w: registered topo-temporal component does not implement TopoTemporalBranch", ErrIncompatible)
		}
		if !p.executionAllowed(component.Metadata()) {
			return ErrExecutionDisabled
		}
		var callErr error
		topoRepresentation, callErr = branch.EncodeContext(ctx, cloneTopoInput(topoInput))
		if callErr != nil {
			return callErr
		}
		return validateRepresentationFor(topoRepresentation, component.Metadata())
	})
	if err != nil {
		return PipelineResult{}, fmt.Errorf("topo-temporal branch: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return PipelineResult{}, err
	}
	topoState := ContributionAvailable
	if input.TopologyState != TopologyComplete || input.HistoryState != HistoryAvailable || eventState == ContributionPartial || eventState == ContributionNoData {
		topoState = ContributionPartial
	}
	general, err := NewGeneralFusion(eventContribution, BranchContribution{Role: RoleTopoTemporal,
		State: topoState, Component: componentRefPointer(topoMetadata), Representations: []Representation{topoRepresentation}})
	if err != nil {
		return PipelineResult{}, fmt.Errorf("build general fusion: %w", err)
	}

	cognitive := NewCognitiveFusion(general, BranchContribution{Role: RoleHomeAdaptation,
		State: ContributionUnavailable, Reason: "optional_home_component_not_registered"})
	if input.Home.State == HomeColdStart || input.Home.State == HomeUnavailable || input.Home.State == HomeReset {
		cognitive.Contributions[0].Reason = "home_adaptation_" + string(input.Home.State)
		if metadata, state, found, describeErr := p.Registry.Describe(RoleHomeAdaptation, ""); describeErr != nil {
			cognitive.Contributions[0].State = ContributionFailed
			cognitive.Contributions[0].Reason = "ambiguous_home_components"
		} else if found {
			cognitive.Contributions[0].Component = componentRefPointer(metadata)
			if state == LifecycleFailed || state == LifecycleStopped {
				cognitive.Contributions[0].State = ContributionFailed
				cognitive.Contributions[0].Reason = "home_component_" + string(state)
			}
		}
	} else {
		homeInput := HomeAdaptationInput{SchemaVersion: HomeInputSchemaVersion, Cutoff: input.EventInput.Cutoff, General: cloneFusion(general), Home: input.Home}
		if err := homeInput.Validate(); err != nil {
			return PipelineResult{}, err
		}
		var homeRepresentation Representation
		homeMetadata, homeState, homeErr := p.Registry.invoke(RoleHomeAdaptation, "", func(component ManagedComponent) error {
			branch, ok := component.(HomeAdaptationBranch)
			if !ok {
				return fmt.Errorf("%w: registered home component does not implement HomeAdaptationBranch", ErrIncompatible)
			}
			if !p.executionAllowed(component.Metadata()) {
				return ErrExecutionDisabled
			}
			var callErr error
			homeRepresentation, callErr = branch.EncodeHomeContext(ctx, cloneHomeInput(homeInput))
			if callErr != nil {
				return callErr
			}
			return validateRepresentationFor(homeRepresentation, component.Metadata())
		})
		if homeErr == nil {
			cognitive.Contributions[0] = BranchContribution{Role: RoleHomeAdaptation, State: ContributionAvailable,
				Component: componentRefPointer(homeMetadata), Representations: []Representation{homeRepresentation}}
		} else {
			state := ContributionUnavailable
			if homeState == LifecycleFailed || homeState == LifecycleStopped {
				state = ContributionFailed
			}
			cognitive.Contributions[0] = BranchContribution{Role: RoleHomeAdaptation, State: state,
				Component: componentRefPointerIfKnown(homeMetadata), Reason: safeErrorCode(homeErr)}
		}
	}
	if err := cognitive.Validate(); err != nil {
		return PipelineResult{}, fmt.Errorf("build cognitive fusion: %w", err)
	}
	return PipelineResult{GeneralFusion: general, CognitiveFusion: cognitive}, nil
}

func componentRefPointerIfKnown(metadata ComponentMetadata) *ComponentReference {
	if metadata.ID == "" {
		return nil
	}
	return componentRefPointer(metadata)
}

func canonicalizeTopoInput(input TopoTemporalInput) TopoTemporalInput {
	input.Entities = append([]AvailableEntity(nil), input.Entities...)
	input.Relations = append([]AvailableRelation(nil), input.Relations...)
	input.Locations = append([]EventLocationAssociation(nil), input.Locations...)
	sort.SliceStable(input.Entities, func(i, j int) bool { return input.Entities[i].Entity.ID < input.Entities[j].Entity.ID })
	sort.SliceStable(input.Relations, func(i, j int) bool {
		return relationSortKey(input.Relations[i]) < relationSortKey(input.Relations[j])
	})
	sort.SliceStable(input.Locations, func(i, j int) bool { return input.Locations[i].Position < input.Locations[j].Position })
	return input
}

func relationSortKey(value AvailableRelation) string {
	relation := value.Relation
	return relation.Subject + "\x00" + relation.Kind + "\x00" + relation.Object + "\x00" +
		sourceIdentity(relation.Source) + "\x00" + timeIdentity(relation.ValidFrom) + "\x00" +
		timeIdentity(relation.ValidTo) + "\x00" + value.ReceivedAt.Format(time.RFC3339Nano)
}

func timeIdentity(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func sourceIdentity(source *stateencoder.SourceRef) string {
	if source == nil {
		return ""
	}
	return source.ID + "\x00" + source.Kind
}

func (p Pipeline) executionAllowed(metadata ComponentMetadata) bool {
	switch metadata.ExecutionClass {
	case ExecutionTrained:
		return true
	case ExecutionTestFixture:
		return p.Config.AllowTestFixtures
	default:
		return false
	}
}

func eventMatchesObservation(event EventRepresentation, available AvailableObservation) bool {
	observation := available.Observation
	return event.Position == available.Position && event.Entity == observation.Entity && event.Attribute == observation.Attribute &&
		event.ReceivedAt.Equal(available.ReceivedAt) && event.Provenance == observation.Provenance &&
		event.EvidenceRef == observation.EvidenceRef && reflect.DeepEqual(event.Source, observation.Source) &&
		reflect.DeepEqual(event.ObservedAt, observation.ObservedAt)
}

func componentRefPointer(metadata ComponentMetadata) *ComponentReference {
	reference := referenceFromMetadata(metadata)
	return &reference
}

func safeErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrComponentFailed):
		return "component_failed"
	case errors.Is(err, ErrComponentUnavailable), errors.Is(err, ErrNotRegistered):
		return "component_unavailable"
	default:
		return "component_contract_or_runtime_error"
	}
}
