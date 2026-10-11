# Existing Synora architecture inventory

## Scope and evidence

Source of truth: `/home/rock/Synora-v1`, branch `master`, HEAD
`effe3eeed01ce86045b3b70b4110f0b976ff5939`. At inventory time the source had
one tracked modification (`Makefile`) and 52 non-ignored untracked files in
`bootstrap/` and `camera/`; these are preserved in this copy. There were 537
tracked files and 52 ordinary untracked files. The extracted tree contains 589
source/config/test files plus the OEM architecture documents.

The repository is one Go module (`module synora`, Go 1.24), 47 Go packages, 17
command directories, a Python 3 vision worker, a React/TypeScript/Vite UI, and
a C++20 camera prototype/mock. There is no Python virtual environment or
second Go module. The Python dependency lock is a pinned
`services/vision-worker/requirements.txt`; the web uses `package.json` and
`package-lock.json`. The Makefile is the primary product/build/deployment
entrypoint, CMake builds the camera mock, and `unittest` runs the Python tests.

The working tree also contained 1.2 GB of `build/` artifacts, 62 MB of
compiled `bin/` executables, 128 MB of `synora-web/node_modules`, Python
bytecode, model weights, and generated web output. Those are not source and
were excluded, except the two small JSON-only V1/V3 dry-run model bundles
(496 KB combined) described in the extraction map. The 3 MB tracked `testdata/` remains because
it is contract and software-test data, not a recorded video dataset. No credential files,
certificates, model weights, private keys, `.env` files, or recordings were
copied. Safe config templates and placeholder-only versioned config examples
remain.

## Executables and entrypoints

| Source path | Responsibility and inputs/outputs | Runtime / direct dependencies | Maturity and OEM destination |
| --- | --- | --- | --- |
| `cmd/synora-bus` | Starts the local message broker; receives service connections and versioned JSON messages over a Unix socket. | Go; `internal/bus`, runtime config; Linux Unix sockets and filesystem permissions. Owns no business state. | IMPLEMENTED software service; `modules/foundation/messaging`. |
| `cmd/synora-core` | Consumes bus events, builds cognitive snapshots, calls the configured MLP backend, applies the safety gate, commits decisions to Universal Store, and publishes outcomes. | Go; bus, cognitivecore, runtime config; persistent writable data path. MLP bundle optional and fail-closed when absent. | IMPLEMENTED software path, PARTIAL product capability because no promoted live bundle/action; `modules/foundation/core` + `modules/intelligence/cognition`. Keep implementation together initially. |
| `cmd/synora-discovery` | Owns external HTTP/HTTPS API, SSE, vision ingress/worker orchestration, device/runtime registry, and event translation to Core. | Go; bus, network, device, vision, security, MediaMTX and face stores; Linux networking commands and root in shipped unit. | IMPLEMENTED software boundary with hardware actions dry-run; `modules/integration/api` + `modules/perception/media` + `modules/platform/runtime`. Mixed responsibilities documented in M2 roadmap. |
| `cmd/synora-runtime-manager` | Handles authenticated bus RPC/command requests and invokes host-service management operations. | Go; bus, `internal/manager`, runtime config; systemd/host commands and root privileges in the unit. | PARTIAL: code exists, host control is Linux-specific; `modules/platform/runtime`. |
| `cmd/synora-connect` | Local connectivity identity/state agent; `run`, `status`, `explain` operations; bus IPC and persisted local identity. | Go; connectivity, config, bus; Linux filesystem and Unix socket. No remote enrollment is implemented. | PARTIAL foundation; `modules/integration/protocols` and `modules/platform/network`. |
| `cmd/synora-api` | Standalone authenticated REST/WebSocket and static-web server. | Go; security/session, systemstate, websocket; HTTP(S), TLS files, static web root. Not present in the shipped systemd unit list; Discovery also hosts a separate API. | IMPLEMENTED standalone entrypoint, deployment relationship is UNKNOWN/UNUSED by shipped units; `modules/integration/api` + `modules/application`. |
| `cmd/synora-network-config` | Operator CLI to inspect/validate/update SynoraNet configuration. | Go; discovery/network; YAML and writable config/backup paths. | IMPLEMENTED CLI; `modules/platform/network`. |
| `cmd/synora-bootstrap-config` | First-boot account/token/session secret/config initialization. | Go; security, bcrypt, YAML; local filesystem and secret generation. | IMPLEMENTED provisioning tool; `modules/platform/runtime`. |
| `cmd/synora-boot-healthcheck` | Read-only boot checks of image identity, mounts, services and API endpoints; emits a report. | Go; boothealth, network, manifests; HTTP to local endpoint. | PARTIAL, host checks only; `modules/platform/runtime`. |
| `cmd/synora-token-rotate` | Explicit operator token-rotation CLI. | Go; security config writer. | IMPLEMENTED maintenance tool; `modules/platform/runtime`. |
| `cmd/synora-version` | Generates/reads version manifest; uses Git metadata and build environment. | Go; version package; writes the requested output when run. | IMPLEMENTED build tool; `modules/platform/runtime`. |
| `cmd/synora-camera-qualification` | Software qualification against configured device/MediaMTX and API evidence. | Go; device, MediaMTX, security; network/API and media endpoint. | PARTIAL; qualification is evidence-gated and not proof of a physical camera unless supplied; `tests`/`modules/perception/media`. |
| `cmd/synora-central-test` | Large hermetic/synthetic integration harness, with opt-in media paths. | Go; most active domains, temporary sockets/files/HTTP test servers. | IMPLEMENTED test harness, not a production process; `tests`. |
| `cmd/synora-cognitive-runtime-bench` | Measures local bundle loading, Core/store and dry-run dispatch performance. | Go; cognitivecore, Discovery dry-run; local MLP bundle. | IMPLEMENTED diagnostic, not runtime; `tests`. |
| `cmd/synora-v1-replay`, `cmd/synora-cognitive-v2-e2e`, `cmd/synora-inject-edge-clip` | Legacy/replay/injection/test entrypoints. | Go; versioned fixtures, test services. | PARTIAL/UNUSED by current shipped runtime; retain as traceable source pending deprecation decision; `tests`. |

## Go package inventory

All Go components below are in the same `synora` module. The `internal/`
boundary is compile-time-private to this module; `pkg/contract` is the primary
shared public contract package.

| Package | Responsibility, data ownership and external boundary | Direct internal dependency / OS / hardware notes | Maturity; proposed domain |
| --- | --- | --- | --- |
| `internal/bus` | Unix-domain JSON bus, client/server lifecycle, ACL, HMAC authentication and replay filtering. Bus owns transient connection/message queues, not business state. | `resourcebudget`, `pkg/contract`; Linux Unix socket, UID/GID and filesystem permissions. | IMPLEMENTED; foundation/messaging. |
| `internal/cognitivecore` | Core event/snapshot pipeline, V1/V3 snapshots, MLP port, safety gate, action ledger and Universal Store WAL/snapshot/replay. Core is sole active business-store writer. | `pkg/contract`; JSON files, append-only journal, fsync/rename. | IMPLEMENTED software / PARTIAL live cognition; foundation/core + intelligence/cognition. |
| `internal/runtimeconfig` | Environment-driven paths, endpoints and timeout defaults shared by services. | Standard library; many Debian `/etc`, `/run`, `/var/lib` defaults. | IMPLEMENTED; foundation/configuration. |
| `internal/configfile` | Safe config file replacement/backup primitives. | Filesystem rename and permissions. | IMPLEMENTED; foundation/configuration. |
| `internal/resourcebudget` | Shared fixed queue, payload and retention limits. | No external service. | IMPLEMENTED; foundation/policy. |
| `internal/idgen` | Process-local IDs/time helpers. | Clock and atomic counter. | IMPLEMENTED; foundation. |
| `internal/outbox` | Durable JSON delivery outbox with retry/ack identity. | `pkg/contract`; local filesystem, single-file persistence. | IMPLEMENTED; foundation/persistence. |
| `internal/recovery` | State-machine/replay support for restored state. | No external service. | IMPLEMENTED; foundation/persistence. |
| `internal/retention` | File retention/removal policy primitives. | Linux filesystem/syscalls. | IMPLEMENTED; foundation/persistence. |
| `internal/state` | Older event/state store, projections and persistence. Explicitly marked legacy and not the active Core store. | `clipstore`, `configfile`, `idgen`, `recovery`, `pkg/contract`; local JSON files. | IMPLEMENTED legacy, UNUSED by current active Core path; keep isolated until separate deletion/migration review. |
| `internal/foundationv1` | Newer abstract topology/device/action/communication/search contracts and a synthetic E2E harness. It does not drive a production device. | `cognitivecore`, `discovery`, `security`, `pkg/contract`; test HTTP/server and temp files. | PARTIAL/reference harness; target boundaries across foundation/intelligence/automation. |
| `internal/intelligence/topology` | YAML topology configuration load/validation and resident/topology semantics. | `configfile`, `pkg/contract`; local YAML config. | IMPLEMENTED configuration semantics, no sensor-derived localization; intelligence/topology. |
| `internal/device` | Configured device registry and device contract validation; no vendor driver. | `configfile`, `pkg/contract`; local YAML. | IMPLEMENTED registry, hardware availability remains PARTIAL; integration/devices. |
| `internal/discovery` | Production composition root: external API/ingress, worker and runtime health, registries, snapshots, event validation/translation, media and device orchestration. | `bus`, clipstore, device, discovery subpackages, face stores, MediaMTX, runtimeconfig/security, `pkg/contract`; Linux listeners/process spawning, in shipped unit root. | IMPLEMENTED software boundary, PARTIAL real hardware; integration/api + perception adapters + platform/runtime. |
| `internal/discovery/ingress` | Authenticated bounded clip/media ingestion and retention hand-off. | clipstore, vision, resourcebudget, retention, `pkg/contract`; HTTP/TLS, local disk. | IMPLEMENTED software; perception/media. |
| `internal/vision` | Worker process manager, Unix socket RPC, clip jobs, retries, health and validated vision evidence publishing. | facedataset, idgen, resourcebudget, runtimeconfig, contract; Python subprocess and Unix socket. | IMPLEMENTED software integration; independent package composed by Discovery. |
| `internal/discovery/runtime` | Device/runtime registry and observed health transitions. | idgen, contract. | IMPLEMENTED software registry; integration/devices. |
| `internal/discovery/network` | SynoraNet host networking lifecycle, AP/firewall/DHCP/DNS and status. Shells out to host utilities. | `pkg/contract`; injected station source; `hostapd`, `dnsmasq`, firewall tools, Linux bridge/nftables/iptables, root. | PARTIAL and highly host-specific; platform/network. |
| `internal/connectivity` | Local identity, access allowlist, rendezvous/tunnel metadata, policy and state. Remote enrollment remains explicitly unimplemented. | bus, configfile, contract(s), crypto; local files. | PARTIAL; integration/protocols. |
| `internal/security` | API/device auth, accounts, pairing, sessions, secrets config, redaction and support bundle. | configfile, YAML, Go crypto; secret files and OS permissions. | IMPLEMENTED software controls; foundation/policy and integration/security. |
| `internal/systemstate` | HTTP state projection/handler types. | Standard library HTTP/JSON. | IMPLEMENTED support layer; integration/api. |
| `internal/manager` | Runtime manager: status collection, service command execution, snapshots/rollback and network config operations. | network config, runtimeconfig, contract; Linux/systemd utilities, root. | PARTIAL; platform/runtime. |
| `internal/boothealth` | Healthcheck report model/aggregation. | `pkg/contract`/standard library. | IMPLEMENTED check library; platform/runtime. |
| `internal/version` | Version manifest model and load validation. | Filesystem/runtime metadata. | IMPLEMENTED; platform/runtime. |
| `internal/modelmanifest` | External model manifest validation. | YAML and filesystem. | IMPLEMENTED manifest checks; perception/inference. |
| `internal/clipstore` | Bounded clip path/storage helpers and integrity metadata. | Filesystem, hashes. | IMPLEMENTED storage helper; perception/media. |
| `internal/facedataset` | Dataset validation, staged promotion and current pointer management. | facestore, contract; files and hashes. | IMPLEMENTED storage lifecycle, not proof of model qualification; perception/vision. |
| `internal/facestore` | Local image/gallery storage, image decode and content integrity. | idgen, contract; disk, image formats. | IMPLEMENTED persistence; perception/vision. |
| `internal/facegallery` | Consent and policy/evidence validation for resident face gallery. | contract; local metadata. | PARTIAL; perception/vision + integration/privacy. |
| `internal/mediamtx` | MediaMTX API client/config utilities and path management. | YAML/HTTP; separate MediaMTX process. | IMPLEMENTED adapter, external server remains dependency; perception/media. |
| `internal/visionsuite` | Vision suite manifests, evidence and qualification checks used by tests/tools. | contract; local fixtures/media paths. | IMPLEMENTED test support, not inference service; tests/perception. |
| `internal/cognitive` | Legacy/empty package directory; no package appears in Go's active package list. | None observed. | UNUSED/UNKNOWN; no source copied for this empty directory. |
| `pkg/contract` | Versioned event, message, decision/action, device, resident, topology, delivery, runtime and vision contracts/validators. | Standard library only. | IMPLEMENTED; shared contract. |
| `pkg/contracts` | Connectivity-state contract package currently containing a narrow state type. | Standard library. | PARTIAL; shared contract; naming and canonical ownership need consolidation review. |

The Go imports remain acyclic as accepted by the current `go list ./...`;
`internal/discovery` is a high-fan-in composition root. This inventory does not
claim every code path has been hardware-qualified.

## Other source components

| Source | Responsibility / inputs and outputs | Dependencies and state | Maturity; target |
| --- | --- | --- | --- |
| `services/vision-worker/worker.py`, `core/`, `edge/`, `modules/` | Python clip/frame inference, postprocessing, edge tracking and evidence contracts; reads local media/model and returns JSON messages on a Unix socket. | Python 3, OpenCV, NumPy, SciPy, RKNN Lite on ARM64, Rockchip runtime/driver/device nodes; model and face data under `/var/lib/synora`. | PARTIAL: detector/model pipelines exist; selected face/runtime capabilities explicitly fail closed or remain unqualified. perception/vision. |
| `services/vision-worker/tests/` | Isolated Python unit/contract tests. | unittest, temporary files/socket pairs; no production service. | IMPLEMENTED test suite; tests. |
| `synora-web/src/` | React UI for authentication, system/intelligence views and configuration surfaces. | Node, TypeScript, React, Vite; talks to standalone API endpoints/WebSocket. | PARTIAL application; application. |
| `camera/v0/` (pre-existing untracked) | C++20 camera daemon abstraction and deterministic simulated evidence producer; central-ingress mock uploads a supplied fixture after auth. | CMake, C++20, ffprobe/curl/openssl for mock upload; no production hardware adapter. | PARTIAL reference simulator; perception/media/reference-adapter. |
| `bootstrap/` (pre-existing untracked) | POSIX-shell host/environment discovery profile and tests, implemented in the preceding bootstrap task. | Minimal POSIX tools; read-only host evidence and optional report output. | PARTIAL first-cut platform discovery; platform/bootstrap. |
| `tools/` | Dataset/manifest generation and audit/qualification runners, some with network download actions when explicitly invoked. | Python, local model/data assets, some external toolkits. | MIXED: implemented operator tooling, not production runtime; tests/tooling. |
| `configs/` | YAML configuration and safe templates for auth, security, network, devices, residents, topology, models and vision. | Local filesystem; secret file paths are external. | IMPLEMENTED schemas/default samples; platform/configuration and shared contracts. |
| `deployments/systemd/` | Six systemd units: bus, core, discovery, runtime manager, connect, MediaMTX. | Debian/Linux + systemd; hardcoded `/opt/synora`, `/etc/synora`, `/var/lib/synora`, `/run/synora`, service users. | IMPLEMENTED current packaging, not portable OEM packaging; platform/supervisors. |
| `deployments/mosquitto/` | Broker config only. No non-test source import of the declared Paho MQTT Go dependency was found. | Mosquitto if enabled externally. | UNUSED/UNKNOWN active integration; do not treat MQTT as an implemented Synora protocol. |
| `testdata/`, Go tests, C++ tests | Contract fixtures, synthetic episodes, replay and harness data. | Local files, mock sockets/services. | IMPLEMENTED tests; tests. No large recorded media copied. |

## Dependencies and operating assumptions

The Go module declares Paho MQTT, UUID, Gorilla WebSocket, `x/crypto`, YAML,
and indirect `x/net`/`x/sync`; Paho has no runtime-source import in the current
tree and is a likely stale dependency. Python requirements pin RKNN Lite only
for ARM64 and pin OpenCV/NumPy/SciPy. The UI pins versions through its lockfile.
MediaMTX is provisioned as an external binary, not a Go package.

The shipped deployment assumes Linux, systemd, filesystem-backed `/etc`,
`/var/lib`, `/run`, Unix sockets, and root access for Discovery and runtime
management. SynoraNet additionally assumes Linux networking tools and hostapd,
dnsmasq and firewall commands. Vision acceleration assumes the Rockchip RKNN
stack when enabled. Other OEM operating systems, supervisors, storage drivers,
and accelerator adapters are not implemented in this source tree.

## Maturity rules and known exclusions

`IMPLEMENTED` means behavior exists and has a corresponding tested/software
path; it does not mean a hardware qualification or production guarantee.
`PARTIAL` means interfaces or a software path exist but a key integration,
qualification, or runtime capability is absent. `PLACEHOLDER` means the
repository documents a future contract but has no functional code. `UNUSED`
means no active runtime composition/reference was found. `UNKNOWN` means this
audit could not establish behavior. Empty future modules were not generated.

Deprecated/legacy examples are `internal/state` and the replay/injector command
line tools; they remain copied for traceability. Hardware adapters for actual
camera sensor/ISP/VENC, physical actuator execution, TTS, inter-camera identity
tracking and remote connectivity enrollment are not implemented. `internal/
foundationv1` is a software/synthetic contract harness rather than a production
hardware integration.
