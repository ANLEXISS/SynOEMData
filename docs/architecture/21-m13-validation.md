# M1.3 validation record

## Baseline provenance

The checkout had no Git metadata. Local Git metadata was initialized in
SynoraOEM only; no remote or global configuration was changed and no baseline
commit was made because the original source contents and generated assets
could not be distinguished safely from earlier milestone work.

The M1.2 validation record reports a serialized full Go suite and build passing
after the M1.2 package moves, plus architecture checks, central synthetic E2E,
Python worker tests, C++ mock tests and UI build. It also records a separate
default-parallel central camera mock failure (5/6) and documents that the
serial final run passed. That failure is retained as a known
`TEST_FLAKINESS`/environment-sensitive baseline; no test was edited to hide it.

An attempted M1.3 pre-change run overlapped with source edits and a second
central harness, so its result is not used as a clean baseline. During the
first post-change serialized run, the central package had a Le2i snapshot
fixture mismatch and a write after a temporary test directory was cleaned.
Both affected central tests passed independently; a second full serialized run
passed. The full suite was rerun once more after the final State injection
compatibility change and also passed. The first failures are classified
`TEST_FLAKINESS`/`ENVIRONMENT_LIMITATION` due to harness overlap/load, not an
extraction regression. The standalone 461-case central run also passed.

## Final results

| Check | Result | Notes |
| --- | --- | --- |
| Go build | PASS | `go build -buildvcs=false ./...` |
| Full Go tests, serialized | PASS | Final `go test -buildvcs=false -p 1 -count=1 ./...`; all packages passed |
| Architecture checker | PASS | `modules=7`, `owned_go_packages=34`, `declared_edges=7`, `actual_module_edges=6` |
| Cognitive / Store / persistence tests | PASS | Cognitive package and new atomic persistence helper tests passed |
| Snapshot compatibility | PASS | V2/V3 golden vectors, immutable V2 prefix, restart/replay, exact canonical snapshot equality across compaction, corruption rejection and erase recovery passed |
| Central synthetic E2E | PASS | 461/461 passed; 460 complete, one expected rejection, zero incomplete. Report: `/tmp/synora-m13-final-central.json` |
| Go race detector | PASS | `-race` passed for `internal/cognitivecore` and `internal/foundation/persistence` on linux/arm64 |
| Python worker tests | PASS | 108 tests passed; runtime initialized locally, no camera was opened |
| C++ mock tests | PASS | 2/2 tests in `/tmp/synora-m13-camera` |
| UI build | PASS | Vite/TypeScript production build passed in a temporary copy under `/tmp` |

The central report labels camera mock E2E `not_qualified` and Foundation V1
`simulated_test`; these are expected qualification labels for this synthetic
run, not hardware or production qualification.

## Scope limit

This milestone adds an injected cognitive state interface and extracts the
atomic materialized-state replacement operation. It does not claim that the
authoritative maps, journal serialization, recovery and persistence policy
have moved out of `internal/cognitivecore`. This preserves existing exported
Go composition fields and persisted field names while documenting the safe
next boundary. Therefore M1.3's full package-level Foundation/Intelligence
separation acceptance remains incomplete; section 17 records the compatibility
blocker and the remaining state-reference limitations.
