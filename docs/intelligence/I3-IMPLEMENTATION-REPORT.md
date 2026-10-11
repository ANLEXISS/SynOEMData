# I3 — Cognitive Dataset Engine Implementation Report

**Status:** IMPLEMENTED / TESTED / EXPERIMENTAL; HUMAN REVIEW REQUIRED  
**Authorization:** I3 execution followed the owner's I2 approval and explicit instruction to proceed to I3.  
**Approval:** NOT APPROVED  
**Source commit/push and I3 documentation publication:** NOT AUTHORIZED  
**Model training/evaluation:** NOT RUN

**I3.1 qualification:** EARLY STOP — Task A/B target identity is underspecified; see `I3.1-DATASET-QUALIFICATION-REPORT.md`. I3 approval remains NOT APPROVED.

## 1. Delivered scope

Added an Intelligence-owned, deterministic synthetic dataset engine and a command-line generator. It is an offline research-data producer. It does not connect to devices, write runtime state, modify production services, or change Core behavior.

The default generator configuration creates 12 synthetic homes. Each example is generated from separately seeded streams for topology/split assignment, schedule, source behavior, faults, clock drift, network delay, scenario selection, and cutoff. The manifest records the generator configuration, versions, seed lineage, split/scenario counts, and per-task class counts.

The five persisted artifact directories are:

1. `ground-truth/` — hidden state/events, future topology edges, and hidden scenario parameters.
2. `source-emissions/` — source records, including records whose receipt time is after cutoff.
3. `availability/` — only emissions received by the prediction cutoff, with per-record `observed_at` and `received_at`.
4. `model-inputs/` — State Encoder batch and encoded representation, plus selected per-record availability metadata. No target, scenario, home, or split fields are included in these files.
5. `targets/` — task labels, horizons, availability times, and provenance.

Artifacts carry schema/generator versions, deterministic digests, and parent digests. The encoder batch time is the prediction cutoff; its batch receipt time is the assembly time. Per-emission receipt times stay in the availability records because the frozen I2 observation type has no per-observation receipt field.

Run the generator with:

```sh
cd /home/rock/SynoraOEM
go run ./cmd/synora-dataset-generate --seed 8142 --homes 12 --start-time 2026-07-01T00:00:00Z --output /tmp/synora-i3-dataset
```

The output directory must be empty. The run is deterministic for the same configuration; output is not written into the repository by default.

## 2. Scenarios, cutoffs, and splits

All twenty scenario templates generate synthetic stimuli. Five templates are excluded from training and included only in validation/test: routine change, unexpected presence, clock drift, new device, and unseen home topology. Whole home groups and topology families stay in one split. The current graph catalog contains six fixed directed ring, chain, and star templates; this is a narrow graph set, not a general household topology simulator.

An emission enters the availability view only when `ReceivedAt <= cutoff`. A late emission is excluded even when `ObservedAt` is before cutoff. A received future-dated observation remains available and keeps its signed negative age. Scenario cutoffs vary by home and scenario. Hidden clock offset, delay, routine deviation, outage duration, conflict count, and future-edge metadata are retained in ground truth rather than model input.

## 3. Label definitions and limitations

The generator emits five labels for every example:

- **A — Next state:** a Boolean occupancy/light state at a one-minute horizon.
- **B — Temporal transition:** steady, rising, or falling between the generated before/after states.
- **C — Topology transition:** unchanged or an added directed edge at the one-minute horizon.
- **D — Behavior anomaly:** a synthetic routine deviation. Daily-routine positives compare two generated observations; unexpected-presence cases are positive. This is not an intrusion or danger label.
- **E — Observation consistency:** whether the generated contemporaneous records agree. Current positives are conflicting-source examples; the label is not a source reliability judgment.

These are initial, simplified labels. In particular, Task A does not yet consistently identify a per-example target entity/attribute across every scenario; Task B has no event-time bins; Task C models edge addition only; Task D has very few training positives and does not model a learned home routine; and Task E covers a narrow synthetic conflict rule. Multiple-occupant movement, missing events, discovery, recovery, and topology changes are compact stimuli, not full physical simulations. No claim is made that the targets establish meaningful temporal, topological, behavioral, or household reasoning.

### Default distribution sample

For seed `8142`, 12 homes, and start time `2026-07-01T00:00:00Z`, the generator emitted 200 examples: 120 train, 40 validation, and 40 test. Home-group counts were 8/2/2. Counts below are label counts, not model predictions or metrics.

| Task / value | Train | Validation | Test |
|---|---:|---:|---:|
| A false | 116 | 35 | 35 |
| A true | 4 | 5 | 5 |
| B falling | 34 | 16 | 12 |
| B rising | 38 | 11 | 13 |
| B steady | 48 | 13 | 15 |
| C edge added | 8 | 2 | 2 |
| C unchanged | 112 | 38 | 38 |
| D false | 116 | 35 | 35 |
| D true | 4 | 5 | 5 |
| E consistent | 112 | 38 | 38 |
| E inconsistent | 8 | 2 | 2 |

**Per-class metrics, baselines, calibration, and model accuracy: NOT MEASURED.** No model was trained. The class imbalance, especially for A and D positives and E inconsistencies, makes this sample unsuitable for claiming task readiness.

## 4. Leakage and reproducibility review

Model-input files do not serialize scenario names, home/split labels, hidden truth, or targets. Availability selection and target artifacts are separate, and tests verify late-record exclusion, provenance/digest chains, split disjointness, and no explicit target fields in model inputs.

A formal statistical leakage audit is **NOT PERFORMED**. Several targets intentionally correlate with observable evidence (for example, a contradiction for E and two differing routine observations for D); scenario-specific record shapes can also reveal which synthetic fixture was used. The opaque IDs are not a substitute for a leakage analysis. No learned performance result exists to assess exploitability. Before training, an owner-reviewed input/target audit and stronger variation within scenarios are required.

Two CLI generations with the same 12-home configuration produced byte-for-byte identical directory trees (`diff -qr` reported no differences). Unit tests also regenerate artifacts from an identical seed and compare all stages.

## 5. Validation record

| Check | Result |
|---|---|
| `make test-dataset-engine` | PASS |
| `go test -race -count=1 ./internal/intelligence/datasetengine` | PASS |
| `go build ./...` | PASS |
| `go test -p 1 -count=1 ./...` | PASS |
| `make architecture-check` | PASS — 7 modules, 40 owned Go packages, 9 declared edges, 8 actual edges; Python RKNN imports isolated |
| `make test-semantic-foundation` | PASS — Go and Python contract/device harness suites |
| Two same-config generated directory trees | PASS — byte-for-byte identical |
| `git diff --check` | PASS |

Shell startup emitted the existing harmless `/etc/bash.bashrc` `PS1: unbound variable` warning for Make invocations; the validation targets still exited successfully.

## 6. Changed files and next decision

Implementation files are `internal/intelligence/datasetengine/`, `cmd/synora-dataset-generate/main.go`, `modules/intelligence.json`, `modules/package-ownership.json`, and the `test-dataset-engine` Make target. The I3 specification, scenario catalog, roadmap, and this report were updated to reflect actual status.

I3 is **IMPLEMENTED / TESTED / EXPERIMENTAL; HUMAN REVIEW REQUIRED**. It is not approved. I4 is not started. No source commit, source push, or publication of these I3 documents was performed. Proposed source commit message, if later authorized: `Implement experimental I3 synthetic dataset engine`.

## 7. I3.1 early-stop finding

I3.1 inspection found that Task A/B labels describe a generic occupancy/light state but the label schema has no target entity/attribute field. The selected truth room can differ from the observed sensor entity; lighting changes the target kind/attribute, and multiple-occupant and fault fixtures still emit the same generic Boolean target. Task B also has no transition-time bins. This prevents an unambiguous prediction unit and baseline definition. The I3.1 mission requires stopping for an owner decision rather than choosing task semantics or changing the model-facing schema. No I3.1 code or schema correction was made. Statistical qualification, adversarial tests, baselines, ablations, stress profiles, and the full compatibility matrix remain NOT MEASURED; refer to the three I3.1 review documents for the bounded findings and decision request.
