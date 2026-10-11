// Package automation owns deterministic action validation and dispatch policy.
// This package deliberately has no physical device adapter.
package automation

import (
	"errors"
	"strings"
)

const BoundarySchemaVersion = "discovery-boundary/v1"

// ActionRequest is abstract and topological. It contains no hardware ID and
// can cross the Core outbox without binding Automation to a device protocol.
type ActionRequest struct {
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	EpisodeID     string `json:"episode_id,omitempty"`
	Action        string `json:"action"`
	Topology      string `json:"topology,omitempty"`
	Capability    string `json:"capability,omitempty"`
	DryRun        bool   `json:"dry_run"`
}

type ActionResult struct {
	SchemaVersion          string `json:"schema_version"`
	RequestID              string `json:"request_id"`
	Status                 string `json:"status"`
	Reason                 string `json:"reason,omitempty"`
	PhysicalActionExecuted bool   `json:"physical_action_executed"`
}

// Dispatcher reports the current controlled dry-run outcome. A false DryRun
// setting blocks the request; it never enables physical execution.
type Dispatcher struct {
	DryRun       bool
	Capabilities map[string]bool
}

func (d Dispatcher) Dispatch(request ActionRequest) (ActionResult, error) {
	if request.SchemaVersion == "" {
		request.SchemaVersion = BoundarySchemaVersion
	}
	if strings.TrimSpace(request.RequestID) == "" || strings.TrimSpace(request.Action) == "" {
		return ActionResult{}, errors.New("action request_id and action are required")
	}
	result := ActionResult{SchemaVersion: BoundarySchemaVersion, RequestID: request.RequestID, Status: "dry_run"}
	switch {
	case !d.DryRun:
		result.Status = "blocked"
		result.Reason = "physical_execution_disabled_in_v1"
	case request.Action == "no_action":
		result.Status = "not_requested"
	case strings.TrimSpace(request.Capability) == "" || !d.Capabilities[request.Capability]:
		result.Status = "unavailable"
		result.Reason = "capability_unavailable"
	}
	return result, nil
}
