# M1.3.3 Validation Record

## Baseline and recovery

The workspace had no Git commits, no tracked files, and no configured remote. A reviewed baseline commit was therefore not possible. The existing `/home/rock/SynoraOEM-m132-recovery-20261010.tar.gz` was verified and preserved; its hash is `5afd8d22376a8f171e2e2439d3c2193642ec188e02962431378299790ae34e27`. It represents a pre-M1.3.2 state, not the current source tree.

A separate pre-implementation M1.3.3 recovery baseline was created at `/home/rock/SynoraOEM-m133-recovery-20261010.tar.gz`, SHA256 `4bbf36b1729eb2d409d4f1578cce87a2c7f331a5109c1f805b7856bd9eb6035c`. The gzip archive was validated and extracted into a temporary directory; representative Store and module ownership files matched. It contains the source, documentation, tests, and project metadata present before implementation edits, while excluding local YAML configuration, datasets, binaries/build output, runtime/cache/temp content, and generated test fixtures. To recover, extract it into a fresh directory. It intentionally does not restore excluded local configuration or generated artifacts. Neither archive was overwritten.

## Validation results

| Validation | Pre-change baseline | M1.3.3 result |
|---|---:|---:|
| Go build (`go build -buildvcs=false ./...`) | PASS, 2.137 s warm cache | PASS |
| Serialized Go suite (`go test -buildvcs=false -p 1 -count=1 ./...`) | PASS, 52.349 s | PASS |
| Architecture checker | PASS: 7 modules, 36 owned Go packages, 8 declared edges, 7 actual edges | PASS: same counts, plus Store-owner assertions |
| Python Vision suite | PASS, 108 tests, 1.094 s | PASS, 108 tests, 1.213 s |
| CMake/CTest mock suite | PASS, 2/2 | PASS, 2/2 |
| UI production TypeScript/Vite build | PASS in isolated copy | PASS in isolated copy; 1,888 modules transformed |
| Synthetic central E2E | PASS, 461/461 (460 complete + 1 expected rejection), 355.414 s | PASS, 461/461 (460 complete + 1 expected rejection), 360.640 s, 0 failed, 0 incomplete |
| Targeted Go race suite: statestore, persistence, cognitivecore, contract | Not run | PASS |
| Focused Store integrity tests | Characterization exposed aliases | PASS after fix |
| Durable concurrent commit/reopen and outbox failure rollback tests | Existing coverage | PASS |

The final central E2E used the established manifest (`manifest_sha256=d33fa1937f3c0ed23203ef350526e58853bf47ae165ae8ec7d898a50db6e68f4`) and wrote `/tmp/synora-m133-final-central.json`. It reported 461 total, 461 passed, 0 failed, 460 pipeline completions, and one expected pipeline rejection. Central E2E was run by itself to avoid overlapping synthetic harnesses.

## Synthetic Store benchmark

Fixture: 128 records with nested fact strings and metadata; the journal/history paths return the full fixture. These figures compare the same machine and benchmark command. The pre-change rows are the clean baseline captured before implementation edits. Final measurements follow the snapshot-read copy reduction.

| API | Baseline ns/op | Baseline B/op | Baseline allocs/op | Final ns/op | Final B/op | Final allocs/op |
|---|---:|---:|---:|---:|---:|---:|
| Snapshot | 290.4 | 64 | 1 | 3,158 | 1,232 | 5 |
| SnapshotV3 | 263.0 | 64 | 1 | 3,413 | 1,232 | 5 |
| Journal | 377,941 | 65,560 | 2 | 2,994,456 | 947,235 | 3,458 |
| Decisions | 7,179 | 6,552 | 2 | 405,075 | 156,057 | 514 |
| ActionOutbox | 8,317 | 8,216 | 2 | 379,779 | 157,721 | 514 |
| History (non-durable) | 333,621 | 65,560 | 2 | 3,029,287 | 947,233 | 3,458 |

The baseline intentionally measured shallow-copy reads. Its low costs included aliasing and are not an equivalent safe result. Deep copying a compound 128-record fixture allocates in proportion to its nested content. Production `ActionRequest` is scalar, while the benchmark action fixture is compound; the action row is therefore a stress bound, not a direct production estimate. Store views remain count-bounded at 512 journal/decision entries and 128 actions. Snapshot reads now avoid a redundant copy. In this stress fixture, safe snapshot reads cost about 11x the baseline latency; full Journal and non-durable History cost about 8x, Decisions about 56x, and ActionOutbox about 46x. These ratios compare aliased baseline returns with detached results, not equal-work implementations.

## Classification and limitations

All required repository validations passed. No production service, active configuration, production dataset, or `/home/rock/Synora-v1` file was changed. Ordinary tests and fsync-path inspection do not constitute power-cut testing. See `31`–`34` for ownership, protocol, compatibility, and remaining risk details.
