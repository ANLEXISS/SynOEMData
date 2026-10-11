package cognitivecore

import (
	"context"
	"time"
)

var HeadOrderV3 = [...]string{"danger", "incident", "task", "action", "communication_intent"}

var LabelsV3 = map[string][]string{
	"danger":               {"none", "low", "medium", "high", "critical"},
	"incident":             {"none", "routine_presence", "perimeter_presence", "threshold_presence", "interior_intrusion", "anomaly", "action_failure", "technical", "resolved"},
	"task":                 {"monitor", "verify", "notify_security", "contain", "close_incident"},
	"action":               {"no_action", "notify", "record", "light", "lock", "siren", "request_review", "announce"},
	"communication_intent": {"none", "neutral_presence_notice", "request_identity", "wellness_check"},
}

const (
	CommunicationNoneV3                  = "none"
	CommunicationNeutralPresenceNoticeV3 = "neutral_presence_notice"
	CommunicationRequestIdentityV3       = "request_identity"
	CommunicationWellnessCheckV3         = "wellness_check"
)

type CommunicationIntentV3 struct {
	Intent     string `json:"intent"`
	TemplateID string `json:"template_id,omitempty"`
}

type MLPOutputV3 struct {
	DangerLabel             string                `json:"danger_label"`
	DangerScore             float32               `json:"danger_score"`
	Incident                string                `json:"incident"`
	IncidentConfidence      float32               `json:"incident_confidence"`
	Task                    string                `json:"task"`
	TaskConfidence          float32               `json:"task_confidence"`
	Action                  string                `json:"action"`
	ActionConfidence        float32               `json:"action_confidence"`
	Communication           CommunicationIntentV3 `json:"communication_intent"`
	CommunicationConfidence float32               `json:"communication_confidence"`
	ModelVersion            string                `json:"model_version"`
}

type CommunicationAssessmentV3 struct {
	Intent              CommunicationIntentV3 `json:"intent"`
	Capability          string                `json:"capability"`
	Status              string                `json:"status"`
	Reasons             []string              `json:"reasons,omitempty"`
	RenderedText        string                `json:"rendered_text,omitempty"`
	PhysicalAudioPlayed bool                  `json:"physical_audio_played"`
}

type CommunicationTemplate struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Text    string `json:"text"`
}

type DecisionV3 struct {
	SchemaVersion          string                    `json:"schema_version"`
	Status                 string                    `json:"status"`
	Mode                   string                    `json:"mode"`
	Source                 string                    `json:"source"`
	HeadOrder              []string                  `json:"head_order"`
	InputDimension         int                       `json:"input_dimension"`
	DangerLabel            string                    `json:"danger_label,omitempty"`
	DangerScore            float32                   `json:"danger_score,omitempty"`
	Incident               string                    `json:"incident,omitempty"`
	Task                   string                    `json:"task,omitempty"`
	Action                 ActionAssessment          `json:"action"`
	Communication          CommunicationAssessmentV3 `json:"communication"`
	HeadLatencyMS          map[string]float64        `json:"head_latency_ms,omitempty"`
	PhysicalActionExecuted bool                      `json:"physical_action_executed"`
	Error                  string                    `json:"error,omitempty"`
	GeneratedAt            time.Time                 `json:"generated_at"`
}

type MLPBackendV3 interface {
	RunV3(context.Context, EncodedSnapshotV3, CognitiveSnapshotV3) (MLPOutputV3, map[string]float64, error)
}

var CommunicationTemplatesV3 = map[string]CommunicationTemplate{
	CommunicationNeutralPresenceNoticeV3: {CommunicationNeutralPresenceNoticeV3, "1.0.0", "Bonjour. Votre présence a été détectée."},
	CommunicationRequestIdentityV3:       {CommunicationRequestIdentityV3, "1.0.0", "Bonjour. Votre présence a été détectée. Si vous êtes autorisé, veuillez vous identifier."},
	CommunicationWellnessCheckV3:         {CommunicationWellnessCheckV3, "1.0.0", "Est-ce que tout va bien ? Une situation inhabituelle a été détectée."},
}

func TemplateIDV3(intent string) string {
	if item, ok := CommunicationTemplatesV3[intent]; ok {
		return item.ID + "@" + item.Version
	}
	return ""
}

func ApplyCommunicationGateV3(snapshot CognitiveSnapshotV3, output MLPOutputV3) CommunicationAssessmentV3 {
	assessment := CommunicationAssessmentV3{Intent: output.Communication, Capability: "announce", Status: "not_requested", PhysicalAudioPlayed: false}
	if output.Communication.Intent == CommunicationNoneV3 {
		assessment.Reasons = []string{"model_intent_none"}
		return assessment
	}
	if !snapshot.BaseV2.Communication.AnnounceAvailable || snapshot.BaseV2.Communication.TTSStatus != TTSAvailable {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"tts_capability_unavailable"}
		return assessment
	}
	if snapshot.BaseV2.Communication.CooldownActive {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"communication_cooldown"}
		return assessment
	}
	if snapshot.Vision.RiskStatus == RiskUncertain || snapshot.Vision.RiskStatus == RiskSuspected || snapshot.Vision.RiskStatus == RiskConfirmed {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"high_risk_communication_forbidden"}
		return assessment
	}
	if output.DangerLabel == "high" || output.DangerLabel == "critical" {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"danger_high_or_critical"}
		return assessment
	}
	if output.Incident == "interior_intrusion" || output.Incident == "perimeter_presence" {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"intrusion_probable"}
		return assessment
	}
	if snapshot.Vision.FaceStatus == FaceV3Unknown && (snapshot.Vision.RiskStatus == RiskSuspected || snapshot.Vision.RiskStatus == RiskConfirmed || output.DangerLabel == "high" || output.DangerLabel == "critical") {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"identity_unknown_high_risk"}
		return assessment
	}
	if snapshot.Vision.CameraUncertainty && (snapshot.Vision.RiskStatus == RiskSuspected || snapshot.Vision.RiskStatus == RiskConfirmed || output.DangerLabel == "high" || output.DangerLabel == "critical") {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"vision_state_uncertain_high_risk"}
		return assessment
	}
	if !snapshot.Vision.RiskQualitySufficient && snapshot.Vision.RiskStatus != RiskNotAvailable && snapshot.Vision.RiskStatus != RiskNotRequested && snapshot.Vision.RiskStatus != "" {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"risk_quality_insufficient"}
		return assessment
	}
	if snapshot.Vision.FallState == FallV3Candidate || snapshot.Vision.FallState == FallV3Confirmed || output.DangerLabel == "critical" {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"high_risk_communication_forbidden"}
		return assessment
	}
	template, ok := CommunicationTemplatesV3[output.Communication.Intent]
	if !ok {
		assessment.Status = "blocked"
		assessment.Reasons = []string{"template_unavailable"}
		return assessment
	}
	assessment.Status = "allowed_dry_run"
	assessment.RenderedText = template.Text
	assessment.Reasons = []string{"template_versioned", "discovery_announce_only", "active_dry_run"}
	return assessment
}

func ApplyActionGateV3(snapshot CognitiveSnapshotV3, output MLPOutputV3) ActionAssessment {
	action := ActionAssessment{Proposed: ActionIntent{Action: output.Action, Topology: snapshot.BaseV2.Topology, Capability: output.Action}, Status: "blocked", PhysicalActionExecuted: false}
	if output.Action == "no_action" {
		action.Status = "not_requested"
		action.Reasons = []string{"model_no_action"}
		return action
	}
	action.Status = "allowed_dry_run"
	action.Reasons = []string{"active_dry_run", "physical_execution_disabled"}
	return action
}

func BuildDecisionV3(snapshot CognitiveSnapshotV3, output MLPOutputV3, latencies map[string]float64, now time.Time) DecisionV3 {
	decision := DecisionV3{SchemaVersion: "cognitive-decision/v3", Status: "available", Mode: "active_dry_run", Source: "mlp-v3-candidate", HeadOrder: append([]string(nil), HeadOrderV3[:]...), InputDimension: CognitiveVectorSizeV3, DangerLabel: output.DangerLabel, DangerScore: output.DangerScore, Incident: output.Incident, Task: output.Task, HeadLatencyMS: latencies, PhysicalActionExecuted: false, GeneratedAt: now.UTC()}
	decision.Communication = ApplyCommunicationGateV3(snapshot, output)
	decision.Action = ApplyActionGateV3(snapshot, output)
	if output.Action == "announce" && decision.Communication.Status != "allowed_dry_run" {
		decision.Action.Status = "blocked"
		decision.Action.Reasons = append(decision.Action.Reasons, "communication_gate_blocked")
	}
	return decision
}
