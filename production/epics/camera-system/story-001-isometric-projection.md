# Story 001: Isometric Projection Setup

> **Epic**: Camera System
> **Status**: Complete
> **Last Updated**: 2026-08-10
> **Layer**: Foundation
> **Type**: Logic
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Dependencies**: None — projection setup is pure math; can be built and tested in isolation.

## Context

**GDD**: `design/gdd/camera-system.md`
**Requirement**: `TR-cam-001`

**ADR Governing Implementation**: ADR-0003: Isometric Camera Setup
**ADR Decision Summary**: Kaiju `StandardCameraOrthographic` with world-space extents. 30° pitch, 45° yaw, 0° roll — fixed, never rotated. Ortho half-height is the canonical zoom parameter. Reference 16:9.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Verify `cameras.NewStandardCameraOrthographic(width, height, viewWidth, viewHeight, position)` API against `kaiju/src/engine/cameras/`.

**Performance**: No measurable runtime impact — camera created once at startup. Matrix update <0.1ms per ADR-0003.

**Control Manifest Rules**: Required: Orthographic 30°/45° fixed. Forbidden: Never allow camera rotation.

## Acceptance Criteria

- [x] GIVEN the game starts, WHEN the first frame renders, THEN the camera uses orthographic projection at 30° pitch / 45° yaw / 0° roll
- [x] GIVEN the camera is rendering, WHEN the player attempts to rotate the view, THEN the camera angle does not change — rotation is disabled
- [x] GIVEN the game window is resized, WHEN the aspect ratio changes, THEN the ortho half-height remains constant and the horizontal extent adjusts

## Implementation Notes

From ADR-0003: Use `cameras.NewStandardCameraOrthographic(width, height, viewWidth, viewHeight, position)`. Width/height are world units, not pixels — this keeps zoom independent of window size. On resize, re-apply width/height from the current zoom level's canonical values. Near/far clip: 1/1000 world units.

## Out of Scope

- Zoom transitions and interpolation (CAM-002)
- Camera follow, pan, and framing (CAM-003)
- Occlusion handling and boundary enforcement (CAM-004)
- Player input handling for zoom/pan/scroll

## Test Evidence

**Test file**: `tests/unit/camera/isometric_projection_test.go` (20 tests, all passing as of 2026-08-10).

## QA Test Cases

> Back-filled by /qa-plan 2026-08-10. Source: GDD CR-1 acceptance criteria.

- **AC-1 — Projection matrix** (GDD CR-1):
  - Given: Any state at any zoom
  - When: Camera renders
  - Then: View matrix encodes pitch = 30° ± 0.1°, yaw = 45° ± 0.1°, roll = 0° ± 0.1°, projection = orthographic. Verified by matrix dump.

- **AC-2 — Ground plane visibility**:
  - Given: Close zoom (O=4u)
  - When: Camera renders
  - Then: Ground plane visible to viewport bottom edge — no void strip. Bottom 5% of viewport contains rendered ground geometry. Repeat at Default (O=10u) and Far (O=20u).

- **AC-3 — Aspect ratio, 16:9 → 21:9**:
  - Given: 16:9 aspect
  - When: Window resized to 21:9
  - Then: Ortho half-height unchanged (vertical preserved), horizontal extent increases proportionally. Verified by extents readout.

- **AC-4 — Resize mid-transition**:
  - Given: Window resize during zoom transition
  - When: Resize completes
  - Then: Ortho extents immediately re-applied from current interpolated O. Verified by extents readout on frame after resize.

- **AC-5 — Minimum aspect ratio**:
  - Given: Unsupported aspect < 16:10
  - When: Resize completes
  - Then: Camera maintains 16:10 minimum (letterboxing or clamping).

- **AC-6 — Rotation disabled**:
  - Given: Camera is rendering
  - When: Player attempts to rotate the view
  - Then: Camera angle does not change — rotation is disabled. Pitch/yaw/roll unchanged within ±0.1°.

**Edge cases**:
- Near clip at each zoom: Extra Close −1.875, Close −3, Default −7.5, Far −15
- Far clip = 1000 world units at all zooms
- `StandardCameraOrthographic` API verified against `kaiju/src/engine/cameras/` before implementation

**Estimated test count**: 30 unit tests

## Completion Notes
**Completed**: 2026-08-10
**Criteria**: 3/3 passing
**Deviations**: None
**Test Evidence**: Logic — `tests/unit/camera/isometric_projection_test.go` (20 tests, all passing)
**Code Review**: Skipped (lean mode)
**Implementation**: `src/camera/isometric.go` (371 lines) — Config, ZoomLevel constants, Isometric struct wrapping Kaiju StandardCameraOrthographic
