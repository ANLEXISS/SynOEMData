# Post-Modularization Roadmap

Architectural completion is separate from product capability, hardware qualification, and commercial readiness.

## A. Architectural debt

1. Split the high-fan-in Discovery composition root along existing API, ingress, Vision, and host-runtime seams without changing public contracts.
2. Continue decomposing Cognitive Core while preserving snapshot, decision, safety, Store, persistence, and event contracts.
3. Decide whether and when the legacy internal/state package can be retired; do not silently migrate persisted data.
4. Determine the production role of the standalone API entrypoint and synthetic internal/foundationv1 harness.
5. Keep package ownership, profile dependency closure, forbidden-import checks, and cycle checks in architecture CI.
6. Revisit the topology package's lack of observed Go consumers when a concrete consumer exists.

## B. Functional improvements

1. Implement Automation rule and policy evaluation only with explicit deterministic authority, persistence, and failure semantics.
2. Add product-approved device capabilities and integration adapters behind the existing abstract contracts.
3. Improve operational health and error reporting without changing security or event semantics.
4. Continue Vision model, privacy, and data lifecycle improvements through the backend-neutral inference boundary.
5. Requalify UI production build after the workspace's locked TypeScript/Vite toolchain is made available.

## C. Cognitive Runtime development

1. Specify and version a structured State Encoder contract.
2. Design the shared backbone and specialized adapter interface as an optional Intelligence-owned runtime.
3. Keep learned outputs as proposals; preserve deterministic Core safety authorization and Foundation state ownership.
4. Add offline evaluation, replay, calibration, and model promotion gates before any runtime integration.
5. Preserve the current MLPBackend and MLPBackendV3 path until a compatible runtime is separately qualified.

## D. Hardware qualification

1. Qualify the selected RKNN model files, NPU runtime, driver, memory use, and thermal behavior on the target ARM64 board.
2. Validate camera and network adapters on representative OEM hardware; keep mock results distinct from hardware evidence.
3. Record CPU fallback behavior and failure modes; no CPU Vision fallback is currently implemented.
4. Validate boot, storage, network isolation, and recovery on the actual deployment image.
5. Treat every hardware result as board, model, driver, and software-version specific.

## E. OEM integration and commercial readiness

1. Define the approved OEM device inventory and the specific static adapter required for each product.
2. Validate authentication, provisioning, updates, rollback, support, privacy, and service operations with the OEM.
3. Establish packaging, signing, licensing, security review, and support lifecycle.
4. Keep commercial readiness separate from modular architecture acceptance; this mission does not certify a product for sale.
