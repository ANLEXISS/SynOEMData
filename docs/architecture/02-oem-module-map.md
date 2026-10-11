# OEM functional module map

The requested domains are functional ownership boundaries, not process
boundaries. One Go module and all executable entrypoints remain. M1.2 moved
Vision from under Discovery into `internal/vision`; remaining package ownership
is recorded in `modules/package-ownership.json`.

| Domain | Existing source components and demonstrated behavior | Inputs → outputs | State / direct dependencies | Classification and future separation |
| --- | --- | --- | --- | --- |
| **Foundation — Core Runtime** | `cmd/synora-core`, `internal/cognitivecore`, `internal/bus` | Versioned bus events → snapshots, decisions, stored commits and outcome events | Core owns Universal Store; bus owns only queues; Go contracts | SYNORA_PRODUCT. Keep Core and active Store together until a stable port is extracted. |
| **Foundation — Event Contracts** | `pkg/contract`, `pkg/contracts`, `internal/connectivity` models | Event/command/RPC JSON ↔ validated typed messages | Shared serialization types and version tags | SHARED_CONTRACT. Review duplicate connectivity state representations before canonicalization. |
| **Foundation — Internal Messaging** | `internal/bus`, `cmd/synora-bus` | Local services ↔ authenticated Unix-socket JSON messages | Bus transient connection state; HMAC/ACL config | SYNORA_PRODUCT + HOST_PLATFORM socket/UID semantics. |
| **Foundation — StateStore** | `internal/cognitivecore/store.go` | Core commits ↔ WAL, snapshot, replay/history | Core sole writer; JSON journal/segments and snapshot | SYNORA_PRODUCT. This is the active canonical business store. |
| **Foundation — Policy / Config / Persistence** | `internal/security`, `runtimeconfig`, `configfile`, `outbox`, `recovery`, `retention`, `resourcebudget`, `idgen` | Configuration, permissions and delivery outcomes ↔ bounded persistent/control behavior | Local filesystem and crypto; no external database | SYNORA_PRODUCT + HOST_PLATFORM paths. Extract stable interfaces only after port tests. |
| **Perception — Vision / Inference** | `services/vision-worker`, `internal/vision`, `internal/facedataset`, `facestore`, `facegallery`, `modelmanifest` | Media jobs and optional configured models → versioned, validated evidence/status | Worker owns transient inference; local face/dataset stores; RKNN runtime optional | Vision has an independent Go package boundary. Actual SoC/driver/camera adapter remains unimplemented. |
| **Perception — Media Acquisition** | `internal/discovery/ingress`, `clipstore`, `internal/mediamtx`, `camera/v0` mock | Authenticated clip upload/RTSP metadata → bounded local spool and job | Disk clips, MediaMTX external service; C++ mock uses an explicitly supplied fixture | Contracts are SHARED_CONTRACT; camera firmware and capture driver are OEM_IMPLEMENTATION/HOST_PLATFORM. C++ mock is a REFERENCE_ADAPTER only. |
| **Perception — Audio / sensors** | No dedicated audio or generic sensor processing implementation found. Some event/device contracts can represent related inputs. | Not implemented | None | PLACEHOLDER in documentation only; future OEM inputs must enter through SHARED_CONTRACT. |
| **Intelligence — Topology / tracking** | `internal/intelligence/topology`, portions of `cognitivecore`, Python `edge/tracker.py`, contracts | Configured topology and observations → validated topology/state and local tracks | Topology config; worker-local tracking; Core state | SYNORA_PRODUCT. A single canonical entity/track lifecycle across edge and Core is future separation; no cross-camera identity merge is claimed. |
| **Intelligence — Correlation / security analysis** | `cognitivecore`, `facegallery`, `security`, `foundationv1` test model | Validated evidence/events → bounded scores, policy decisions and store records | Core store and security policy | SYNORA_PRODUCT. `foundationv1` paths are synthetic only; MLP availability is bundle-dependent and fail-closed. |
| **Intelligence — Behavior learning** | Cognitive bundle loader/runtime and training/audit tools; no online learner in the service runtime | Offline bundle → optional inference results | Model files are external resources | PARTIAL. Model training/qualification is not an OEM module or active online learning loop. |
| **Automation — Engine / decision validation** | Core safety gate, `internal/discovery` action boundary, action contracts | Decision proposal → gate status and dry-run action result | Core action ledger/store; Discovery boundary and permissions | SYNORA_PRODUCT. Automation proposals exist as software, but the shipped mode is `active_dry_run`; no physical action execution. |
| **Automation — Device dispatch** | `internal/device`, `discovery/runtime`, manager/network host commands | Abstract device/service request → registry status, dry-run result or host operation | Config state and runtime manager; Linux service/network commands | Current host operations are HOST_PLATFORM. Physical device command adapters remain unimplemented; OEM_IMPLEMENTATION is expected for vendor APIs. |
| **Integration — REST / WebSocket / SSE** | `cmd/synora-api`, `internal/discovery` web/API handlers | Authenticated HTTP/WebSocket/SSE requests/events ↔ redacted state and bus events | Discovery/Core readers; security sessions; configured TLS | API behavior is SYNORA_PRODUCT; HTTP path/auth contract is SHARED_CONTRACT. Two server compositions need an explicit deployment decision. |
| **Integration — Device Registry / external protocols** | `internal/device`, `internal/discovery/runtime`, connectivity; MediaMTX client | YAML device records and runtime checks ↔ registry, connectivity state and external media API | Local config/state; Unix bus and HTTP | Registry/policies are SYNORA_PRODUCT; OEM device API is OEM_IMPLEMENTATION; MediaMTX client is REFERENCE_ADAPTER. MQTT is not evidenced as an active runtime integration. |
| **Platform — Host detection / runtime / supervision** | `bootstrap/`, `internal/manager`, `boothealth`, `runtimeconfig`, systemd units | Host facts and health probes → JSON/health reports and service actions | `/proc`, `/sys`, `/etc`, `/run`, `/var`, systemd | Generic host profile is PARTIAL SYNORA_PRODUCT; current supervisor and filesystem behavior are HOST_PLATFORM. Systemd is the only shipped supervisor implementation. |
| **Platform — Network / storage / acceleration** | `internal/discovery/network`, stores, RKNN worker adapters, model manifests | Host config/media/model status ↔ system service state and inference status | Linux commands, local disk, MediaMTX, RKNN libraries/device nodes | Current Linux adapter is HOST_PLATFORM/REFERENCE_ADAPTER; alternate OEM OS/network/storage/NPU implementations are future work. |
| **Application — Web UI** | `synora-web/src`, `cmd/synora-api` | User session and UI requests ↔ API/WebSocket state | Browser state; API server | Application is SYNORA_PRODUCT. Branding, identity provider and OEM configuration UX are shared extension points, not yet a white-label product. |

## Ownership rules

Synora supplies event semantics, Core/Store behavior, inference/evidence
semantics, policy boundaries and its current APIs. Manufacturers supply the
camera/device/sensor hardware, firmware, hardware identifiers, electrical
constraints, vendor control interfaces and product lifecycle. Shared contracts
are versioned and should not embed a vendor's device model. Reference adapters
remain optional examples. Linux drivers, kernel facilities, systemd and
accelerator runtimes are host-platform responsibilities.

No functional algorithm was duplicated during extraction. Cross-domain source
is mapped once to its primary responsibility; Discovery and Core are noted as
composition roots, not copied into multiple future modules.

M1.2 status and current dependency limits are in
[`11-module-boundaries.md`](11-module-boundaries.md) and
[`12-module-dependency-matrix.md`](12-module-dependency-matrix.md).
