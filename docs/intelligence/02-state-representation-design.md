# I1 State Representation Design Proposal

**Status:** I1 PROPOSAL; I2 V1 IMPLEMENTED / EXPERIMENTAL

**Milestone boundary:** The approved I2 implementation is an isolated versioned encoder. This does not replace existing CognitiveSnapshot V1/V2/V3 contracts or integrate a new encoder into runtime inference.

## Why the existing snapshots are not the target encoder

Current CognitiveSnapshot and its V1/V2/V3 numerical encoders are compatibility-oriented decision projections. They combine selected security, presence, sensor, episode, Vision, capability, and action-result facts. Their fixed 86/64/86 feature orders and separate snapshot schemas are consumed by existing CPU models. They do not enumerate arbitrary entities, retain every observation source/status, or encode topology and temporal history generically. Some bool/count/enum offsets rely on neighboring known/status fields; they are not a general missing-data mask.

The State Encoder must therefore be proposed as a new Intelligence-owned boundary. Reusing the existing projections as the canonical cross-domain state representation would silently discard device, source, freshness, unsupported-value, and graph semantics. No compatibility bridge is selected in I1.

## Proposed input record shape

The future external adapter may map existing contracts into an internal batch with these concepts:

    ObservationBatch (proposal)
      schema_ref
      batch/event_time and received_at kept distinct
      observations[]
        entity_ref {stable_id, kind}
        attribute_ref {namespace, name, value_type}
        value (typed scalar/category/vector) OR no value
        semantic_status {observed, unknown, unavailable, unsupported,
                         stale, conflicting}
        source_ref and provenance/evidence_ref
        observed_at, received_at, freshness/expiry policy
        confidence plus explicit meaning/calibration source when available
      relations[]
        subject_ref, relation_kind, object_ref
        validity interval, source/provenance
        optional coordinate_frame_ref for geometry

This is a field inventory, not a committed JSON schema. The canonical ID namespace, conflicting-source resolution, unit registry, event-time versus ingestion-time precedence, topology relation direction, confidence calibration, and per-attribute expiry require owner decisions before I2 freezes a representation.

## Numerical representation proposal

An encoded item should include:

1. **Stable feature order/version** and explicit shape.
2. **Typed value channels** for bounded numeric and enumerated fields; categorical identifiers map through a versioned vocabulary, not arbitrary string hashing without collision policy.
3. **Validity channels** distinguishing observed value, unknown, unavailable, unsupported, stale, conflicting, and absent/not-reported where those distinctions apply.
4. **Provenance channels/references** for source kind and evidence, without copying raw media or biometrics into the Core vector.
5. **Temporal channels** such as event-time deltas, age/freshness, and sequence order; ingestion time remains separately represented.
6. **Graph channels** referencing stable node/entity IDs and typed edges. Pixel, normalized image, PTZ, and floor coordinates must cite frames and explicit transforms.
7. **Bounded dimensions** and a deterministic overflow policy for entity count, edge count, and history length.

Unknown is never encoded as false; unavailable is never encoded as zero; an unobserved feature remains masked. The model may abstain when required context is missing.

## Proposed output contract

An experimental task output should identify:

    PredictionProposal (proposal)
      schema_ref
      task_id
      model_ref and encoder_ref
      input_observation_refs or snapshot digest
      subject/entity_ref and predicted attribute/value
      horizon_start and horizon_end (or explicit next-event horizon)
      probability distribution or calibrated score with meaning
      uncertainty/abstention and reason
      evidence_refs and provenance
      generated_at
      status {predicted, abstained, unavailable}

This output must not contain authorized, executed, or confirmed as model claims. Deterministic Core/Automation contracts own those lifecycle states. A prediction is not an observation and is not written as authoritative physical state.

## Ownership, state, failure, and test policy

The Semantic Input Adapter and numerical encoder belong to Intelligence. Their only production input dependencies should be existing Foundation-owned contracts and optional typed source adapters; Vision is one optional producer, not an initialization dependency. Encoder/model weights are immutable inference artifacts for a run. There is no home-specific state or online update in I1/I2.

Input schema mismatch, unsupported value, missing identity, invalid timestamp, unknown graph node, stale source, or conflicting observation must remain explicit and either produce a supported masked encoding or an abstention/error. It must never be silently defaulted into a confident fact. Test with golden source-to-observation mappings, missing/unknown/unavailable permutations, stable ordering, timestamp edge cases, changed device inventories, and held-out graph shapes.

## I1 proposal questions and I2 disposition

| I1 design question | I2 v1 decision or deferred work |
|---|---|
| Canonical entity and attribute references | Namespaced opaque entity/source IDs and namespaced attribute names are validated. Mapping existing Foundation IDs into this namespace is deferred; I2 adds no adapter. |
| Input fields and compatibility | `synora.observation-batch/v1` is a typed Go contract. The encoder rejects other declared schema versions. JSON decoding policy for unknown fields and future schema migration is deferred to a future adapter. |
| Time and freshness | Event time and receive time are required and distinct. Observation age is computed from event time and observation time. Future observation times fail. No stale threshold or time source precedence is inferred. |
| Values, units, categories, bounds | Number, boolean, and category are supported. Numeric values require a unit token; no unit conversion or category vocabulary is introduced. Fixed per-batch limits are recorded below. |
| Conflicting sources and confidence | Source-specific records remain separate; conflict status is explicit and no source wins automatically. Confidence is omitted pending semantics and calibration. |
| Graph relations and coordinates | Directed typed relations and optional validity intervals are preserved. Coordinate frames, geometry, topology inference, and graph feature generation are deferred. |
| Status and feature representation | Seven explicit statuses and a fixed 14-name structured feature contract are implemented; existing snapshot feature arrays remain frozen. |
| First adapter and unsupported inputs | No production or Foundation adapter is added. Unsupported value types are rejected; unsupported status is represented explicitly without a value. |

I2 implements only the versioned Intelligence-owned encoder and deterministic fixtures. It adds no persistence, user rules, online adaptation, Core integration, action proposals, or Vision dependency. Existing snapshots and model bundles remain frozen.

## I2 v1 implementation decisions

The implementation lives in `internal/intelligence/stateencoder` and provides `synora.intelligence.state-encoder/v1`. It accepts the typed `synora.observation-batch/v1` contract and returns `synora.state-encoder/v1` version `1.0.0`. These are new Intelligence-owned contracts; no adapter writes to Core or modifies the existing CognitiveSnapshot encoders.

- Entity and source IDs use a namespaced opaque string form; entity kind, attribute namespace/name, and source kind are validated independently. Attribute names and source IDs remain symbolic metadata rather than being hashed into numeric values.
- Supported values are finite numbers with an explicit unit token, booleans, and categories. Numeric values are carried as float32 in the declared unit without conversion. Categories are preserved as strings; I2 does not create or infer category vocabularies or ordinal codes.
- Every observation retains one explicit status channel: observed, unknown, unavailable, unsupported, stale, conflicting, or absent. Unknown/unavailable/unsupported/absent cannot carry a value. Stale and conflicting records may retain a source-specific value with its timestamp. The encoder does not coerce missing values, infer conflicts, or choose a winning source.
- Event time and received time are required and kept separate. An observation timestamp is required when an observed value is supplied. Age is derived as event time minus observation time; future observation times are rejected. No stale threshold is inferred and no time is synthesized.
- Relations retain caller-supplied direction, kind, source, and optional validity bounds. The encoder does not infer topology or turn edges into learned graph features.
- Limits are 512 entities, 2,048 observations, and 2,048 relations per batch. Identifiers are limited to 192 bytes; provenance/evidence references are limited to 256 bytes. Exceeding a limit is an error; inputs are never silently truncated.
- Entity, observation, and relation ordering is canonicalized. Duplicate entity IDs, duplicate entity/attribute/source observations, duplicate relations, undeclared relation endpoints, malformed tagged values, and unsupported schema versions are rejected.
- The encoded result contains a versioned 14-name feature contract: seven status channels, three declared-type channels, typed value fields, and an age field. It is a structured encoder output, not a fixed-size, vocabulary-complete model vector. Unit normalization, category embeddings/vocabularies, confidence calibration, source conflict policy, and a production adapter remain unresolved and are not claimed by I2.

Deterministic fixtures under `internal/intelligence/stateencoder/testdata/v1/` cover all seven statuses, typed values, two source-specific conflict candidates, separate event/received times, directed relations, and invalid identity input. Package tests also cover stable ordering, no silent defaulting, future timestamps, duplicate inputs, and overflow rejection. I2 remains experimental until its review report records validation and owner acceptance.
