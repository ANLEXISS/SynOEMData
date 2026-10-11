# M1.2 modularization validation

> M1.3 validation and the remaining Foundation extraction boundary are
> recorded in [21-m13-validation.md](21-m13-validation.md).

## Baseline before edits

`SynoraOEM` had no `.git` directory, so no OEM-local Git status or pre-existing
OEM edits could be enumerated. The reference `/home/rock/Synora-v1` status was
read before and after the work; its pre-existing `Makefile`, `bootstrap/` and
`camera/` changes remain untouched. No services were started and no package
was installed.

| Baseline check | Result |
| --- | --- |
| Go package tests, `go test -count=1 ./...` | Failed in `cmd/synora-central-test`: `TestLe2iAggregateEvidenceTraversesCore` reported a V3 snapshot schema/dimension/head mismatch and `TestCameraMockE2EIsTransportOnlyAndPreservesProvenance` reported 5/6 simulated cases. Remaining listed Go packages passed. |
| Focused central harness package rerun | `go test -count=1 ./cmd/synora-central-test` passed. This indicates an order/load-sensitive baseline failure; root cause is not yet isolated. |
| Pre-edit architecture map | Reported 47 Go packages, six deployed services, active Universal Store ownership in Core, and Vision under `internal/discovery/vision`. |

The full Go test run exercises process-heavy synthetic scenarios across
packages. Its two failures are retained as baseline failures; the later
serialized full-suite and focused reruns passed. The initial failure cause was
not conclusively isolated.

## Changes and post-edit validation

| Check | Result |
| --- | --- |
| Vision package move | `internal/discovery/vision` moved to `internal/vision`; all seven Go import sites updated. No wire protocol, algorithms or entrypoints changed. |
| Topology package move | The standalone, currently unimported topology package moved from `internal/topology` to `internal/intelligence/topology`; there were no production import sites to rewrite. |
| Platform dependency inversion | Network no longer imports `internal/device`; its consumer-owned `StationSource` is injected by Discovery. Focused adapter tests cover mapping, filtering and allowlist output. |
| Focused Go packages | `internal/vision`, `internal/discovery`, `internal/discovery/ingress`, `internal/discovery/network`, and `cmd/synora-central-test` passed after the moves. |
| Architecture enforcement | `make architecture-check` passed: 7 descriptors, 33 owned Go packages, 7 declared dependency edges, 6 observed cross-module edges; no module or package cycles, undeclared edges, OS imports in shared contracts, or forbidden imports. |
| Full Go tests | `go test -buildvcs=false -p 1 -count=1 ./...` passed all packages, including the moved topology package. The initial default-parallel baseline had two synthetic central test failures; the serialized final run passed. Cause of the baseline flake remains unknown. |
| Go build | `go build -buildvcs=false ./...` passed. |
| Python worker tests | 108 passed with bytecode output disabled. The suite initializes the local RKNN runtime/model in a test; no camera was connected and this is not hardware qualification. |
| Native camera mock | CMake build and CTest under `/tmp`; 2/2 passed. |
| Central synthetic E2E | 461/461 passed, 0 failed; 460 pipeline completions, 1 expected rejection, 0 incomplete. `camera_mock_e2e=not_qualified`; `foundation_v1=simulated_test`. This was rerun after the Vision and station-source extractions; the only later Go change was relocating the unused standalone topology package. |
| UI build | TypeScript/Vite passed from a temporary copy using existing dependencies; output stayed under `/tmp`. |

The checker uses only Python's standard library and `go list`. It runs through
`make architecture-check` and is a prerequisite of `make test`. The checker
also verifies descriptor fields, source paths, declared dependency IDs and
acyclicity. It checks the Go package graph, not Python import ownership.

## Target architecture

Run all repository checks after each boundary change. Add focused contract
tests when a new external adapter is implemented. Keep hardware qualification
separate from software tests.

## Not implemented / remaining risk

The Vision package still contains the concrete `os/exec` process adapter next
to its injected process-executor port; moving that adapter cleanly requires an
API-compatible default-construction decision. Core cognition and Universal
Store remain co-located. Python is not included in the import-policy checker.
Physical camera and target-board qualification were not performed.
