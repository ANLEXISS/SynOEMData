# M1.3.3 State Ownership Audit

## Scope and result

This audit covers the extracted M1.3.2 Store and its current OEM callers. The implementation remains the single authority under `internal/foundation/statestore`; Cognitive Core owns cognitive algorithms and typed record definitions, not Store state. No package/module ownership changed in M1.3.3.

## Authoritative state inventory

| State | Owner and constructor | Controlled mutations | Reads and persistence |
|---|---|---|---|
| Revision, current V1/V3 snapshots, bounded journal, decisions, action outbox, processed event IDs, action claims | `statestore.UniversalStore`; `NewUniversalStore` or `OpenUniversalStore` | `Commit`, `ClaimPendingActions`, `AcknowledgeAction`, `Compact`, `EraseAll` | Locked getters; JSON materialized state and append-only WAL; replay on open |
| Resident gallery records and idempotency keys | `statestore.UniversalStore`; initialized with the Store | Store resident mutation methods | Locked Store reads, validated on open, included in materialized state |
| WAL/segment files and materialized state file | Foundation persistence helpers perform filesystem mechanics; Store coordinates schema and ordering | Store transaction/lifecycle methods | `persistence.AppendSynced`, `WriteSynced`, `ReplaceSynced`, and `SyncDir` |
| Cognitive snapshot, decision, V3 snapshot and action record values | Typed definitions in `internal/cognitivecore`; values are passed into Store | Created by Cognitive Core, accepted as cloned values by `Commit` | Store returns detached typed values; contracts retain their JSON field names |

## Operation classification

- **CONTROLLED_MUTATION:** commit, resident updates, action claim/acknowledgement, compaction, and erase. State transitions occur while holding the Store write lock.
- **CONTROLLED_READ:** revision, resident lookups, bounded journal, decisions, outbox, and current snapshots. Read methods hold the read lock and now return values detached from Store-owned mutable references.
- **SNAPSHOT_EXPORT:** `Snapshot`, `SnapshotV3`, `Journal`, `Decisions`, `ActionOutbox`, `History`, and their JSON views. JSON exports own their byte buffers; typed exports own nested slices/maps/pointers through record clone methods.
- **PERSISTENCE_INTERNAL:** state serialization, WAL append/replay, segment validation, retention, and marker recovery. These do not expose the in-memory state.
- **LEGACY_COMPATIBILITY:** Cognitive Core aliases/wrappers for commit types and Store access. They route through Foundation and do not reveal the generic Store's fields.
- **UNCONTROLLED_MUTATION:** none identified in active Store consumers after this change. Before remediation, nested values from several read methods could mutate Store state.

## Locking and ordering

One `sync.RWMutex` protects Store state. Mutations, commit revision assignment, WAL append, materialized-state replacement, compaction, claim/ack, resident changes, and erase coordinate under the write lock. Read methods copy under the read lock. Replay is used while opening a newly constructed Store and takes the write lock while applying records. Compaction retains the lock across rotation so a concurrent commit cannot fall between the captured WAL and new active WAL. This keeps ordering simple at the cost of blocking writers during filesystem operations.

The commit sequence remains: validate and detach caller values; under the write lock reject duplicate IDs, assign the next revision, append and sync the WAL record, apply the record to memory, then atomically replace and sync `state.json`. A crash after WAL durability but before state replacement is recovered by replay. A returned state-write error may therefore mean the WAL (and possibly the state replacement) already contains the commit; callers should use the stable event ID for safe retry.

## External consumers and exceptions

Core, CoreV3, Foundation V1 adapters, and the central harness use Store read APIs. The production paths inspected consume values and do not rely on mutating getter results. The compatibility aliases preserve type-level integration and their historical reflection package-path differences remain described in the earlier M1.3.2 documentation.

The Store's generic typed contract now requires `Clone() T`; snapshot records also require `Normalized() T` to return an independently owned value. This is a compile-time contract for Store integrations. Event payloads are cloned recursively for common Go reference shapes and JSON-friendly exported structs. Reflection cloning preserves concrete types and cycles, but unexported reference-bearing fields in custom payload structs are copied as part of the struct and are not recursively traversed. Such payloads must be treated as immutable or provide a future explicit typed clone contract. Current production payload contracts inspected use exported/JSON-friendly data.

`Replay(events, callback)` does not commit these events or retain them: the caller supplied the input slice. Its callback receives each supplied event in order. This is an explicit callback compatibility boundary, not a Store getter.

## Remaining legacy exceptions

- Public contract records remain mutable Go values for compatibility; ownership is enforced at Store ingress/egress, not by making fields private.
- The persistence protocol is local-filesystem based and assumes one active Core owns a Store directory. There is no cross-process lock.
- Durable read/history operations can allocate proportional to returned record content. Bounded in-memory views cap journal/decision/outbox counts at 512/512/128.
