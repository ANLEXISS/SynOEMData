# M1.3.2 Store compatibility record

## Compatibility surface

`internal/cognitivecore` continues to expose `UniversalStore`, constructors,
`Commit`, `CommitResult`, `ActionRequest`, `CompactionOptions`,
`StoreDiskReport`, `PersistenceHooks`, resident gallery operations, and the
existing read, replay, erase, compaction, and validation methods. The concrete
adapter contains one pointer to Foundation statestore and delegates storage
operations. The executable composition root still calls
`cognitivecore.OpenUniversalStore`; process boundaries and entrypoints remain
unchanged.

`Commit` and `CommitResult` are Go aliases to Foundation generic
instantiations. Field names, types, order, and JSON tags are preserved.
Nested pointer/map/slice behavior remains shallow as before. Snapshot
normalization and V3 projection remain cognitivecore responsibilities.

## Persistence compatibility

The adapter passes the existing cognitive types directly to the Foundation
JSON codec. The state and journal schemas remain `universal-store/v1`, with
existing `state.json`, `journal.jsonl`, `segments/*.jsonl`, and `erase.marker`
paths. Existing V1/V3 snapshot fields, decision/outbox fields, resident
metadata, and optional-field behavior are retained. No production state was
opened or rewritten during this change.

## Observable limits

The public `cognitivecore.Commit` and `CommitResult` names resolve through
aliases, so code relying on reflection package paths observes the new
Foundation package path. `UniversalStore` remains a distinct compatibility
wrapper. This migration does not claim compatibility for unobserved external
callers using private fields (which were never exported), unsafe layout
assumptions, or reflection metadata. Existing repository source, tests, exact
synthetic fixtures, build, and architecture validation provide the evidence
recorded in `30-m132-validation.md`.
