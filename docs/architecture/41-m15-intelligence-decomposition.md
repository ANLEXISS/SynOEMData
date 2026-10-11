# M1.5 — Intelligence Decomposition

**Status:** Complete after validation

## Result

The CPU MLP bundle loaders and inference code now live in the Intelligence-owned package `internal/intelligence/mlp`. That package implements the existing `cognitivecore.MLPBackend` and `cognitivecore.MLPBackendV3` interfaces. The cognitive core continues to own snapshot projection, feature encoding, decision orchestration, decision ordering, safety gates, StateStore coordination, and the shared inference contracts.

No model format, inference calculation, output mapping, trace schema, gate, persistence format, or executable entrypoint was intentionally changed. The current composition roots load the same CPU bundle through the new package and inject it into Cognitive Core.

## Actual dependency shape

```text
cmd/synora-core ───────────────┐
cmd/synora-cognitive-runtime-bench ─┤
cmd/synora-central-test ────────────┼──> internal/intelligence/mlp
                                   │              │
                                   │              └──> internal/cognitivecore contracts
                                   └─────────────────> internal/cognitivecore orchestration
```

The adapter depends on Core contracts; Core does not import the adapter. The architecture checker now enforces this one-way boundary and verifies that the package is owned by `synora.intelligence`.

## Scope and limits

- V1 and V3 CPU bundle validation, loading, feed-forward execution, and trace generation moved to `internal/intelligence/mlp`.
- Core's V1/V3 interfaces, output and trace contracts, snapshot/encoder contracts, and deterministic safety behavior remain in Cognitive Core.
- `internal/intelligence/topology` remains a separate Intelligence package. A source import search found no Go consumers in the workspace; it remains descriptor-owned, and this milestone does not claim it is connected to the cognitive orchestration path.
- This is an internal package boundary inside the existing Go module, not a new process, service, module, or dynamic plugin mechanism.
- No learned Cognitive Runtime or new inference dependency was added.

## Validation

- `go build -buildvcs=false ./...` — PASS.
- `go test -buildvcs=false -p 1 -count=1 ./...` — PASS.
- `go test -buildvcs=false -race -p 1 -count=1 ./internal/cognitivecore ./internal/intelligence/mlp ./internal/foundation/statestore` — PASS.
- `make GO=<Go 1.24 toolchain> architecture-check` — PASS: 7 logical modules, 37 owned Go packages, 8 declared module edges, 7 actual module edges; RKNN import isolation and MLP adapter direction enforced.
- Synthetic central E2E: 461 passed, 0 failed; camera mock E2E reported `not_qualified` by the harness.

## Files and package ownership

- `internal/intelligence/mlp/v1.go`, `v3.go`: moved CPU inference implementations.
- `internal/intelligence/mlp/mlp_trace_test.go`, `core_integration_test.go`: moved/adjusted the associated inference trace and Core integration coverage.
- `internal/cognitivecore/decision.go`, `decision_v3.go`: retain contracts and orchestration; concrete CPU inference implementations removed.
- Core, benchmark, and central test composition roots: updated to load from `internal/intelligence/mlp`.
- `modules/intelligence.json`, `modules/package-ownership.json`, `tools/architecture/check.py`: ownership registry and enforced dependency direction updated.
