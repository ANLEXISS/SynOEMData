# Contract audit and compatibility plan

## Existing contract families

| Family | Current source | Purpose / transport | Compatibility notes |
| --- | --- | --- | --- |
| Bus envelope | `pkg/contract/message.go` (`Message`, message kinds) | JSON envelope over authenticated Unix socket; source/target, timestamp, priority, request/response correlation and payload. | Transport message and domain `Event` are distinct types. Preserve both layers and existing serialized field names. |
| Domain events | `pkg/contract/event.go`, `event_chain.go`, `vision_evidence_v1.go` | Versioned events, validated payloads, provenance and chain metadata. | Existing validators reject unsafe/unknown payload patterns; additions need explicit versioning and tests. |
| RPC / delivery | `rpc.go`, `delivery.go`, `runtime.go` | Bus request/response, durable delivery and runtime health/control result. | Runtime manager commands are not device action contracts. |
| Decisions and actions | `decision.go`, `action.go`, `action_plan.go`, `danger.go` | Cognitive decision, action proposal/request/result and risk semantics. | Core gate and Discovery boundary are separate enforcement stages; physical execution remains dry-run. |
| Devices | `device.go`, `internal/device` | Device identity/capabilities/configuration and patches/validated registry. | Configured record is not proof of a connected/available device. Vendor identity and commands should stay behind OEM adapters. |
| State | `public_snapshot.go`, `runtime.go`, cognitivecore snapshots and `internal/state/types.go` | Public redacted read view, runtime health, Core snapshots, and legacy event state. | Multiple state-shaped structures serve distinct owners. Do not merge them by name alone. Active Universal Store is `internal/cognitivecore`; legacy `internal/state` is retained but inactive. |
| Residents / topology | `resident.go`, `topology.go`, `internal/intelligence/topology` | Resident-safe views/patches and configured topology/links/nodes. | Public resident view is narrower than persisted record. Topology configuration is not physical localization evidence. |
| Vision / media | `camera.go`, `clip.go`, `vision_clip_summary.go`, `vision_segment_ready.go`, `vision_evidence_v1.go`, Python `core/events.py` | Camera, clip, segment and validated inference evidence/summary contracts. | Go and Python both validate/construct related schemas; exact JSON golden fixtures are essential to compatibility. Raw frames/embeddings are intentionally bounded from central event payloads. |
| Connectivity | `pkg/contracts/connectivity.go` plus `internal/connectivity` state/config/access records | Local connectivity status and access policy. | Package name overlap and duplicated internal state types deserve canonical-schema ownership decision; Paho/MQTT is not used by these contracts. |
| HTTP/WebSocket | `internal/discovery/web_v1.go`, `cmd/synora-api`, systemstate handlers, `synora-web` | HTTP routes, SSE/WebSocket event projections, authentication and redacted state. | There are two Go server compositions. Wire compatibility includes path, method, auth role, JSON names, status codes and no-store/redaction requirements. |
| Foundation synthetic ports | `internal/foundationv1` and `PipelineInput` fixture | Abstract observation/decision/action/device/test ports used by a synthetic E2E harness. | Test/reference contract only; not a replacement for active production event or device schemas. |

## Canonical target concepts (proposed, not installed)

| Canonical concept | Existing source to preserve | Gap / migration requirement |
| --- | --- | --- |
| **Observation** | Validated `VisionEvidenceV1`, clip summaries, event envelope and camera/segment records. | Define one producer-neutral envelope for camera, audio and generic sensors without expanding the current privacy boundary. Audio/sensor producers do not exist yet. |
| **Entity** | Device contract, Resident view, local vision track metadata and topology nodes. | These represent different identities today. Define references and ownership without merging biometrics, residents, devices and transient tracks. |
| **State** | Cognitive snapshots and Universal Store journal; public snapshots; device/topology configuration; legacy `internal/state`. | State owner and projection must be explicit. Preserve Core sole-writer rule and keep public views redacted. Retire legacy store only after usage and serialized-data migration audit. |
| **Inference** | Python model output and `VisionEvidenceV1`; optional CPU MLP backend and decision-head result. | No single stable cross-backend inference result covers status, provenance, model qualification and uncertainty. Do not imply runtime presence equals a qualified model. |
| **Decision** | `pkg/contract.Decision` and Core `Decision`/Safety Gate records. | Preserve separation between inference score, policy gate and decision; reconcile naming and schema versions without changing behavior. |
| **Action** | `ActionRequest`, `ActionResult`, plan and dry-run boundary. | Add no physical action implementation in this milestone. Future commands need capability, permission, idempotency, feedback, timeout and failure states. |

## Duplications and risks found

- `pkg/contract` (singular) is the broad canonical-looking package;
  `pkg/contracts` (plural) contains a narrow connectivity state contract.
- `internal/connectivity` has persistence/config/state models beside that
  public state contract.
- `internal/cognitivecore` and legacy `internal/state` both contain state and
  persistence shapes; only the former is named as active Core authority in the
  architecture and current composition.
- Go contract validators and Python builders validate related Vision JSON.
  Golden cross-language fixtures are the practical compatibility safeguard.
- Discovery's own API and `cmd/synora-api` expose overlapping HTTP concepts but
  are different implementations/compositions.

No contract was changed during extraction. Migration order: produce a schema
catalog from actual serialized JSON; assign one canonical owner per concept;
add compatibility tests for Go↔Python and both HTTP compositions; add versioned
adapters; only then deprecate duplicate or legacy representations.

M1.2 did not change serialized contracts. Vision continues to publish the
existing `synora.vision.evidence/v1` and worker protocol. The new lifecycle and
network station-source seams are in-process Go interfaces; neither adds a wire
format. The architecture checker enforces that shared contract packages stay
free of OS packages and non-standard-library dependencies.
