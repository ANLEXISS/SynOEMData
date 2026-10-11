package automation

import "testing"

func TestDispatcherControlledDryRunOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		dispatcher Dispatcher
		request    ActionRequest
		wantStatus string
		wantReason string
	}{
		{
			name:       "available capability is dry run only",
			dispatcher: Dispatcher{DryRun: true, Capabilities: map[string]bool{"lock": true}},
			request:    ActionRequest{RequestID: "req-1", Action: "lock", Capability: "lock"},
			wantStatus: "dry_run",
		},
		{
			name:       "unavailable capability fails closed",
			dispatcher: Dispatcher{DryRun: true},
			request:    ActionRequest{RequestID: "req-2", Action: "lock", Capability: "lock"},
			wantStatus: "unavailable",
			wantReason: "capability_unavailable",
		},
		{
			name:       "disabled execution stays blocked",
			dispatcher: Dispatcher{Capabilities: map[string]bool{"lock": true}},
			request:    ActionRequest{RequestID: "req-3", Action: "lock", Capability: "lock"},
			wantStatus: "blocked",
			wantReason: "physical_execution_disabled_in_v1",
		},
		{
			name:       "no action is not requested",
			dispatcher: Dispatcher{DryRun: true},
			request:    ActionRequest{RequestID: "req-4", Action: "no_action"},
			wantStatus: "not_requested",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.dispatcher.Dispatch(test.request)
			if err != nil {
				t.Fatalf("Dispatch() error = %v", err)
			}
			if got.Status != test.wantStatus || got.Reason != test.wantReason {
				t.Fatalf("Dispatch() = status %q reason %q, want %q %q", got.Status, got.Reason, test.wantStatus, test.wantReason)
			}
			if got.PhysicalActionExecuted {
				t.Fatal("dry-run dispatcher reported physical execution")
			}
			if got.SchemaVersion != BoundarySchemaVersion || got.RequestID != test.request.RequestID {
				t.Fatalf("result contract mismatch: %#v", got)
			}
		})
	}
}

func TestDispatcherRequiresRequestIdentityAndAction(t *testing.T) {
	for _, request := range []ActionRequest{{Action: "lock"}, {RequestID: "req-1"}} {
		if _, err := (Dispatcher{DryRun: true}).Dispatch(request); err == nil {
			t.Fatalf("Dispatch(%#v) unexpectedly succeeded", request)
		}
	}
}
