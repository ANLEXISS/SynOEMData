package mlp

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	cognitivecore "synora/internal/cognitivecore"
	"synora/pkg/contract"
)

func loadedTestCore(t *testing.T) (*cognitivecore.Core, *cognitivecore.UniversalStore) {
	t.Helper()
	bundle, err := LoadCPUBundle("../../../build/cognitive-mlp-v1")
	if err != nil {
		t.Skipf("V1 bundle not available in this checkout: %v", err)
	}
	store := cognitivecore.NewUniversalStore()
	return &cognitivecore.Core{Store: store, MLP: bundle, Gate: cognitivecore.SafetyGate{DryRun: true}, Now: func() time.Time { return time.Unix(600, 0).UTC() }}, store
}

func testHarnessEvent(t *testing.T, inference bool) contract.Event {
	t.Helper()
	evidence := evidenceEventForTest(t, "test-event-1")
	return contract.Event{ID: "test-event-1", Type: contract.EventVisionEvidenceV1, Source: "api", Timestamp: evidence.Timestamp, Payload: map[string]any{
		"provenance": "test-harness", "test": true, "test_inference": inference, "vision_evidence": evidence.Payload,
	}}
}

func TestProcessTestUsesRealBundleWithoutPersistenceOrAction(t *testing.T) {
	core, store := loadedTestCore(t)
	result, err := core.ProcessTest(context.Background(), testHarnessEvent(t, true))
	if err != nil {
		t.Fatal(err)
	}
	if result.Commit.Decision.Trace == nil || result.Commit.Decision.Trace.SchemaVersion != cognitivecore.MLPTraceSchemaVersion {
		t.Fatalf("missing MLP trace: %#v", result.Commit.Decision)
	}
	if !result.Commit.Decision.Trace.Redacted || !result.Commit.Decision.Trace.Test || result.Commit.Decision.Trace.Provenance != "test-harness" {
		t.Fatalf("test provenance/redaction missing: %#v", result.Commit.Decision.Trace)
	}
	if result.Result.Action != nil || len(store.Journal()) != 0 || store.Revision() != 1 {
		t.Fatalf("test inference touched business state or action outbox: result=%#v journal=%d revision=%d", result.Result, len(store.Journal()), store.Revision())
	}
}

func TestProcessTestWithoutInferenceReturnsNoCalculation(t *testing.T) {
	core, store := loadedTestCore(t)
	event := testHarnessEvent(t, false)
	result, err := core.ProcessTest(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if result.Commit.Decision.Status != "not_requested" || result.Commit.Decision.Trace != nil || len(store.Journal()) != 0 {
		t.Fatalf("non-inference scenario produced a calculation or persistence: %#v", result.Commit)
	}
}

func TestLegacyVisionCatalogTypesAreRejectedByTestCore(t *testing.T) {
	for _, catalogCase := range contract.TestInferenceCatalog() {
		if !catalogCase.TriggersInference {
			continue
		}
		t.Run(catalogCase.ID, func(t *testing.T) {
			core := &cognitivecore.Core{Store: cognitivecore.NewUniversalStore(), MLP: cognitivecore.UnavailableMLP{}, Gate: cognitivecore.SafetyGate{DryRun: true}}
			event := testHarnessEvent(t, true)
			event.Type = catalogCase.EventType
			if _, err := core.ProcessTest(context.Background(), event); err == nil {
				t.Fatalf("legacy event %q reached Core", catalogCase.EventType)
			}
		})
	}
}

func TestServiceTestEnvelopePublishesOnlyRedactedDecisionToAPI(t *testing.T) {
	core, store := loadedTestCore(t)
	bus := &integrationServiceBus{}
	service := &cognitivecore.Service{Bus: bus, Core: core, Name: "core"}
	evidence := evidenceEventForTest(t, "test-envelope")
	payload, err := json.Marshal(map[string]any{"event_type": contract.EventVisionEvidenceV1, "provenance": "test-harness", "test": true, "test_inference": true, "vision_evidence": evidence.Payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Handle(context.Background(), contract.Message{ID: "test-envelope", Type: contract.EventValidationTestInference, Kind: contract.KindEvent, Source: "api", Target: "core", Timestamp: time.Unix(600, 0).UTC(), Payload: payload}); err != nil {
		t.Fatal(err)
	}
	if len(bus.sent) != 1 || bus.sent[0].Type != "core.decision" || bus.sent[0].Target != "api" || len(store.Journal()) != 0 {
		t.Fatalf("test envelope crossed an unsafe boundary: sent=%#v journal=%d", bus.sent, len(store.Journal()))
	}
	if string(bus.sent[0].Payload) == "" || string(bus.sent[0].Payload) == "null" {
		t.Fatal("empty test decision payload")
	}
}

func TestServiceRejectsUncataloguedTestEventType(t *testing.T) {
	core, _ := loadedTestCore(t)
	service := &cognitivecore.Service{Bus: &integrationServiceBus{}, Core: core}
	payload := json.RawMessage(`{"event_type":"unknown.test.event","provenance":"test-harness","test":true,"test_inference":true}`)
	if err := service.Handle(context.Background(), contract.Message{ID: "unknown-test", Type: contract.EventValidationTestInference, Kind: contract.KindEvent, Source: "api", Target: "core", Timestamp: time.Unix(600, 0).UTC(), Payload: payload}); err == nil {
		t.Fatal("uncatalogued test event type was accepted")
	}
}

func evidenceEventForTest(t *testing.T, id string) contract.Event {
	t.Helper()
	body, err := os.ReadFile("../../../pkg/contract/testdata/v1/vision-evidence-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := contract.DecodeVisionEvidenceV1(body)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	return contract.Event{ID: id, Type: contract.EventVisionEvidenceV1, Source: "api", Timestamp: evidence.WindowEnd, Payload: payload}
}

type integrationServiceBus struct{ sent []contract.Message }

func (b *integrationServiceBus) Send(message contract.Message) error {
	b.sent = append(b.sent, message)
	return nil
}
func (b *integrationServiceBus) SubscribeChannel(string) <-chan contract.Message { return nil }
