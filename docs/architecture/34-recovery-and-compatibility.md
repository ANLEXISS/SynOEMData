# M1.3.3 Recovery and Compatibility

## Persisted formats

This change preserves the existing `universal-store/v1` schema for `state.json`, active journal records, and archived journal segments. Typed snapshots, decisions, resident records, processed IDs, claims, and action outbox entries retain their existing JSON tags and optional-field behavior. No schema version was incremented, no data migration was performed, and no production data was opened or modified.

Snapshot normalization, event/decision ordering, revision assignment, and JSON serialization remain owned by the same typed contracts and StateStore. The additional `Clone` methods do not alter serialized fields. Durable history uses records decoded from the existing JSON representation; in-memory journal reads preserve the same bounded order and return detached copies.

## Open and replay

Open first handles a pending erase marker, loads and validates `state.json` if present, replays later WAL records, validates archived segments, and checks configured bounds. Replay rejects malformed JSON, unsupported schema markers, invalid zero revisions, and scanner errors. Records at or below the loaded state revision and already processed event IDs are skipped; accepted records are applied in revision order from the source logs. Duplicate `History` revisions are de-duplicated and output is sorted by snapshot revision.

A commit is written to the WAL before in-memory application and materialized-state replacement. A restart after a durable WAL append can reconstruct a commit whose state-file replacement did not finish. Re-submitting the same event ID is idempotent while the processed-ID window retains that ID. The processed-ID cache is bounded, so long-term idempotency beyond that window is not promised by this cache alone.

## Failure scenarios

| Failure point | Result and recovery |
|---|---|
| WAL open/write/sync/close error | Commit returns an error before in-memory application; a partial record, if produced by an I/O failure, is rejected on next open rather than silently repaired |
| Crash after WAL sync, before state replacement | Replay applies the WAL record after loading prior state |
| Temporary snapshot write/sync/close error | Old `state.json` remains the named file; temporary file may remain and is ignored |
| Rename succeeds but parent-directory sync fails | Caller gets an error; the new file is visible, but power-loss durability is uncertain. For outbox claim/ack, in-memory state is retained to match the visible replacement; WAL remains the recovery source for commits |
| Truncated/invalid active WAL or segment record | Open/history fails closed with a corruption/read error; no automatic truncation or salvage |
| Compaction interrupted before archive sync | Source WAL remains; a partial segment may be rejected on open |
| Compaction interrupted after segment and directory sync but before WAL removal | Both copies can be read; revision de-duplication prevents duplicate history |
| Compaction after WAL removal | Durable segment exists first; an empty active WAL is recreated and synced |
| Erase interrupted while marker exists | Open repeats removal/reset; prior journal history is not replayed |
| Outbox claim/ack state write fails before replacement | The newly changed claim is rolled back in memory; WAL/outbox entries remain authoritative for the existing protocol |
| Outbox claim/ack reports an error after rename | The in-memory claim remains aligned with the visible state file, while the caller receives the directory-sync error and power-loss durability remains uncertain. `ClaimPendingActions` also returns the claimed entries so they can be handled |

These are code-path assessments supported by unit/fault-injection and reopen tests. They are not hardware power-loss guarantees.

## Compatibility caveats

Current exported Store method signatures and persisted JSON contracts remain stable. The generic Store now has a compile-time `Clone() T` requirement, which affects external source integrations that instantiate it. Existing Cognitive Core records satisfy this requirement. Compatibility aliases still route to Foundation and preserve their prior reflection-path behavior.

The clone contract for arbitrary event payload structs can only recursively inspect exported fields. Current production event payloads inspected are JSON-friendly; custom types with unexported reference-bearing fields must be immutable or add an explicit safe cloning strategy before being stored.
