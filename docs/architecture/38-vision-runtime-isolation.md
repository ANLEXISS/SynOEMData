# M1.4 — Vision runtime isolation

**Status:** IMPLEMENTED for inference runtime types and worker resource cleanup. Hardware use remains Rockchip-specific through the reference adapter.

## Extraction

`core/rknn_backend.py` owns the only production Python import of `rknnlite`, runtime construction, model loading, runtime initialization, inference, output shape logging, NPU core-mask constants, and `release()`. `core/model_runner.py` owns static selection, compatibility factory arguments, and capability status. The perception pipeline does not receive an RKNN runtime object or NPU enum.

Each model stage continues to own its own preprocessing and decoding because the existing models do not share one tensor layout. The processing graph, software tracking, face identity thresholds, event production, and worker IPC are retained. No production CPU adapter was added.

## Worker lifecycle

```mermaid
sequenceDiagram
  participant Go as Go WorkerLifecycle / supervisor
  participant Socket as Existing Unix socket
  participant Worker as Python VisionWorker
  participant Pipeline as Pipeline / detector adapters
  participant Model as InferenceBackend
  Go->>Worker: start process (existing supervisor policy)
  Worker->>Socket: bind existing socket and accept protocol v1
  Worker->>Pipeline: construct configured components
  Pipeline->>Model: initialize through static factory
  Go->>Socket: existing protocol.hello and work requests
  Socket->>Worker: request
  Worker->>Pipeline: process request
  Pipeline->>Model: infer tensor
  Model-->>Pipeline: output tensors
  Pipeline-->>Worker: existing event/result payload
  Worker-->>Socket: protocol v1 JSON response
  Go->>Worker: process stop
  Worker->>Pipeline: close owned detector/pipeline components
  Pipeline->>Model: close and release runtime resources
```

Go process management and its injected `WorkerLifecycle` remain unchanged. Python `VisionWorker.close()` is called from `finally` around the existing worker entrypoint and releases detector executors and unique model components. Detector wrappers close the underlying detector and runner; cleanup errors are logged at worker scope.

## Actual module dependencies

```mermaid
flowchart TD
  F[Foundation: StateStore and persistence]
  I[Intelligence / Cognitive Core]
  V[Vision: Go orchestration + Python worker]
  R[Integration: composition and adapters]
  P[Platform]
  VO[Voice]
  A[Automation]
  I -->|injected Foundation contract| F
  R --> V
  R --> I
  R --> VO
  R --> A
  R --> P
  V -. no backend edge to Cognitive Core .-> I
  F -. no dependency .-> V
```

The seven logical modules and Go package ownership remain unchanged. The architecture checker continues to enforce Go module edges and now additionally parses Python imports to require that `rknnlite` imports remain in `core/rknn_backend.py` only. This check is deliberately limited to one concrete runtime boundary.

## Remaining platform assumptions

- RKNNLite and an RK3588/RKNPU2-compatible runtime/model are required for the current real models.
- NPU core fan-out uses RKNNLite constants, now obtained only through the adapter.
- Existing model paths remain configuration inputs; this milestone does not alter production paths or deploy hardware configuration.
- `cv2`, NumPy, Linux AF_UNIX sockets, and the worker process remain platform dependencies.
- **NOT_IMPLEMENTED:** hardware-independent production inference. Python import success alone does not establish portability.
- **TESTED:** backend substitution with fake runner; actual RK3588 model quality, throughput, and device lifecycle were not qualified in this run.
