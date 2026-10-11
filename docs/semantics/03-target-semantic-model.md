# Target Semantic Model (Proposal Only)

No wire, persistence, Go, Python, IPC, API, or OEM contract is changed by this audit. The following is a direction for additive follow-up, not a canonical schema implementation.

## Semantic layers

| Record kind | Meaning | Required distinctions |
|---|---|---|
| ENTITY | Stable referenced thing: device, endpoint, room, resident/person, zone, sensor/actuator | Stable ID, entity kind, display/config data separate from observations, optional parent/topology refs |
| OBSERVATION | Source-reported measurement or detection at a time | Source identity, event time, optional ingestion time, subject/entity refs, typed value/unit/frame, quality, provenance, schema version |
| RELATION | Typed link between entities or an entity and a place | Relation kind, subject/object, validity interval, source/provenance; do not encode geometry as arbitrary metadata |
| STATE | Current or derived value with explicit validity/freshness | State owner, value/status, `valid_from`, `observed_at`, optional expiry, source observation refs |
| EVENT | Immutable occurrence or lifecycle transition | Event ID/type/source, event and ingest times, correlation/causation IDs, entity refs, typed data/version |
| CONDITION | Predicate over typed state/events/time/capabilities | Operand types, operator, missing/unknown behavior, time basis, stable version; bounded composition |
| INTENT | Requested outcome before policy authorization | Initiator/principal, target, desired action, reason/context, correlation, expiry; not proof of permission |
| COMMAND | Authorized, concrete request to an adapter/device | Capability/operation, target endpoint, arguments, authorization reference, idempotency and deadline |
| EXECUTION_RESULT | Adapter/device outcome | Accepted/started/finished times, outcome enum including unsupported/unavailable/denied/timeout/unknown, error details, actual state/evidence refs |

## Workflow boundaries

```text
OBSERVATION -> (optional) INFERENCE -> DECISION -> AUTHORIZATION -> COMMAND -> EXECUTION_RESULT
```

- An observation records what a source reported. An inference records a computed claim and references evidence, model/algorithm revision, and uncertainty. A sensor value can itself be an observation; a camera classification is an inference unless the source explicitly reports a device-generated classification.
- A decision selects a policy outcome and explains inputs. It does not grant authority.
- Authorization evaluates actor, target, operation, policy, context, and freshness; returns an explicit allowed/denied/indeterminate result with reason and policy version.
- A command is sent only after authorization. The execution result reports adapter/device evidence; command acceptance is not state confirmation.
- Unsupported capability, unknown enum, missing field, stale value, and unresolved entity reference remain explicit states. They must not silently coerce to false, zero, or a broad event payload.

## Stable core primitives

Use a small set of versioned primitives rather than enumerating every device property:

- `Identifier { namespace, value }` and typed `EntityRef { id, kind? }`.
- `Timestamp { value, clock/source quality? }` with separate event-time and ingest-time semantics; UTC RFC3339/JSON time representation at Go boundaries unless an existing compatibility contract dictates otherwise.
- `Measurement { quantity, value, unit, quality, observed_at }`; no untyped numeric sensor without unit.
- `Confidence { value, meaning, calibration/provenance? }`; avoid assuming all current float confidences share calibration.
- `EvidenceRef { kind, id, digest?, retention class? }`; do not move raw media/biometrics into Core records.
- `SchemaRef { name, version }`; additive field evolution and explicit version policy for persisted/wire records.
- `Extension { namespace, version, typed data }` for genuine vendor/domain-specific additions. Validate and preserve unknown extension data only where safe; unknown commands remain non-executable.
- `CapabilityDescriptor { id, version, operations, constraints, availability }`, with endpoint and support state. Existing `[]string` values remain legacy projections during transition.
- `CoordinateFrame { frame_id, kind, units, origin/orientation, dimensions }` and geometry values that cite a frame. Never compare pixel, normalized-image, PTZ, and floor coordinates without a declared transform.

Do not create an enum for every device feature. Keep stable lifecycle/result/status enums limited to cross-vendor semantics; represent capabilities through versioned descriptors and typed extensions. Unknown values should survive deserialization where safe, but policy/dispatch must fail closed on unknown high-impact operations.

## Mapping existing records without breaking them

| Proposed layer | Existing sources | Additive bridge idea (future milestone) |
|---|---|---|
| ENTITY | `contract.Device`, `CameraObservation`, resident records, topology Nodes, synthetic `foundationv1.Peripheral`/`Zone` | Export canonical IDs and entity kind from existing production IDs; keep reference-harness records clearly marked as nonproduction; preserve current JSON field names. |
| OBSERVATION | `contract.Event`, `CameraObservation`, VisionEvidenceV1, sensor events | Add new typed versioned record alongside current event envelope; adapter reads existing JSON and maps known events only. |
| RELATION | topology links, device node/room fields, resident baseline room map, synthetic zone transitions | Represent spatial relations separately while leaving source config shape intact; do not treat reference transitions as camera geometry. |
| STATE | `CognitiveSnapshot*`, incident, runtime/camera health; Foundation Store | Keep authoritative Store serialization unchanged until migration and dual-read/write plan is tested. |
| CONDITION | currently scattered Core gates and scenario fixture expectations | Start with internal typed test scenario predicates; do not claim these are a user automation rules engine. |
| INTENT / AUTHORIZATION | `Decision`, `ActionRequest`, Core safety gate, bus/API auth | Add a separate authorization result with actor/policy/context references before physical dispatch exists. |
| COMMAND / RESULT | `ActionRequest`, `ActionResult`, Automation dry-run Dispatcher | Preserve request/result fields; differentiate simulated result from adapter/device-confirmed execution. |

## Compatibility policy

The audit found Go JSON structs, Python-produced JSON, three selected JSON Schemas, Bus/Vision IPC formats, and existing persisted formats, but no single repository-wide schema registry. Before adding typed schema fields, capture golden fixtures for current Go, Python, API, IPC, and `universal-store/v1` records. Add a new schema name/version; do not repurpose fields or reinterpret existing strings. Unknown fields should follow current per-contract behavior, which is not uniform: Go JSON decoding commonly ignores unknown fields, while validators may reject selected unknown values. Document each boundary's behavior. Plan persistence migration separately, with backup, old-reader/new-writer compatibility, rollback and restart-recovery tests.
