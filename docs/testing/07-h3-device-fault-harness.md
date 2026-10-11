# H3 Deterministic Device and Fault Harness

**Status:** VALIDATED  
**Schema:** `synora.oem-test-scenario/v1`  
**Scope:** Test-only Go and Python simulators driven by shared synthetic JSON scenarios.

## Harness design

`testkit` provides fixed logical time, a scripted one-device simulator, request and observation capture, capability checks, deterministic faults, reset, and expected-result comparison. It uses only the Go standard library. The Vision worker test suite has a Python reader/runner for the same fixture file. Neither simulator opens device sockets, starts services, performs real inference, writes production state, or executes physical actions.

Schema V1 deliberately supports exactly one device per scenario because its expected result currently exposes one final device projection. Both runners reject or assert against a different shape. Multi-device orchestration, stochastic scheduling, a general property-testing framework, real adapter fault injection, and hardware execution are outside this milestone.

## Initial scenarios

`testdata/device-harness/scenarios-v1.json` contains ten named scenarios covering:

1. camera unavailable then recovery;
2. valid Vision observation;
3. nested Vision payload;
4. out-of-order observations;
5. duplicate observations;
6. unsupported capability/operation;
7. abstract action remains dry-run;
8. Store write interruption and outbox retry;
9. worker unavailability and connection interruption followed by recovery;
10. malformed, missing, stale, and sequence-gap observations plus restart/reset behavior.

Every case has a synthetic ID, fixed initial clock, declared capability set, scripted steps and fault, expected trace, final device state, physical execution flag, cleanup assertion, and final clock. Each fixture is run twice in Go and Python and compared with its expected result.

## Production seam coverage

| Behavior | Deterministic fixture | Existing production boundary exercised |
|---|---|---|
| Camera observation, unavailable, recovery | A | Discovery Registry producer and snapshot/event sequence (`internal/discovery/runtime/device_contract_baseline_test.go`) |
| Vision observation and nested payload | B/C | Python EventBuilder contract, Go typed decode, actual Unix socket response path (`internal/vision/contract_baseline_test.go`; Python contract baseline) |
| Out-of-order, duplicate, missing, malformed, stale, sequence gap | D/E/J | H1 decoder and Core temporal characterization; these tests document acceptance/rejection without claiming Core normalizes ordering |
| Unsupported capability and abstract action safety | F/G | Existing Automation policy and dry-run tests (`internal/automation/action_test.go`) |
| Persistence interruption and retry | H | Existing WAL crash/recovery, replay, and outbox tests (`internal/cognitivecore/store_persistence_test.go`) |
| Worker and connection unavailable/recovery | I | Existing Vision runtime and worker-manager fault tests |
| Reset and cleanup | A/J and Go reset test | Testkit resets state, requests, observations, and logical clock; no production state is touched |

The fixture simulator is not itself a substitute for those production seam tests. The dedicated Make target includes the owner packages so the real boundaries are regression-tested with their existing test doubles.

## Commands

- `make test-device-harness` runs Go fixture tests and the Python shared-scenario runner with bounded timeouts.
- `make test-semantic-foundation` runs contract baseline plus the H3 harness.

The default `make test` target is unchanged. No third-party dependencies, daemons, scheduled tasks, production services, or hardware requirements were added.

## Validation and limits

All ten scenarios pass twice in both languages with the checked-in expected traces and no physical execution. The broader milestone checks are recorded in [`06-h1-remediation-report.md`](06-h1-remediation-report.md). The scenario seed is metadata for repeatability and is not used to randomize runs; there is no randomness or sleep in the runner. Delayed response advances logical time. Out-of-order, stale, and sequence-gap faults are recorded as outcomes but do not synthesize Core policy.

The fixture's store-write/outbox scenario is a deterministic model. Production persistence fault coverage comes from the separate injected WAL tests. Worker and Discovery simulator scenarios are linked to separate package tests; the shared H3 runner does not orchestrate production modules end-to-end. No physical camera, network device, audio path, or actuator has been exercised.
