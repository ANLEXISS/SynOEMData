# SynoraOEM modular transformation progress

**Goal:** Complete M1.5 through M1.10 while preserving the seven existing domains and observable behavior.
**Current milestone:** M1.10 — Architecture Stabilization, complete.
**Workspace:** `/home/rock/SynoraOEM` only.

## Audited starting point

- M1.1–M1.4 implementation and records were checked against the workspace before continuing.
- Initial local Git commit: `806f316 chore: establish audited SynoraOEM baseline`. It uses the already configured global identity; no Git remote or global setting was changed.
- The baseline excludes local config YAML. The three YAML templates remain unstaged because their auth/hash fields require a separate safe-template review.
- Verified pre-M1.5 recovery archive: `/home/rock/SynoraOEM-m15-baseline-20261010.tar.gz`, SHA-256 `ff0478545dfaa8131c51ecb52cea4c1532bf39f7f17367c8f2872f104cd9ac4b`. The archive passed `gzip -t`, contained no YAML, credentials, model/media files, logs, or bytecode, and a representative source file matched the workspace.
- Go build, serialized full Go tests, and architecture check pass at the M1.5 baseline. Python Vision tests are 115/115; C++ is 2/2; synthetic central is 461/461; UI production build passes per the M1.4 final validation.

## Milestone tracking

| Milestone | Status | Validation / record |
|---|---|---|
| M1.1–M1.4 | COMPLETE per repository code and prior validation records | `docs/architecture/01`–`40`; M1.4 details in `36`–`40` |
| M1.5 Intelligence Decomposition | COMPLETE | `docs/architecture/41-m15-intelligence-decomposition.md`; Go build/full suite, targeted race tests, architecture check, and synthetic central E2E 461/461 pass |
| M1.6 Automation Independence | COMPLETE | `docs/architecture/42-m16-automation-independence.md`; Go build/full suite, Automation/Discovery/Core/StateStore race tests, architecture check, synthetic central E2E 461/461 pass |
| M1.7 OEM Integration Boundaries | COMPLETE | docs/architecture/43-m17-oem-integration-boundaries.md; Go build/full suite, Discovery/network/device race tests, architecture check, synthetic central E2E 461/461 pass |
| M1.8 Platform Independence | COMPLETE | docs/architecture/44-m18-platform-independence.md; Go build/full suite, network/Discovery/manager race tests, architecture check, synthetic central E2E 461/461 pass |
| M1.9 OEM Composition Profiles | COMPLETE | docs/architecture/45-m19-oem-composition-profiles.md; full Go build/suite, selected profile build/smoke/race, architecture check, synthetic central E2E 461/461 pass |
| M1.10 Architecture Stabilization | COMPLETE | docs/architecture/51-m110-architecture-stabilization.md; final Go, race, architecture, reduced-profile, Python, C++, and central validation recorded; UI production build passes from the locked offline dependency set |

## M1.5 initial audit

The initial M1.5 audit found CPU MLP bundle loading and inference embedded in Cognitive Core beside existing MLPBackend / MLPBackendV3 contracts. The completed extraction moved the CPU adapters behind those contracts without changing snapshot encoding, decision ordering, safety gates, state writes, or persisted formats. The separate topology package remains without observed Go consumers.

## M1.5 completed extraction

- CPU V1/V3 MLP bundle loading, validation, inference, and trace generation are owned by `internal/intelligence/mlp`; it consumes Cognitive Core contracts.
- Core retains cognitive orchestration and does not import the concrete adapter. The architecture check enforces this direction and Intelligence ownership.
- Full synthetic central E2E: 461 passed, 0 failed; 460 pipelines completed and one expected pipeline rejection. Camera mock E2E remains `not_qualified` per harness output.
- Go build/full serialized test suite, MLP/Core/StateStore race tests, and architecture check passed.

## M1.6 completed extraction

- `internal/automation` owns abstract action request/result contracts and deterministic dry-run dispatch policy. Discovery retains bus ingress and result event-envelope composition; source-compatible action aliases remain at its boundary.
- Cognitive Core remains the deterministic safety gate and authoritative action outbox/ledger owner. Physical execution remains disabled.
- Full Go build and serialized suite pass; targeted race checks pass for Automation, Discovery, Core, and Foundation StateStore.
- Architecture check: 7 logical modules, 38 owned Go packages, 9 declared edges, 8 actual module edges; dependency graph acyclic and boundaries enforced.
- Synthetic central E2E: 461 passed, 0 failed; 460 pipelines completed, one expected rejection, and 49 allowed dry-run / 6 safety blocked / 371 no-action suppressed. Camera mock E2E reports `not_qualified`.

## M1.7 completed extraction

- Discovery composition supports a static OEM-provided network station source through NewManagerWithStationSource; the existing constructor and installed config adapter remain the default.
- HTTP and media ingress retain explicit publisher, subscription, authentication, queue, and device-tracker contracts.
- Integration descriptor declares Platform network interface consumption; the architecture checker enforces the dependency.
- Full Go build and serialized package suite pass; targeted race checks pass for Discovery, network, and device packages.
- Architecture check: 7 modules, 38 owned Go packages, 9 declared edges, 8 actual edges.
- Synthetic central E2E: 461 passed, 0 failed; 460 pipelines completed, one expected rejection. Action lifecycle: 49 allowed dry-run, 6 blocked, 371 suppressed. Camera mock E2E reports not_qualified.

## M1.8 completed extraction

- Platform network operations use HostCommandExecutor and the same default OSCommandExecutor. Discovery composition can statically supply a different executor and station source.
- Network bridge, firewall, hostapd, Wi-Fi capability, and dnsmasq command calls route through the port. Tests cover bridge command sequencing and firewall stdin.
- The architecture checker enforces that network-package os/exec usage stays in command.go.
- Full Go build and serialized suite pass; race checks pass for network, Discovery, and Runtime Manager. Synthetic central E2E passes 461/461; camera mock reports not_qualified.
- Host paths, process signals, Linux networking, systemd, and RKNN remain platform/hardware specific; no non-Linux qualification is claimed.

## M1.9 completed composition

- Added and validated the reduced synora.oem.core-only profile selecting Foundation and Intelligence with existing Bus/Core entrypoints. Vision, Voice, Automation dispatch, Integration, and Platform runtime are excluded.
- Architecture check verifies required dependency closure and transitive internal package ownership against the selected modules.
- The Make target builds exactly the two selected executables and runs a real Unix Bus plus Core service smoke test; the event reached the Foundation-backed Core Store without Discovery or Vision.
- Full Go build and serialized suite pass; the profile smoke test passes under the race detector. Architecture check passes.
- Synthetic central E2E passes 461/461; 460 completed and one expected rejection. Camera mock reports not_qualified.

## M1.10 final validation

- Go build and complete serialized suite pass. Targeted race suite passes for Automation, Core, MLP, Discovery, Platform network, Runtime Manager, StateStore, and the reduced-profile smoke test.
- Architecture check passes: 7 logical modules, 38 owned Go packages, 9 declared edges, 8 actual edges, no dependency cycles.
- Reduced Bus/Core profile build and smoke test pass. Vision Python tests pass 115/115; C++ camera mock tests pass 2/2; synthetic central E2E passes 461/461.
- UI production build passes after restoring only package-lock.json dependencies from the local cache with lifecycle scripts disabled; package manifests remain unchanged. Physical camera, network, and RKNN hardware qualification remain UNVERIFIED.
- Final architecture, maturity, readiness, roadmap, and stabilization records are docs/architecture/47 through 51.

## Overall result

The architectural transformation M1.5–M1.10 is implemented, validated, and documented. The final UI production build passes using the existing locked package set. All software acceptance checks pass; physical hardware qualification remains UNVERIFIED as allowed by the mission.

## Limitations and decisions

- Do not implement learned reasoning or add model/runtime dependencies.
- Keep Foundation authoritative for StateStore and persistence.
- Preserve Go entrypoints and current CPU MLP behavior; production inference remains the existing injected backend contract.
- Voice remains NOT_IMPLEMENTED and optional.
- No architectural or software-validation blocker remains. Hardware qualification is outside this software-only refactoring mission.

## Closure

All defined software milestones and deliverables are complete. Physical hardware qualification is a separate future activity.
