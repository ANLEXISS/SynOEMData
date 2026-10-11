# M1.3.1 validation record

## Workspace and baseline

The task was run in `/home/rock/SynoraOEM`. The original prototype
`/home/rock/Synora-v1` and production paths were not changed. The local Git
checkout had no commits and all existing project files were untracked, so no
baseline commit was created: the source contents could not be reliably
distinguished from generated and milestone artifacts. No remote or global Git
configuration was changed. `.gitignore` excludes build products, environment
secrets, models, node modules and Python caches; ignored generated model JSON
under `build/` was left in place. A filename scan found no local private-key,
certificate or `.env` files to add.

The attempted M1.3.1 baseline Go run was not a clean baseline: the serial test
process remained active while extraction edits began, and then was interrupted
after reporting a synthetic central test failure and a compile-cache/import
error in the package being changed. Those results are contaminated by overlap
with edits and are not attributed to the original tree or to the completed
change. The parallel baseline build exited successfully, but it also overlapped
with edits and is not used as baseline evidence. No pre-change run is claimed.
The prior M1.3 record reports its isolated central synthetic run at 461/461;
that is historical evidence, not an M1.3.1 post-change run.

## Post-extraction results

| Check | Result | Evidence |
| --- | --- | --- |
| Focused Foundation contracts tests | PASS | `go test ./internal/foundation/contracts`; exact synthetic JSON, restore/validation, and optional field semantics |
| Cognitive Core / Store tests | PASS | `go test ./internal/cognitivecore`; includes resident persistence and existing snapshot/journal/decision behavior |
| Go build | PASS | `go build -buildvcs=false ./...` |
| Full Go tests, serialized | PASS | `go test -buildvcs=false -p 1 -count=1 ./...`; all workspace packages passed, including `cmd/synora-central-test`, cognitivecore, snapshot, persistence and decision paths |
| Architecture checker | PASS | `make architecture-check`; `modules=7`, `owned_go_packages=35`, `declared_edges=7`, `actual_module_edges=6` |
| Snapshot restore and replay | PASS | Existing cognitivecore Store/persistence/snapshot tests passed within the serialized full suite |
| Decision serialization | PASS | Existing Store and decision tests passed within the serialized full suite; decision definitions were unchanged |
| Central synthetic E2E | PASS | 461/461 passed; 460 completed, 1 expected rejection, 0 incomplete. Report: `/tmp/synora-m131-final-central.json`. `camera_mock_e2e=not_qualified`, `foundation_v1=simulated_test`. |
| Python worker tests | PASS | 108 tests passed in 0.973s; local RKNN model/runtime initialization warning was emitted by tests; no camera was connected |
| C++ mock tests | PASS | CMake/CTest built under `/tmp/synora-m131-camera`; 2/2 passed |
| UI production build | PASS | Vite/TypeScript production build passed from `/tmp/synora-m131-ui-final` using existing local dependencies; no install performed |
| Targeted Go race tests | PASS | `go test -race` passed for `internal/foundation/contracts` and `internal/cognitivecore` on linux/arm64; this is targeted coverage, not a comprehensive race guarantee |

The initial in-flight baseline attempt described above produced a central
failure and an import/cache error while the source was being changed; these are
not valid pre-change results and are not counted as regressions. The clean
post-change serialized suite and standalone central harness both passed. No
comprehensive race-safety claim is made from targeted race testing.

## Scope result

The resident metadata definition is physically owned by
`internal/foundation/contracts`; its cognitivecore alias preserves the
historical Go spelling. The authoritative Store, journal/replay, serializers,
snapshot/decision domain contracts, and shallow-copy semantics remain in
`internal/cognitivecore`. This is a bounded, completed type extraction, not
full Foundation isolation.
