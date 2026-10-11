package hybrid

import (
	"fmt"
	"math"
	"time"
)

type CognitiveHeadInput struct {
	SchemaVersion  string         `json:"schema_version"`
	DecisionCutoff time.Time      `json:"decision_cutoff"`
	Fusion         FusionEnvelope `json:"cognitive_fusion"`
}

func (i CognitiveHeadInput) Validate() error {
	if i.SchemaVersion != HeadInputSchemaVersion || i.DecisionCutoff.IsZero() {
		return fmt.Errorf("%w: invalid cognitive-head input schema", ErrIncompatible)
	}
	if err := i.Fusion.Validate(); err != nil {
		return err
	}
	if i.Fusion.Stage != FusionCognitive {
		return fmt.Errorf("%w: cognitive head requires cognitive fusion", ErrIncompatible)
	}
	return nil
}

type HeadOutcome string

const (
	OutcomeAssessment HeadOutcome = "assessment"
	OutcomePrediction HeadOutcome = "prediction"
	OutcomeNoEvent    HeadOutcome = "no-event"
	OutcomeNoAction   HeadOutcome = "no-action"
	OutcomeAbstain    HeadOutcome = "abstain"
)

type UncertaintyEstimate struct {
	Kind  string  `json:"kind"`
	Value float32 `json:"value"`
}

type ActionProposal struct {
	ProposalID string `json:"proposal_id"`
	Summary    string `json:"summary"`
}

type SituationAssessment struct {
	Summary        string          `json:"summary"`
	RiskEstimate   *float32        `json:"risk_estimate,omitempty"`
	ProposedAction *ActionProposal `json:"proposed_action,omitempty"`
}

type FutureEventPrediction struct {
	EventKind     string   `json:"event_kind"`
	EntityRef     string   `json:"entity_ref,omitempty"`
	HorizonMillis int64    `json:"horizon_millis"`
	RiskEstimate  *float32 `json:"risk_estimate,omitempty"`
}

type AssistanceProposal struct {
	Summary            string   `json:"summary"`
	TargetEntityRef    string   `json:"target_entity_ref,omitempty"`
	RequiredCapability string   `json:"required_capability,omitempty"`
	UsefulnessEstimate *float32 `json:"usefulness_estimate,omitempty"`
}

// HeadOutput is a proposal/interpretation contract only. It deliberately has
// no authorization result, dispatch method, or physical-action field.
type HeadOutput struct {
	SchemaVersion  string                 `json:"schema_version"`
	DecisionCutoff time.Time              `json:"decision_cutoff"`
	Head           HeadKind               `json:"head"`
	Component      ComponentReference     `json:"component"`
	Outcome        HeadOutcome            `json:"outcome"`
	Situation      *SituationAssessment   `json:"situation,omitempty"`
	FutureEvent    *FutureEventPrediction `json:"future_event,omitempty"`
	Assistance     *AssistanceProposal    `json:"assistance,omitempty"`
	Uncertainty    *UncertaintyEstimate   `json:"uncertainty,omitempty"`
	EvidenceRefs   []string               `json:"evidence_refs,omitempty"`
}

func (o HeadOutput) Validate() error {
	if o.SchemaVersion != HeadOutputSchemaVersion || o.DecisionCutoff.IsZero() || !validHead(o.Head) || o.Component.Role != RoleCognitiveHead || o.Component.Head != o.Head || o.Component.ID == "" {
		return fmt.Errorf("%w: invalid head output metadata", ErrIncompatible)
	}
	if err := o.Component.Validate(RoleCognitiveHead); err != nil {
		return err
	}
	if o.Uncertainty != nil {
		if o.Uncertainty.Kind == "" || !probability(o.Uncertainty.Value) {
			return fmt.Errorf("%w: invalid uncertainty estimate", ErrIncompatible)
		}
	}
	for _, ref := range o.EvidenceRefs {
		if ref == "" {
			return fmt.Errorf("%w: empty evidence reference", ErrIncompatible)
		}
	}
	switch o.Head {
	case HeadSituationAssessment:
		if o.FutureEvent != nil || o.Assistance != nil {
			return fmt.Errorf("%w: situation output contains another head payload", ErrIncompatible)
		}
		switch o.Outcome {
		case OutcomeAssessment:
			if o.Situation == nil || o.Situation.Summary == "" {
				return fmt.Errorf("%w: assessment outcome requires situation", ErrIncompatible)
			}
		case OutcomeNoAction, OutcomeAbstain:
			if o.Situation != nil {
				return fmt.Errorf("%w: no-action/abstain must not contain a situation payload", ErrIncompatible)
			}
		default:
			return fmt.Errorf("%w: invalid situation outcome %q", ErrIncompatible, o.Outcome)
		}
		if o.Situation != nil && o.Situation.RiskEstimate != nil && !probability(*o.Situation.RiskEstimate) {
			return fmt.Errorf("%w: invalid situation risk estimate", ErrIncompatible)
		}
	case HeadEventPrediction:
		if o.Situation != nil || o.Assistance != nil {
			return fmt.Errorf("%w: event output contains another head payload", ErrIncompatible)
		}
		switch o.Outcome {
		case OutcomePrediction:
			if o.FutureEvent == nil || o.FutureEvent.EventKind == "" || o.FutureEvent.HorizonMillis <= 0 {
				return fmt.Errorf("%w: prediction outcome requires event and positive horizon", ErrIncompatible)
			}
		case OutcomeNoEvent, OutcomeAbstain:
			if o.FutureEvent != nil {
				return fmt.Errorf("%w: no-event/abstain must not contain event payload", ErrIncompatible)
			}
		default:
			return fmt.Errorf("%w: invalid event prediction outcome %q", ErrIncompatible, o.Outcome)
		}
		if o.FutureEvent != nil && o.FutureEvent.RiskEstimate != nil && !probability(*o.FutureEvent.RiskEstimate) {
			return fmt.Errorf("%w: invalid future-event risk estimate", ErrIncompatible)
		}
	case HeadDomesticAssistance:
		if o.Situation != nil || o.FutureEvent != nil {
			return fmt.Errorf("%w: assistance output contains another head payload", ErrIncompatible)
		}
		switch o.Outcome {
		case OutcomePrediction:
			if o.Assistance == nil || o.Assistance.Summary == "" {
				return fmt.Errorf("%w: assistance proposal outcome requires proposal", ErrIncompatible)
			}
		case OutcomeNoAction, OutcomeAbstain:
			if o.Assistance != nil {
				return fmt.Errorf("%w: no-action/abstain must not contain proposal payload", ErrIncompatible)
			}
		default:
			return fmt.Errorf("%w: invalid assistance outcome %q", ErrIncompatible, o.Outcome)
		}
		if o.Assistance != nil && o.Assistance.UsefulnessEstimate != nil && !probability(*o.Assistance.UsefulnessEstimate) {
			return fmt.Errorf("%w: invalid usefulness estimate", ErrIncompatible)
		}
	}
	return nil
}

func probability(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0) && value >= 0 && value <= 1
}
