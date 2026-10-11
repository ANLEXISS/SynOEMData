# Synora Intelligence Roadmap

This roadmap describes planned milestones; it is not authorization to implement them. The owner must approve I1 before I2 begins.

| Milestone | Description | Status | Prerequisites / deliverables | Tests and known blockers | Review / approval / revision |
|---|---|---|---|---|---|
| I1 | Cognitive architecture and research baselines | IMPLEMENTED / TESTED / EXPERIMENTAL; REVIEW REQUIRED | Source audit, component proposal, task objectives, candidate comparison, isolated next-state experiment and report | Required regression commands run. Current full and targeted central runs complete episode-sequence-03; the previously reported incomplete action-result observation was not reproduced. Synthetic model result is not household evidence. | Human approval pending; no approval date, source commit, or documentation commit assigned. |
| I2 | Structured State Encoder | PLANNED / NOT AUTHORIZED | Approved observation/entity/value/status contract; versioned encoder, masks, mappings, and fixtures | Blocked on owner decisions listed in 02-state-representation-design.md; must preserve unknown/unavailable/stale distinctions and frozen V1/V2/V3 contracts. | Not reviewed or approved. |
| I3 | Cognitive Dataset Engine | PLANNED / NOT AUTHORIZED | I2 contract approval; deterministic synthetic worlds, provenance, group splits, leakage controls | Needs independent labels, home/graph grouping, privacy policy, and anomaly-label validation. | Not reviewed or approved. |
| I4 | Shared Backbone and Specialized Heads | PLANNED / NOT AUTHORIZED | I2/I3 acceptance; compare A/B and per-task controls | Must beat simple baselines on held-out groups; quantify negative transfer, uncertainty, and resource use. No production replacement. | Not reviewed or approved. |
| I5 | Cognitive Evaluation and Qualification | PLANNED / NOT AUTHORIZED | I4 candidates and evaluation suite | Needs calibration, abstention, unseen-home/graph, missingness, and failure-mode evidence. | Not reviewed or approved. |
| I6 | Home Adaptation and Continual Learning | PLANNED / NOT AUTHORIZED | I5 evidence plus privacy, retention, opt-out, and rollback decisions | No home-specific training state or adaptation is implemented. | Not reviewed or approved. |
| I7 | Cognitive Runtime / Core Integration | PLANNED / NOT AUTHORIZED | Approved model/output contracts, runtime/fallback and lifecycle design | Runtime must stay disabled by default, preserve Foundation authority and existing event/store compatibility, and pass deterministic fallback tests. | Not reviewed or approved. |
| I8 | Automation and Decision Integration | PLANNED / NOT AUTHORIZED | I7 approval plus proposal/authorization semantics | Models may propose only; deterministic authorization retained; physical execution is not authorized here. | Not reviewed or approved. |
| I9 | Edge Runtime Optimization | PLANNED / NOT AUTHORIZED | Qualified model/runtime and authorized target hardware | Requires actual board-specific latency, memory, thermal and backend evidence; desktop estimates are not qualification. | Not reviewed or approved. |

## I1 review gate

I1 implementation and experiment are complete for review, not approved. A prior central report recorded episode-sequence-03 as incomplete. A fresh full suite and a targeted rerun now show all required stages, including action_result_recorded, with pipeline accounting valid. Review the architecture report, representation proposal, candidate comparison, task definitions, experiment results, and remaining design decisions before deciding whether I1 is approved or requires revision.

No commit or publication SHA is recorded. Any such SHA must be added only after separate explicit authorization and completion of that action.
