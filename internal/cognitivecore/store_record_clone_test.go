package cognitivecore

import (
	"testing"
	"time"

	"synora/pkg/contract"
)

func TestCognitiveSnapshotV3CloneDetachesNestedEvidence(t *testing.T) {
	evidence := &contract.VisionEvidenceV1{
		PriorityReasons:  []string{"baseline"},
		RuntimeAggregate: &contract.VisionRuntimeAggregateV1{RiskStatus: "none"},
	}
	snapshot := CognitiveSnapshotV3{
		BaseV2:                   CognitiveSnapshotV2{SchemaVersion: SnapshotSchemaVersionV2, CapturedAt: time.Unix(1, 0).UTC()},
		VisionEvidence:           evidence,
		VisionEvidenceProjection: VisionEvidenceProjectionV3{Facts: []string{"baseline"}},
	}
	cloned := snapshot.Clone()
	evidence.PriorityReasons[0] = "input mutation"
	evidence.RuntimeAggregate.RiskStatus = "input mutation"
	snapshot.VisionEvidenceProjection.Facts[0] = "input mutation"
	if cloned.VisionEvidence.PriorityReasons[0] != "baseline" || cloned.VisionEvidence.RuntimeAggregate.RiskStatus != "none" || cloned.VisionEvidenceProjection.Facts[0] != "baseline" {
		t.Fatalf("V3 clone retained input references: %+v", cloned)
	}
}

func TestDecisionCloneDetachesTraceAndMaps(t *testing.T) {
	decision := Decision{
		HeadOrder:     []string{"danger"},
		HeadLatencyMS: map[string]float64{"danger": 1},
		Action:        ActionAssessment{Reasons: []string{"safe"}},
		Trace: &MLPTrace{
			Topology:    MLPTopology{Heads: []MLPHeadTopology{{Layers: []MLPLayerTopology{{ID: "layer"}}}}},
			Activations: []MLPActivationSummary{{ActiveNodes: []MLPActiveNode{{NodeID: "node"}}}},
			ActivePaths: []MLPActivePath{{From: "a", To: "b"}},
		},
	}
	cloned := decision.Clone()
	decision.HeadOrder[0] = "changed"
	decision.HeadLatencyMS["danger"] = 99
	decision.Action.Reasons[0] = "changed"
	decision.Trace.Topology.Heads[0].Layers[0].ID = "changed"
	decision.Trace.Activations[0].ActiveNodes[0].NodeID = "changed"
	decision.Trace.ActivePaths[0].From = "changed"
	if cloned.HeadOrder[0] != "danger" || cloned.HeadLatencyMS["danger"] != 1 || cloned.Action.Reasons[0] != "safe" || cloned.Trace.Topology.Heads[0].Layers[0].ID != "layer" || cloned.Trace.Activations[0].ActiveNodes[0].NodeID != "node" || cloned.Trace.ActivePaths[0].From != "a" {
		t.Fatalf("decision clone retained mutable references: %+v", cloned)
	}
}

func TestDecisionV3CloneDetachesNestedFields(t *testing.T) {
	decision := DecisionV3{
		HeadOrder:     []string{"danger"},
		HeadLatencyMS: map[string]float64{"danger": 1},
		Action:        ActionAssessment{Reasons: []string{"safe"}},
		Communication: CommunicationAssessmentV3{Reasons: []string{"dry_run"}},
	}
	cloned := decision.Clone()
	decision.HeadOrder[0] = "changed"
	decision.HeadLatencyMS["danger"] = 99
	decision.Action.Reasons[0] = "changed"
	decision.Communication.Reasons[0] = "changed"
	if cloned.HeadOrder[0] != "danger" || cloned.HeadLatencyMS["danger"] != 1 || cloned.Action.Reasons[0] != "safe" || cloned.Communication.Reasons[0] != "dry_run" {
		t.Fatalf("V3 decision clone retained mutable references: %+v", cloned)
	}
}
