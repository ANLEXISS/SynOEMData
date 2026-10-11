# Component maturity register

`IMPLEMENTED` describes tested software behavior, not a blanket production or
hardware qualification. `PARTIAL` marks a concrete software path whose
capability, adapter or deployment is incomplete. `PLACEHOLDER` is documentation
or a contract without functional implementation. `UNUSED` means no current
runtime composition was found. `UNKNOWN` means inspection did not establish a
defensible status.

| Component | Maturity | Evidence / qualification boundary |
| --- | --- | --- |
| Go Unix message bus, ACL/HMAC, reconnect and bounded queues | IMPLEMENTED | Package tests and running service entrypoint; Linux Unix-socket packaging. |
| Active Core event/snapshot/Safety Gate/Universal Store pipeline | IMPLEMENTED | Broad tests and hermetic harness; physical actions remain dry-run and promoted model bundle absent by default. |
| Legacy `internal/state` store | IMPLEMENTED (legacy), UNUSED in active Core path | Existing persistence/replay tests; architecture marks it isolated legacy. |
| Discovery external boundary, APIs, ingress and runtime composition | IMPLEMENTED software; PARTIAL OEM portability | Broad tests; root/systemd/network command dependencies and multiple responsibilities. |
| Standalone `cmd/synora-api` | IMPLEMENTED executable; UNKNOWN active deployment | Builds/tests, but no systemd unit; overlaps Discovery API composition. |
| Device registry/configuration | IMPLEMENTED software; PARTIAL hardware integration | Config/contracts and tests; no general vendor driver or observed physical feedback requirement satisfied by config alone. |
| Topology load/validation | IMPLEMENTED configuration and software transitions | Tests; no real sensor localization or inferred home map. |
| Vision worker protocols and validators | IMPLEMENTED software | Python tests and Go worker manager tests; real model/backend qualification is per-runtime/model. |
| RKNN detector runtime path | IMPLEMENTED adapter path; PARTIAL qualified capability | Host includes RKNN runtime/model; tests can initialize it. Model/driver/toolkit compatibility is hardware- and model-specific. |
| Face identity/gallery capability | PARTIAL / fail-closed | Source marks face backend unqualified; consent/dataset/gallery foundations exist, but qualified face runtime is not established. |
| MediaMTX adapter and packaging | IMPLEMENTED integration; external dependency | API client and systemd unit; binary/model/media lifecycle external to Go module. |
| Actual camera sensor/ISP/VENC driver | PLACEHOLDER / not implemented | `camera/v0` interfaces and simulator only; README explicitly reserves hardware implementation. |
| Camera upload mock | IMPLEMENTED simulator/reference | Native tests prove software queue/upload behavior with fixture; not camera or inference qualification. |
| SynoraNet host manager | PARTIAL | Linux command implementations and config/tests; requires privileges, hostapd/dnsmasq/firewall and target Wi-Fi hardware. |
| Local connectivity agent | PARTIAL | Local identity/policy/state logic; docs explicitly say remote enrollment is not implemented. |
| Runtime manager host actions | PARTIAL | Manager and command handler exist; systemd/root assumptions and broad privileges. |
| Boot health report | PARTIAL | Read-only checks/report; it does not prove all hardware or service correctness. |
| API/session/security policy | IMPLEMENTED software controls | Extensive tests/config; production requires generated secrets/TLS and deployment hardening. |
| Foundation V1 abstract workflows | PARTIAL / synthetic reference | Test harness and ports; no physical action, TTS, peripheral adapter or MLP runtime qualification. |
| Cognitive behavior-learning/training | PARTIAL / offline | Bundles and training/audit tools exist; no active online learning service. |
| Audio perception/ASR/TTS | PLACEHOLDER / not implemented | No production implementation located. |
| Generic sensor driver/fusion | PLACEHOLDER / not implemented | Contracts may accept observations; no generic adapter/fusion service located. |
| MQTT integration | UNUSED/UNKNOWN | Paho declared in `go.mod` and Mosquitto config exists; no application source import/runtime path found. |
| React application | PARTIAL product surface | Source and build config exist; OEM-specific white-label/provisioning is not established. |
| POSIX host bootstrap | PARTIAL first-cut discovery | Previous task's code exists and is copied; limitations are documented in its README and tests. |
| Current systemd deployment | IMPLEMENTED for current prototype | Six units, fixed paths/users and Debian/Linux assumptions; no alternate supervisor packaging. |

This register is a starting architecture ledger. Every future maturity change
must cite executable code, tests and, where relevant, target hardware evidence;
file existence alone is insufficient.

M1.2 adds machine-checked logical descriptors and an import-boundary check.
The Vision package move does not change its runtime maturity or qualify a
hardware backend. See [`11-module-boundaries.md`](11-module-boundaries.md).
