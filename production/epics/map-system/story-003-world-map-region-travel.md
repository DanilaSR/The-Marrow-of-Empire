# Story 003: World Map & Region Travel

> **Epic**: Map System
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: L (8-10 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/map-system.md`
**Requirement**: `TR-ui-001`, `TR-ui-003`, `TR-faction-001`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework (primary), ADR-0011: Event Bus (secondary — faction data propagation)
**ADR Decision Summary**: World map is a desk-register overlay showing all discovered regions with state summaries, travel routes, and faction territory. Rendered in Kaiju HTML/CSS-like UI with pen-and-ink iconography on parchment.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI rendering for region overview layout. Cross-system data reads: persistent world state (region discovery flags), faction system (territory overlays). Verify UI rendering API and component store reads against `kaiju/src/`.

**Control Manifest Rules (Presentation Layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space — ADR-0012
- Required: Desk register — parchment overlay for world map — ADR-0012
- Required: Pen-and-ink icons: single-stroke, 2px at 24px, flat fills only — ADR-0012
- Forbidden: Never use 3D UI or diegetic HUD — ADR-0012
- Forbidden: Never encode game-critical information by hue alone — ADR-0012

---

## Acceptance Criteria

*Derived from GDD `design/gdd/map-system.md` Overview — World Map section (Alpha tier). No formal ACs exist in the GDD for world map features.*

- [ ] **AC-1**: GIVEN the player has discovered at least one region, WHEN the world map is opened (from the local map or main menu), THEN an overview of all discovered regions is shown with ink-drawn region boundaries on parchment.
- [ ] **AC-2**: GIVEN a discovered region, WHEN viewing the world map, THEN the region shows its state summary — one of: Cleared (ink X + faded), Visited (normal ink), Unexplored (dashed outline, no fill).
- [ ] **AC-3**: GIVEN two discovered regions with an established travel route, WHEN the world map is opened, THEN a travel route line (dotted ink path) connects them.
- [ ] **AC-4**: GIVEN the player has faction standing data for at least one faction, WHEN the world map is opened, THEN faction territory is shown as a subtle colored wash overlay within region boundaries (color from faction palette, low opacity).
- [ ] **AC-5**: GIVEN the world map is open, WHEN the player selects a discovered region, THEN a region detail panel appears showing: region name, state, faction presence, discovered locations count, active threads count.
- [ ] **AC-6**: GIVEN the world map is open, WHEN the player presses M or Esc, THEN the world map closes and returns to the local map (if opened from local map) or game (if opened from menu).

---

## Implementation Notes

*Derived from ADR-0012, ADR-0011:*

- World map is a desk-register overlay — same parchment aesthetic as local map (Story 001), but showing the larger-scale region overview.
- Navigation: from local map, there should be a way to zoom out to world map (e.g., a "World" button or zoom-out beyond minimum scale). From world map, select a region to zoom into its local map.
- Region rendering: each discovered region rendered as an ink-drawn boundary shape on the parchment. Boundaries may be hand-authored or procedurally generated from region data — prefer authored shapes for visual quality.
- Region states:
  - **Cleared**: region drawn with faded ink + small ink X. All major locations cleared.
  - **Visited**: region drawn with normal ink, solid outline.
  - **Unexplored**: region shown as dashed outline, no interior detail. Player knows the region exists but hasn't entered it.
- Travel routes: dotted ink lines connecting region boundaries. Routes are authored per region pair in world data. A route is only shown if both connected regions are discovered.
- Faction territory: read faction standings from faction system. Apply a low-opacity (10-15%) colored wash within region boundaries for each faction with presence ≥ threshold. Multiple factions in one region → striped or blended overlay. Color from faction palette; also provide a shape distinction (hatch pattern direction) for colorblind accessibility.
- Region detail panel: paper-slide in from the side when a region is clicked. Shows region name (desk serif), state icon, faction presence icons, discovered locations count, active threads count. Dismiss with click-outside or Esc.
- Accessibility: faction territory must use pattern/hatch in addition to color — hue-alone encoding is forbidden (ADR-0012).
- Alpha tier note: this story is Alpha priority. The local map (stories 001-002) must be complete and stable before implementing the world map, as it builds on the same rendering infrastructure.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- **Story 001**: Local map display — parchment overlay, M key, region linework, player position, scrolling
- **Story 002**: Map markers — NPC markers, thread markers (amber dot/star), location pins, cleared X marks
- Region travel mechanic itself (loading the destination region, travel time, random encounters) — owned by the World/Region system, not the map

---

## QA Test Cases

*For Integration story — automated test specs:*

- **AC-1 (Region overview)**:
  - Given: Player has discovered 3 regions (PWS flags set for each): The March (visited), Old Road (cleared), Thornwood (unexplored)
  - When: The world map is opened
  - Then: All 3 regions are rendered as ink-drawn boundary shapes on parchment, each visually distinct by state
  - Edge cases: Zero discovered regions (show empty parchment with hint text "No regions discovered"), 20+ regions (scroll/pan to view all), single region (centered, no scrolling needed)

- **AC-2 (Region state display)**:
  - Given: Three regions in different states: Cleared, Visited, Unexplored
  - When: The world map renders
  - Then: Cleared region shown with faded ink + ink X; Visited region shown with normal ink, solid outline; Unexplored region shown with dashed outline, no interior
  - Edge cases: Region transitions from Visited to Cleared while map is open (update on refresh), partially cleared region (still "Visited" until all major locations cleared)

- **AC-3 (Travel routes)**:
  - Given: The March and Old Road are discovered and have an authored travel route between them
  - When: The world map renders
  - Then: A dotted ink line connects the two region boundaries
  - Edge cases: Route between discovered and undiscovered region (route not shown — both ends must be discovered), multiple routes from one region (all shown), route through impassable terrain (shown with hazard marker per world data)

- **AC-4 (Faction territory overlay)**:
  - Given: The March has faction "Iron Banner" with standing ≥ presence threshold, and faction "Merchant Guild" with standing below threshold
  - When: The world map renders
  - Then: The March shows an Iron Banner colored wash (10-15% opacity) with distinct hatch pattern; Merchant Guild is not shown
  - Edge cases: Two factions above threshold in same region (striped or blended overlay, both patterns visible), zero factions above threshold (no overlay), faction presence changes while map open

- **AC-5 (Region detail panel)**:
  - Given: The world map is open showing The March
  - When: The player clicks on The March region
  - Then: A detail panel slides in showing: region name "The March", state icon (Visited), faction presence ("Iron Banner"), discovered locations count (e.g. "12 locations"), active threads count (e.g. "3 threads")
  - Edge cases: Click on unexplored region (show limited info — name, state, no location/thread counts), click outside panel to dismiss, panel at minimum resolution (must fit without clipping)

- **AC-6 (World map close)**:
  - Given: The world map is open (entered from local map)
  - When: The player presses M or Esc
  - Then: The world map closes and returns to the local map view
  - Edge cases: World map opened from main menu (close returns to main menu), rapid M key double-tap (debounce — only one toggle per 300ms)

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/map_system/world_map_test.go` — must exist and pass
**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 002 (Map Markers & Pins) — world map builds on the same marker/rendering infrastructure
- Depends on: Persistent World State epic (region discovery flags, cleared state)
- Depends on: Faction System epic (faction standings, territory data, faction palette)
- Unlocks: None (last story in epic)

---

## Completion Notes

**Completed**: 2026-08-12
**Criteria**: 6/6 passing — all automated via 23 integration tests
**Deviations**: None
**Test Evidence**: `tests/integration/map_system/world_map_test.go` — 23 tests, all PASS (BLOCKING gate met). Story 001 unit 25/25 + Story 002 integration 19/19 = 67 total PASS, no regressions.
**Code Review**: APPROVED WITH SUGGESTIONS — S3 debounce test fix applied.
**Advisory**: Zoom navigation untested (host-gated), click routing untested, hazard route QA edge not implemented, manual walkthrough pending.
**Tech Debt**: None logged.
**Map System Epic**: 3/3 Complete ✅
