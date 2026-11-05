# Story 003: World-Anchored Health Bars

> **Epic**: Combat UI
> **Status**: Ready
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: 3h
> **Manifest Version**: 2026-08-09
> **Last Updated**: [set by /dev-story when implementation begins]

## Context

**GDD**: `design/gdd/combat-ui.md`
**Requirement**: `TR-ui-002`, `TR-ui-001`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: World register — health bars, damage numbers, targeting indicators are anchored in world space (not screen space), subject to Pillar 3 readability requirements ("darkness never eats a decision"). Amber fill depletes to oxblood. Faction-fixed hues — never encode game-critical information by hue alone; all colors must have shape/icon/value/sound backup.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju world-space UI anchoring — verify how HTML/CSS-like elements attach to world positions (screen-space projection of 3D coordinates). Post-cutoff APIs, no LLM training data exists.

**Control Manifest Rules (this layer)**:
- Required: World register — anchored bars/numbers in world space, subject to Pillar 3 readability; amber fill depleting to oxblood; faction-fixed hues with non-color backup
- Forbidden: Never encode game-critical information by hue alone — health bar fill level (shape) and numeric display (value) must also convey the information
- Guardrail: UI render (static) <1ms/frame; no frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/combat-ui.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN combat with party members and enemies, WHEN combat is active, THEN each combatant has a health bar anchored above them in world space showing current/max HP with amber fill that depletes to oxblood as HP decreases, and faction-fixed hues distinguish party (e.g., warm amber), enemies (e.g., oxblood-dominant), and neutral (e.g., grey-umber).
- [ ] **AC-2**: GIVEN a combatant takes damage, WHEN their HP decreases, THEN the health bar fill shrinks smoothly (150ms ease-out) from current to new value, with the amber portion receding and oxblood portion expanding from left to right.
- [ ] **AC-3**: GIVEN a combatant reaches 0 HP, WHEN they die, THEN the health bar plays a shatter animation (breaks into fragments that fade out over 300ms) and the combatant's portrait in the turn order bar greys out.
- [ ] **AC-4**: GIVEN health bars are displayed, THEN each bar is readable at all three zoom levels (Close 4u, Default 10u, Far 20u) — scaling appropriately so the bar is never smaller than 4px effective height at reference resolution.

---

## Implementation Notes

*Derived from ADR-0012:*

- **World-space anchoring**: Health bars are Kaiju HTML/CSS-like elements positioned by projecting the combatant's world-space position (head/top of entity) to screen coordinates. This requires reading the entity's Transform from the entity store and projecting through the camera matrix. Update each frame or on camera move. Use the matrix math from `kaijuengine.com/matrix` — never `gonum/mathgl`.
- **Bar design**: Horizontal bar, fixed width in screen space (e.g., 48px at reference 1920×1080), scaling with camera zoom for readability. Two-color fill: amber (#D4A040) from left, transitioning to oxblood (#4A2020 or art-bible-defined oxblood) as HP decreases. The bar is effectively a single fill where the color is amber when `HP/maxHP > 0.5` and interpolates toward oxblood as HP drops. Alternatively: amber fill on left (healthy portion), oxblood "empty" portion on right — the boundary between them moves as HP changes.
- **Faction-fixed hues**: 
  - Party members: Amber-dominant (#D4A040 fill), umber background. 
  - Enemies: Oxblood-dominant (#6B2020 fill), dark umber background.
  - Neutral: Grey-umber (#8B7355 fill), muted background.
  - **Non-color backup**: Each bar also displays the numeric HP value (e.g., "42/56") in combat face tabular figures next to or overlaid on the bar. The fill level (shape/width) is the primary information channel — color is reinforcement, never the sole channel.
- **Deplete animation**: On `HealthChanged` event (entity ID, old HP, new HP, max HP): animate bar fill width via CSS `transition: width 150ms ease-out`. The numeric display counts down in real-time. If multiple hits land in the same frame, animate the total change from pre-first-hit to post-last-hit (don't queue animations).
- **Death animation**: On `EntityDied` event → add CSS class `health-bar--shatter`. Animation: bar splits into 3-5 fragment pieces that translate outward and fade to opacity 0 over 300ms (ease-in). Use CSS `@keyframes` with clip-path or transform on child elements. After animation completes, remove the health bar element. Simultaneously, emit event or directly set the portrait in the turn order bar to greyed/destroyed state (coordinated with Story 001 — use EventBus `PortraitStateChanged` event).
- **Zoom scaling**: At Close (4u): bar is larger on screen (more detail visible — show numeric HP by default). At Default (10u): bar at standard 48px width. At Far (20u): bar scales down but must stay ≥4px effective height. Use the camera zoom level (available via EventBus or camera system query) to adjust bar CSS scale. If the bar would be smaller than 4px at Far zoom, clamp the scale and consider showing only the fill (hide numeric text) at Far zoom.
- **Performance**: Health bars update every frame (world-space projection). Keep the number of DOM mutations minimal — update bar width and position only when values change or camera moves. Batch position updates in a single requestAnimationFrame or Kaiju equivalent. Target: <0.5ms for all health bar updates combined (typically 7-15 combatants).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Turn order bar, combat start/end animations, Field register layout, portrait rendering/hover
- Story 002: AP meter, ability menu, End Turn button
- Story 004: Floating damage numbers (these appear near the health bar but are separate elements), crit effects, combat audio
- Story 005: Targeting reticle, range line, flanking/cover indicators
- Story 006: Victory/defeat screens
- Health/damage calculation: Handled by Tactical Combat Engine — this story only displays the HP values it receives

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1** (health bars with faction hues and numeric backup):
  - Setup: Combat with 2 party members, 3 enemies, 1 neutral NPC
  - Verify: Each combatant has a health bar above them in world space. Party bars: amber fill on umber. Enemy bars: oxblood fill. Neutral bar: grey-umber fill. Each bar shows "current/max" HP in combat face tabular figures.
  - Pass condition: All bars visible, faction colors distinct, numeric HP display present on every bar

- **AC-2** (health bar deplete animation on damage):
  - Setup: In combat, deal 15 damage to an enemy with 50/50 HP
  - Verify: Bar fill shrinks from 100% width to 70% width over ~150ms (ease-out). Numeric display: "50/50" → counts down to "35/50". Oxblood portion expands from right as amber recedes.
  - Pass condition: Smooth fill transition, numeric countdown correct, no teleport

- **AC-3** (death shatter animation + portrait grey):
  - Setup: Deal lethal damage to an enemy (reduce to 0 HP)
  - Verify: Health bar fragments into pieces that fly outward and fade over 300ms. Turn order portrait for that enemy greys out (desaturated, lowered opacity).
  - Pass condition: Shatter animation visible, portrait greyed within 300ms of death

- **AC-4** (readability at all zoom levels):
  - Setup: Combat with health bars visible. Toggle through zoom levels: Close (4u), Default (10u), Far (20u).
  - Verify: Close: bars larger, numeric HP clearly readable. Default: bars at standard size, numeric HP readable. Far: bars smaller but still ≥4px effective height, numeric HP may hide but bar fill level still readable.
  - Pass condition: Bars readable at all three zoom levels, no sub-4px bars

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `production/qa/evidence/combat-ui-health-bars-evidence.md` or interaction test

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Combat HUD Framework — combat lifecycle events must be established; camera system must be operational for world-space projection)
- Unlocks: Story 004 (Damage Numbers — damage numbers appear near health bars and share the world-space anchoring pattern)
