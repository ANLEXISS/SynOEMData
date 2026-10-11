# Synora Camera V0

One native C++ process for one EM1103B/RV1103B camera. It is intentionally an
edge-evidence producer, never an identity or scene-understanding system.

## Current prototype

The default binary runs a deterministic hardware simulation on a normal Linux
machine. It produces 5 compressed H.265-shaped access units per second, stores
only a bounded 10-second compressed prebuffer, opens a `person_suspect` event
after a simulated NPU human probability, associates boxes with a tiny local
tracker, and writes the resulting event contract plus clip spool to its data
directory.

```sh
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build
ctest --test-dir build --output-on-failure
./build/synora-camera --duration 35 --data-dir ./runtime
```

`runtime/outbox/events.jsonl` is the durable hand-off point for the future
central uploader. A clip is written directly to `runtime/clips`; it is not
held in memory. The simulated `.h265` bytes are deliberately not a playable
video. On the board, `HardwarePipeline` will be implemented with Rockchip
VI/ISP + VENC H.265 + RKNN, preserving this same interface and contract.

## Central-ingress software mock

`synora-camera-mock` is a separate development executable. It runs the V0
`CameraDaemon` and `SimulatedPipeline` in a C++20 `std::jthread`, and prints a
machine-readable `simulated_camera:true` marker. On `person_suspect.close`, it
copies the explicitly supplied fixture to `DATA_DIR/pending/` and uploads that
MP4; it never submits the synthetic H.265 spool. `ffprobe` must confirm a video
stream before the mock starts. The durable `DATA_DIR/mock-run.json` repeats the
simulation declaration and fixture hash. Pending MP4 files survive restart and
are only removed after an exact HTTP 202 ACK with `status:"queued"` and the
matching `clip_id`. Timeout, rejection and bounded retry exhaustion retain the
file; a sidecar preserves the retry count across restarts. Use at least 23 seconds
to include the unchanged 20-second V0 postroll and observe a finalized clip.

The ingress URL must be the actual Discovery URL including `/vision`. Device
authentication follows the current ingress contract: `X-Synora-Device`,
`X-Synora-Timestamp`, and `X-Synora-Signature`, where the signature is HMAC-SHA256
over `device_id + timestamp + fixture_sha256`, keyed with the configured
transport verifier. Provide that verifier only through
`SYNORA_CAMERA_SECRET_HASH`; do not put it in source or shell history. TLS
verification remains enabled by curl defaults. Runtime dependencies are
`curl`, `openssl`, and `ffprobe` on `PATH`.

Reproducible local demonstration (all values are supplied by the operator):

```sh
cmake -S camera/v0 -B build/camera-v0 -DCMAKE_BUILD_TYPE=Release
cmake --build build/camera-v0 --target synora-camera-mock
SYNORA_CAMERA_SECRET_HASH="$SYNORA_CAMERA_SECRET_HASH" \
  build/camera-v0/synora-camera-mock \
  --camera-id "$CAMERA_ID" \
  --data-dir "$MOCK_DATA_DIR" \
  --fixture-clip "$FIXTURE_MP4" \
  --ingest-url "$VISION_INGRESS_URL" \
  --duration 30 --scenario success
```

`--scenario` also accepts `timeout`, `offline`, `duplicate`, and `queue-full`.
These modes exercise uploader behavior only. A successful ingress ACK proves a
software upload, not Vision model availability or camera qualification. This
mock cannot mark `qualify-camera`, J2, J3, or J4 qualified and makes no claim
about RK3588/EM1103B hardware, capture quality, or model performance.

`make test-central-v1` runs the C++ mock tests and the hermetic Bus/Core/Store
suite; it does not start the HTTP clip ingress or call `/api/system/state` from
this mock. Therefore a complete fixture-upload-to-system-state central E2E is
still `not_qualified`. Run the demo against an explicitly configured central
Discovery only when its paired camera identity, device verifier, Vision worker,
and API session are available.

## Frozen V0 boundary

The only event names are `person_suspect`, `motion_unknown`, `tamper`, and
`health`. `person_suspect` has `open`, `update`, and `close` phases. The open
event declares a 10-second prebuffer and 20-second postroll; the close event
names the completed H.265 clip. The prebuffer is capped at 4 MiB by default;
no decoded image is persisted.

There is no face detector, facial embedding, recognition, pose, object class,
animal classifier, re-identification, database, rule engine, Python runtime,
container, or GStreamer pipeline in this component.

## Board integration next

Implement one `HardwarePipeline` adapter, not new services:

1. Rockchip VI/ISP and VENC supply already compressed H.265 access units.
2. The same source offers a 320x320 inference image only while motion/PIR is active.
3. RKNN runs one INT8 human-probability model at 3–5 fps, one inference at a time.
4. A bounded, eMMC-backed uploader transmits events/clips and deletes a clip only after the central acknowledgement.

The OpenRC file is the Alpine deployment supervisor; the systemd unit is for
the prototype machine. Both restart the sole daemon if it exits. The board
adapter will also feed the hardware `/dev/watchdog`; this is deliberately not
faked on the proto host. The update mechanism stays outside this daemon and
must only invoke verified signed A/B update artifacts; the daemon reports the
`signed-only` policy in health rather than accepting remote update commands.
