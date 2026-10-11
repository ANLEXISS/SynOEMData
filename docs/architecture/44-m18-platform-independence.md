# M1.8 — Platform Independence

**Status:** Complete.

## Result

The Platform-owned network package now owns host process execution through HostCommandExecutor and its current OSCommandExecutor implementation. Bridge creation, firewall application, Wi-Fi capability detection, hostapd startup/reload, and dnsmasq startup/reload route through this port. Network policy and command arguments remain in their existing Linux adapter code; this change does not claim that Linux network behavior is portable.

Network Manager and Discovery expose explicit composition constructors accepting the existing station source plus a host command executor. Their existing constructors continue to install the same OS executor and configured device-registry station source.

## Boundary and enforcement

- OS process construction is centralized in internal/discovery/network/command.go.
- The architecture checker rejects direct os/exec usage elsewhere in that package.
- Host-specific paths, signals, firewall rules, hostapd/dnsmasq behavior, and systemd remain Platform-owned.
- Runtime Manager already uses its injected Executor for service commands; its filesystem and disk probes remain OS-specific Platform implementations.
- Vision's RKNN runtime remains isolated behind the existing backend boundary.

## Validation

- go build -buildvcs=false ./... — PASS.
- go test -buildvcs=false -p 1 -count=1 ./... — PASS.
- go test -buildvcs=false -race -p 1 -count=1 ./internal/discovery/network ./internal/discovery ./internal/manager — PASS.
- make GO=<Go 1.24 toolchain> architecture-check — PASS: 7 modules, 38 owned Go packages, 9 declared edges, 8 actual edges; raw network command execution isolated.
- Synthetic central E2E — PASS: 461/461; action lifecycle 49 allowed dry-run, 6 safety blocked, 371 no-action suppressed. camera_mock_e2e=not_qualified.

## Limits

This is a software boundary, not qualification of another operating system, network stack, device, or accelerator. A non-Linux implementation and portable deployment profiles remain future work.
