# Existing Semantic Grammar Inventory

**Audit date:** 2026-10-11  
**Scope:** contracts and production source in the SynoraOEM checkout at this revision. Status labels describe implemented code, not the maturity claims in milestone reports. This is an inventory of externally meaningful grammar and state families; it is not a generated listing of every Go struct used as an implementation detail.

## Ownership and formats

| Semantic area | Current owner / defining code | Wire or persisted form | Producers and consumers |
|---|---|---|---|
| Shared events, entities, actions, incidents, residents, topology | Foundation-owned `pkg/contract` (`event.go`, `device.go`, `camera.go`, `action*.go`, `incident.go`, `resident.go`, `topology.go`, `vision*.go`) | Go structs serialized as JSON; selected config structs also YAML | Discovery/API and Vision emit; Bus transports; Cognitive Core consumes/projects; API and UI read snapshots |
| Local IPC | Foundation `internal/bus` (`types.go`, `server.go`, `client.go`) | newline-delimited JSON `Message` over Unix domain socket; JSON body may encode event/RPC | Service processes; socket auth/ACL in Bus |
| Vision worker IPC | Vision `internal/vision/protocol.go`, `runtime.go`, Python `services/vision-worker/worker.py` | newline-delimited JSON over Unix socket; versioned hello and operation names | Go runtime sends clip/segment/face requests; Python worker responds and publishes event payloads |
| Camera/evidence/segment contracts | Foundation `pkg/contract/camera.go`, `vision_evidence_v1.go`, `vision_segment_ready.go`, clip summary contract | versioned JSON; JSON Schemas exist for evidence, suite manifest, and module registry | Discovery produces camera observations; Python/Go Vision produces evidence and segment events; Core maps them to snapshots/decisions |
| Cognitive records and projections | Intelligence-owned `internal/cognitivecore` (`snapshot*.go`, `decision*.go`, `store.go`, `vision_evidence.go`) | typed JSON records, versions V1/V2/V3; Foundation StateStore persists generic typed records | Core creates snapshot, decision and action records from input event and existing state |
| Authoritative persistence | Foundation `internal/foundation/statestore`; reusable filesystem mechanics in `internal/foundation/persistence`; domain record definitions remain in Cognitive Core | `universal-store/v1` materialized JSON plus newline-delimited JSON WAL, immutable segments, revisioned outbox | Core is the active writer; StateStore serializes/replays; event and action history readers project state |
| Legacy persistence | Foundation-owned legacy `internal/state` | versioned JSON `PersistedState` / FilePersistence | Still has broad typed state and tests, but architecture reports say it is not the active Core writer; avoid treating it as canonical live state |
| Synthetic reference contracts | `internal/foundationv1/contracts.go` | Software-only JSON structs; generic StorePort | Reference harness models statuses, camera health, peripherals, coarse capability availability, zone coverage/provenance, topology transitions, search and dry-run communication/action. Package comment says these are not connected to production peripherals or qualified inputs; do not count them as production semantic support |
| Device, topology, and residents configuration | Integration `internal/device`, Intelligence `internal/intelligence/topology`, `configs/devices.yaml`, `topology.yaml`, `residents.yaml` | YAML config into Go structs; device config is projected into JSON `contract.Device` | Discovery loads configuration; Core consumes topology/resident facts; API edits selected records |
| Python observation pipeline | Vision `services/vision-worker/core/{clip_pipeline_v1.py,events.py,enrichment_v*.py,edge/*}` | Python dataclasses/enums and JSON event payloads | Detector/tracker/enrichers produce candidate observations; EventBuilder emits allowed event names; Discovery/Core consume |
| External API | Integration `internal/discovery/{api_server,web_v1,ingress}` and `cmd/synora-api` | JSON HTTP/WebSocket DTOs; no OpenAPI source found in checkout | UI and integrations call authenticated API; handlers translate to internal events/RPC |

Logical domains are registered in `modules/manifest.json`. The executable remains one Go module (`module synora`). The actual source package map is `modules/package-ownership.json`; notable split ownership under `internal/discovery` is enforced by nonrecursive Integration ownership plus explicit Platform network ownership.

## Event catalog

`pkg/contract/event.go` defines the canonical Go string constants and the envelope. Families currently include:

| Family | Existing event names / shape | Status and notable limitations |
|---|---|---|
| Vision detections | `vision.identity`, `.end`, `.unknown`, `.uncertain`, `.motion`, `.weapon`, `.fall`, `.fight`, `.tamper` | IMPLEMENTED as event names and consumers; observations are coarse event types with largely open payloads, not a generic typed detection record |
| Vision contracts | `synora.vision.clip-summary/v1`, `preliminary-alert/v1`, `clip-observation/v1`, `segment-ready/v1`, `segment-gap/v1`, `continuity-reset/v1`, `enrichment/v3` | IMPLEMENTED versioned subsets; not a unified Observation schema |
| Device/discovery | `device.trigger`, `device.offline`, `discovery.camera.observed/online/offline`, worker started/stopped/crashed/unavailable, network degraded, ingress/runtime status | IMPLEMENTED for camera/service/network lifecycle; not general device health/capability-change events |
| Clips/runtime | `clip.ready/processing/processed/failed`, runtime flapping/model missing | IMPLEMENTED operational events |
| System/security | `system.state.changed`, `system.presence.updated`, `system.unknown`, `security.mode.changed`, `system.state.reset`, `manual.risk` | IMPLEMENTED product/system summaries; access-control authorization lifecycle is not modeled as a unified event family |
| Actions | `action.request`, `action.result`, `action.service.started` | IMPLEMENTED abstract request/result lifecycle. Dispatch is controlled dry-run; no physical actuator implementation |
| Validation/sensors/web | `validation.test-inference`, `sensor.normal`, `sensor.anomaly`, `discovery.web.command` | IMPLEMENTED as bounded event names. Sensor contracts remain generic and lack units/ranges/typed measurement metadata |

Envelope fields include `id`, `type`, `source`, `timestamp`, open `payload map[string]any`, device/node IDs, epoch/sequence/received-at, identity/resident/confidence, priority, grouping/track/clip/activation/sequence identifiers, and validation flags. Timestamps are `time.Time` JSON values. `received_at` exists but producer/consumer use is not uniform. Unknown event type values remain possible because `Type` is a string. Some input paths validate an allow-list; others consume named constants or versioned payloads.

## Entity, state, relation, and time concepts

| Concept | Existing representation | Classification |
|---|---|---|
| Device | `contract.Device`: string type/status/role/zone role, room/node, booleans enabled/trusted, string capability list, open config/metadata; synthetic `foundationv1.CapabilityState` has coarse name/state/mode | PARTIALLY_REPRESENTED; no production typed capability descriptors, endpoint/function distinction, health freshness or capability revision |
| Camera | `CameraObservation`: version, stable IDs, endpoint/firmware, string capabilities, online, `LastSeen`; camera config/status elsewhere | IMPLEMENTED for discovery identity and reachability; PARTIALLY_REPRESENTED for analytics configuration and PTZ coordinate/preset association |
| Node/room/topology | `contract.Topology`, `Node`, links; Intelligence topology loader; rooms in resident baselines and device config | PARTIALLY_REPRESENTED; spatial graph is node/link-level and has no typed geometry, containment semantics, floors/coordinates or directional relations |
| Synthetic zones | `foundationv1.Zone` has ID, public/protected/service/unknown type, coverage, uncertainty, provenance, expiry and device IDs; `foundationv1.Topology` has zone transitions | PARTIALLY_REPRESENTED in software-only reference harness; not production topology or camera-zone association, and it has no geometry/frame/rules |
| Resident/person identity | `contract.Resident`, identity profile, gallery consent/status, incident identity kind, event identity/resident IDs; Python face/identity status | IMPLEMENTED for resident gallery and known/unknown/uncertain visual identity; authorization principal, role grants, credential, visitor/household membership lifecycle are separate or absent |
| Tracks/presence | Python track and episode records; legacy `internal/state` tracks/presence; cognitive snapshots contain resident presence/episode facts | PARTIALLY_REPRESENTED; active track state is primarily transient/evidence-oriented, with no general entity-state relation graph |
| Sensor state | cognitive snapshot `SensorFacts`; `sensor.normal/anomaly` events | PARTIALLY_REPRESENTED; lacks typed measurement kind, unit, scale, calibration, sampling/quality and sensor device reference as a required contract |
| Security state | security mode string, danger levels, incident status, identity kind, decision safety fields | IMPLEMENTED for narrow Synora intrusion and incident flow; PARTIALLY_REPRESENTED for general security zone, alarm source, escalation and incident closure semantics |
| Time | event timestamp, receive timestamp, created/updated/started/last-event timestamps, clip frame/segment times, temporal IDs | PARTIALLY_REPRESENTED; event-time and ingest-time are not mandatory or uniformly interpreted; sequence IDs do not encode ordering/gap policy themselves |
| Relationships | node links, resident-to-room baseline weights, event/decision/action IDs, camera/node IDs, incident references | PARTIALLY_REPRESENTED; relationships are embedded scalar references or config edges, not typed first-class versioned relations |

## Decision, action, persistence, and IPC

`contract.Decision` captures score/effective score, reason, state/node, danger source/level, clip/track/group/sequence references, graph use, validation flags, policy action recommendations, blocked actions, final action plan and decision reason. Cognitive Core also owns richer V1/V2/V3 decision and snapshot projections with schema versions, confidences and vision evidence projection. Decision is distinct in code from an `ActionRequest`, but authorization has no independently versioned grant/authorization record: Core's deterministic safety gate and policy assessment are implementation-local decision logic.

`ActionRequest`/`ActionResult` carry identifiers, source event and decision refs, target/device/node, data maps, timestamps, timeout/retry/idempotency/correlation metadata, status/error and duration. `internal/automation/action.go` validates abstract requests and returns controlled dry-run results. `ActionPlanItem` and `PolicyActionDecision` are decision-side plan/policy records, not a general rule language. No physical command completion claim follows from a dry-run result.

The separate `internal/foundationv1` software-only harness includes illustrative contracts (`Peripheral`, `CapabilityState`, `Zone`, `Topology`, `ActionProposal`, `CommunicationIntent`, `SearchState`, `Correlation`) and conservative statuses such as `simulated_test`, `dry_run`, `unavailable`, and `unknown`. It is explicitly not connected to production peripherals or qualified inputs. These types show useful prototype concepts but do not close production grammar gaps or establish persistence/API compatibility.

Incident records have lifecycle `new`, `viewed`, `acknowledged`, `resolved`; typed identity kind resident/unknown/uncertain/none; cause, timeline event refs, camera/node/person refs, scores and event/clip refs. These are substantial security concepts, but a single `SecurityState` remains a string and statuses do not define a general escalation/suppression/false-positive lifecycle.

Foundation StateStore owns commits, revision assignment, deduplication, WAL/replay, compaction and outbox consistency. Its disk wrapper has `schema_version`, snapshot/decision V1/V3, action request/result, event history, sequence and resident gallery records. It is JSON and journal line oriented. Legacy `internal/state` has overlapping DeviceState, CameraState, NodeState, Track, ResidentTrack, EntityTrack, PresenceState and SystemState types and a separate `PersistedState` version. Treat the overlap as DUPLICATED/legacy, not two active authorities.

`internal/bus.Message` is the generic JSON IPC envelope, with type/source/payload and request/correlation-oriented fields. Bus uses Unix sockets, bounded message queues, and auth/ACL. RPC contracts live in `pkg/contract/rpc.go`. Vision worker protocol has its own hello version and operation strings (clip, segment, episode release, enrichment/face operations). Those protocol families are not represented as one schema registry. No Protocol Buffers, Avro, or OpenAPI spec files were found; JSON Schema is present only for selected Vision test/registry contracts and bootstrap host profile.

## Producer / consumer trace and serialization

| Producer | Emitted concept | Consumer | Encoding / validation boundary |
|---|---|---|---|
| Vision Python pipeline | detection/enrichment/clip observations | Go worker/runtime, Discovery/Core | Python mapping/dataclass -> JSON event; event allow-list and selected versioned schema checks |
| Discovery camera poller | `CameraObservation` and lifecycle events | Core, API | canonical Go JSON; `EnsureID`, schema version and identifier validation |
| Core | snapshots, decisions, incidents, action requests/results, state-change events | StateStore, Bus, API/UI | Go typed JSON; persisted by Foundation Store; selected versioned projections |
| API/UI / clients | RPC and web commands | Discovery/Core | JSON over local Bus and HTTP/WebSocket; handler-specific validation and auth |
| Core outbox | pending abstract action request | Automation dispatcher | persisted typed JSON outbox -> dry-run request/result; no hardware dispatch |

## Status vocabulary applied in this audit

IMPLEMENTED = typed or explicitly handled code exists and has direct tests. PARTIALLY_REPRESENTED = some fields or narrow workflow exist, but target concept is not carried end-to-end. MISSING = no production contract/behavior found. DEPRECATED = retained compatibility/legacy path not used as active owner. DUPLICATED = overlapping representations have distinct owners/formats. INCOMPATIBLE = same concept has conflicting shape/meaning across active boundaries. A concept can be duplicated and legacy simultaneously; the matrix records the dominant condition and explains overlap.
