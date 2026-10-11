# Extraction validation report

## Source baseline

Validation was performed against the source working tree before extraction,
including its pre-existing `Makefile`, `bootstrap/` and `camera/` changes. Go
toolchain 1.24.0 was available in the local module cache; network module access
was disabled (`GOPROXY=off`). All build/test output and caches were directed to
`/tmp` or the external Go cache. The source Git status was checked before and
after these commands and remained `M Makefile`, `?? bootstrap/`, `?? camera/`.

| Check | Baseline command / result | Classification |
| --- | --- | --- |
| Go package compilation/tests | `go test ./...` — all 47 packages passed; packages with no tests reported normally. | PASS |
| Go build | `go build ./...` — passed. | PASS |
| Go package listing | `go list ./...` — 47 packages. | PASS |
| Python worker tests | `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests -p 'test_*.py'` — 108 tests passed. Output included expected missing-model warnings and a local RKNN runtime/model initialization exercised by tests; no camera was connected. | PASS; test output is hardware-sensitive. |
| Native camera tests | CMake configured/built under `/tmp`; CTest — 2/2 passed. | PASS |
| Central synthetic E2E | `go run ./cmd/synora-central-test --manifest testdata/central-e2e-v1/manifest.json --out /tmp/...` — 461/461 passed, 0 failed; 460 pipeline completions and 1 expected rejection. `camera_mock_e2e=not_qualified`; no hardware qualification claim. | PASS |
| Web build | Dependencies are installed only in the source's ignored `node_modules`; no dependency installation was performed. Build validation is run from a temporary copy of OEM UI source using those existing local dependencies, with output outside both repositories. | See final result below. |

## Extraction validation

Results are recorded after running the same non-installing checks against this
workspace. Go module and package paths were preserved; no import path rewrite
was needed. Python bytecode and CMake outputs are not stored in this checkout.

| Check | OEM result | Regression classification |
| --- | --- | --- |
| Go package listing | `go list -buildvcs=false ./...` — 47 packages. | PASS |
| Go test/build | `go test -count=1 ./...` — all packages passed. `go build -buildvcs=false ./...` — passed. An initial unflagged `go build ./...` tried to stamp VCS state from the unrelated parent `/home/rock/.git` because the OEM tree intentionally has no `.git`; the existing Makefile already specifies `-buildvcs=false`. | PASS using repository build flags; unflagged command is an environment limitation |
| Python unit suite / syntax | 108 tests passed with `PYTHONDONTWRITEBYTECODE=1`; no source bytecode generated. | PASS |
| Native C++ mock suite | CMake build under `/tmp`; CTest 2/2 passed. | PASS |
| Central synthetic E2E | Run against the copied workspace with output under `/tmp`; 461/461 passed, 0 failed; 460 pipeline completions, 1 expected rejection, 0 incomplete. `camera_mock_e2e=not_qualified`; `foundation_v1=simulated_test`. Report hash and summary match the source baseline. | PASS; no regression |
| UI TypeScript/Vite build | `npm run build` from a temporary copy of OEM UI source; local source-installed `node_modules` was linked only into that temp tree; TypeScript/Vite build passed to `/tmp`. | PASS |

No production service, systemd unit, network interface, camera, or live
deployment was started or modified. No package was installed. Any redacted or
hardware-dependent qualification limitation remains explicit in the test
output; a software test does not qualify physical hardware.

## M1.2 active modularization

M1.2 baseline and post-move package validation are recorded in
[`16-modularization-validation.md`](16-modularization-validation.md). The
original M1.1 results above remain historical and are not post-M1.2 validation
of the moved package.
