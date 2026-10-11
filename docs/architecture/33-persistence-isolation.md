# M1.3.3 Persistence Isolation

## Dependency direction and responsibilities

`internal/foundation/statestore` imports `internal/foundation/persistence`. Persistence does not import StateStore, Cognitive Core, or the domain record packages. The architecture checker continues to require Foundation ownership of StateStore and Intelligence ownership of Cognitive Core.

StateStore owns the authoritative records, typed JSON schema, transaction sequencing, WAL record format, replay semantics, state restoration, compaction lifecycle, outbox consistency, resident invariants, and erase protocol. Persistence owns reusable filesystem mechanics: complete writes, append plus file sync, temporary-file replacement plus file and directory sync, and directory sync/error propagation. This isolates mechanics without splitting transaction authority or inventing a repository framework.

## Durable write protocol

For a commit, StateStore marshals the typed `universal-store/v1` record and appends one newline-delimited JSON record to `journal.jsonl`. `AppendSynced` opens in append mode, writes the complete buffer (handling short writes), calls file `Sync`, and checks close errors. StateStore then applies the value under its lock and serializes materialized state. `ReplaceSynced` writes a same-directory temporary file, writes all bytes, syncs and closes it, renames it to `state.json`, and syncs the containing directory. Errors are propagated.

The WAL append syncs file contents. On first journal creation the WAL helper does not sync the parent directory itself; the successful subsequent state replacement syncs the Store directory, persisting both directory entries. If that state replacement fails before directory sync, the first journal name may not be power-loss durable even though its file was synced. The implementation does not claim stronger guarantees for that failure case. A directory-sync error after snapshot rename is explicitly distinguishable: the new file is visible, but its power-loss durability is uncertain.

## Compaction and retention

Under the Store write lock, compaction first persists current materialized state, reads and validates the active journal, creates `segments/`, syncs the Store directory for that directory entry, writes and syncs an immutable segment, syncs `segments/`, then removes the active journal and creates/syncs an empty active journal. Thus the archive is durable before the source WAL is removed. Retention removes eligible segments and syncs the segment directory. Recovery reads both archived segments and the active WAL, de-duplicates by revision, and sorts history in revision order.

## Erase and outbox lifecycle

Erase writes and syncs `erase.marker`, syncs the Store directory, removes the WAL and archived segments, resets in-memory state, persists the empty materialized state, removes the marker, and syncs the directory. If interrupted with a marker present, open repeats the history clear and state reset before removing the marker. Outbox claim/ack changes are persisted with the same atomic state replacement. A write failure before replacement rolls back the new in-memory claim; an error after rename retains the mutation so memory agrees with the visible state file, while reporting the uncertain directory durability. `ClaimPendingActions` returns its claimed records with that post-replacement error so the caller can process the committed claim.

## Limitations

- File and directory `fsync` semantics depend on the operating system and filesystem. No device power-cut test was performed.
- The Store holds its write lock across filesystem operations to preserve serialization and ordering; slow storage blocks other Store operations.
- There is no cross-process Store-directory lock; the ownership model expects one active Core per directory.
- A partial/truncated WAL line is rejected during open; recovery fails closed and does not silently truncate or salvage it.
- If rename succeeds but directory sync reports an error, the replacement may already be visible. The caller receives an error, while recovery uses the WAL as its source of truth.
- On a newly created Store directory, directory-entry durability for the directory itself is not explicitly synced at creation time by `OpenUniversalStore`.
