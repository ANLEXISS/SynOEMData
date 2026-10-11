# M1.6 — Automation Independence

**Status:** Complete.

## Result

A production Automation package now owns the existing abstract action request/result contracts and deterministic controlled dry-run outcome policy: `internal/automation`. Discovery remains responsible for receiving the bus command and publishing the Integration event envelope. Its established exported action types remain source-compatible aliases to the Automation-owned types.

The extraction was limited to code that already executes in the production action path. The similarly named `internal/foundationv1` policy and dry-run executor remain a synthetic reference harness and were not treated as production Automation.

## Runtime path

```text
Cognitive Core outbox
  -> Discovery bus ingress (Integration)
  -> internal/automation.Dispatcher
  -> Discovery result event envelope (Integration)
  -> Cognitive Core action-result ledger and Foundation StateStore
```

Cognitive Core retains proposal generation, safety gates, durable outbox/ledger semantics, and StateStore coordination. Automation validates required request identity/action and applies the existing deterministic outcomes: dry-run, unavailable capability, blocked when dry-run execution is disabled, and no action requested. Every result reports `physical_action_executed: false`; no device identifier or physical adapter was added.

## Dependency and descriptor changes

- `synora.automation` owns `internal/automation` and provides `synora.automation.action-dispatch/v1`.
- Integration declares the Automation capability and depends on the Automation module.
- Automation has no Foundation, Intelligence, Integration, or Platform imports; the current implementation is a pure policy boundary.
- Seven domain names and the existing Go module/service structure remain unchanged.

## Validation

- `go build -buildvcs=false ./...` — PASS.
- `go test -buildvcs=false -p 1 -count=1 ./...` — PASS.
- `go test -buildvcs=false -race -p 1 -count=1 ./internal/automation ./internal/discovery ./internal/cognitivecore ./internal/foundation/statestore` — PASS.
- `make GO=<Go 1.24 toolchain> architecture-check` — PASS: 7 modules, 38 owned Go packages, 9 declared edges, 8 actual edges; Integration-to-Automation ownership/dependency direction enforced.
- Synthetic central E2E — PASS: 461/461; 49 allowed dry-run outcomes, 6 safety blocks, 371 suppressed no-action results. `camera_mock_e2e=not_qualified`.

## Limits

Automation remains partial. There is no rules engine, physical executor, vendor device adapter, or independent process. The deterministic Core safety gate remains authoritative for action authorization; this extraction does not move or weaken that gate.
