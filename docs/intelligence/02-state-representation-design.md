# I1 State Representation Design Proposal

**Status:** PROPOSED / NOT IMPLEMENTED  
**Milestone boundary:** Design for I2 review; this is not an I2 implementation.

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

## I2 prerequisites and proposal

Before implementing I2, review and decide:

- canonical entity and attribute references and how existing IDs map;
- the exact input observation fields and forward/backward compatibility policy;
- event-time/received-time/freshness semantics and ordering;
- supported value types, units, categorical vocabularies, and overflow bounds;
- how conflicting sources and confidence values are represented;
- graph edge semantics and any coordinate-frame/transform scope;
- output feature mask and status representation;
- which existing typed inputs are mapped in the first adapter and which remain unsupported.

Then implement only a versioned Intelligence-owned encoder and deterministic fixture suite. Keep existing snapshots and model bundles frozen. Reject unsupported inputs explicitly. Do not add persistence, user rules, online adaptation, Core integration, action proposals, or Vision dependence in I2 without separate authorization.
