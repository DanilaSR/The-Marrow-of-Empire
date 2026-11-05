# Story 005: Targeting Reticle & Positioning Indicators

> **Epic**: Combat UI
> **Status**: Ready
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: 4h
> **Manifest Version**: 2026-08-09
> **Last Updated**: [set by /dev-story when implementation begins]

## Context

**GDD**: `design/gdd/combat-ui.md`
**Requirement**: `TR-ui-002`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: World register elements — targeting reticle, range line, flanking indicator, and cover indicator are anchored in world space. Thread-red for the range line (investigation connector color, reused for tactical readability). All colors have shape/icon/value/sound backup. Field register (umber panels) must remain readable against targeting overlays.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju world-space line/arc rendering — verify whether the HTML/CSS-like UI system supports arbitrary line drawing or if targeting indicators need custom rendering. Line from attacker to cursor requires per-frame update. Post-cutoff APIs, no LLM training data exists.

**Control Manifest Rules (this layer)**:
- Required: World register for anchored indicators; thread-red for range line (reuse investigation connector color); all colors have shape/icon/value/sound backup
- Forbidden: Never encode game-critical information by hue alone — flanking (blue arcs) also changes hit chance number, cover (shield icon) has a shape, valid/invalid targets differ in brightness not just hue
- Guardrail: UI render (static) <1ms/frame; targeting updates must not drop frames during mouse/cursor movement

---

## Acceptance Criteria

*From GDD `design/gdd/combat-ui.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN a flanking attack, WHEN targeting, THEN the flanking indicator (blue arcs) appears behind the target and the hit chance display includes the +2 flanking bonus in its calculation.
- [ ] **AC-2**: GIVEN an attack or ability is selected, WHEN the player moves the cursor over the battlefield, THEN a thread-red range line is drawn from the attacker to the cursor position, valid targets are highlighted (brightened outline or glow), and invalid targets (out of range or invalid type) are dimmed.
- [ ] **AC-3**: GIVEN the cursor hovers over a valid target, WHEN targeting is active, THEN the hit chance percentage is displayed (e.g., "75%") next to the cursor or above the target, calculated from the combat engine's attack resolution formula.
- [ ] **AC-4**: GIVEN a combatant is in cover, WHEN targeting or during combat, THEN a shield icon (pen-and-ink, umber) appears on or near the combatant. Half cover: shield with "½" marker. Full cover: solid shield icon, and the combatant cannot be targeted by ranged attacks (blocked LoS).
- [ ] **AC-5**: GIVEN targeting mode is active, WHEN the player right-clicks or presses Escape, THEN targeting is cancelled, the ability is deselected, range line and highlights are removed, and the UI returns to the Ability Menu state.

---

## Implementation Notes

*Derived from ADR-0012:*

- **Targeting mode entry**: Triggered when the player selects an attack or ability (from Story 002's ability menu). On `AbilitySelected` event: enter targeting mode. The cursor changes to a crosshair or target reticle (OS cursor change or Kaiju custom cursor). The ability menu remains visible but dims slightly — the player is now in a modal targeting state.
- **Range line**: A line rendered from the attacker's world position to the current cursor/mouse world position. Color: thread-red (#8B2020 or art-bible-defined thread-red — same as investigation contradiction connectors per ADR-0012). The line updates every frame as the cursor moves. Render via:
  - If Kaiju UI supports SVG-like elements: use a `<line>` or canvas-drawn line.
  - If not: use a thin div rotated to the correct angle via CSS `transform: rotate()` — compute angle and length from attacker-to-cursor vector.
  - Performance: line update is a single transform change per frame — must stay under 0.1ms.
- **Valid/invalid target highlighting**: Query the combat engine for valid targets given the selected ability (range, target type, LoS). Valid targets: add a brightened outline or subtle green-gold glow (amber-adjacent, not pure green — must fit the amber/oxblood palette). Invalid targets: dim to 40% opacity or desaturate. Non-targetable entities (dead, out of range, wrong type): dim further or no highlight.
  - **Accessibility note**: Valid/invalid distinction uses brightness (not hue alone) — valid targets are visibly brighter. The cursor also changes on hover over valid vs invalid targets.
- **Hit chance display**: On hover over a valid target, query the combat engine's attack resolution: `d20 + accuracy >= evasion` → hit chance = `max(5, min(95, (accuracy - evasion + 11) * 5))`% (d20 probability clamped 5-95%). Display as "75%" in combat face tabular figures next to the cursor or floating above the target. Update in real-time as different targets are hovered. Include flanking (+2 accuracy per TR-combat-005) and cover modifiers in the calculation.
- **Flanking indicator**: When the attacker is positioned such that the target is flanked (attacker and an ally on opposite sides of the target per TR-combat-005), display cold indigo/blue arcs behind the target. The arcs are two curved lines (like parentheses) on either side of the target — pen-and-ink style, 2px stroke. Color: cold indigo (#3A4A8A or art-bible-defined). The indicator appears automatically when the flanking condition is met — no player action needed. The hit chance display already incorporates the +2 flanking bonus.
- **Cover indicator**: For combatants in cover (half cover: +2 evasion, full cover: blocks LoS per TR-combat-005):
  - Half cover: Shield icon (pen-and-ink, umber) with a small "½" marker. Positioned near the health bar or on the combatant.
  - Full cover: Solid/filled shield icon. Combatant cannot be targeted by ranged attacks — the targeting system refuses to highlight them as valid. Melee attacks may still target if in range.
  - Cover status comes from the combat engine's positioning system — query per combatant.
- **Targeting cancel**: Right-click or Escape during targeting → deselect ability, remove range line, remove all highlights, clear hit chance display, exit targeting mode, re-enable ability menu at full brightness. Emit `TargetingCancelled` event.
- **Targeting confirm**: Left-click on a valid target → emit `TargetSelected` event with target entity ID and selected ability → combat engine executes the attack. Remove range line and highlights. Transition targeting mode off.
- **Grid awareness**: The range line follows the square grid (1 unit = 1 meter per TR-combat-001). The line can be drawn as a direct line (not path-found) because the targeting check validates range, not path. However, if the combat engine requires path-based LoS, the line should show the actual path — start with direct line for MVP.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Turn order bar, combat start/end animations, Field register
- Story 002: Ability menu (selection triggers targeting, but the menu itself is Story 002)
- Story 003: World-anchored health bars (cover icon appears near health bar but is a separate element)
- Story 004: Damage numbers, crit effects, combat audio
- Story 006: Victory/defeat screens
- Flanking/cover calculation: Handled by Tactical Combat Engine — this story only queries the combat engine and displays the indicators

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1** (flanking indicator with hit chance bonus):
  - Setup: Position attacker and ally on opposite sides of an enemy (flanking condition). Select Attack.
  - Verify: Blue/cold-indigo arcs appear behind the target (two curved lines, pen-and-ink style). Hover over the flanked target — hit chance shows a value that includes the +2 flanking accuracy bonus (compare: hit chance without flanking should be 10 percentage points lower).
  - Pass condition: Blue arcs visible, hit chance correctly incorporates +2 flanking

- **AC-2** (range line + valid/invalid target highlighting):
  - Setup: Select an attack with 4-tile range. Move cursor across the battlefield.
  - Verify: Thread-red line drawn from attacker to cursor, updates smoothly with cursor movement. Enemies within 4 tiles: brightened outline. Enemies beyond 4 tiles: dimmed. Allies, dead enemies, objects: not highlighted. Cursor moves outside valid targets → no target highlighted.
  - Pass condition: Line visible and tracking, valid targets clearly distinguishable from invalid/non-targets

- **AC-3** (hit chance on hover):
  - Setup: Targeting mode active. Hover over enemy with evasion 12 while attacker has accuracy 14.
  - Verify: Hit chance displays as "65%" (d20: accuracy 14 - evasion 12 = 2; hit on 9+, so 12/20 = 60%... wait — let the combat engine compute this). Display updates when hovering different targets. No display when not hovering a valid target.
  - Pass condition: Hit chance percentage correct per combat engine formula, updates per target, combat face tabular figures

- **AC-4** (cover indicator — shield icon):
  - Setup: Combat with enemy in half cover (+2 evasion). Select a ranged attack.
  - Verify: Shield icon with "½" marker visible on the enemy. Hit chance reflects +2 evasion from cover (lower than without cover). Full cover enemy: solid shield icon, not targetable by ranged (not highlighted when hovered). Melee attack may still target full cover enemy if adjacent.
  - Pass condition: Correct shield icon per cover level, cover modifier reflected in hit chance, full cover blocks ranged targeting

- **AC-5** (targeting cancel):
  - Setup: Targeting mode active with range line and highlights visible
  - Verify: Press Escape → range line removed, highlights cleared, ability deselected, ability menu returns to full brightness. Right-click → same result. No ability executed.
  - Pass condition: All targeting visuals removed, ability deselected, menu restored

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `production/qa/evidence/combat-ui-targeting-evidence.md` or interaction test

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Combat HUD Framework — combat lifecycle, Field register), Story 002 (AP Meter & Ability Menu — ability selection triggers targeting mode), Tactical Combat Engine (must provide hit chance calculation, flanking/cover status, valid target queries)
- Unlocks: None directly — targeting is a leaf feature within Combat UI
