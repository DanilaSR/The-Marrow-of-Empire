# Story 002: Zoom System

> **Epic**: Camera System
> **Status**: Complete
> **Last Updated**: 2026-08-10
> **Layer**: Foundation
> **Type**: Logic
> **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 001

## Context

**GDD**: `design/gdd/camera-system.md`
**Requirement**: `TR-cam-002`

**ADR**: ADR-0003 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [x] GIVEN Default zoom (10u), WHEN mouse wheel up, THEN camera transitions to Close (4u) over 300ms
- [x] GIVEN Default zoom, WHEN mouse wheel down, THEN camera transitions to Far (20u) over 300ms
- [x] GIVEN Close zoom, WHEN mouse wheel up, THEN no further zoom in — already at minimum
- [x] GIVEN Far zoom, WHEN mouse wheel down, THEN no further zoom out — already at maximum
- [x] GIVEN any zoom transition, WHEN measured, THEN the transition uses smooth lerp over 300ms

## Implementation Notes

From ADR-0003: Three discrete zoom registers stored as ortho half-height: Close 4u, Default 10u, Far 20u. No free scroll wheel — discrete steps only. On zoom change, lerp the ortho extents from current to target over 300ms.

## QA Test Cases

> Back-filled by /qa-plan 2026-08-10. Source: GDD CR-2 acceptance criteria.

- **AC-1 — Start zoom**:
  - Given: Game starts
  - When: First Exploration frame renders after world load
  - Then: Camera in Exploration at Default zoom (O=10u), centered on party centroid within 32 screen pixels. Waived if centroid within one viewport half-extent of region boundary.

- **AC-2 — Zoom in**:
  - Given: Player scrolls wheel up at Default
  - When: Wheel processed
  - Then: O transitions monotonically from 10u to 4u within `zoom_transition_duration` (300ms), no overshoot.

- **AC-3 — Zoom out**:
  - Given: Player scrolls wheel down at Default
  - When: Processed
  - Then: O transitions to 20u within `zoom_transition_duration`.

- **AC-4 — Clamp at min**:
  - Given: Close zoom (O=4u)
  - When: Wheel up
  - Then: No effect (< 0.05u change over 500ms).

- **AC-5 — Clamp at max**:
  - Given: Far zoom (O=20u)
  - When: Wheel down
  - Then: No effect.

- **AC-6 — Extra Close (accessibility)**:
  - Given: Extra Close Zoom enabled
  - When: Player scrolls up at Close
  - Then: O transitions to 2.5u.
  - Given: Extra Close Zoom disabled
  - When: Wheel up at Close
  - Then: No effect.

- **AC-7 — Rapid input during transition**:
  - Given: 3+ detents during transition
  - When: Transition in flight
  - Then: At most one pending level queued. Intermediate levels skipped. Last-input-wins.

- **AC-8 — Zoom persistence across context**:
  - Given: Player at Close (exploration)
  - When: Enter Conversation → exit
  - Then: Camera returns to Close (last Exploration zoom), not Default.

- **AC-9 — Save/load zoom**:
  - Given: Game saved at Far zoom
  - When: Loaded
  - Then: Camera at Far at saved position (not party centroid).

**Edge cases**:
- Zero-size viewport (minimized window) → zoom transition paused, resumed on restore
- Delta-time-based lerp: frame times ≥ 45ms → duration stretches to maintain smoothness, no pop

**Estimated test count**: 15 unit tests
**Test file**: `tests/unit/camera/zoom_system_test.go`

## Completion Notes
**Completed**: 2026-08-10
**Criteria**: 5/5 passing
**Deviations**: None
**Test Evidence**: Logic — `tests/unit/camera/zoom_system_test.go` (12 tests, all passing; 40 camera tests total)
**Code Review**: Skipped (lean mode)
**Implementation**: `src/camera/zoom.go` (230 lines) + `src/camera/camera.go` (extended) — ZoomTo, ZoomStep, Update, CurrentHalfHeight, SetContext, ZoomState/SetZoomState
