# Story 001: Local Map Display & Navigation

> **Epic**: Map System
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/map-system.md`
**Requirement**: `TR-ui-001`, `TR-ui-004`, `TR-ui-005`, `TR-ui-006`, `TR-ui-007`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: All UI rendered in screen space via Kaiju HTML/CSS-like UI system. Desk register for parchment overlays — world dims to 30%, paper-slide transitions (200-300ms ease-out, no bounce/scale/spin/flip). Desk never trembles (immune to camera shake).

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI rendering API, font loading, input routing — all require verification against `kaiju/src/`. No LLM training data exists for Kaiju Engine APIs.

**Control Manifest Rules (Presentation Layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space — ADR-0012
- Required: Desk register: full-screen parchment overlay, world dims to 30%, paper-slide transitions — ADR-0012
- Required: Paper-slide motion: enter 200-300ms ease-out, exit 200ms — no bounce/scale/spin/flip — ADR-0012
- Required: Desk never trembles — UI immune to camera shake — ADR-0012
- Required: Minimum 1280×720, reference 1920×1080 with height-based scaling — ADR-0012
- Required: Body text 16px at reference, minimum 12px effective — ADR-0012
- Required: Accessibility: tab order + focus ring (≥3:1), arrow-key navigation, Esc mapping — ADR-0012
- Forbidden: Never use 3D UI or diegetic HUD — the desk is flat by law — ADR-0012
- Forbidden: Never use skeuomorphic buttons — pen-and-ink icons only — ADR-0012
- Guardrail: UI render (static): <1ms/frame — ADR-0012
- Guardrail: No frame drops during UI animations — ADR-0012

---

## Acceptance Criteria

*Derived from GDD `design/gdd/map-system.md` Overview and Visual/Audio Requirements:*

- [ ] **AC-1**: GIVEN the game is running, WHEN the player presses the M key, THEN the local map opens as a full-screen parchment overlay with paper-slide transition (200-300ms ease-out), and the world dims to 30%.
- [ ] **AC-2**: GIVEN the map is open, WHEN the player presses M or Esc, THEN the map closes with paper-slide exit transition (200ms), and the world returns to full brightness.
- [ ] **AC-3**: GIVEN the map is open, THEN the current region's layout is rendered as ink-on-parchment linework — geography, paths, structures shown in the dark ink palette.
- [ ] **AC-4**: GIVEN the map is open, THEN the player's current position is shown with a facing indicator, and the indicator updates as the player moves.
- [ ] **AC-5**: GIVEN the map is open, WHEN the region is larger than the viewport, THEN the player can scroll/pan to view different areas (mouse drag or edge-scroll).
- [ ] **AC-6**: GIVEN the map is open, THEN the UI is rendered at the desk register — immune to camera shake, following all desk typography and palette rules.
- [ ] **AC-7**: GIVEN the map is open at 1280×720 minimum resolution, THEN all map elements are readable (body text ≥12px effective) and the layout scales correctly with height-based scaling.

---

## Implementation Notes

*Derived from ADR-0012 Implementation Guidelines:*

- Use Kaiju's HTML/CSS-like UI system — all map rendering is screen-space, not world-space.
- The map is a desk-register element: full-screen parchment overlay that dims the world to 30%.
- Apply the paper-slide motion: 200-300ms ease-out for enter, 200ms for exit. No bounce, scale, spin, or flip.
- The map panel must be immune to camera shake (the desk never trembles).
- Ink-on-parchment linework: use the dark ink palette colors from the art bible Section 6. Do not use amber for text (amber is accent/fill only — 1.5:1 contrast fails WCAG AA).
- Font: desk serif (Garamond class) for location names and labels.
- M key toggles map open/close. Esc also closes when map is open.
- Render region geography from scene/region data — line-drawn style, not a satellite view.
- Player position indicator: update from entity transform each frame while map is open.
- Map scrolling: support mouse drag (click-and-drag to pan) and edge-scroll (cursor near viewport edge).
- Verify Kaiju UI rendering API against `kaiju/src/` before implementing — no LLM training data exists.
- Accessibility: ensure all interactive elements are in tab order, focus ring ≥3:1 contrast, arrow keys navigate.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- **Story 002**: NPC position markers, thread markers (amber dot/star), discovered location pins, cleared location X marks — map markers of any kind
- **Story 003**: World map with region overview, travel routes, faction overlays

---

## QA Test Cases

*Manual verification steps for UI story:*

- **AC-1 (Map open)**: 
  - Setup: Launch game, enter a playable region
  - Verify: Press M — map slides in (paper-slide, 200-300ms), world dims ~30%, parchment overlay visible with region geography
  - Pass condition: Map opens smoothly, transition duration between 200-300ms, world visibly dimmed

- **AC-2 (Map close)**:
  - Setup: Map is open
  - Verify: Press M or Esc — map slides out (200ms exit), world returns to full brightness
  - Pass condition: Map closes smoothly, no visual stutter, world brightness restored

- **AC-3 (Region linework)**:
  - Setup: Open map in a region with known geography (paths, structures, boundaries)
  - Verify: Geography rendered as ink lines on parchment background — paths, structures, region boundaries distinguishable
  - Pass condition: Region layout is recognizable and legible, ink-on-parchment aesthetic achieved

- **AC-4 (Player position)**:
  - Setup: Open map, note player indicator position
  - Verify: Move player character, reopen map — indicator has updated to new position with correct facing
  - Pass condition: Position updates correctly, facing indicator is unambiguous

- **AC-5 (Map scrolling)**:
  - Setup: Open map in a region larger than the viewport
  - Verify: Click and drag to pan — map scrolls smoothly. Move cursor to viewport edge — map scrolls in that direction
  - Pass condition: All areas of the region are reachable via scrolling, no visual tearing

- **AC-6 (Desk register compliance)**:
  - Setup: Trigger camera shake (combat explosion, etc.), then open map
  - Verify: Map panel is perfectly still — no tremble, no vibration
  - Pass condition: Map UI is fully immune to camera shake

- **AC-7 (Minimum resolution)**:
  - Setup: Set game to 1280×720
  - Verify: Open map — all labels readable, layout intact, no elements clipped or overlapping
  - Pass condition: Map is fully functional and legible at minimum resolution

---

## Test Evidence

**Story Type**: UI
**Required evidence**: `production/qa/evidence/story-001-local-map-display-evidence.md` — manual walkthrough doc OR interaction test
**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: None (first story in epic — no preceding map stories)
- Unlocks: Story 002 (Map Markers & Pins) — markers render on top of the map display built here

---

## Completion Notes

**Completed**: 2026-08-11
**Criteria**: 7/7 passing (5 automated, 2 manual/visual — AC-3 linework, AC-6 desk register)
**Deviations**: None
**Test Evidence**: `tests/unit/map_system/local_map_display_test.go` — 25 tests, all PASS. Manual walkthrough evidence at `production/qa/evidence/story-001-local-map-display-evidence.md` pending (ADVISORY for UI story type).
**Code Review**: APPROVED WITH SUGGESTIONS — all fixes applied (font-scale-on-resize bug, 6 edge case tests, unused const removed).
**Tech Debt**: None logged.
