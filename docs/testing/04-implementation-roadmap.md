# Semantic Audit and Harness Implementation Roadmap

Priority is based on cross-module leverage and contract risk. This roadmap is not authorization to implement product semantics in this audit.

| Milestone | Scope and deliverables | Exit evidence | Compatibility / dependency notes |
|---|---|---|---|
| H0 — Baseline and audit records | Preserve current seven modules; publish this grammar, gap matrix, model, spatial case study, harness audit/spec, scenarios and roadmap. Record exact checks and known hardware limitations. | Architecture check and existing regression commands recorded with revision/output; docs reviewed against code. | Docs-only; no wire or persisted changes. |
| H1 — Contract fixture inventory and remediation | Index current Go JSON fixtures, Python-produced event fixtures, Bus/Vision IPC samples, API DTOs and `universal-store/v1` records; add missing golden tests; correct the Vision payload characterization; cover temporal and persistence edges; repair segment `node_id` decoding without changing serialized shape. | Go/Python fixture and runtime-boundary tests; segment, API, camera, persistence, and temporal characterization; full regression recorded in the H1 reports. | Characterization tests plus a narrow allowlisted segment decoder correction; no new wire or persisted semantics. |
| H2 — Observation envelope proposal and compatibility spike | Design new typed versioned observation/evidence envelope and legacy adapters for a narrow camera/sensor example. Keep old event and payload fields intact. | Dual-read characterization; old payloads round-trip; unknown fields behavior documented; adapter rejects unsafe unknown commands. | Additive JSON only; separate migration review before persistence. Cross-language fixtures required. |
| H3 — Shared fake device and fault harness | Build small Go fake interfaces and Python equivalents with fixed clock, capabilities, scripted failure, out-of-order delivery and reset. Avoid a new framework. | Same ten scenario scripts reproduce outputs and cleanup in Go and Python; owner module regressions pass; bounded execution. | Test-only; schema V1 supports one device per scenario; reuse current interface seams where possible, do not redesign module ownership. |
| H4 — Device capabilities and command authorization | Define capability descriptors, unsupported result, distinct authorization grant/denial record and command/execution-result semantics; start with lock and camera capability. | Contract, API, Core, Foundation and Automation tests prove denied != failed != unsupported != simulated success. | Additive; preserve strings and action wire fixtures; no actuator until later milestone. Persistence migration separate. |
| H5 — Spatial frames, geometry and static zone rules | Define frame-bound geometry and camera-zone association; implement offline validation/evaluation only after contract approval; include privacy masks separately. | Deterministic polygon/line, direction, dwell, gap, PTZ/preset and privacy policy fixtures. | New versioned contracts; no claiming ONVIF conformance. Coordinate migration and camera adapter mapping required. |
| H6 — External protocol adapters and hardware qualification | Add Matter/ONVIF/vendor adapters as Integration-owned boundaries; connect actual camera/device test labs and qualification artifacts. | Protocol conformance evidence and physical device test reports for exact models/firmware. | Standards version and device support matrix required; compliance only after relevant official test programs. |

## Recommended first implementation pass

Start with **H1: fixture inventory and characterization tests**. It has low product risk and establishes the facts needed for all semantic evolution: current JSON field names, Go/Python interpretation, Bus/Vision IPC messages, and persisted `universal-store/v1` behavior. Then build one shared fake device script for network availability and unsupported capability outcomes, since both map to existing seams and do not depend on unimplemented spatial detection.

Do not start with a universal event payload or a full user automation language. Keep new concepts typed, versioned and narrowly scoped; preserve the seven module domains; treat stored data migration as a separate deliverable with rollback tests.

## H1 completion

H1 is validated. The current production contract families, shared Vision worker fixtures, representative API and camera goldens, segment ready/gap/reset payloads, actual `universal-store/v1` state/WAL serializer output, and temporal behavior are indexed in `testdata/contract-baseline/index.json`. The repeatable entrypoint is `make test-contract-baseline`. See [`05-h1-contract-compatibility-baseline.md`](05-h1-contract-compatibility-baseline.md) and [`06-h1-remediation-report.md`](06-h1-remediation-report.md) for findings and validation evidence.

## H3 completion

H3 is validated. `testdata/device-harness/scenarios-v1.json` drives the Go `testkit` runner and the Python Vision test runner with ten deterministic scenarios. `make test-device-harness` runs both fixture runners and the mapped production owner package tests. The simulator is isolated and test-only; it does not control real devices or make physical qualification claims. Schema V1 is one-device-per-scenario because the expected final-state projection is singular. See [`07-h3-device-fault-harness.md`](07-h3-device-fault-harness.md).

H2 remains a proposal and compatibility spike; no new semantic envelope has been implemented. Capability operation binding remains a later H4 design task.
