# I2 — Structured State Encoder Implementation Repor

**Status:** IMPLEMENTED / TESTED / EXPERIMENTAL; HUMAN APPROVED
**Authorization:** I2 implementation authorized after I1 approval
**Source baseline:** `9b25f24` (`I1`); I2 source changes are local and uncommitted.
**Human approval:** Owner explicitly approved I2 on 2026-10-11. Approval is for the experimental milestone only; it does not imply production readiness or authorize commit, push, documentation publication, or I3 execution.

## 1. Scope

I2 adds an isolated structured observation contract and deterministic encoder at `internal/intelligence/stateencoder`. It is registered as `synora.intelligence.state-encoder/v1` and consumes `synora.observation-batch/v1`, producing `synora.state-encoder/v1` version `1.0.0`.

The package does not import Core, Foundation, Vision, or model packages. It adds no persistence, online adaptation, production input adapter, runtime integration, action proposal, or model invocation. Existing CognitiveSnapshot and V1/V2/V3 feature orders, bundles, and behavior are unchanged.

## 2. Contract and encoding behavior

- A batch requires separate event and received timestamps, declared entities, and at least one observation. Entity and source IDs are opaque namespaced strings with validated length and syntax. Attribute namespace, name, value type, and unit are explicit.
- Values support finite numbers, booleans, and category strings. Numeric values require a unit token and are carried as float32 without unit conversion. Booleans and category strings remain typed; categories are not hashed or assigned implicit ordinal values.
- Each observation has an explicit status: observed, unknown, unavailable, unsupported, stale, conflicting, or absent. Non-value statuses cannot be defaulted to false or zero. Observed records require a value and observation time. Stale/conflicting records can preserve source-specific retained values and timestamps.
- Source and evidence references are preserved. Multiple sources and multiple timestamps for one entity/attribute remain separate; only exact duplicate observation records are rejected. The encoder neither infers conflicts nor chooses a winner.
- Event, observation, and received times remain separate and are preserved as declared. Age is the signed `event_time - observed_at`; future observation times are retained with a negative age. No freshness threshold, timestamp correction, or status is inferred.
- Relations retain caller-supplied direction, type, source, and validity interval. The encoder does not infer missing edges or produce learned graph features.
- The versioned feature order contains seven status bits, three declared-type bits, typed value fields, and observation age. IDs, category text, units, source, evidence, and relations remain named structured data, not hashed numeric features.
- Per-batch limits are 512 entities, 2,048 observations, and 2,048 relations. Identifier length is at most 192 bytes; provenance/evidence references are at most 256 bytes. Overflow and malformed input return errors without truncation or default substitution.
- Entities, observations, and relations are sorted canonically. The encoder does not mutate its input. Directed cycles and isolated entities are preserved.
- Numeric inputs are checked for finite float32 range; non-zero values that underflow to float32 zero are rejected. Representable float64 values can round when narrowed, as specified by this experimental representation.

The complete field contract and deliberate limitations are recorded in `02-state-representation-design.md`.

## 3. Files added or changed

- Added `internal/intelligence/stateencoder/types.go` and `encoder.go`.
- Added `internal/intelligence/stateencoder/encoder_test.go`, `i21_test.go`, and v1 fixtures under `internal/intelligence/stateencoder/testdata/v1/`.
- Registered package ownership and the provided interface in `modules/package-ownership.json` and `modules/intelligence.json`.
- Updated the I2 implementation details in `02-state-representation-design.md` and milestone status in `ROADMAP.md`.

No production services, Core contracts, existing snapshot encoders, model artifacts, or device integrations were changed.

## 4. Validation

- `go test ./internal/intelligence/stateencoder` — PASS.
- `go test -buildvcs=false -p 1 -count=1 ./...` — PASS across all Go packages.
- `make architecture-check` — PASS; seven logical modules, 39 owned Go packages, nine declared module edges, eight actual module edges. The check enforces state encoder ownership and prevents its imports from reaching Core/Foundation contracts or runtime packages; Python RKNN import isolation also passed.
- Tests cover all seven supplied statuses, number/boolean/category types, source-specific conflict candidates, distinct event/observation/receipt times, positive/zero/negative age, future timestamps, nanoseconds, wide and overflowing timestamp deltas, delayed receipt, replay, same-attribute records across sources/times, deterministic ordering, full input immutability, directed/cyclic relations, isolated entities, topology changes, invalid values/references, numeric boundaries, exact resource limits, and overflow rejection. I2.1-specific results are recorded in `I2.1-CONTRACT-STABILIZATION.md`.

## 5. Limits and unresolved work

- This output is a structured encoding contract, not a fixed-size model-ready vector. Category vocabularies/embeddings and normalized numeric ranges require separate versioned decisions.
- Unit conversion/registry, confidence semantics/calibration, automatic source conflict policy, canonical mapping from existing Foundation contracts, and per-attribute freshness policy are not implemented.
- No production adapter, Core integration, persistence, temporal-history window, topology reasoning, model training, household dataset, or runtime performance qualification was performed.
- The output remains a structured intermediate, not a fixed-size model-ready tensor. Causal input availability/cutoff selection and temporal history are deferred. I2 remains experimental and awaits human review. I3 is NOT AUTHORIZED.

## 6. Approval state

**I2 approval: APPROVED by the owner on 2026-10-11.** No source commit, tag, merge, documentation publication, or production rollout is performed or implied by this approval.

## 7. I2.1 stabilization status

The owner resolved the earlier temporal-policy stop with the decision that the State Encoder is translation-only. I2.1 now preserves all declared timestamps, accepts future observation times, and represents signed age without clamping. The broad entity/attribute/source duplicate rule was replaced with exact-record duplicate rejection so records from different times or sources remain representable. Full input immutability is now asserted by serializing the complete input before and after encoding. See `ADR-STATE-ENCODER-RESPONSIBILITY.md` and `I2.1-CONTRACT-STABILIZATION.md`.

I2 is `IMPLEMENTED / TESTED / EXPERIMENTAL; HUMAN APPROVED` as of 2026-10-11. The I3 engine and scenario catalog are specification-only. I3 implementation remains NOT AUTHORIZED.
