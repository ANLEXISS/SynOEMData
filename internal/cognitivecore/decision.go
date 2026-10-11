package cognitivecore

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	DecisionSchemaVersion = "cognitive-decision/v1"
	BundleManifestSchema  = "synora.cognitive-v1-manifest/v1"
)

var ErrModelUnavailable = errors.New("V1 MLP model unavailable")

var HeadOrder = [...]string{"danger", "incident", "task", "action"}

var AbstractActions = [...]string{"no_action", "notify", "record", "light", "lock", "siren", "request_review"}

type ActionIntent struct {
	Action     string `json:"action"`
	Topology   string `json:"topology,omitempty"`
	Capability string `json:"capability,omitempty"`
}

type MLPOutput struct {
	DangerLabel        string       `json:"danger_label"`
	DangerScore        float32      `json:"danger_score"`
	IncidentConfidence float32      `json:"incident_confidence,omitempty"`
	TaskConfidence     float32      `json:"task_confidence,omitempty"`
	ActionConfidence   float32      `json:"action_confidence,omitempty"`
	Incidents          []string     `json:"incidents,omitempty"`
	Task               string       `json:"task,omitempty"`
	Action             ActionIntent `json:"action"`
	Trace              *MLPTrace    `json:"trace,omitempty"`
}

const MLPTraceSchemaVersion = "synora.mlp-trace/v1"

// MLPTrace is a bounded, redacted observation contract. It never contains
// input values, embeddings, identities, media, secrets, or model weights.
type MLPTrace struct {
	SchemaVersion string                 `json:"schema_version"`
	InferenceID   string                 `json:"inference_id"`
	ModelVersion  string                 `json:"model_version"`
	Provenance    string                 `json:"provenance,omitempty"`
	Test          bool                   `json:"test,omitempty"`
	Topology      MLPTopology            `json:"topology"`
	Activations   []MLPActivationSummary `json:"activations"`
	ActivePaths   []MLPActivePath        `json:"active_paths,omitempty"`
	Proposed      string                 `json:"proposed_output,omitempty"`
	DurationMS    float64                `json:"duration_ms"`
	Redacted      bool                   `json:"redacted"`
}

type MLPTopology struct {
	SchemaVersion string            `json:"schema_version"`
	ModelVersion  string            `json:"model_version"`
	Heads         []MLPHeadTopology `json:"heads"`
}

type MLPHeadTopology struct {
	Name   string             `json:"name"`
	Layers []MLPLayerTopology `json:"layers"`
}

type MLPLayerTopology struct {
	ID         string `json:"id"`
	InputSize  int    `json:"input_size"`
	OutputSize int    `json:"output_size"`
	Activation string `json:"activation"`
}

type MLPActivationSummary struct {
	LayerID     string          `json:"layer_id"`
	Count       int             `json:"count"`
	Minimum     float32         `json:"minimum"`
	Maximum     float32         `json:"maximum"`
	Mean        float32         `json:"mean"`
	ActiveNodes []MLPActiveNode `json:"active_nodes,omitempty"`
}

type MLPActiveNode struct {
	NodeID     string  `json:"node_id"`
	Activation float32 `json:"activation"`
}

type MLPActivePath struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Strength float32 `json:"strength"`
}

type Decision struct {
	SchemaVersion      string             `json:"schema_version"`
	Status             string             `json:"status"`
	Mode               string             `json:"mode"`
	Source             string             `json:"source"`
	Provenance         string             `json:"provenance,omitempty"`
	Test               bool               `json:"test,omitempty"`
	HeadOrder          []string           `json:"head_order"`
	InputDimension     int                `json:"input_dimension"`
	DangerLabel        string             `json:"danger_label,omitempty"`
	DangerScore        float32            `json:"danger_score,omitempty"`
	IncidentConfidence float32            `json:"incident_confidence,omitempty"`
	TaskConfidence     float32            `json:"task_confidence,omitempty"`
	ActionConfidence   float32            `json:"action_confidence,omitempty"`
	Incidents          []string           `json:"incidents,omitempty"`
	Task               string             `json:"task,omitempty"`
	Action             ActionAssessment   `json:"action"`
	HeadLatencyMS      map[string]float64 `json:"head_latency_ms,omitempty"`
	Trace              *MLPTrace          `json:"trace,omitempty"`
	Error              string             `json:"error,omitempty"`
	GeneratedAt        time.Time          `json:"generated_at"`
}

type ActionAssessment struct {
	Proposed               ActionIntent `json:"proposed"`
	Status                 string       `json:"status"`
	Reasons                []string     `json:"reasons,omitempty"`
	PhysicalActionExecuted bool         `json:"physical_action_executed"`
}

type MLPBackend interface {
	Run(context.Context, EncodedSnapshot, CognitiveSnapshot) (MLPOutput, map[string]float64, error)
}

type UnavailableMLP struct{ Reason string }

func (u UnavailableMLP) Run(context.Context, EncodedSnapshot, CognitiveSnapshot) (MLPOutput, map[string]float64, error) {
	if u.Reason == "" {
		return MLPOutput{}, nil, ErrModelUnavailable
	}
	return MLPOutput{}, nil, fmt.Errorf("%w: %s", ErrModelUnavailable, u.Reason)
}

type SafetyGate struct {
	DryRun            bool
	Capabilities      map[string]bool
	AllowedTopologies map[string]bool
	LastAction        map[string]time.Time
	Cooldown          time.Duration
}

func (g SafetyGate) Apply(output MLPOutput, snapshot CognitiveSnapshot, now time.Time) ActionAssessment {
	assessment := ActionAssessment{Proposed: output.Action, Status: "not_requested", PhysicalActionExecuted: false}
	if output.Action.Action == "" {
		assessment.Proposed.Action = "no_action"
	}
	if assessment.Proposed.Action == "no_action" {
		return assessment
	}
	if !isAbstractAction(assessment.Proposed.Action) {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"action_not_in_abstract_catalog"}
		return assessment
	}
	if len(g.AllowedTopologies) > 0 && !contractTopologyAllowed(g.AllowedTopologies, snapshot.Topology) {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"topology_not_allowed"}
		return assessment
	}
	if len(g.Capabilities) > 0 && (assessment.Proposed.Capability == "" || !g.Capabilities[assessment.Proposed.Capability]) {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"capability_unavailable"}
		return assessment
	}
	if g.Cooldown > 0 && g.LastAction != nil {
		if at, ok := g.LastAction[assessment.Proposed.Action]; ok && now.Sub(at) < g.Cooldown {
			assessment.Status = "blocked"
			assessment.Reasons = []string{"cooldown"}
			return assessment
		}
	}
	if g.DryRun {
		assessment.Status = "allowed_dry_run"
		assessment.Reasons = []string{"dry_run"}
		return assessment
	}
	assessment.Status = "allowed"
	return assessment
}

func isAbstractAction(value string) bool {
	for _, action := range AbstractActions {
		if action == value {
			return true
		}
	}
	return false
}
func contractTopologyAllowed(values map[string]bool, topology string) bool {
	if len(values) == 0 {
		return false
	}
	return values[topology]
}

func makeUnavailableDecision(now time.Time, err error) Decision {
	return Decision{SchemaVersion: DecisionSchemaVersion, Status: "unavailable", Mode: "active_dry_run", Source: "mlp", HeadOrder: append([]string(nil), HeadOrder[:]...), InputDimension: CognitiveVectorSize, Action: ActionAssessment{Proposed: ActionIntent{Action: "no_action"}, Status: "not_requested", PhysicalActionExecuted: false}, Error: err.Error(), GeneratedAt: now.UTC()}
}
