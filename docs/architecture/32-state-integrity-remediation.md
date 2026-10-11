# M1.3.3 State Integrity Remediation

## Aliasing inventory and corrections

| API/path | Previously exposed references | Remediation |
|---|---|---|
| `Snapshot()` | Generic snapshot values could retain nested refs; Cognitive V1 slice fields were mutable | Snapshot records require `Clone`; Cognitive V1 normalization returns copied slices; the Store's `Normalized` contract explicitly requires independent ownership |
| `SnapshotV3()` | Evidence pointers, nested priority reasons/runtime aggregate, and projection facts slice | `CognitiveSnapshotV3.Clone` copies those nested references |
| `Journal()` / non-durable `History()` | Outer slice was copied, but event payload, snapshots, decisions, V3 values, and action pointers could alias | `Journal` and non-durable `History` use `cloneCommit`; durable `History` decodes independent records from disk |
| `Decisions()` | Outer slice copied; nested decision slices/maps and trace pointers remained shared | Typed `Decision.Clone` and `DecisionV3.Clone` copy nested slices/maps and MLP trace structures |
| `ActionOutbox()` / `ClaimPendingActions()` | Outer slice copied; generic action records could carry references | Action records require `Clone`; getters and claim results clone each record |
| `CommitResult.Action` | Returned action could share caller/store references | Result action is cloned |
| `Commit` ingress | Caller could mutate nested records after submission | Event, snapshots, decisions, optional V3 records, and actions are cloned before validation/application |
| Event payloads | Nested maps/slices/pointers could be retained by reference | `Event.Clone` recursively copies common map/slice/pointer shapes, preserving concrete types and cycles |

The characterization test was run before remediation and demonstrated that nested getter mutations changed Store-owned values. It is now a regression test asserting independent records, including caller-input mutation after commit.

## Controlled mutation interface

No broad mutation API was introduced. Existing `Commit`, resident mutation methods, outbox claim/acknowledgement, compaction, and erase remain the only state-changing entry points. The remediation adds a typed clone contract to the generic Store constraints, not an untyped payload map or reflection-based Store schema.

Outbox claim and acknowledgement roll back their in-memory claim change when state persistence fails before replacement. If rename has installed the new state but directory sync fails, the Store keeps the claim aligned with the visible file. `ClaimPendingActions` also returns those claimed entries alongside the durability error so the caller can still handle them. Commit behavior intentionally preserves write ordering: the WAL is durable before applying state and the materialized state is replaced afterward. If state persistence fails after WAL append, replay can still recover the accepted WAL record.

## Compatibility

Exported Store method signatures, JSON names, persisted schema version (`universal-store/v1`), field omission rules, and record ordering remain unchanged. Generic Store implementations must now implement `Clone() T`; this is a source compatibility requirement for any external instantiation of the generic Store. In-repository record types implement it. No persisted migration is required.

Event payload clone behavior preserves Go concrete types instead of round-tripping through JSON, avoiding numeric type changes. Recursive cloning handles maps, slices, and pointers with cycle tracking. Struct fields that reflection cannot safely inspect (unexported fields) remain shallow within that struct; payload contract authors must not place mutable shared references there.

## Cost and tradeoff

Deep copies are bounded by the returned value size and Store limits. This costs more than exposing shared references, but makes concurrent caller mutation independent of Store locks. The synthetic stress fixture benchmark is reported in `35-m133-validation.md`; it deliberately uses 128 compound records and is substantially more allocation-heavy than a scalar action record. Snapshot getter copying was reduced to one independent normalization copy by making that requirement explicit in `SnapshotRecord`.

No production consumer was found that intentionally mutates a getter result to update Store state, so no consumer migration or new mutation API was necessary.
