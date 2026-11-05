# Story 002: Map Markers & Pins

> **Epic**: Map System
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: M (5-7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/map-system.md`
**Requirement**: `TR-ui-001`, `TR-ui-003`, `TR-kg-005`, `TR-thread-001`, `TR-thread-004`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework (primary), ADR-0005: NPC Knowledge Graph (secondary — entity references), ADR-0010: Thread Investigation (secondary — thread data)
**ADR Decision Summary**: Map markers are rendered as pen-and-ink icons on the desk register. NPC positions sourced from entity transforms via ADR-0001. Thread markers sourced from thread investigation system via ADR-0010. Entity references via ADR-0005 enable cross-referencing NPCs to map positions.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI rendering for dynamic icon placement. Cross-system data reads: entity transform store (ADR-0001), NPC knowledge graph (ADR-0005), thread investigation PlayerKnowledge (ADR-0010), persistent world state (ADR-0013). Verify all component store read APIs against `kaiju/src/`.

**Control Manifest Rules (Presentation Layer)**:
- Required: Pen-and-ink icons: single-stroke, 2px at 24px, flat fills only for palette colors, no gradients/bevels — ADR-0012
- Required: Desk register — all map UI in screen space, parchment overlay — ADR-0012
- Required: Never encode game-critical information by hue alone — all colors have shape/icon/value/sound backup — ADR-0012
- Forbidden: Never use amber-on-parchment text — amber is accent/fill only — ADR-0012

---

## Acceptance Criteria

*From GDD `design/gdd/map-system.md` Acceptance Criteria:*

- [ ] **AC-1**: GIVEN a discovered NPC location, WHEN the map is opened, THEN the NPC is shown at their last known position with their name.
- [ ] **AC-2**: GIVEN an active thread referencing a location, WHEN the map is opened, THEN an amber dot marks that location.

*Derived from GDD Overview (additional marker types described but not formally specified):*

- [ ] **AC-3**: GIVEN a thread is prioritized (revelation_ready — 3+ connected facts), WHEN the map is opened, THEN an amber star marks its referenced location (distinct from the amber dot for regular threads).
- [ ] **AC-4**: GIVEN a location has been discovered and named by the player, WHEN the map is opened, THEN a named pin marks that location with its discovered name.
- [ ] **AC-5**: GIVEN a location has been cleared or destroyed, WHEN the map is opened, THEN an ink X marks that location.
- [ ] **AC-6**: GIVEN the map is open and the player moves to a different region, WHEN the map refreshes, THEN only markers relevant to the current region are shown.
- [ ] **AC-7**: GIVEN an NPC has not been seen by the player, THEN that NPC does NOT appear on the map (no omniscient marker placement).

---

## Implementation Notes

*Derived from ADR-0012, ADR-0005, ADR-0010:*

- All markers use pen-and-ink iconography: single-stroke, 2px line weight at 24px icon size, flat fills only for palette colors. No gradients, bevels, or skeuomorphic pins.
- Amber is accent/fill only — never use amber for marker text. Marker labels use dark ink text (desk serif).
- NPC markers: read NPC entity transforms from the spatial index (ADR-0001 uniform grid). Display name from the entity's display name component. Only show NPCs the player has discovered (tracked via PWS or PlayerKnowledge).
- Thread markers: read active threads from `PlayerKnowledge` component (ADR-0010). For each thread with a location entity reference, place a marker at that entity's position.
  - Regular active thread → amber dot (single circle fill)
  - Prioritized thread (chain status = `revelation_ready`) → amber star (5-point, outlined with fill)
- Location pins: read discovered location entities from PWS flags. Render as named pin icon with location name in desk serif.
- Cleared/destroyed marks: read PWS flags for cleared/destroyed locations. Render as ink X over the location.
- Marker placement: convert entity world position to map screen coordinates. Account for map scroll/pan offset.
- Accessibility: amber dot and amber star must have distinct shapes (dot vs star) — not hue alone. The difference is visible to colorblind players.
- Cross-system data reads are read-only — map does not mutate any system state.
- Verify all component store read APIs against `kaiju/src/` before implementing.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- **Story 001**: The map display itself — parchment overlay, region linework, M key toggle, scrolling, player position indicator
- **Story 003**: World map with region overview, travel routes, faction overlays

---

## QA Test Cases

*For Integration story — automated test specs:*

- **AC-1 (NPC marker)**:
  - Given: An NPC entity exists at position (100, 50) in the current region, and the player has discovered this NPC (PWS flag set)
  - When: The map is opened
  - Then: An NPC marker icon is rendered at the screen position corresponding to world (100, 50), with the NPC's display name shown adjacent
  - Edge cases: NPC at region boundary (within 1 tile of edge), NPC with empty display name, NPC in different region (must not appear)

- **AC-2 (Thread amber dot)**:
  - Given: An active investigation thread references a location entity at (200, 300), and the thread is NOT in revelation_ready state
  - When: The map is opened
  - Then: An amber dot marker is rendered at the screen position corresponding to world (200, 300)
  - Edge cases: Thread references an entity that no longer exists, thread references multiple locations (show dot at each), thread with no location reference (no marker)

- **AC-3 (Prioritized thread amber star)**:
  - Given: An active thread chain has status = revelation_ready and references location (150, 250)
  - When: The map is opened
  - Then: An amber star marker (visually distinct from amber dot) is rendered at that location
  - Edge cases: Thread transitions from regular to prioritized while map is open (marker updates on next refresh), multiple prioritized threads at same location (both stars visible or combined indicator)

- **AC-4 (Discovered location pin)**:
  - Given: A location at (400, 100) has been discovered and the player has named it "Old Watchtower"
  - When: The map is opened
  - Then: A named pin icon is rendered at that location with the label "Old Watchtower" in desk serif
  - Edge cases: Location with no custom name (use generated/default name), very long name (truncate with ellipsis at 24 characters)

- **AC-5 (Cleared location X)**:
  - Given: A location at (350, 200) has been cleared (PWS flag: `location.cleared`)
  - When: The map is opened
  - Then: An ink X mark is rendered over that location
  - Edge cases: Location both discovered AND cleared (show both pin and X), location cleared then revisited (X persists)

- **AC-6 (Region-scoped markers)**:
  - Given: Two regions, Region A and Region B, each with NPCs and threads
  - When: The map is opened while the player is in Region A
  - Then: Only markers for Region A are shown; Region B markers are not rendered
  - Edge cases: Player transitions between regions while map is open (refresh markers on region change event)

- **AC-7 (Undiscovered NPC exclusion)**:
  - Given: An NPC exists in the region but has never been seen by the player
  - When: The map is opened
  - Then: No marker appears for that NPC
  - Edge cases: NPC seen briefly then despawns (marker shows at last known position), NPC seen but player doesn't know their name (marker shows "???" or generic label)

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/map_system/map_markers_test.go` — must exist and pass
**Status**: [x] Created — `tests/integration/map_system/map_markers_test.go`

---

## Dependencies

- Depends on: Story 001 (Local Map Display & Navigation) — markers render on top of the map display
- Depends on: NPC Knowledge Graph epic (entity references for NPC markers)
- Depends on: Thread Investigation epic (PlayerKnowledge for thread markers)
- Depends on: Persistent World State epic (PWS flags for discovered/cleared locations)
- Unlocks: Story 003 (World Map & Region Travel)

---

## Completion Notes

**Completed**: 2026-08-11
**Criteria**: 7/7 passing — all automated via 19 integration tests
**Deviations**: None
**Test Evidence**: `tests/integration/map_system/map_markers_test.go` — 19 tests, all PASS (BLOCKING gate met). Story 001 unit suite 25/25 PASS (no regressions).
**Code Review**: APPROVED WITH SUGGESTIONS — all required fixes applied (H1 defensive copy in Markers(), M1 star-arm count check, M2 comment fix, M3 boundary tests).
**Advisory**: H2 same-location star stacking (design ruling deferred), H3 renderer coverage (manual walkthrough pending), provider-contract QA edges (untestable until game-layer providers exist).
**Tech Debt**: None logged.
