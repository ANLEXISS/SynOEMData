# Existing Test Harness Audit

**Audit date:** 2026-10-11. No product code or test framework was changed for this audit.

## Entrypoints and repository conventions

| Harness | Existing entrypoint | Scope and dependencies | Determinism / isolation notes |
|---|---|---|---|
| Architecture | `make architecture-check` -> `tools/architecture/check.py` | Seven logical modules, package ownership, imports/cycles, selected invariant checks and profile closure; Python standard library and Go source inspection | Deterministic source inspection; does not validate runtime semantics or contract coverage |
| Go package tests | `go test ./...` (serialized `-p 1 -count=1` used in prior stabilization evidence) | Unit/contract/component tests across Go packages; locked Go modules from `go.mod` | Most tests use `t.TempDir`, fakes and fixed time; separate live-process/network and filesystem edge cases need explicit environment review |
| Combined regression | `make test` | Architecture + central harness + C++ camera mock + Go suite + Python compileall + Python `unittest` worker suite | Broad existing local regression entrypoint. Requires Go, Python, CMake/CTest and C++ compiler; no install target should be invoked |
| Central synthetic E2E | `make test-central-v1` or direct `go run ./cmd/synora-central-test --manifest testdata/central-e2e-v1/manifest.json --out /tmp/...` | Core/Store/Discovery/MLP event lifecycle with fixture seed/date and generated cases; includes camera C++ mock target | Synthetic deterministic seed and logical date; output report goes to `/tmp`; explicitly reports camera mock `not_qualified` |
| C++ camera mock | `make test-camera-mock` | CMake build and CTest under `build/camera-v0`; mock target | Mock tests are not physical camera qualification; local build directory is mutable output |
| Vision Python unit/component | `python3 -m unittest discover -s services/vision-worker/tests -p 'test_*.py'` | Python stdlib unittest; backend fakes and fixture files; some model/backend tests use local available runtime/model conditionally | Tests use deterministic fakes in many suites; tests that inspect local model/runtime availability must report that dependency explicitly |
| Race checks | `go test -race -p 1 -count=1` targeted package list per stabilization report | Bus, Core, MLP, Discovery, network, Runtime Manager, StateStore, reduced-profile smoke | Race instrumentation only applies to those runs/packages; does not establish all-package race freedom |
| Fuzz tests | Go fuzz targets in Connectivity, Security, Discovery network | Go built-in fuzzing; seeds run under normal `go test`; extended fuzzing requires explicit `-fuzz` and time budget | Reproducible seed corpus; randomized fuzz campaigns need recorded seed/corpus and duration |
| Performance | Store benchmark and cognitive runtime benchmark entrypoint | Go benchmark / `cmd/synora-cognitive-runtime-bench`; runtime bundle may be required | Not part of ordinary regression; machine/runtime dependent, no stable performance gate documented |
| Hardware qualification | Camera qualification, external media/model/RKNN paths | Physical camera/network/RKNN and device-specific setup | NOT RUN / UNVERIFIED in M1.10 report; mock and synthetic cases must not be reported as hardware proof |

The Makefile `test` path is the closest existing integrated entrypoint. It does not install dependencies. `make install`, `make install-deps`, qualification targets that touch physical services, and commands writing under `/etc`, `/opt`, `/var/lib`, `/run` are outside an isolated harness and must not be used as ordinary tests.

## Audit-run validation evidence

Run during this audit from the SynoraOEM workspace:

| Check | Result |
|---|---|
| `make test` with central E2E output redirected to `/tmp/synora-semantic-audit-e2e.json` | PASS; the Make target does not install dependencies |
| Architecture checker | PASS: 7 modules, 38 owned Go packages, 9 declared edges, 8 actual edges; Vision RKNN import isolation verified |
| Native camera mock | PASS: 2/2 CTest cases; mock-only, not hardware qualification |
| Central synthetic E2E | PASS: 461/461, 460 completed, one expected rejection, zero incomplete; `camera_mock_e2e=not_qualified`; Foundation V1 `simulated_test` |
| Go package suite | PASS: all packages with tests passed; packages without tests reported as such |
| Vision Python compile and unittest | PASS: 115/115; diagnostic logs include a missing-model fixture and local RKNN runtime initialization/error-path exercise; no physical qualification claim |
| Existing Git state | Three pre-existing untracked `configs/*.yaml.template` files remain untouched; audit adds only the eight requested Markdown files |

The login shell printed repeated `/etc/bash.bashrc` warnings that `PS1` was unset; the command returned exit code 0. These shell warnings are environmental noise, not regression failures. No physical camera/network/RKNN qualification was performed.

## Module-level coverage snapshot

| Logical module | Existing test footprint and classifications | Main blind spots |
|---|---|---|
| Foundation | Unit/contract: `pkg/contract/*_test.go`, Bus lifecycle/security/concurrency, persistence and StateStore tests; one Store benchmark; race evidence for Bus/StateStore | No single Foundation harness reset contract across Bus + Store; golden cross-language fixtures limited to selected V1 records; true power-loss durability not tested |
| Intelligence | Unit/component: Cognitive Core Core/V2/V3, snapshots, decisions, service, persistence, evidence, gallery, topology and MLP tests; central E2E; CPU benchmark | Topology has no observed production consumer in report; no typed observation grammar or broad temporal/order permutation contract; learned runtime is absent |
| Vision | Go runtime/worker/segment/clip manager tests; Python unittest covering worker, pipeline, backend interfaces, tracking, enrichment, manifest, camera health, dataset; C++ camera mock tests | Hardware camera, RKNN and media availability; static analytics zones/PTZ and physical object scenes not represented; local backend condition can vary |
| Voice | None | Correctly NOT_IMPLEMENTED; no production package or module tests |
| Automation | Unit tests in `internal/automation/action_test.go`, Core safety/action tests, Discovery and central lifecycle integration; race evidence | Dispatcher is dry-run; no physical device adapter, user rule/condition grammar, actor authorization or real actuator confirmation |
| Integration | Discovery/API/ingress/network station source, device, connectivity, security and API tests; central synthetic E2E; race evidence | No broad conformance adapter suite for actual Matter/ONVIF or OEM vendor protocols; behavior across all service entrypoints not one fixture-driven test |
| Platform | Network config/host command, runtime manager, boot health, version, MediaMTX tests; race evidence for network and Runtime Manager | Privileged Linux changes, systemd, real network tools, power failure and alternative OS are not deterministic hermetic tests |

## Test-quality observations

- Test counts measure the amount of exercised code paths, not coverage of semantic concepts. A passing 461-case synthetic E2E demonstrates the fixture matrix it contains, not unrepresented zone/lock/voice/Matter behavior.
- Fixtures have useful existing strengths: central manifest has seed/logical date, package tests use fake backends and temporary directories, Python worker tests use fake detector/model runners, and contract tests keep JSON fixtures.
- Failure isolation is strongest in package tests with injected filesystem/runtime interfaces; weakest when a test consults installed models, host utilities, process signals, or local hardware/runtime availability.
- Timeout handling exists at several service/runtime boundaries but no common harness-wide deadline/reset protocol is documented.
- Race coverage is targeted, not repository-wide. Fuzz target discovery is present, but regular suite only executes seeds; no stable fuzz campaign duration is recorded.
- No evidence of a shared module conformance fixture format, common fake device registry, or per-module contract coverage gate was found.
