# H1 Contract Compatibility Remediation

**Status:** H1/H3 targeted checks PASS; central synthetic suite has one incomplete reference journey, recorded below  
**Date:** 2026-10-11  
**Scope:** Contract characterization, test fixtures, and one minimal segment decoder correction. No new product semantics.

## Findings and remediation

The earlier H1 write-up incorrectly stated that Go discarded Python Vision event payloads. Source review showed `internal/vision.Event.Payload` is `map[string]any`; the runtime clones it and preserves nested values while decoding worker responses. Tests now compare the complete payload and carry a Python `EventBuilder` result through the actual Unix socket worker boundary. Additional cases characterize missing, null, empty, malformed, duplicate-key, unknown-event-name, and 1 MiB payload behavior. No new payload-size limit was introduced.

Segment coverage found one actual decoder defect. `DecodeVisionSegmentReadyV1` reused a strict payload sanitizer that removed `node_id`, although the typed segment validator requires it. The decoder now restores only the explicitly allowed identity field before validation. The JSON shape and serialized bytes are unchanged. New goldens cover ready, gap, and continuity reset records, including output from the actual Vision segment publisher.

Discovery API handler fixtures now cover `/api/v1/state`, `/api/v1/health`, the existing subscription event stream, and resident gallery status. Synthetic state includes an incident projection. No separate incident route exists in the inspected API, so no route was invented. Device observation fixtures exercise online, offline, and recovery reporting.

Persistence characterization extends the existing V1 state/WAL serializer and replay baseline with V1/V2/V3 record references and fixed-clock temporal checks. Event time drives snapshot `CapturedAt`; missing event time falls back to injected `Core.Now`; `ReceivedAt` remains unset; history is in commit arrival order; the Store de-duplicates by event ID; epoch and sequence are carried as metadata and are not generally enforced by Core.

The source-backed capability inventory is in [`05-device-capability-inventory.md`](../semantics/05-device-capability-inventory.md) and `testdata/contract-baseline/device-capabilities-v1.json`. It separates configured identifiers, Vision runtime feature status, and abstract action labels. It makes no physical support or qualification claim.

## Validation

| Check | Result |
|---|---|
| Focused Go tests for Core temporal behavior, Discovery camera availability, Vision worker socket, segment contract, and H3 runner | PASS |
| `make test-contract-baseline` | PASS |
| `go build ./...` | PASS |
| `go test -p 1 -count=1 ./...` | PASS |
| `make architecture-check` | PASS |
| Vision Python full unittest suite | PASS |
| H3 repeatability and module regression checks | PASS |
| Central synthetic E2E | Command PASS (461/461 aggregate); report contains 1 incomplete pipeline journey, `episode-sequence-03`, at `action_result_recorded` (`result_not_observed`). This remains an explicit validation limitation and is not described as a fully complete central pipeline. |
| C++ mock checks | PASS; 2/2 CTest cases |
| UI build | PASS |
| Targeted race checks | PASS |

The central report also records 460 completed pipeline journeys, one expected rejection, and one incomplete journey. Its camera mock remains `not_qualified` and Foundation V1 remains `simulated_test`. The H1 contract and H3 scenario targets pass independently; no hardware qualification is implied by synthetic or mock checks. This report does not treat the central aggregate exit status as evidence that the incomplete reference journey completed.

## Limits

The Vision worker response decoder has no configured payload byte limit; this report records current behavior. Core does not generally enforce event ordering, sequence monotonicity, epoch transitions, or stale-event rejection. API incident data is a state projection, not a separate endpoint. Physical cameras, inference accelerators, supplier models, Voice, ONVIF/Matter, and actuator operations were not qualified or enabled. The central fixture's missing structured action-result observation is outside the new H1/H3 fixtures and remains unresolved; it must be addressed before claiming full central pipeline completion.

## Related records

- [`05-h1-contract-compatibility-baseline.md`](05-h1-contract-compatibility-baseline.md)
- [`07-h3-device-fault-harness.md`](07-h3-device-fault-harness.md)
- [`testdata/contract-baseline/index.json`](../../testdata/contract-baseline/index.json)
- [`testdata/contract-baseline/device-capabilities-v1.json`](../../testdata/contract-baseline/device-capabilities-v1.json)
