# M1.4 — Vision runtime independence validation

**Status:** COMPLETE for M1.4 acceptance; physical RK3588 model qualification remains outside this validation.  
**Scope:** `/home/rock/SynoraOEM` only. No live service, production camera, production model path, or external project was intentionally changed.

## Recovery baseline and source boundary

- The OEM workspace had no commits, no tracked files, and no configured remote before implementation. A reviewed Git commit could not be established without staging the full pre-existing untracked project, so recovery uses an archive.
- Pre-change recovery archive: `/home/rock/SynoraOEM-m14-recovery-20261010.tar.gz`, SHA-256 `a77e7c9d1010995598f36b9f5039031654134e98e19ef44a95c289d2296b8687`.
- Archive integrity was checked with `gzip -t`; it was extracted to a temporary verification directory and representative source files matched. It excludes `.git`, build/output/dependency directories, local YAML configuration/templates, runtime/cache/data/media/model artifacts, and derived test fixtures. Earlier M1.3.2 and M1.3.3 archives remain preserved.
- No secret/key/certificate files, production video/model binaries, or credential files were included in the archive.
- The only intended workspace is `/home/rock/SynoraOEM`; Synora-v1, production services, and production paths were not used for execution or modified.

## Baseline before implementation

| Check | Result |
|---|---|
| Go build (`go build -buildvcs=false ./...`) | PASS |
| Full serialized Go suite (`go test -buildvcs=false -p 1 -count=1 ./...`) | PASS |
| Architecture check | PASS: 7 modules, 36 owned Go packages, 8 declared edges, 7 actual module edges |
| Vision Worker Python suite | PASS: 108/108 |
| C++ camera/mock suite | PASS: 2/2 |
| UI production build | PASS |
| Synthetic central E2E | PASS: 461/461, 0 failures; 460 completed, 1 expected rejection; manifest SHA-256 `d33fa1937f3c0ed23203ef350526e58853bf47ae165ae8ec7d898a50db6e68f4` |
| CPU inference runtimes | ONNX Runtime, Torch, TensorFlow, OpenVINO, and TFLite Runtime were not available; no dependencies were installed |

## Final validation

| Check | Result |
|---|---|
| Go build (`go build -buildvcs=false ./...`) | PASS |
| Full serialized Go suite (`go test -buildvcs=false -p 1 -count=1 ./...`) | PASS |
| Architecture check, including narrow Python AST import guard | PASS: 7 modules, 36 owned Go packages, 8 declared edges, 7 actual module edges; `rknnlite` import isolated to `core/rknn_backend.py` |
| Vision Worker Python suite | PASS: 115/115 (baseline 108; seven new inference-boundary tests) |
| C++ camera/mock suite | PASS: 2/2 |
| UI production build | PASS: TypeScript build and Vite production bundle, 1,888 modules |
| Targeted Go race tests | PASS: `internal/vision`, `internal/discovery`, `internal/discovery/ingress` |
| Synthetic central E2E | PASS: 461/461, 0 failures; 460 completed, 1 expected rejection, 0 incomplete; 356,761.582 ms. Report `/tmp/synora-m14-final-central.json`; report SHA-256 `3596d4ae2e934d3b4fbbdc0e0db580d873d234bcd47c20cbb1443db178f3c620`; manifest SHA-256 matches baseline `d33fa1937f3c0ed23203ef350526e58853bf47ae165ae8ec7d898a50db6e68f4` |

Harnesses were run sequentially. The central harness runs alone and does not start production services or use a camera/model; it validates synthetic system contracts, not Vision inference.

## Backend and pipeline evidence

- **IMPLEMENTED:** `InferenceBackend`, model metadata, runtime config, typed initialization/execution errors, static factory selection, RKNN adapter, and resource cleanup.
- **TESTED:** An injected deterministic fake passes through existing YOLO preprocessing, model output normalization/decoding, detector filtering, clip-backend normalization and timestamp propagation, and cleanup. Synthetic execution failure is contained at the existing detector boundary.
- **MOCKED:** Fake model arrays only. They validate wiring and compatibility, not neural-network accuracy or latency.
- **RKNN reference:** Runtime/model loading and inference now live in `core/rknn_backend.py`; the RKNN runtime class is not imported in the generic pipeline, model stages, or worker. No production model was run through the adapter in this validation.
- **CPU backend:** NOT_IMPLEMENTED. No suitable installed CPU/ONNX runtime was found. This milestone does not claim production inference portability.
- **Hardware qualification:** No validated physical RK3588 model run or comparable NPU benchmark was conducted. The runtime remains tied to RKNNLite, RK3588/RKNPU2 model artifacts, core masks, and Linux/native dependencies.
- **Environment observation:** RKNNLite and runtime/driver libraries were importable and reported RK3588/RKNPU2 metadata, but the worker's no-core-mask initialization probe returned `init_runtime ret=-1`; no production model was loaded or inferred. Therefore this confirms library presence only, not an operational model/device qualification.

## Performance review and limitation

The RKNN adapter passes NumPy tensors through `numpy.ascontiguousarray`, preserving the no-copy behavior when an input is already contiguous. This is a code-path property, not a measured end-to-end performance result. No comparable before/after RK3588 benchmark is available; initialization cost, inference overhead, and NPU throughput require target hardware and production-compatible model artifacts. No performance equivalence claim is made.

## Compatibility and ownership

The seven logical modules, Go package ownership, Foundation state authority, Cognitive Core, Go `WorkerLifecycle`, process supervisor, socket path, protocol identifier (`synora.vision.v1`), request/response envelope, and event names remain unchanged. Existing hello capability model entries gain additive status/metadata fields; event and request schemas do not change. The final existing IPC, event, and model tests passed as part of the suites above.

## Remaining work and readiness

M1.4 isolates runtime access and demonstrates substitution at the model-stage boundary. Before Vision can claim independently deployable hardware portability, an OEM integration still needs a production adapter for another runtime, matching model artifacts, target-specific qualification, and measured performance. M1.4 does not implement M1.5 or production CPU inference.
