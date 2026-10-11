# M1.2 OEM adaptation surfaces

## Current implementation

| Replaceable implementation | Boundary / inputs → outputs | Errors and lifecycle | Minimum / optional capability | Classification |
| --- | --- | --- | --- | --- |
| Vision worker process | `WorkerLifecycle` plus `ProcessExecutor`; JSON requests and versioned observations | Start/stop, health snapshot, bounded retry/backoff; failures become unavailable evidence | Required: protocol hello, correlation IDs, bounded request/response. Optional: selected model capabilities | Synora product boundary with current Python/RKNN reference implementation |
| Video input | `internal/vision` clip/segment contracts and local media reference | Validation, timeout, unavailable/rejected observation; no direct sensor lifecycle in current implementation | Required: accepted media window and provenance; optional: camera health/continuity | Replaceable input adapter; actual sensor/ISP/VENC adapter is not implemented |
| RKNN inference | Python detector backend/model configuration and aggregate observation output | Backend reports unavailable/degraded; model qualification must be explicit | Required: load status and bounded inference result; optional: hardware acceleration | Reference adapter; no ONNX Runtime, TensorRT, OpenVINO or QNN adapter is implemented |
| MediaMTX | `PathController` HTTP API; path list/add/delete | HTTP error/status; external process lifecycle remains separate | Required: path control if MediaMTX is selected | Reference adapter; replaceable streaming server integration |
| Host process supervisor | `internal/manager` commands/runtime snapshots | Command exit/error, bounded waits and rollback where implemented | Required: start/stop/status; optional: rollback and health integration | Linux/systemd-oriented platform implementation |
| Host networking | `internal/discovery/network`; configured network and injected station data to host state | Privileged command failures surfaced; cleanup and rollback are operation-specific | Required: report capability and apply result; optional: AP/DHCP/firewall features | Host platform implementation; highly Linux-specific |
| Durable state | Core-owned Universal Store files/WAL | fsync/rename/replay errors surfaced to Core; Core remains sole writer | Required: ordered durable commits and replay; optional: snapshots/compaction | Synora product state contract; storage backend not generalized |
| Internal message transport | `contract.Message` through bus client/server | ACL/HMAC, correlation, queue bounds and disconnect handling | Required: preserve message identity and delivery errors | Synora transport implementation; external broker adapters not implemented |

Replaceable parts must keep business semantics in shared contracts. Platform
adapters translate lifecycle, errors and capabilities; they must not decide
cognitive or automation behavior. `modules/*.json` records the current
capabilities and maturity without claiming hypothetical adapters exist.

## Target architecture

Choose adapters per OEM product only after its OS, supervisor, storage,
accelerator and media contracts are qualified. Keep adapter interfaces small
and close to consumers. A future vendor camera SDK belongs outside Vision's
processing and observation contracts.

## Not implemented

No OEM camera SDK, alternate supervisor, storage backend, or accelerator
backend is included. Existing `camera/v0` is a simulator/reference path, not a
physical camera adapter.
