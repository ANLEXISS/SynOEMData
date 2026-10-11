# H1 Contract Compatibility Baseline

**Status:** VALIDATED  
**Date:** 2026-10-11  
**Scope:** Characterize existing production contracts; no new product semantics or production behavior.

## Result

The H1 baseline inventory indexes 11 production contract families in the SynoraOEM source file `testdata/contract-baseline/index.json`. Fixtures characterize the Go/Python Vision exchange, StateStore state and WAL, camera observations and availability, segment records, Discovery API responses, and device capability identifiers. Existing V1 goldens were reused rather than copied. Tests trace examples to their owning source, producer, consumer, format, version, validation behavior, unknown-field behavior, and compatibility requirement.

The inventory preserves the seven existing logical modules. `internal/foundationv1` is treated as a synthetic reference only. The fixtures and tests introduce no dependency, production socket, service, device, migration, or storage-path changes. Store tests use temporary directories; the Vision worker runs in dry-run mode.

## Added baseline artifacts

| Artifact | Purpose |
|---|---|
| `testdata/contract-baseline/index.json` | Machine-readable contract inventory and test traceability; validated for JSON syntax and referenced fixture presence |
| `testdata/contract-baseline/vision/go-clip-request-v1.json` | Exact Go `Request` serializer shape; Python worker consumes the fixture in dry-run mode |
| `testdata/contract-baseline/vision/python-worker-response-v1.json` | Deterministic Python `EventBuilder` output consumed by the Go response decoder with nested payload preservation |
| `testdata/contract-baseline/vision/python-worker-empty-payload-v1.json` | Empty payload shape compatibility case |
| `testdata/contract-baseline/persistence/state.json` | Actual `UniversalStore` serializer output including V1 materialized snapshot, V2/V3 snapshot data, decision records, and action outbox |
| `testdata/contract-baseline/persistence/journal.jsonl` | Actual `universal-store/v1` WAL serialization used by replay |
| `pkg/contract/testdata/camera-observation-v1.json` | Canonical camera observation and RFC3339 timestamp |
| `internal/discovery/testdata/resident-gallery-response-v1.json` | Representative API response DTO serializer output |
| `internal/discovery/testdata/state-response-v1.json`, `health-response-v1.json`, `websocket-event-v1.json` | State, health, and event stream handler goldens |
| `pkg/contract/testdata/vision-segment-ready-v1.json` and `testdata/contract-baseline/vision/segment-{gap,continuity-reset}-v1.json` | Segment identity, gap, and continuity reset goldens |
| `testdata/contract-baseline/device-capabilities-v1.json` | Machine-readable source-backed capability identifier inventory |
| `internal/vision/contract_baseline_test.go` | Go request serialization, Python response decoding, and malformed-response behavior |
| `services/vision-worker/tests/test_contract_baseline.py` | Python consumption of Go request fixture, Python event golden, and existing event-name rejection |
| `internal/cognitivecore/contract_baseline_test.go`, `temporal_contract_baseline_test.go` | Byte-for-byte Store/WAL serializer comparison, temporary-directory replay/outbox assertions, and fixed-clock temporal behavior characterization |
| `pkg/contract/camera_contract_baseline_test.go` | Camera fixture round-trip, timestamp and machine-index fixture reference checks, Message timestamp/unknown-field compatibility |
| `internal/discovery/api_contract_baseline_test.go` | API response serializer fixture assertion |

The dedicated entrypoint is `make test-contract-baseline`. It runs existing Go tests in the relevant owner packages and the new Python `unittest` cases with a bounded timeout. `make test` remains unchanged.

## Compatibility findings

- The Go Vision request serializer emits zero-valued `started_at` and `ends_at` timestamps as `0001-01-01T00:00:00Z`, despite `omitempty` tags. The shared fixture preserves this existing wire shape, and the Python worker accepts it.
- Python restricts Vision event names through `EventBuilder`. Go's current response decoder checks response shape and request correlation but accepts structurally valid event names. An unrecognized name remains data at this boundary; this decoder does not map it to an executable action.
- Go's `internal/vision.Event` models `Payload map[string]any`; the runtime clones and preserves Python event payloads through typed decode and worker socket handling. Complete fixture comparison now covers nested and event-specific fields. Missing, null, and empty payloads remain distinct zero/nil cases; arrays and scalar payloads fail typed decoding. Duplicate JSON keys follow Go's standard decoder behavior (last value wins). The decoder has no configured payload byte limit; a 1 MiB payload is accepted, which records current behavior rather than defining an approved protocol maximum.
- Bus messages use newline JSON and a custom timestamp decoder. Tests record RFC3339Nano, integer seconds, and integer milliseconds behavior, unknown-field tolerance, zero-time fallback for an invalid timestamp shape, and rejection of an invalid numeric sequence.
- `universal-store/v1` state and WAL fixtures are generated by the actual production Store serializer and compared byte-for-byte. Tests reopen the fixtures from temporary directories and verify V2/V3 snapshot data, journal revision, and the dry-run action outbox replay.
- Core temporal behavior is now characterized with a fixed clock: an absent event timestamp uses `Core.Now`, snapshot `CapturedAt` follows event time, `ReceivedAt` is not synthesized, history follows commit arrival order, and unique event IDs with repeated sequence values are accepted. Event IDs are the Store duplicate key; Core does not generally interpret epoch changes, sequence gaps, stale timestamps, or out-of-order delivery.
- The segment ready decoder previously passed through a shared sanitizer that removed `node_id`, even though segment validation requires it. The narrow decoder now restores this allowlisted identity field before final validation; the serialized schema is unchanged.
- Discovery state, health, event-stream, and resident gallery status handlers are exercised through their existing routes. Incident facts are present in the state projection; no separate incident API route was found.
- Unknown JSON fields are generally ignored by the owning Go typed decoders. Unknown string and enum behavior remains owner-specific: some values are normalized, while validated schemas reject unsupported values.

## Validation results

| Check | Result |
|---|---|
| `make test-contract-baseline` | PASS; relevant Go contract/runtime/store/API packages and Python contract tests |
| `go build ./...` | PASS |
| `go test -p 1 ./...` | PASS; all packages with tests passed |
| `make architecture-check` | PASS; 7 modules, 38 owned Go packages, 9 declared edges, 8 observed module edges, Vision RKNN import isolation verified |
| Vision Python compile and full unittest suite | PASS; 118/118 (the prior 115 plus 3 H1 tests) |
| `make test-camera-mock` | PASS; 2/2 CTest cases |
| `make test-central-v1 CENTRAL_E2E_OUT=/tmp/synora-h1-h3-central-e2e.json` | Command PASS; 461/461 aggregate, 460 completed, one expected rejection, one incomplete pipeline journey (`episode-sequence-03`, missing structured action-result observation); camera mock reports `not_qualified`; Foundation V1 remains `simulated_test` |
| `npm run build` in `synora-web` | PASS; TypeScript and Vite production build |

No prior contract or regression test was weakened. Build and test commands printed an environment shell warning that `PS1` was unset; each command returned success. The central command's aggregate result does not make its incomplete reference journey complete; see [`06-h1-remediation-report.md`](06-h1-remediation-report.md). The synthetic and mock checks do not establish physical camera, RKNN, network, or actuator qualification.

## Remaining coverage gaps

- The Python-to-Go Vision decoder has no explicit payload byte limit. This baseline records observed behavior and does not choose a maximum.
- The API has no separately identified incident endpoint; incident data is available through the state projection. Other anonymous DTOs not listed in the H1 fixtures remain outside this baseline.
- Capability values are inventoried, but remain distributed among device configuration, worker status metadata, and abstract decision/action labels. There is no exhaustive device operation or vendor qualification catalog.
- State/WAL V1, V2, and V3 snapshot data is covered in the persisted baseline. Independent standalone V2/V3 snapshot golden files and legacy production-data samples are not present; no production records were read or migrated.
- This baseline does not claim Voice, spatial-zone analytics, physical device execution, or new semantic grammar as implemented.

## H2 and H3 readiness

H2 can use this baseline to propose a narrow versioned observation envelope and legacy adapter, while preserving the Python event payload and existing wire fixtures. H3 now has a shared deterministic Go/Python fixture harness and boundary-level coverage; see [`07-h3-device-fault-harness.md`](07-h3-device-fault-harness.md). Neither readiness statement claims a new product semantic has been implemented.

## Documentation publication

The local report is ready for the repository's approved Markdown security scan and synchronization workflow. Publication status and remote commit SHA are recorded after the synchronization command completes.
