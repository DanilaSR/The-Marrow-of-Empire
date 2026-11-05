# Story 004: Occlusion & Boundaries

> **Epic**: Camera System
> **Status**: Complete
> **Last Updated**: 2026-08-11
> **Layer**: Foundation
> **Type**: Integration
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 001

## Context

**GDD**: `design/gdd/camera-system.md`
**Requirement**: `TR-cam-003`, `TR-cam-004`

**ADR**: ADR-0003 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [x] GIVEN a structure between the camera and the player, WHEN the player moves behind it, THEN the structure fades to transparent (material alpha)
- [x] GIVEN the player enters an interior, WHEN the camera is inside, THEN the roof is not rendered
- [x] GIVEN the camera at a region boundary, WHEN the player attempts to pan beyond it, THEN the camera stops — no void space visible
- [x] GIVEN a 21:9 aspect ratio, WHEN the game renders, THEN the player sees ~33% more horizontal world at the same zoom level
- [x] GIVEN interactable entities behind geometry, WHEN they are within interaction range, THEN they highlight through the occluding geometry

## Implementation Notes

From ADR-0003: Occlusion uses material alpha fade for structures between camera and party. Interiors detect roof entities and disable rendering. Boundary enforcement reads camera bounds from the scene file (Asset Pipeline). Region transitions: fade to black 300ms, reposition, fade in 300ms. "A wall never eats a decision."

## QA Test Cases

> Back-filled by /qa-plan 2026-08-10. Source: GDD CR-5, CR-6, CR-8, Accessibility acceptance criteria.

**Occlusion (CR-8)**:
- **AC-1 — Structure fade**:
  - Given: Structure's screen-space AABB overlaps party screen rect and structure between party and camera in depth
  - When: Occlusion detected
  - Then: Structure fades to `occluder_fade_alpha` (20%) within `occluder_fade_duration` (150ms). Verified via fade-alpha debug readout (not pixel-sampling).

- **AC-2 — Fade hysteresis**:
  - Given: Structure in faded state
  - When: Occlusion clears
  - Then: Structure stays faded ≥ `occluder_fade_min_hold` (300ms), clear must persist ≥ `occluder_fade_min_clear` (300ms) before restore. Walking-alongside-wall test: no alpha cycle faster than one per 2s.

- **AC-3 — Roof removal**:
  - Given: Party enters interior
  - When: Interior is active region
  - Then: Roof/ceiling geometry above party elevation (tags "roof", "ceiling") absent from render (draw-call log). Roofs restore within 150ms on exit.

- **AC-4 — Interactable highlight**:
  - Given: Interactable occluded
  - When: Tab pressed
  - Then: 2px solid outline in parchment `#E3D5B0` (±8/255), unlit, visible through geometry.

- **AC-5 — Combat silhouette**:
  - Given: Combatant occluded during combat
  - When: In frustum
  - Then: Outline visible through geometry: solid (friendly, amber `#D9A441`), dashed (hostile, red `#C4483A`). Shape distinguishable by protanopia-simulating tester at 2px. Always active — no key press.

- **AC-6 — Combat silhouette precedence**:
  - Given: Entity both combatant and interactable during Combat
  - When: Tab held
  - Then: Combat silhouette wins (hostile dashed red / friendly solid amber); parchment suppressed.

- **AC-7 — Shadow exclusion**:
  - Given: Faded structure
  - When: Alpha < 50%
  - Then: Shadow draw instances deactivated. Re-enabled when alpha > 95% on restore. Verified via shadow-pass draw-call log.

**Boundary enforcement (CR-6)**:
- **AC-8 — Boundary clamp**:
  - Given: Camera at region boundary
  - When: Player pan-beyond
  - Then: Camera clamps such that viewport shows no void (using AABB of 45°-rotated viewport rect). Automated check: all four edges + corners at 16:9 + 21:9.

- **AC-9 — Edge scroll over UI**:
  - Given: Cursor over UI panel
  - When: At screen edge
  - Then: Edge scroll does NOT fire. Keyboard/drag/gamepad pan functional.

**Camera shake (CR-5)**:
- **AC-10 — Critical hit shake**:
  - Given: Critical hit
  - When: Hit connects
  - Then: Lateral offset achieves peak 1.27% ± 0.2% of shorter viewport dimension at 8 Hz sine's first crest with linear falloff, decays to zero by 200ms (±20ms). Deterministic replay within 1 logical pixel.

- **AC-11 — Environmental collapse shake**:
  - Given: Environmental collapse
  - When: Event fires
  - Then: Offset achieves peak 4.8% ± 0.3% with ease-out falloff, decays to zero by 800ms (±40ms). No achieved peak exceeds 5.0% — the hard cap.

- **AC-12 — Reduce Camera Motion = Disabled**:
  - Given: Reduce Camera Motion = Disabled
  - When: Any shake event triggers
  - Then: Offset = zero.

- **AC-13 — Reduce Camera Motion = Reduced**:
  - Given: Reduce Camera Motion = Reduced
  - When: Shake triggers
  - Then: Achieved peak = 50% ± 5% of standard, duration = 50% ± 10% of standard.

- **AC-14 — Shake suppressed during conversation**:
  - Given: Conversation active
  - When: Any shake triggers
  - Then: Offset = zero regardless of Reduce Camera Motion setting.

- **AC-15 — Overlapping shake events**:
  - Given: Two shake events overlap (critical hit 200ms + explosion 300ms, 50ms apart)
  - When: Both active
  - Then: Offset = signed offset with largest magnitude (not sum). Sign preserved.

- **AC-16 — Shake at boundary**:
  - Given: Shake at region boundary
  - When: Offset would show void
  - Then: Amplitude reduced for that frame. Waveform phase continues.

**Accessibility**:
- **AC-17 — First-launch prompt**:
  - Given: First launch (fresh profile)
  - When: First frame renders
  - Then: Accessibility prompt displayed: Reduced Camera Motion + Subtitles, keyboard/gamepad-navigable, no timeout. Escape/Skip = accessible defaults (Reduced + subtitles ON).

- **AC-18 — Accessibility persistence**:
  - Given: Reduce Camera Motion = Disabled
  - When: Relaunched
  - Then: Setting persists from profile.

- **AC-19 — Reduced motion behavior**:
  - Given: Reduce Camera Motion = Reduced (first-launch default)
  - When: Any state
  - Then: Zoom transitions = instant snaps, no conversation re-frames after initial, combat framing = snaps.

- **AC-20 — Extra Close zoom**:
  - Given: Extra Close Zoom enabled
  - When: Player zooms in from Close
  - Then: Camera transitions to O = 2.5u.

**Edge cases**:
- Occlusion detection: zero spatial-index range queries per frame. Structure list built at region load.
- Foliage materials use higher alpha floor (35% vs 20% solid)
- Viewport_height_px = 0 (minimized) → skip shake, no NaN
- Shake never uses rotation (strongest vestibular trigger)

**Estimated test count**: 9 integration + 20 unit tests
**Test file**: `tests/integration/camera/occlusion_boundaries_test.go` + `tests/unit/camera/occlusion_test.go`

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 6/20 core implemented, 14 deferred
**Implemented**: AC-1 (fade 20%), AC-2 (hysteresis), AC-3 (roof removal), AC-4 (interactable highlight), AC-8 (boundary clamp), AC-9 (edge scroll over UI)
**Deferred**: AC-5/6 (combat silhouettes — needs combat system), AC-7 (shadow exclusion — needs renderer), AC-10-16 (camera shake — separate subsystem), AC-17-19 (accessibility — needs settings), AC-20 (Extra Close — implemented in CAM-002)
**Test Evidence**: `tests/unit/camera/occlusion_test.go` (20 tests) + `tests/integration/camera/occlusion_boundaries_test.go` (9 tests), all passing
**Code Review**: Skipped (Lean mode)
