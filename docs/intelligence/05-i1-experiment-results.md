# I1 Minimal Learning Experiment Results

**Status:** EXPERIMENTAL / REPRODUCIBLE; **I1 human review: APPROVED.**  
**Experiment:** `synora-i1-next-state-mlp/v1`  
**Implementation:** `research/i1/next_state_experiment.py` (Python standard library only)  
**Result artifact:** `research/i1/results.json`

## Question and protocol

The experiment asks whether a small learned model can recover a next-state relation from structured synthetic observations and outperform simple baselines on homes not used for training. It tests that narrow question only.

The generator creates 60 synthetic homes with 128 sequential records each (7,680 total). For each home it independently samples a door-open probability uniformly from 0.18 to 0.62, a motion probability uniformly from 0.28 to 0.72, and an initial binary occupancy state. At each step it samples the two sensor bits from that home's probabilities. It sets `next_occupancy = current_occupancy XOR (door_open AND motion)`, then independently flips that target with probability 0.02. The resulting target becomes the current occupancy at the next step. The transition law is identical for every home; only sampled rates, initial state, and resulting sequences differ.

## Exact input and model

The 31-parameter model receives exactly three ordered binary features on each record:

1. `current_occupancy_bit` — the current state, also the preceding generated record's target within a home.
2. `door_open_bit` — whether the generated door sensor is open at this step.
3. `motion_bit` — whether generated motion is present at this step.

The network is a fully connected 3-input, 6-unit hidden layer with `tanh` activations and one `sigmoid` output representing the probability that `next_occupancy_bit` is 1. Parameter count is `3×6 + 6 + 6 + 1 = 31` (input-to-hidden weights, hidden biases, hidden-to-output weights, output bias). It is trained with binary cross-entropy and deterministic mini-batch SGD (batch 32, learning rate 0.08, L2 0.0001), up to 180 epochs. The checkpoint with the lowest validation log loss is retained; early stopping selected epoch 91.

The record also contains `home` and `step` for generation and split bookkeeping. Neither is passed to the model. There are no timestamps, device/entity identifiers, graph/topology features, source/freshness indicators, missing-value indicators, or history-window vectors in its input. The only state/history information passed is the current occupancy bit.

## Home split and held-out variation

The split is by contiguous ranges of whole home IDs, with no home or within-home sequence shared between splits:

| Split | Homes | Records | Home IDs |
|---|---:|---:|---|
| Train | 36 | 4,608 | `synthetic-home-000`–`synthetic-home-035` |
| Validation | 12 | 1,536 | `synthetic-home-036`–`synthetic-home-047` |
| Test | 12 | 1,536 | `synthetic-home-048`–`synthetic-home-059` |

Held-out homes have independently sampled door-open and motion rates, initial occupancy states, sensor draws, and noisy generated sequences. The sampled per-home rate ranges were:

| Split | Door-open probability range | Motion probability range |
|---|---:|---:|
| Train | 0.199593–0.619924 | 0.301575–0.718794 |
| Validation | 0.183577–0.610798 | 0.288269–0.706933 |
| Test | 0.186551–0.593573 | 0.280342–0.716899 |

These ranges overlap substantially and all were sampled from the same configured ranges (`0.18–0.62` for door-open and `0.28–0.72` for motion). Their targets use the same transition rule as training homes. The test is therefore a whole-home group holdout under one shared synthetic mechanism, not a test of unseen rules, household populations, topology, or deployment conditions.

## Label leakage and target coupling

There is no direct target-column leakage: `label`/`next_occupancy_bit`, future records, home ID, and step index are not model features. The current occupancy feature is the preceding state in the generated sequence; using it to predict the next state is part of the stated forecasting task, rather than access to the current record's target.

There is, however, strong intentional feature-to-label coupling. The target is computed directly from the three input bits by `current XOR (door AND motion)`, then independently flipped only 2% of the time. Thus the inputs almost completely determine the label under the generator's rule. The same rule is used in every split, so the test measures whether training recovers that synthetic rule under shifted per-home sensor frequencies. It does not establish learning of household semantics or generalization to independently defined target mechanisms. No separate leakage stress test or alternative-rule test was performed: **NOT MEASURED**.

## Test-set class distribution and metrics

The test set has 800 class-0 targets and 736 class-1 targets (1,536 total). The table below reports one-vs-rest precision, recall, and F1 by target class. These values are calculated from the saved confusion counts in `results.json`; the experiment artifact itself records confusion counts and aggregate macro-F1, not a per-class metric table.

| Predictor | Class | Support | Precision | Recall | F1 |
|---|---:|---:|---:|---:|---:|
| Learned MLP | 0 | 800 | 0.988622 | 0.977500 | 0.983030 |
| Learned MLP | 1 | 736 | 0.975839 | 0.987772 | 0.981769 |
| Last-state persistence | 0 | 800 | 0.832704 | 0.827500 | 0.830094 |
| Last-state persistence | 1 | 736 | 0.813765 | 0.819293 | 0.816520 |
| Train-majority predictor | 0 | 800 | 0.520833 | 1.000000 | 0.684932 |
| Train-majority predictor | 1 | 736 | 0.000000 | 0.000000 | 0.000000 |

| Predictor | Accuracy | Macro-F1 | Brier score | Log loss | Correct / total |
|---|---:|---:|---:|---:|---:|
| Learned 3-6-1 MLP | 0.982422 | 0.982399 | 0.017314 | 0.089348 | 1,509 / 1,536 |
| Last-state persistence | 0.823568 | 0.823307 | 0.176432 | 4.875006 | 1,265 / 1,536 |
| Train majority class | 0.520833 | 0.342466 | 0.479167 | 13.239864 | 800 / 1,536 |

The majority predictor emits class 0 for every test record because class 0 is the more common training label. Its class-1 precision is defined as zero here when it makes no class-1 predictions, matching the experiment's zero-denominator convention for F1.

## Interpretation of 98.24% and the persistence baseline

The 98.24% accuracy means 1,509 of these 1,536 generated test records were classified correctly at a 0.5 probability threshold. It is a result on one fixed, reproducible synthetic test set whose target is generated from the same simple rule used to create training labels. It is not an estimate of real-home accuracy, future product performance, or cognitive capability. The independent 2% target-flip process also means perfect rule recovery would not imply perfect accuracy on every finite sample.

Last-state persistence predicts the next occupancy equals current occupancy, ignoring door and motion. It gets 1,265 records right (82.36%) because many generated transitions leave occupancy unchanged; it fails on transitions where the door-and-motion conjunction toggles state, as well as on independently flipped targets. Its confusion counts are TN 662, FP 138, FN 133, TP 603. The MLP improves by 0.158854 absolute accuracy on this test set by using the two sensor bits and learning their conjunction with current occupancy. This comparison supports recovery of the generator's relation; it does not measure temporal forecasting over a history window.

## What this result demonstrates

- A small model can be trained from three structured numeric bits to approximate a deliberately simple synthetic next-state rule.
- With the fixed seeds and whole-home split, this MLP outperforms the selected constant-majority and current-state persistence baselines on the generated held-out records.
- The standard-library experiment and saved dataset/model hashes make the recorded run reproducible in the checked environment.

## What remains unproven

- **Temporal reasoning:** NOT DEMONSTRATED. Records are generated sequentially, but the model sees no ordered history, timestamps, duration, event timing, recurrence, delay, concurrency, or temporal encoder. A current-state bit and sequential data generation alone do not show learned temporal reasoning.
- **Topological reasoning:** NOT DEMONSTRATED. The model sees no graph, edges, adjacency, coordinates, movement, or topology variation. Home IDs are grouping metadata only.
- Generalization to real households, different transition rules, real sensor distributions, unseen device types, or independently generated environments.
- Semantics, anomaly detection, missing/unknown/unavailable/stale values, source conflicts, calibration guarantees, privacy approval, production integration, or runtime quality.
- Embedded inference time, peak memory, quantization, or accelerator performance. Peak memory was not measured; the reported roughly 8.8-second run time is environment-specific experiment time, not an embedded benchmark.

## Hypotheses for later milestones (not authorization)

These are questions for future approved work, not findings or approval to begin I2/I3.

- **I2 — structured state representation:** Test whether explicit typed value and status masks preserve known, unknown, unavailable, stale, and conflicting observations without collapsing them to a boolean/default. Specify identity, source, time/freshness, units, and bounded-cardinality semantics, then test encoder behavior and feature leakage. I1 did not measure these questions.
- **I3 — dataset and evaluation:** Test whether conclusions survive multiple independently parameterized transition mechanisms and controlled domain shifts. Keep homes, sequences, and scenario/template families disjoint at split time; derive targets independently from features where appropriate; compare against persistence and rule-based baselines; include missing, stale, delayed, conflict, failure, and imbalanced-label cases. State synthetic limits and assess leakage explicitly. I1 did not measure these questions.

## Reproduction and artifact integrity

From the repository root, reproduce without overwriting the preserved result artifact:

    python3 research/i1/next_state_experiment.py > /tmp/synora-i1-reproduced.json
    cmp /tmp/synora-i1-reproduced.json research/i1/results.json

The experiment uses fixed data/model seeds (104729 and 130363). Dataset SHA-256: `062d5119b74a468d8e85449b95ee70dd32d1a8301f3cbb86fea82c27941c9119`. Selected model SHA-256: `bf2673e0d19b8f55f6a6db63dd263099dbfde34523608e2983fc8f822d6387e9`. The reproduction check does not write to the saved result artifact.
