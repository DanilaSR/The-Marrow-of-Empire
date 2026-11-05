# ADR-0003: Isometric Camera Setup

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Rendering |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | Camera System GDD (approved), `kaiju/src/engine/cameras/` |
| **Post-Cutoff APIs Used** | `cameras.NewStandardCameraOrthographic(width, height, viewWidth, viewHeight, position)` |
| **Verification Required** | Orthographic camera API stability in Kaiju nightly, aspect ratio handling, occlusion/fade support |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — camera reads entity positions for framing) |
| **Enables** | All rendering — the camera IS the viewpoint |
| **Blocks** | Scene rendering — no camera, no visible game |
| **Ordering Note** | Must be Accepted before any rendering work begins |

## Context

The Marrow of Empire uses a fixed isometric orthographic camera at 30° pitch / 45° yaw with three discrete zoom levels (Close 4u / Default 10u / Far 20u). The player cannot rotate the view. Kaiju Engine provides `NewStandardCameraOrthographic`. This ADR defines how the camera is configured, how occlusion is handled, and how aspect ratios are managed.

## Decision

**Kaiju `StandardCameraOrthographic` with world-space extents.** The camera is created with world-unit dimensions (not pixel viewport dimensions) so zoom levels are independent of window resize. The orthographic half-height is the canonical zoom parameter. Three zoom registers: Close at 4u, Default at 10u, Far at 20u.

### Key Configuration

| Parameter | Value |
|-----------|-------|
| Projection | Orthographic |
| Pitch | 30° from horizontal |
| Yaw | 45° from cardinal axes |
| Roll | 0° |
| Near/Far clip | −0.75 × O / 1000 world units (zoom-derived near — amended 2026-08-10, see note) |
| Reference aspect | 16:9 |
| Minimum aspect | 16:10 |
| Zoom registers | Close 4u / Default 10u / Far 20u (ortho half-height) |

> **Amendment 2026-08-10 (Story CAM-001)**: The near clip is zoom-derived —
> `near = −0.75 × O` (Extra Close −1.875, Close −3, Default −7.5, Far −15) —
> not the originally specified +1. A fixed +1 near plane clips the ground plane
> at the viewport bottom edge, producing a 22–33% void strip at every zoom
> (GDD CR-1 revision 2026-08-09). Kaiju `SetNearPlane` accepts negative values
> (its orthographic default is −1). Far clip remains 1000 world units.
> Implemented and verified in `src/camera/camera.go`.

### Occlusion Handling
Structures between camera and party fade to transparent via material alpha. Interiors render without roofs when occupied. Interactable entities highlight through geometry. "A wall never eats a decision."

### Aspect Ratio
The orthographic half-height is constant across aspect ratios. At 21:9, the player sees ~33% more horizontal world — acceptable per the approved Camera GDD.

## GDD Requirements Addressed
Camera System GDD: CR-1 (Isometric Projection), CR-2 (Zoom Levels), CR-6 (Boundary Enforcement), plus occlusion handling and aspect ratio strategy from the reviewed GDD.

## Performance Implications
Camera matrix update: <0.1ms per frame. Occlusion checks: <1ms per frame (frustum culling + fade material swap).

## Related Decisions
ADR-0001: Entity Architecture — camera reads entity positions for follow/framing.
