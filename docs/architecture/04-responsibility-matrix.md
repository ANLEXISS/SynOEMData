# Synora and OEM responsibility matrix

| Capability | Ownership | Evidence in current tree | OEM integration point / limit |
| --- | --- | --- | --- |
| Core event processing, state transitions and decision pipeline | SYNORA_PRODUCT | `internal/cognitivecore`, active Core entrypoint and tests | Remains vendor-neutral; no OEM fork of decision algorithms. |
| Event/message/action/device/state schemas | SHARED_CONTRACT | `pkg/contract`, `pkg/contracts`, JSON/YAML manifests | Versioned compatibility required; current packages have contract duplication noted in audit. |
| Vision event semantics, tracking/evidence reduction and inference adapters | SYNORA_PRODUCT | `internal/vision`, Python worker, evidence contracts | Hardware feed is replaceable; no claim every model/backend is qualified. |
| Camera firmware, sensor/ISP/VENC, board wiring, physical controls | OEM_IMPLEMENTATION | Not implemented; camera V0 is simulation/mock | OEM supplies vendor SDK, driver, device lifecycle, capability declaration and feedback. |
| Linux camera/NPU driver, kernel modules, device-node permissions | HOST_PLATFORM | `/dev`/RKNN runtime assumptions and systemd packaging | Must be qualified for each board/kernel; never inferred from SoC name alone. |
| Video/clip ingress contract and authentication | SHARED_CONTRACT | Discovery ingress and signed mock uploader | Keep payload, size, signature and status semantics stable/versioned. |
| MediaMTX control integration | REFERENCE_ADAPTER | `internal/mediamtx`, packaged external MediaMTX unit | OEM may substitute a media source/server if contract conformance is proven. |
| Local bus protocol, ACL and HMAC | SYNORA_PRODUCT + SHARED_CONTRACT | `internal/bus`, `pkg/contract` | Unix socket is current Linux reference; other IPC requires same identity, replay and authorization properties. |
| REST, WebSocket and SSE semantics | SHARED_CONTRACT / SYNORA_PRODUCT | `cmd/synora-api`, `internal/discovery/web_v1.go` | Resolve duplicate server compositions; preserve auth and redaction behavior. |
| Physical device catalog and policy | SYNORA_PRODUCT / SHARED_CONTRACT | `internal/device`, action contracts | OEM implementation advertises hardware capabilities and feedback; registry alone does not prove availability. |
| Physical action driver/control | OEM_IMPLEMENTATION or REFERENCE_ADAPTER | Current production mode is dry-run; no physical executor is active | Requires explicit capability, permission, feedback, safety qualification and separate activation review. |
| Host process supervision | HOST_PLATFORM | systemd units only | OEM supplies its supervisor adapter and shutdown/restart semantics. |
| Runtime manager policy/API | SYNORA_PRODUCT | `internal/manager`, RPC contracts | Platform command backend is HOST_PLATFORM; narrow privilege rather than root-wide control. |
| Network policy and pairing semantics | SYNORA_PRODUCT / SHARED_CONTRACT | `internal/discovery/network`, connectivity config/contracts | Hostapd/dnsmasq/nftables implementation is Linux reference; OEM networking stack may differ. |
| Filesystem/state persistence policy | SYNORA_PRODUCT / SHARED_CONTRACT | Universal Store/outbox/recovery | OEM provides storage durability, mount layout, encryption, capacity and replacement behavior. |
| RKNN runtime and board acceleration | HOST_PLATFORM / REFERENCE_ADAPTER | Python requirements and RKNN backends | OEM owns runtime/driver packaging. Runtime presence is not model compatibility proof. |
| Identity provider, product enrollment, branding, provisioning UX | OEM_IMPLEMENTATION with shared auth contracts | Local accounts, bootstrap config and UI login exist | OEM-specific cloud/account lifecycle is not in this repository. |
| Web UI and diagnostic views | SYNORA_PRODUCT | `synora-web` and Go APIs | OEM may theme/extend presentation without changing event/state semantics. |
| Audio capture, ASR/TTS, generic sensor fusion | Not implemented; future ownership undecided | No production module found | Document only; do not imply product functionality or synthesize empty modules. |

An implementation can span several ownership types. For example the vision
algorithm is Synora Product, the video job/event boundary is Shared Contract,
the MediaMTX client is a Reference Adapter, and a camera's firmware/driver is
OEM Implementation/Host Platform.

The Go Vision package is independently owned at `internal/vision`; see
[`14-oem-adaptation-surfaces.md`](14-oem-adaptation-surfaces.md) for its
current ports and the adapters that remain host-specific.
