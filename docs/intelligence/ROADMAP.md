# Synora Intelligence Roadmap

This roadmap records milestone state; implementation, approval, and publication each require their separate authorization gates. I3 implementation is complete locally and awaits human review.

| Milestone | Description | Status | Prerequisites / deliverables | Tests and known blockers | Review / approval / revision |
|---|---|---|---|---|---|
| I1 | Cognitive architecture and research baselines | IMPLEMENTED / TESTED / EXPERIMENTAL; HUMAN APPROVED | Source audit, component proposal, task objectives, candidate comparison, isolated next-state experiment and report | Required regression commands run. Current full and targeted central runs complete episode-sequence-03; the previously reported incomplete action-result observation was not reproduced. Synthetic model result is not household evidence. | Owner approved after the I1 final-review clarification; approval applies to the experimental milestone, not production readiness. |
| I2 | Structured State Encoder | IMPLEMENTED / TESTED / EXPERIMENTAL; HUMAN APPROVED 2026-10-11 | Versioned observation-batch contract, Intelligence-owned encoder, explicit status/type channels, signed temporal deltas, deterministic fixtures and implementation report | I2.1 aligns translation-only responsibility and preserves future timestamps. Validation is reported in I2.1-CONTRACT-STABILIZATION.md. Output remains structured, not a complete neural tensor. | I2 approval itself did not authorize source commit, push, documentation publication, or I3 execution. The owner later authorized I3 as recorded in the I3 row. |
| I3 | Cognitive Dataset Engine | IMPLEMENTED / TESTED / EXPERIMENTAL; REVIEW REQUIRED | Deterministic synthetic generator, five separated artifact directories, receipt-time cutoff selection, whole-home/topology-family splits, five task labels, 20 scenario stimuli, CLI and reproducibility checks | I3.1 stopped at a task-contract ambiguity: A/B do not identify the target entity/attribute; full qualification, baselines, and adversarial audit remain incomplete. See `I3.1-DATASET-QUALIFICATION-REPORT.md`. | I3 execution was authorized after I2 approval. I3 is not approved; no source commit/push or I3 documentation publication is authorized. I4 remains unauthorized. |
| I3.3 | Triple-MLP hybrid architecture skeleton | STRUCTURAL CONTRACTS IMPLEMENTED / TESTED / EXPERIMENTAL; HUMAN APPROVED 2026-10-11 | Event, Topo-Temporal, optional Home interfaces; lifecycle registry; versioned representations; deterministic structural fusion; typed head contracts; no trained inference or training | Full validation results and limitations in `I3.3-IMPLEMENTATION-REPORT.md`. I3.1/I3.2 semantics and dataset work remain DEFERRED; no mappings fabricated. | Owner priority override authorized I3.3 implementation and the owner approved the structural milestone on 2026-10-11. No commit, push, publication, or I4 authorization. |
| I4 | Shared Backbone and Specialized Heads | PLANNED / NOT AUTHORIZED | I2/I3 acceptance; compare A/B and per-task controls | Must beat simple baselines on held-out groups; quantify negative transfer, uncertainty, and resource use. No production replacement. | Not reviewed or approved. |
| I5 | Cognitive Evaluation and Qualification | PLANNED / NOT AUTHORIZED | I4 candidates and evaluation suite | Needs calibration, abstention, unseen-home/graph, missingness, and failure-mode evidence. | Not reviewed or approved. |
| I6 | Home Adaptation and Continual Learning | PLANNED / NOT AUTHORIZED | I5 evidence plus privacy, retention, opt-out, and rollback decisions | No home-specific training state or adaptation is implemented. | Not reviewed or approved. |
| I7 | Cognitive Runtime / Core Integration | PLANNED / NOT AUTHORIZED | Approved model/output contracts, runtime/fallback and lifecycle design | Runtime must stay disabled by default, preserve Foundation authority and existing event/store compatibility, and pass deterministic fallback tests. | Not reviewed or approved. |
| I8 | Automation and Decision Integration | PLANNED / NOT AUTHORIZED | I7 approval plus proposal/authorization semantics | Models may propose only; deterministic authorization retained; physical execution is not authorized here. | Not reviewed or approved. |
| I9 | Edge Runtime Optimization | PLANNED / NOT AUTHORIZED | Qualified model/runtime and authorized target hardware | Requires actual board-specific latency, memory, thermal and backend evidence; desktop estimates are not qualification. | Not reviewed or approved. |

## I1 review decision

I1 was explicitly approved by the owner after the final-review clarification in I1-REVIEW-REPORT.md. The approval is for the experimental milestone only. The synthetic result is not household evidence, and the decision does not authorize later milestones by itself.

## I2 review gate

I2 and I2.1 were approved by the owner on 2026-10-11 for the experimental milestone. The owner-decided temporal rule is documented in `ADR-STATE-ENCODER-RESPONSIBILITY.md`; event, observation, and receipt times stay distinct and age is signed. That approval did not authorize a source commit, push, documentation publication, production integration, or I3 execution; the owner subsequently authorized I3. The encoder remains isolated from Core runtime behavior and frozen snapshot/model contracts.

## I3 implementation boundary

`I3-DATASET-ENGINE-SPECIFICATION.md` and `I3-SCENARIO-CATALOG.md` record the implemented research scope and remaining limitations. The generator applies causal cutoff selection separately from State Encoder translation so a late record cannot enter a prediction input merely because its source timestamp is old. Model training and I4 remain NOT AUTHORIZED.

I2 source and documentation remain subject to the repository's milestone validation and documentation synchronization workflow. Source and documentation-repository revisions are reported separately when recorded.
