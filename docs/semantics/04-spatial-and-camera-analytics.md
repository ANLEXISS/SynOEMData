# Spatial and Camera Analytics Audit

**Finding:** Synora has topology labels and track/camera references, but does not currently define product contracts for analytics zones or camera-view geometry. This document specifies missing contracts only; zone detection and geometry processing are not implemented here.

## Existing reusable structures

- `pkg/contract.Topology`, `Node`, `TopologyLink`, and Intelligence topology config model logical nodes and links.
- `contract.Device` has `NodeID`, `ZoneRole`, `RoomName`; `CameraObservation` has camera identity, capabilities, online and last-seen.
- Vision detections/tracks have camera/clip/track references and frame-oriented evidence. Python edge tracking preserves object continuity for a clip, and camera health/continuity contracts represent gaps and uncertainty.
- `internal/foundationv1/contracts.go` defines a software-only `Zone` with category, coverage, uncertainty, provenance, expiry and device references, plus topology transitions. The package states its contracts are not connected to production peripherals or qualified inputs. It provides reference vocabulary, but no geometry, camera-to-zone association, rule execution or production support.
- ONVIF support is not present in the integration inventory; there is no claim of ONVIF support or protocol conformance.

These structures can anchor future references, but `ZoneRole` and `room_name` are not a geometry model.

## Minimal additive contracts to design

1. **SpatialFrame** — stable frame ID, coordinate kind (`image_pixels`, `image_normalized`, `ptz`, `building_local`, `geodetic`), units, dimensions/origin/orientation, camera/preset/stream reference, and transform/version when conversion is supported.
2. **ZoneDefinition** — stable ID/version, semantic purpose, camera/source association, geometry reference, inclusion/exclusion role, lifecycle, and privacy policy reference. Separate analytics zones from privacy masks because they have different policy and retention semantics.
3. **Geometry** — tagged point/polyline/simple polygon with frame ref and validation rules (finite values, normalized bounds if applicable, minimum points, non-self-intersecting polygon). Geometry has no implicit coordinate frame.
4. **RuleDefinition** — rule ID/version, scope, enabled state, subject/object filters, condition (`inside`, `enter`, `exit`, `cross`, `dwell`), duration/window, direction where relevant, confidence threshold and unknown-data behavior.
5. **ZoneObservation / BoundaryEvent** — source/time/ingest-time, camera and zone refs, track/entity ref, event (`entered`, `exited`, `crossed`, `loitering_started`, `loitering_ended`), direction, confidence, evidence refs, rule revision and correlation ID.
6. **PTZAssociation** — camera ID, PTZ coordinate frame, preset token or calibrated pose, transform/version, and rules valid for that view. For moving PTZ, rule evaluation must bind observations to pose/preset and define behavior during transition or unknown pose.
7. **TopologyRelation** — typed relation between zone and room/node (within, adjacent-to, entry-for, visible-from) with source and validity. Physical and view-space containment remain distinct.

Minimal status values should include active/disabled/unsupported/invalid and observation outcomes known/unknown/unavailable. Do not infer an exit when tracking is lost; emit a continuity gap/unknown unless policy explicitly defines closure.

## Case study coverage

| Capability | Current code evidence | Missing contract |
|---|---|---|
| Detection / semantic zones | `ZoneRole` string and logical room/node in production; coarse `foundationv1.Zone` in software-only reference harness | Production zone ID/geometry/version/source association and runtime semantics |
| Exclusion zones | None found | Explicit include/exclude semantics and boundary policy |
| Privacy masks | No privacy-mask geometry contract found | Separate mask geometry, enforcement point, audit/version, fail-safe behavior |
| Polygons | No production polygon type found | Frame-bound validated simple polygon |
| Virtual lines | No line detector/rule type found | Ordered polyline, side/direction semantics, boundary crossing policy |
| Entry / exit | Vision event families are detection categories, not zone transitions | Stateful crossing semantics and tracking-gap behavior |
| Line crossing | No production contract found | Direction, track and line refs, time and evidence |
| Dwell / loitering | Episode/temporal keys exist but no zone-scoped threshold rule | Duration/window, start/end behavior, reset and gap policy |
| Direction | No typed movement vector/bearing in shared contract found | Direction in coordinate frame plus uncertainty |
| Object class filters | Python detection classes human/vehicle/animal/unknown and V1 subject classes | Versioned class vocabulary / extension strategy and rule filter semantics |
| Confidence thresholds | Many numeric confidence fields exist | Calibrated meaning, threshold scope and unknown handling |
| Camera-to-zone | Camera ID + topology references are independent | Versioned association and source-of-geometry provenance |
| PTZ frames/presets | No typed PTZ/preset association found | Coordinate transform and pose/preset at observation time |
| Physical topology mapping | Nodes/links, room names and room baselines | Floor/room geometry, containment/adjacency relation and transform |

## Interoperability references (not a compliance claim)

ONVIF Analytics Service describes scene descriptions, analytics modules and rules; its analytics model includes normalized polygon/polyline geometry, field/line detectors, loitering rules, and PTZ-status/preset context for view-dependent rules. ONVIF Profile M covers analytics metadata/events and generic object classification. These are useful decompositions for an adapter boundary, not proof that Synora implements them. [ONVIF Analytics Service](https://www.onvif.org/specs/2406/ONVIF-Analytics-Service-Spec-v2406.pdf), [Profile M](https://www.onvif.org/profiles/profile-m/).

Matter's data model is organized around nodes/endpoints, device types and functional clusters, with optional/vendor additions; its device vocabulary can inform capability discovery, but its domain is not a substitute for a camera analytics geometry schema. Matter 1.5 announced camera, closure and energy-management additions. Synora has not been checked or certified against Matter. [Matter specification](https://csa-iot.org/wp-content/uploads/2023/10/Matter-1.2-Core-Specification.pdf), [Matter 1.5 release](https://csa-iot.org/newsroom/matter-1-5-introduces-cameras-closures-and-enhanced-energy-management-capabilities/).

## Safe future sequence

First define frame and geometry validation with golden JSON fixtures. Then add static camera/zone associations and offline rule evaluation using fake detections. Only after that consider ONVIF/vendor adapters and PTZ. Validate privacy-mask behavior independently. Keep all schema work additive and preserve existing camera and event payloads until a migration is explicitly designed.
