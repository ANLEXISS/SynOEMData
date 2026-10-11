# Semantic Coverage Gap Matrix

This matrix is a code-based audit of the current checkout, not a compliance assessment. External standards are comparison points only.

| Target area | Existing Synora representation | Status | Material gap / evidence |
|---|---|---|---|
| Spatial relationships and geometry | Node/link topology; node/room IDs; resident room probability map; device `room_name` and `zone_role`; software-only `foundationv1.Zone` and `Topology` | PARTIALLY_REPRESENTED | Reference harness has zone category/coverage/provenance and transitions, but no production geometry, coordinate reference system, floor/containment predicates, polygon/line types, or canonical relation IDs. |
| Camera analytics and detection zones | Vision detection events/tracks and software-only topology labels; ONVIF not integrated | PARTIALLY_REPRESENTED | `foundationv1` is not production; no production configured zone geometry, exclusion region, privacy mask, line crossing rule or camera-to-zone association. |
| Temporal reasoning and event sequences | timestamps, clip intervals, sequence/activation/group keys, event-chain concepts, episode records | PARTIALLY_REPRESENTED | No first-class temporal window/ordering/gap/clock-quality contract shared across services. Arrival/out-of-order policy differs by consumer. |
| Identity, roles, authorization | resident roles owner/resident/child/guest/caregiver; visual identity states; Bus/API auth and ACL | PARTIALLY_REPRESENTED | Human identity/household role is separate from authenticated principal and scoped permission. No end-to-end actor -> requested capability -> policy decision -> grant provenance record. |
| Security and incident lifecycle | Security modes, danger assessment, safety gate, incidents new/viewed/acknowledged/resolved | PARTIALLY_REPRESENTED | Strong narrow intrusion flow, but no generic alarm incident taxonomy, escalation/assignment/suppression lifecycle, or explicit acknowledgement actor authorization record. |
| Device capability descriptions | `Device.Capabilities []string`; camera capabilities; cognitive capability facts; runtime model manifests; coarse software-only `foundationv1.CapabilityState` | PARTIALLY_REPRESENTED | The reference state is not production and has only name/state/mode; no production typed operations with schema/version, constraints, endpoint, availability, or supported/unsupported command-result semantics. |
| Automation rules and actions | action plan/policy decisions; abstract Action/Request/Result; deterministic safety gate; dry-run Dispatcher | PARTIALLY_REPRESENTED | No user-authored rule/trigger/condition/temporal expression lifecycle; no physical adapter; condition grammar not a module-level contract. |
| Environmental sensing | generic sensor.normal/anomaly; `SensorFacts`; device type sensor | PARTIALLY_REPRESENTED | No typed measurement quantity/unit, calibrated value, quality, sample time, source entity, threshold/evidence association. |
| Energy and power management | runtime disk/resource budget and device availability; no home energy domain | MISSING | No power source state, energy measurement, load/production/tariff/time-series or energy control semantic records. |
| Doors, locks and physical access | DeviceTypeLock; abstract action; security mode; unauthorized command safety is implied in Core logic | PARTIALLY_REPRESENTED | No typed lock/door state, latch/bolt/obstruction, access credential, actor grant, command authorization result, or physical execution receipt. |
| Audio and voice interactions | speaker device type and V3 communication intent/template fields | MISSING (Voice); PARTIALLY_REPRESENTED (communication) | No production Voice package, speech events, utterance/session, speaker identity, intent confidence, transcript privacy or TTS execution contract. V3 fields do not constitute Voice implementation. |
| Health and home assistance | fall/pose risk, face/resident gallery and caregiver role, health-related vision signals | PARTIALLY_REPRESENTED | Narrow camera-derived fall-risk path only; no user health measurement, care plan, emergency contact action lifecycle, or clinical provenance/consent contract. |
| Device diagnostics and availability | camera online/last-seen, runtime health, worker/network status, camera health v3 statuses | IMPLEMENTED for operations; PARTIALLY_REPRESENTED for semantic diagnostics | No common device health record with reason, observation time, freshness, fault code, capability degradation and recovery. |
| Uncertainty, evidence and provenance | confidence fields, known/unknown/uncertain identity, VisionEvidenceV1 with availability/support, source event refs, qualification provenance | PARTIALLY_REPRESENTED | Evidence/provenance is strongest in Vision; general events and decisions lack typed evidence references, calibrated confidence semantics, model/version provenance, explicit inferred-vs-observed marker. |

## Existing grammar classification

| Concept family | Classification | Rationale |
|---|---|---|
| Event taxonomy | PARTIALLY_REPRESENTED / DUPLICATED | Constants in `pkg/contract/event.go`, Python `ALLOWED_EVENT_TYPES`, versioned Vision contract names, and event-specific consumers are not driven from one registry. Some names/semantics overlap between coarse event and versioned records. |
| Entity and state | PARTIALLY_REPRESENTED / DUPLICATED | `contract.Device`, Core snapshots/incident records, and legacy `internal/state` states overlap in device/camera/node/track/presence. Legacy State is not active authority but remains compiled and tested. |
| Incident lifecycle | IMPLEMENTED (narrow) | Typed Go status and validation, timeline and identity classification; no broader alarm/response lifecycle or actor-permission linkage. |
| Camera/evidence | IMPLEMENTED (selected versions) | Camera observation canonicalization and versioned Vision Evidence schemas/tests exist. Geometry and analytics-rule association do not. |
| Action contract | IMPLEMENTED (abstract); PARTIALLY_REPRESENTED (execution) | Versioned compatibility JSON tests, typed request/result and dry-run. No physical executor or persisted authorization artifact. |
| Universal Store | IMPLEMENTED (current contract); DUPLICATED legacy schema | Foundation owns active Store machinery and `universal-store/v1`; legacy `internal/state` has `PersistedState` and distinct record shapes. Do not silently merge/migrate. |
| RPC/API/IPC | PARTIALLY_REPRESENTED / DUPLICATED | Go DTOs and Bus/Vision protocols are tested, but no repository-wide schema catalog/OpenAPI and multiple JSON envelope conventions exist. |
| Voice | MISSING | Architecture registry reserves the logical domain; no production package or module-specific tests. |

## Highest-value gaps

1. A compact typed observation envelope with separate observed facts and inference assertions, stable source/entity references, event/ingest times, schema version, evidence, and unknown-field/value policy. Do not replace existing envelopes during this audit.
2. Typed device capability declarations and capability-specific command results, while preserving existing string lists as compatibility projections.
3. A coordinate-aware spatial primitive set for camera views and physical topology, with explicit frame and camera/preset association.
4. Distinct authorization decision/receipt and execution result records for access and other high-impact commands.
5. Deterministic cross-module test fakes that exercise existing contracts without asserting absent zone, voice, or hardware behavior.
