# I1 Initial Training Task Specifications

These are research task definitions, not new product semantics. Horizons below are proposed evaluation units; before a product encoder freezes them, I2/I3 must obtain approval for source-specific time bases and label policy.

## Task A — Next-state prediction

| Field | Specification |
|---|---|
| Question | Given a validated state and its recent supporting observations, predict the next observed value/status of a selected entity attribute. |
| Input examples | Entity/type and attribute references; current typed value/status; source and event/received time; age/freshness and validity masks; relevant sensor observations and optional topology refs. Do not serialize the whole Store to text. |
| Target / horizon | The next ground-truth observation for that attribute. In a synthetic benchmark, one generated transition step. A real-time duration is domain-specific and remains open. |
| Output | Per-entity typed class/value distribution plus uncertainty and predicted/abstained/unavailable status. Output is an inference, never an observed fact. |
| Loss | Categorical cross-entropy for categorical state, or an appropriately scaled likelihood/loss for continuous state. Mask unavailable/unknown targets; do not treat them as the negative/zero class. |
| Missing-data policy | Include explicit input masks; train on observed targets only; report metrics by missingness/freshness strata. If critical inputs are absent or contradictory, abstain. |
| Metrics | Accuracy and macro-F1 for categorical values; MAE/RMSE for continuous attributes with units; Brier/log loss/calibration and risk-coverage for uncertainty. |
| Baselines | Last-state persistence, train-set majority/frequency, per-attribute transition table, and a simple linear classifier/regressor where appropriate. Evaluate on identical held-out homes. |
| Failure criteria | No improvement over last-state/transition-table control on held-out groups; excess false transitions; poor calibration; unknown/unavailable collapsed into a value; unsupported horizon semantics. |

## Task B — Temporal/topological transition prediction

| Field | Specification |
|---|---|
| Question | Predict the next transition of an entity through a declared household graph, using an ordered bounded history. |
| Input examples | Current entity/node ref, prior typed observations with event and receive times, sequence/epoch metadata when trustworthy, graph nodes and edges, source/freshness masks, and relation validity intervals. |
| Target / horizon | Next observed node/edge transition within the next 5 generated events in the first offline benchmark, including no_transition and unknown. This is an experiment proposal, not a runtime promise. |
| Output | Distribution over valid destination-node/transition labels, or abstention when the graph, node, or ordering is unknown. |
| Loss | Masked categorical cross-entropy over validated labels; no loss is assigned to unknown destination/invalid graph records. |
| Missing-data policy | Preserve gaps, delay, stale observations, event-time versus receive-time, and graph disconnection as distinct conditions. Do not reorder by assumption. |
| Metrics | Top-1 accuracy, macro-F1, reciprocal rank, Brier/log loss, calibration, and accuracy by history length, graph size, and missingness. Split by both home and graph template. |
| Baselines | Last valid transition, per-home transition frequency, a first-order transition table, and Candidate A without sequence/graph context. |
| Failure criteria | No gain on unseen homes and graph shapes; invalid-edge predictions; performance depends on leaked entity IDs; out-of-order events silently treated as ordered. |

## Task C — Behavioral anomaly scoring

| Field | Specification |
|---|---|
| Question | Rank a current state/history window by deviation from normal behavior in a declared context; do not decide that a security incident occurred. |
| Input examples | Typed current state and a bounded history window, time context, topology, event source/quality/freshness, and explicit missingness. |
| Target / horizon | A scenario/window-level anomaly annotation for the current synthetic 10-minute window. Real-world labels require independently reviewed, privacy-approved, context-specific annotations; they are not defined in I1. |
| Output | Bounded anomaly score, uncertainty, evidence refs, and abstention status. No action, identity claim, or incident confirmation. |
| Loss | Binary cross-entropy for independently labeled synthetic windows, or ranking loss if labels provide only ordering. Do not derive labels from the evaluation baseline's scoring heuristic. |
| Missing-data policy | Missing sensors are not themselves anomalies unless the separately defined target says so. Report score/abstention behavior by missingness and context. |
| Metrics | PR-AUC, recall at fixed false-positive rate, false-positive rate on rare-but-normal scenarios, calibration, and risk-coverage. |
| Baselines | Train-set anomaly frequency, context-conditioned transition frequency, and a transparent statistical deviation score whose labels are generated independently. |
| Failure criteria | Label generation is circular with the baseline; false alarms on rare normal routines; poor unseen-home precision; missing inputs increase confidence; model output is presented as a confirmed incident. |

## Shared protocol

The first experiment trains only Task A on fully observed synthetic records. It uses separate whole-home train/validation/test groups, a deterministic seed, and both majority and persistence baselines. It does not establish the missing-data policy, real-time horizon, topological semantics, or anomaly validity of Tasks B/C. Those require I2/I3 decisions and separate experiments.
