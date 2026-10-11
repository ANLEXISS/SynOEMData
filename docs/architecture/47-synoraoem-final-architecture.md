# SynoraOEM Final Architecture

**Architecture state:** Stabilized logical module boundaries, one Go module, existing executable boundaries retained.
**Hardware state:** Physical RKNN, camera, network, and OEM device qualification remain unverified.

## Logical modules and current package ownership

| Domain | Current ownership |
| --- | --- |
| Foundation | Shared contracts and message bus; configuration, persistence primitives, authoritative Foundation StateStore; legacy state package remains separate from the active Core Store path |
| Intelligence | Cognitive Core orchestration and contracts, snapshot projection, topology configuration, and CPU MLP bundle adapters |
| Vision | Vision worker client/runtime, Python worker and backend interface; face/model/dataset support and qualification helpers |
| Voice | No production package; NOT_IMPLEMENTED |
| Automation | Abstract action request/result contracts and deterministic controlled dry-run outcome policy |
| Integration | Discovery composition, APIs, authenticated ingress, event translation, device registry, connectivity and security adapters |
| Platform | Host network commands and policy, runtime manager, boot/version support, MediaMTX adapter, deployment/bootstrap support |

The full package ownership ledger is modules/package-ownership.json. The logical registry remains seven domains; the repository remains one Go module.

## Module dependency graph

Actual dependency direction (source depends on target):

    Intelligence -> Foundation
    Vision       -> Foundation
    Platform     -> Foundation
    Integration  -> Foundation, Intelligence, Automation, Vision, Platform
    Voice        -> no implementation and no package dependencies

The eight actual inter-module edges are all declared. The architecture checker validates package ownership, forbidden cross-domain imports, declared dependencies, and Go package cycles.

## Runtime composition

- Full system entrypoints remain Bus, Core, Discovery, API, Connectivity, Runtime Manager, Network Config, and existing maintenance/qualification tools.
- Integration composes API and media-ingress ports with the local bus, Vision runtime, configured device data, Automation action boundary, and optional Platform network/runtime adapters.
- Foundation owns active durable StateStore commits, journal/replay and the Core action outbox. Cognitive Core remains the business-facing compatibility adapter over the Foundation store.
- The reduced profile modules/profiles/oem-core-only.json composes only Bus and Core from Foundation and Intelligence. It has a real Unix Bus/Core smoke test and excludes Vision, Voice, Automation dispatch, Integration, and Platform services.

## Contracts and boundaries

- Foundation provides the shared contracts, local JSON bus protocol, persistent contracts, StateStore, and persistence primitives.
- Intelligence owns Core snapshot/decision orchestration and depends on injected MLPBackend and MLPBackendV3 interfaces. internal/intelligence/mlp implements the CPU V1/V3 backends; Core does not import the adapter.
- Vision exposes its backend-neutral inference boundary. The concrete RKNN Lite import remains inside the Python RKNN backend adapter.
- Automation validates abstract actions and returns controlled dry-run outcomes. It does not own Core's safety gate, action ledger, authoritative state, or a physical executor.
- Integration bridges external API/ingress and bus contracts. It can supply an OEM station source and host command executor through static composition.
- Platform owns OS/network/process-specific implementations. Network host commands are behind HostCommandExecutor; the default remains the current Linux utility implementation.
- Voice is optional and NOT_IMPLEMENTED.

## Persistence and authority

Foundation is authoritative for active durable Cognitive Core state. Cognitive Core retains decision algorithms, deterministic safety gates, snapshot projection, and compatibility-facing Store APIs. Persistence formats, journal/replay semantics, outbox ordering, and IPC/event contracts were preserved through these milestones and exercised by the Go suite and synthetic E2E.

## Remaining coupling

- internal/cognitivecore remains a broad package containing cognitive orchestration, snapshots, encoders, decision contracts, safety gates, and the Foundation Store compatibility wrapper.
- internal/discovery remains a high-fan-in Integration composition root.
- internal/foundationv1 remains a synthetic reference harness, not a production runtime.
- internal/state is a legacy Store, not the active Foundation StateStore.
- Platform defaults remain Linux-specific and use fixed runtime paths and host utilities.
- Voice, physical device execution, alternative operating systems, and physical accelerator qualification are not implemented.

See docs/architecture/48-synoraoem-module-maturity.md for per-module maturity and docs/architecture/50-post-modularization-roadmap.md for follow-up work.
