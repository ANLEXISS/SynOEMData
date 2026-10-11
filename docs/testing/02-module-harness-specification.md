# Deterministic Module Harness Specification

This is a harness design using repository conventions. It specifies future harness behavior; it does not add a framework or claim that unimplemented behavior is tested.

## Shared harness rules

1. Keep Go `testing`, Python `unittest`, CTest and current manifest-driven central harness. Do not add a test framework without a demonstrated gap.
2. Each test gets an isolated `t.TempDir()`/`TemporaryDirectory`, private Unix socket, fixed logical clock, fixed IDs/seed and fresh fake device state. Never write to deployed paths or share a mutable default Store.
3. Inject clocks, worker/process adapters, command executors, bus publishers, inference backends, media sources and persistence hooks through existing interfaces. Fakes record calls and can return error, timeout, malformed/unsupported, duplicate, stale and reordered results.
4. Assert both positive outcomes and boundary contracts: JSON golden compatibility, schema/version, source/time/entity/evidence references, authorization vs execution distinction, Store revision/dedup/replay, and cleanup.
5. Bound each operation with a context/deadline; on timeout capture goroutine/process/socket state, cancel, close, wait for workers, and fail without hanging the package. No sleeps for ordering; use channels/barriers/fake clock.
6. Reproducibility report includes code revision, seed, logical clock, selected backend, fixture hashes, test command, hardware/runtime availability and result. A fake/fixture success is labeled software test, not device qualification.

## Module specifications

| Module | Entrypoint and scope | Fixture / fake adapters | Assertions, injected failures, reset and cleanup |
|---|---|---|---|
| Foundation | Existing Go packages: `go test ./pkg/contract ./internal/bus ./internal/foundation/... ./internal/state`; contract subset also included in `go test ./...`. Layers: unit, contract, component, race, fuzz, benchmark. | Golden JSON under `pkg/contract/testdata`; fresh temp StateStore; fake `PersistenceHooks`; in-process Bus clients and temp socket; fixed clock/IDs. | JSON round-trip and legacy fixture compatibility; event clone isolation; schema/version validation; queue/ACL behavior; WAL append/replay/dedup/revision/outbox; injected short write/sync/rename/dir-sync/close failure; crash simulation via close/reopen at checkpoints; reset by fresh temp directory; stop clients and remove socket. Race Store and Bus suites; fuzz contract validation/decoding. |
| Intelligence | `go test ./internal/cognitivecore ./internal/intelligence/...`; reduced composition via existing `make oem-profile-core-only`; central E2E covers service composition. Unit/component/contract/integration/race. | Fixed-clock typed event sequences, Store temp dir, stub MLP V1/V3, fixed topology/residents, fake Bus publisher and capability facts. | Snapshot/decision outputs, evidence references, deterministic IDs/order, safety-gate outcomes, idempotent duplicate inputs, unknown/stale inputs and capability-missing behavior. Inject backend unavailable/error/timeout, corrupt record, Store failures and event reorder/gap. Fresh Core and Store per case; close loop/bus and discard temp files. |
| Vision | Go `go test ./internal/vision ./internal/visionsuite ./internal/facedataset ./internal/facegallery`; Python `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s services/vision-worker/tests -p 'test_*.py'`; C++ `make test-camera-mock`. Unit/contract/component/integration/fuzz; hardware separate. | Existing Python fake detectors/runners and JSON fixtures; fake worker protocol over temp Unix socket; fake process executor; synthetic frames/tracks; existing fixture manifests; no RKNN device required. | Protocol hello/version, JSON schema and event allow-list, one tracking owner, confidence/evidence/source/time preservation, worker lifecycle, segment gaps and recovery, face privacy/consent boundaries. Inject malformed JSON, process exit, timeout, backend unavailable, partial segment, duplicate/out-of-order frames and cleanup error. Terminate worker, close socket, release episode context, clear temp media. Mark actual camera/RKNN qualification separately. |
| Voice | No entrypoint or implementation exists; status `NOT_IMPLEMENTED`. | None until approved product contract exists. Future harness must start with a typed Voice port and fake audio/TTS adapters. | No Voice tests should be described as coverage. Once implemented, deterministic audio fixtures, fixed clock, redacted transcript, privacy/consent and unavailable/timeout results are mandatory; no microphone required in unit tests. |
| Automation | `go test ./internal/automation ./internal/cognitivecore ./internal/outbox`; central harness for action lifecycle; race package run. Unit/contract/component/integration. | Fake `Dispatcher` device/action adapter (current implementation remains dry-run), Core safety policy, Foundation outbox temp Store, fixed event sequence. | Validate command vocabulary/target, policy decision and authorization distinct from result; idempotency; request/result correlation; dry-run marker; unavailable/unsupported/timeout outcomes; dispatch error after claim and ack/retry behavior. Inject denied/indeterminate, adapter error, crash after claim/before ack, duplicate request. Recreate Store to verify replay; drain workers and clear temp outbox. |
| Integration | `go test ./internal/discovery/... ./internal/device ./internal/connectivity ./internal/security ./cmd/synora-api`; central harness. Unit/contract/component/integration/race. | Fake station source, HTTP clients, Bus, auth/session, device registry YAML in temp, fake network source; fake Vision worker and host command executor; deterministic API clock. | Boundary mapping, authentication/authorization, canonical camera observation, unsupported adapter/protocol, device online/offline, retry and event correlation; ensure secrets are redacted. Inject bad config, 401/403, malformed payload, timeout, duplicate/out-of-order observation and source restart. Temp config/data/socket only; close listener and goroutines. |
| Platform | `go test ./internal/discovery/network ./internal/manager ./internal/boothealth ./internal/version ./internal/mediamtx ./cmd/synora-boot-healthcheck`; unit/component/race. Avoid install targets. | Existing `HostCommandExecutor` fake, in-memory process executor, temp configs, fake systemd/status and MediaMTX HTTP responder; fixed command outputs. | Exact args/env and allowed command boundaries, process supervision state transitions, timeout/cancel, idempotent start/stop, host config rendering and atomic file errors. Inject command not found, nonzero exit, partial output, signal/timeout, corrupt status. Do not call privileged real commands. Ensure children, temp files and sockets are removed. |

## Proposed repeatable commands

The following are suggestions assembled from existing entrypoints, not new targets:

```text
make architecture-check
go test -buildvcs=false -p 1 -count=1 ./...
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s services/vision-worker/tests -p 'test_*.py'
make test-camera-mock
make test-central-v1 CENTRAL_E2E_OUT=/tmp/synora-central-e2e-audit.json
```

Run race/fuzz/performance/hardware suites as separate tagged jobs with explicit budgets and evidence. Avoid `make test` when its local CMake output or generated test report location is not controlled; its current components and side effects should be reviewed before CI adoption.

