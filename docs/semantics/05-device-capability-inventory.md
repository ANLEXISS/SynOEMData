# Device Capability Inventory

**Status:** Source-backed inventory; not a qualification matrix  
**Schema:** `synora.oem-device-capability-inventory/v1`

## Scope and method

This inventory records capability identifiers that appear in installed device configuration, Vision worker hello metadata, and abstract cognitive/automation decisions. It distinguishes descriptive identifiers from executable operations. The machine-readable version is [`device-capabilities-v1.json`](../../testdata/contract-baseline/device-capabilities-v1.json).

The device configuration was inspected by capability blocks only. Network, firmware, credential, and device identity fields are outside this inventory. The three previously identified authentication/hash-bearing YAML templates are excluded from synchronization.

## Current capability families

| Source and owner | Identifiers | Representation and validation | Operation mapping and current behavior |
|---|---|---|---|
| Device configuration; `internal/device` | `vision`, `infrared`, `motion_detection`, `cry_detection`, `microphone`, `speaker`, `tts`, `wakeword` | String arrays on device records. Device validation checks identifier syntax and duplicates, not vendor support. Discovery camera observations canonicalize capability strings. | No configured identifier maps directly to a device operation. `voice_relay` values are present in configuration, but a Voice command runtime is not implemented. |
| Vision worker; `services/vision-worker` | `face_detection`, `face_recognition`, `object_detection`, `weapon_detection`, `fall_detection`, `pose_enrichment_v3`, `risk_object_enrichment_v3` | JSON map of identifiers to status and optional mode/reason. Values vary with dry-run mode and qualified backends. | These describe inference features. They do not authorize actuator actions. A capability can be unavailable, not available, or available in dry-run mode. |
| Cognitive/Automation decisions | `no_action`, `notify`, `record`, `light`, `lock`, `siren`, `request_review`, `announce` | Abstract action strings in decision/policy contracts. Automation accepts capability data through an opaque map. | These are proposed action labels, not a complete device operation catalog. Dispatch remains dry-run; no physical actuator execution is claimed. |

## Validation, consumers, and gaps

The device registry validates identifiers with the pattern `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$` and rejects duplicates. That syntax check establishes neither that the device supports the value nor that an operation is available. Discovery preserves normalized camera capability identifiers. Vision worker metadata reports feature readiness separately. Automation policy can deny a proposed action when an explicit capability is absent, but there is no common descriptor tying a configured identifier to an authorized command, adapter, and execution result.

The H3 simulator uses fixture-declared capabilities only to exercise supported versus unsupported outcomes. It cannot establish real device support. No ONVIF or Matter conformance, hardware qualification, Voice implementation, physical camera inference qualification, or actuator command path is established by this inventory.

## Source and test references

- Device configuration identifiers: `configs/devices.yaml` (capability blocks only).
- Identifier syntax and device validation: `internal/device/devices.go`.
- Camera capability normalization and observation: `pkg/contract/camera.go` and `internal/discovery/runtime/device.go`.
- Vision feature status: `services/vision-worker/worker.py`.
- Abstract decision/dispatch behavior: `internal/cognitivecore` and `internal/automation/action_test.go`.
- Repeatable capability and unsupported-operation scenario: `testdata/device-harness/scenarios-v1.json` and `testkit/scenario_test.go`.
