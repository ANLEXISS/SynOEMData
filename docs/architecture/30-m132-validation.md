# M1.3.2 validation record

## Preparation and baseline

The work ran in `/home/rock/SynoraOEM`. The original `/home/rock/Synora-v1`
workspace and production paths were not changed. Git metadata contains no
commits and all project files are untracked, so a safe baseline commit was not
possible. No remote or global Git setting was changed. A verified local
recovery archive was created outside the workspace at
`/home/rock/SynoraOEM-m132-recovery-20261010.tar.gz`; it excludes runtime
configuration, model/data assets, generated builds, and runtime state. Archive
integrity, extraction, and representative source-file comparisons passed.
SHA-256: `5afd8d22376a8f171e2e2439d3c2193642ec188e02962431378299790ae34e27`.

Pre-change checks, before implementation edits:

| Check | Result | Duration / evidence |
| --- | --- | --- |
| `go build -buildvcs=false ./...` | PASS | 42.04s; `/tmp/synora-m132-baseline-build.log` |
| `go test -buildvcs=false -p 1 -count=1 ./...` | PASS | 81.32s; `/tmp/synora-m132-baseline-test.log` |
| Architecture checker | PASS | 7 modules, 35 owned packages, 7 declared edges, 6 observed module edges |
| Central synthetic E2E | PASS | 461/461; 460 completed, one expected rejection; 366.16s; `/tmp/synora-m132-baseline-central.json` |

The active ARM64 Go 1.24 toolchain was selected explicitly with
`GOTOOLCHAIN=local` and `GOPROXY=off`. No external dependency download was
required. Central E2E runs were kept sequential and isolated.

## Post-extraction results

| Check | Result | Evidence |
| --- | --- | --- |
| Go build | PASS | `go build -buildvcs=false ./...` |
| Full Go suite, serialized | PASS | `go test -buildvcs=false -p 1 -count=1 ./...`; includes `internal/foundation/statestore` and all cognitivecore Store, snapshot, persistence, journal/replay and compatibility tests |
| Foundation Store focused tests | PASS | Typed durable records, restart, revision, V3, action dry-run, duplicate event and WAL recovery after state-write failure |
| Architecture checker | PASS | 7 modules, 36 owned packages, 8 declared edges, 7 observed module edges; Foundation statestore owns implementation and cognitivecore is Intelligence-owned |
| Central synthetic E2E | PASS | 461/461, zero failed, 460 completed, one expected rejection; `/tmp/synora-m132-central.json`; camera mock remains not qualified and Foundation V1 remains simulated-test |
| Python worker | PASS | compileall and 108 tests; emitted expected missing-model/RKNN diagnostics in test cases |
| Native C++ tests | PASS | CMake build and CTest 2/2; build output under `/tmp/synora-m132-camera` |
| UI production build | PASS | TypeScript/Vite build from the isolated `/tmp/synora-m131-ui-final` checkout |
| Targeted Go race tests | PASS | `go test -race -buildvcs=false -p 1 -count=1 ./internal/foundation/statestore ./internal/cognitivecore`; both packages passed |

The central run took approximately six minutes, consistent with the isolated
pre-change run. The full serialized Go suite completed successfully after the
Foundation-specific tests were added. No overlapping synthetic harnesses were
used.

## Scope and limits

The authoritative Store implementation, maps, lock, journal/WAL, restore,
replay, resident mutations, outbox state, compaction, erase, and persistence
calls are physically in `internal/foundation/statestore`. The cognitivecore
Store is a compatibility adapter with cognitive V3-to-V1 projection. Module
ownership and actual package edges pass the checker. Persisted schemas,
paths, fields, and normal runtime entrypoints remain unchanged.

The compatibility evidence covers repository source and synthetic fixtures,
not unknown external callers using reflection package paths or unobserved
production state. The Go API aliases for `Commit` and `CommitResult` expose a
Foundation reflection package path. No production state was opened, migrated,
or written. Targeted race tests passed for the extracted Store and compatibility adapter; this is not a comprehensive race-freedom claim.
