# M1.9 — OEM Composition Profiles

**Status:** Complete.

## Reduced composition

The profile at modules/profiles/oem-core-only.json selects Foundation and Intelligence and composes only the existing Bus and Cognitive Core entrypoints.

It intentionally excludes Vision, Voice, Automation dispatch, external Integration, and Platform runtime services. This supports local event processing, deterministic Cognitive Core behavior, and authoritative Foundation state without camera ingestion, external API ingress, physical command dispatch, or host network management.

The profile smoke test runs an isolated Unix Bus server and Cognitive Core service, sends a versioned device-offline event through the Bus, and confirms the Core commits it. It uses a temporary socket and in-memory Store and starts no production service or host adapter.

## Validation mechanism

- - go build -buildvcs=false ./... — PASS.
- go test -buildvcs=false -p 1 -count=1 ./... — PASS.
- make GO=<Go 1.24 toolchain> oem-profile-core-only — PASS: selected Bus/Core build and reduced composition smoke test.
- go test -race ./cmd/synora-core -run TestCoreOnlyCompositionProcessesBusEvent — PASS.
- make GO=<Go 1.24 toolchain> architecture-check — PASS: profile schema, required dependency closure, entrypoint existence, transitive Go package ownership, seven modules and acyclic dependency graph.
- Synthetic central E2E — PASS: 461/461; 49 allowed dry-run, 6 blocked, 371 no-action suppressed; camera_mock_e2e=not_qualified.
- Existing module registry, seven domain names, and executable entrypoints remain unchanged.

## Limits

This reduced profile intentionally has no Discovery result dispatcher. Core remains in deterministic dry-run mode, so proposed physical actions are not executed. The profile demonstrates a supported reduced composition; it does not establish hardware qualification or full product operation.
