# I1 Cognitive Architecture and Source Audit

**Milestone:** I1 — Cognitive Architecture & Research Baselines  
**Status:** IMPLEMENTED / TESTED / EXPERIMENTAL / REVIEW REQUIRED  
**Approval:** NOT APPROVED; human review is pending  
**Source revision at start:** f64964c9fee6ed5a8460c950a118be43d7e0aa4d

This report distinguishes the current implementation from a proposed target. The target is not a production contract, and this milestone changes no production behavior.

## Current Intelligence implementation

The production path is a snapshot-driven decision pipeline, not a generalized shared cognitive runtime:

    contract.Event
        -> Cognitive Core.Process
        -> composeSnapshot
        -> SnapshotEncoder v1 (86 values)
        -> injected MLPBackend
        -> deterministic SafetyGate
        -> Decision and optional dry-run ActionRequest
        -> Foundation authoritative StateStore

Vision Evidence V1 is one optional typed input path into the snapshot. The topology package parses a node tree and links, but grep found no production Core/MLP consumer.

Core.Process checks the event envelope, rejects legacy Vision event types, validates Vision Evidence V1 where applicable, supplies Core.Now when event time is absent, projects selected facts into the prior snapshot, normalizes and validates it, encodes it, calls the injected backend, applies the deterministic gate, then commits the event, snapshot, decision, and optional dry-run action request. The durable action outbox and journal belong to Foundation through the typed Store interface. Core retains business projection and safety policy.

### Existing contracts and coupling

| Area | Current source-backed behavior | Reuse constraint |
|---|---|---|
| Input | pkg/contract.Event, camera observations, and versioned Vision Evidence V1 coexist. There is no single cross-domain observation batch contract. | Do not reinterpret an existing event or silently normalize unknown/unavailable values. |
| Snapshot and encoder | CognitiveSnapshot V1 feeds a frozen 86-value V1 feature order. V2 is a distinct 64-value projection; V3 has a distinct 86-value contract that appends selected Vision fields to its V2 base. | These are compatibility contracts for current paths, not a general entity/state encoder. Do not repurpose offsets. |
| Temporal facts | Events carry timestamp, received_at, epoch, and sequence. Core uses event time for snapshot time, fills a missing timestamp from its injected clock, and does not synthesize received_at. Store history follows commit order and de-duplicates by event ID. | Core does not generally enforce monotonic sequence, epoch transition, stale-event, or out-of-order rules. Proposed temporal semantics must be separately versioned and approved. |
| Topology | internal/intelligence/topology loads and validates a node tree and undirected neighbor links. Search found no production consumer in Core/MLP. | Node IDs/links are not yet mapped into a Core feature contract; no coordinate frames or geometric transforms are defined. |
| V1 backend | MLPBackend is injected. The CPU V1 bundle loads separately described per-head models, each consuming the fixed 86-value vector. Current heads are danger, incident, task, and action. | Inference adapter only; repository search found no cognitive training pipeline that emits these bundles. |
| V3 backend | MLPBackendV3 is separately injected. CPU V3 loads a shared backbone plus per-task heads; V3 Core is an active-dry-run candidate path. | Existing V3 is not evidence of a general encoder, temporal/topological model, promoted runtime, or production action flow. |
| Output and safety | MLP output contains predictions/confidences and an abstract action intent. V1/V3 Core gates remain deterministic; the Store forces action outbox entries to dry-run. | Confidence is not authorization. Learned outputs must not execute or confirm a physical action. |
| State ownership | Foundation owns durable state, WAL, replay, and outbox. Cognitive Core uses a typed compatibility-facing adapter. | No transient embedding or prediction may become authoritative physical state. |
| Vision/Voice | Vision is an optional source; the reduced composition can run Core without Vision. Voice remains NOT_IMPLEMENTED. | The proposed Intelligence input must be source-neutral and must not require either module. |

## Proposed target architecture

The following is an I1 design proposal only. The numerical encoder contract belongs to I2; no Go interface, schema, IPC field, or persisted format is created here.

    Existing source contracts
        -> Semantic Input Adapter
        -> Synora State Encoder (I2)
        -> Temporal Context Encoder
        -> Topological Context Encoder
        -> Optional shared representation
        -> Specialized task heads
        -> Prediction/proposal assembly
        -> Deterministic Core policy and safety
        -> Foundation StateStore / existing action lifecycle

Offline evaluation and model lifecycle manage candidate artifacts, but do not become part of the production Core data path by default. Semantic adapter output must keep source facts distinct from derived claims. The encoder should emit a versioned numerical representation with explicit validity/missingness masks and bounded feature ordering. Temporal and graph inputs remain structured until their dedicated encoders. The optional shared model returns task predictions, not authorized commands. Existing Core policy and Foundation persistence remain the authorities.

## Component specification

| Component | Input → output and responsibility | Owner; runtime/training dependencies; state | Failure behavior, testing, and estimated cost |
|---|---|---|---|
| Semantic Input Adapter | Existing Foundation Event, CameraObservation, and selected typed contracts → validated source-neutral observations with entity/source/time/status/evidence references. | Intelligence; imports Foundation contracts only; no training dependency; stateless. | Reject unsupported schemas and retain unknown/unavailable as explicit status. Contract fixture and cross-source mapping tests. Low CPU/memory; proportional to observation count. |
| Synora State Encoder | Validated observations and entity capabilities/state → versioned numeric features plus masks and feature-name/order metadata. Responsibilities include type/domain validation and stable identity mapping. | Intelligence; no Vision/Voice requirement; no trained weights required for the first encoder; stateless per call. | Return typed invalid/unsupported input, never coerce unknown to false/zero. Golden feature fixtures, property tests, round-trip semantic checks. Cost O(entities × selected features); exact width/memory is an I2 decision. |
| Temporal Context Encoder | Bounded, ordered observations with event and ingestion time → durations, transition/context representation, and masks. | Intelligence; no external runtime required initially; trained temporal weights are a later candidate; no durable state. | Missing/stale/conflicting sequence remains explicit or abstains. Test duplicates, gaps, delay, out-of-order arrival, and fixed-clock replay. Cost depends on window length; bounded window required. |
| Topological Context Encoder | Entity/node references and validated graph edges → adjacency/context representation tied to stable node IDs. | Intelligence; may reuse topology parser after a reviewed mapping; no Vision dependency; no durable state. | Unknown node, disconnected graph, or unframed geometry yields explicit unsupported/unknown output. Test held-out graph shapes and identity stability. Simple aggregation O(V+E); learned message passing O(E×d), estimate only. |
| Shared Backbone | State/temporal/topology vectors → an optional internal representation shared by tasks. | Intelligence; optional CPU inference runtime; trained weights/artifact are offline inputs; no authoritative state. | Missing, corrupt, incompatible, timeout, or unavailable model yields inference unavailable and deterministic fallback; no Store mutation. Replay, artifact-integrity, timeout, and numerical-bound tests. Cost candidate-dependent; compare with separate heads before adopting. |
| Specialized Heads | Shared or task-specific representation → typed task prediction and uncertainty. | Intelligence; same model runtime as backbone; task-specific training labels; stateless per inference. | Unsupported task, invalid scores, or low evidence results in abstention. Per-task fixtures, metrics, calibration, and failure-mode tests. Small output layers are cheap relative to backbone; no measurements yet. |
| Optional Expert/Adapter Router | Input/task context → selected optional expert or adapter set. | Intelligence; no router in the first experiment; training may require routing labels/regularization; no state. | Fixed baseline path when absent; route failure abstains. Only add after controlled evaluation shows benefit. Adds routing compute, parameters, and failure modes. |
| Cognitive Output Contracts | Task head outputs plus model/input metadata → typed predicted/inferred/abstained/unavailable records. | Intelligence; Foundation references only; no training dependency; immutable output. | Reject invalid task/value/version, retain unsupported/unknown. Schema, compatibility, provenance, and adversarial-output tests. Negligible versus inference. |
| Proposal Assembly | Compatible task outputs/evidence → a candidate cognitive proposal. Does not authorize, dispatch, or claim execution. | Intelligence; Core policy boundary consumes only after later approval; no training dependency; no persistent state. | Incompatible/missing head or conflicting claims produces no proposal/indeterminate. Test conflict and abstention. Negligible compute. |
| Evaluation Interfaces | Versioned datasets, generator provenance, model outputs → metrics, group split checks, and failure reports. | Research tooling under research/; Python standard library in I1; offline only; no household state. | Invalid split/label provenance fails evaluation. Hash fixtures, split-isolation assertions, deterministic reruns. CPU/memory proportional to dataset; I1 experiment is small. |
| Model Lifecycle Manager | Manifest, encoder/model versions, artifact hashes, runtime/device profile → load/disabled/unavailable status and safe model selection. | Intelligence; filesystem/model runtime only after separately reviewed design; no model training at runtime; no home-learning state. | Disabled by default; reject hash/version mismatch and return deterministic fallback. Test rollback, corrupt artifacts, version mismatch. Memory dominated by selected model; no production implementation in I1. |

### Ownership and specialization

- A **head** is a task-specific prediction/output component.
- An **adapter** specializes a shared representation for a task/domain.
- An **expert** is a separately parameterized model/subnetwork.
- A **router** chooses experts/heads/adapters at runtime.

I1 proposes task heads for the first three tasks. It does not select adapters, experts, or routing. Routing remains unjustified until measured benefit exists.

## Compatibility and safety boundary

I1 adds no production behavior, API, event, persistence schema, model manifest, or runtime dependency. The research script consumes only synthetic triples. Existing CPU model bundles and fixed V1/V2/V3 encoders are not modified or replaced. No real household, camera, identity, biometric, or supplier data is used. Outputs remain experimental predictions; Core authorization, Foundation state, and dry-run action behavior are unchanged.

## I3.3 structural architecture status (2026-10-11)

The later I3.3 milestone installs model-independent branch, lifecycle, representation, fusion, and output contracts in `internal/intelligence/hybrid`. This is an experimental architecture skeleton only: deterministic test fixtures are not trained models, the pipeline does not invoke cognitive heads, and no inference result is produced. Its structural fusion preserves branch representations separately and does not concatenate or learn cross-branch weights. The I1 research findings and earlier architecture proposal remain historical; this section does not revise their results. See `I3.3-TRIPLE-MLP-ARCHITECTURE.md` and `I3.3-IMPLEMENTATION-REPORT.md` for current implementation scope and review status.
