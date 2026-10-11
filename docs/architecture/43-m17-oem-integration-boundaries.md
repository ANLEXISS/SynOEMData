# M1.7 — OEM Integration Boundaries

**Status:** Complete.

## Result

Integration already has several real adapter contracts rather than one vendor-specific implementation:

- discovery.WebAPI receives event publishing, subscriptions, history, health, and resident-gallery services through small interfaces.
- discovery/ingress.Config receives authentication, ingress gating, device tracking, queue, and message publisher interfaces.
- discovery/network.Manager receives network.StationSource, whose output is the platform network package's narrow KnownStation type.
- discovery.Boundary translates external commands into existing versioned Core events; its action outcome policy now delegates to Automation (M1.6).

The production composition previously hard-coded the configured-device-to-network-station adapter. NewManagerWithStationSource now allows a statically supplied OEM adapter at composition time while the existing NewManager entrypoint continues to use the installed device registry adapter.

## Boundary ownership

    OEM device configuration/adapter
      -> Integration station-source seam
      -> Platform network manager
      -> host network operations

    External API / authenticated ingress
      -> injected Integration event and queue ports
      -> existing local bus and versioned contracts

    External action request
      -> Integration bus bridge
      -> Automation controlled dry-run policy
      -> Integration result event bridge

The custom station source can translate an OEM registry into the existing narrow station input. It does not add vendor protocols, physical device execution, or a dynamic adapter loader. The current default remains unchanged.

## Descriptor and enforcement changes

Integration declares its existing Automation dependency and Platform network interface consumption. The architecture checker rejects an undeclared Integration-to-Platform edge and verifies that Integration consumes synora.platform.network/v1. The existing seven module names and service entrypoints remain unchanged.

## Validation

- go build -buildvcs=false ./... — PASS.
- go test -buildvcs=false -p 1 -count=1 ./... — PASS.
- go test -buildvcs=false -race -p 1 -count=1 ./internal/discovery ./internal/discovery/network ./internal/device — PASS.
- make GO=<Go 1.24 toolchain> architecture-check — PASS: 7 modules, 38 owned Go packages, 9 declared edges, 8 actual edges; Integration-to-Platform network consumption checked.
- Synthetic central E2E — PASS: 461/461; 49 allowed dry-run outcomes, 6 safety blocks, 371 suppressed no-action results. camera_mock_e2e=not_qualified.

## Limits

The alternate station-source seam is software-level only. It does not qualify any physical network adapter. Discovery remains the current high-fan-in integration composition root; splitting its unrelated API, Vision, storage, and host-runtime responsibilities is outside this focused milestone.
