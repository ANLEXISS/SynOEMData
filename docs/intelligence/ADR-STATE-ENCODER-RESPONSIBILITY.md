# ADR — State Encoder Responsibility and Temporal Fidelity

**Decision status:** OWNER DECIDED; I2 HUMAN APPROVED 2026-10-11 (EXPERIMENTAL)
**Decision scope:** Intelligence-owned `synora.state-encoder/v1` translation contrac
**I3:** SPECIFIED ONLY / NOT AUTHORIZED

## Contex

The earlier I2 implementation rejected `observed_at > event_time` and computed age using a non-negative `time.Duration`. The owner clarified that the State Encoder must transmit temporal information faithfully and must not make plausibility judgments. This decision resolves that I2.1 stop condition. It does not approve I2, authorize I3 implementation, or change Foundation/Core authority.

## Decision

> State Encoder is a deterministic, faithful translation layer.
>
> It performs technical validation and representation transformations only.
>
> It does not perform semantic interpretation, behavioral filtering, temporal plausibility assessment, source conflict resolution, or authorization.
>
> Those responsibilities belong to the cognitive models or deterministic Core policies as appropriate.

Foundation/StateStore remains authoritative for validated stored facts, event history, persistence, provenance, replay, and recovery. Cognitive models may learn and interpret relationships, temporal patterns, consistency, anomalies, and behavior. Deterministic Core/Automation retains security, authorization, safety, action lifecycle, dispatch, and execution-result verification. Predictions and proposals are not authoritative state or authorization.

## Temporal rule

The encoder preserves the declared `event_time`, `observed_at`, and `received_at` independently, including their original time-zone representations. It computes `age_seconds = event_time - observed_at` as a signed float64 seconds value. A future observation is valid under this rule and yields a negative age. A zero delta yields zero; a past observation yields a positive age. No skew tolerance, clamping, timestamp correction, freshness threshold, or status inference is applied.

This representation rule does not authorize future-data leakage. A future runtime or training input assembler must select only records actually available by its prediction cutoff, using receipt/availability provenance and an explicit cutoff policy. That assembler is not implemented in I2.1.

## Other representation rules

- The seven supplied statuses remain distinct and input-owned: `observed`, `unknown`, `unavailable`, `unsupported`, `stale`, `conflicting`, and `absent`. The encoder does not infer one from timestamps or values.
- Different source records and different-time records for the same entity and attribute are retained. Only exact duplicate observation records are rejected as duplicates. No source is preferred and no conflict is resolved.
- Number, Boolean, and category remain typed. Numeric values are narrowed to float32 only when finite and representable; a non-zero value that underflows to zero is rejected. Ordinary float32 rounding remains part of the declared representation and is documented.
- Entity identities and directed relation endpoints/kinds/validity/source are preserved. Isolated entities and cyclic directed relations are permitted. The encoder does not create, reverse, symmetrize, or infer edges.
- Technical schema, identifier, value-tag, reference, duplicate, and resource-budget violations fail the complete batch. There is no truncation or partial output.

## Consequences and limitations

The current 14-name output is a structured intermediate, not a complete neural tensor. Numeric channels include status/type masks, float32 values, and signed age seconds. Boolean values remain Boolean; categories, entity/attribute/source IDs, units, evidence, and graph endpoints remain symbolic; relations remain directed records. A model-facing projection still needs versioned category vocabularies, numeric normalization, entity mapping, graph encoding, and any temporal history/window representation. I2.1 adds none of those mechanisms.

Input limits are experimental v1 resource budgets: 512 entities, 2,048 observations, 2,048 relations, 192-byte qualified IDs, and 256-byte provenance/evidence strings. They are fixed package constants, not deployment-wide or universal SynoraOEM limits.

## Validation and status vocabulary

- **IMPLEMENTED:** signed temporal encoding, timestamp preservation, exact-record duplicate identity, representability checks, and deterministic ordering are implemented in the isolated encoder.
- **TESTED:** focused and regression commands are recorded in the I2.1 report; the test suite does not establish model quality or production readiness.
- **EXPERIMENTAL:** I2 v1 and its limits/output remain experimental pending human review.
- **DEFERRED:** input availability/cutoff assembly, learned temporal/topological reasoning, categorical vocabularies, graph-to-tensor projection, and numeric normalization.
- **NOT_IMPLEMENTED:** runtime adapter, training, inference, persistence changes, or changes to existing snapshot encoders.
- **NOT_AUTHORIZED:** I3 implementation and all later milestones.

## Approval gates

The owner approved the experimental I2 milestone on 2026-10-11. This does not authorize source commit, source push, documentation publication, or next-milestone execution. Each remains separately gated under `AGENTS.md` and `SYNORA_DOCUMENTATION_WORKFLOW.md`.
