# Story 002: AP Meter, Ability Menu & End Turn

> **Epic**: Combat UI
> **Status**: Ready
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: 3h
> **Manifest Version**: 2026-08-09
> **Last Updated**: [set by /dev-story when implementation begins]

## Context

**GDD**: `design/gdd/combat-ui.md`
**Requirement**: `TR-ui-002`, `TR-ui-004`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: Field register (umber combat chrome, no world dim) for combat HUD. Combat face font (grotesque sans, tabular figures) for all combat numbers — AP, HP, initiative. Paper-slide motion only. Pen-and-ink icons for ability buttons — no skeuomorphic buttons. Amber is accent/fill only — never amber-on-parchment text.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI system — verify CSS animation for bar fill/depletion, keyboard event routing for hotkeys 1-6, button click handling. Post-cutoff APIs, no LLM training data exists.

**Control Manifest Rules (this layer)**:
- Required: Combat face (grotesque sans, tabular figures) for AP numbers; paper-slide motion; pen-and-ink icons only; amber accent/fill only
- Forbidden: Never use skeuomorphic buttons; never allow amber-on-parchment text; never encode game-critical information by hue alone
- Guardrail: UI render (static) <1ms/frame; no frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/combat-ui.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN a character with 7 AP, WHEN they spend 2 AP on an attack, THEN the AP bar depletes from 7 to 5 with the amber fill shrinking in real-time and the numeric display counting down from 7 to 5.
- [ ] **AC-2**: GIVEN the combat HUD, WHEN it is a player-controlled character's turn, THEN the ability menu shows 6 actions (Move [1], Attack [2], Abilities [3], Items [4], Defend [5], Wait [6]) as pen-and-ink icon buttons with hotkey numbers, and the End Turn button is visible (amber, prominent) on the right.
- [ ] **AC-3**: GIVEN a character with 2 AP remaining, WHEN the ability menu is displayed, THEN abilities costing more than 2 AP (Attack at 2 AP, Abilities at 1-3 AP) are greyed out — except those whose cost <= 2 AP remain active.
- [ ] **AC-4**: GIVEN the player clicks an ability button or presses its hotkey (1-6), WHEN the ability is selected, THEN a soft select sound plays and the targeting state begins (handled by Story 005). GIVEN the player clicks End Turn, WHEN clicked, THEN the current turn ends and transitions to the next combatant.
- [ ] **AC-5**: GIVEN the current turn begins, WHEN AP resets to the character's maximum, THEN the AP bar refills with an amber pulse animation (200ms, brief brightness spike at completion) and the numeric display updates to the max value.

---

## Implementation Notes

*Derived from ADR-0012:*

- **AP meter**: Bottom-left of the combat HUD. Horizontal bar with amber fill (#D4A040 or art-bible-defined amber) on umber background. Fill width = `(currentAP / maxAP) * 100%` — animate via CSS `transition: width 150ms ease-out` for smooth depletion. Numeric display: `"currentAP / maxAP"` in combat face grotesque sans with tabular figures (prevents layout shift as digits change). Position: overlaid on the bar or immediately adjacent.
- **AP depletion animation**: On AP spend event (`APChanged` from EventBus): update the bar fill width and numeric display. The transition should be visible but fast (150ms) — the player should see the AP drain, not teleport. Real-time countdown: if AP depletes sequentially (e.g., Move costs 1, then Attack costs 2), each decrement animates individually. If multiple AP are spent in one action, animate the total change in one transition.
- **AP refill on turn change**: On `TurnChanged` event for a new player turn: animate fill from current (depleted) to maxAP over 200ms. At completion, apply a brief amber pulse glow (box-shadow brighten-dim cycle, 150ms). Numeric display counts up to max. This is coordinated with Story 001's turn change animation.
- **Ability menu grid**: Bottom-center. 2×3 or 1×6 grid of ability buttons. Each button: pen-and-ink icon (not skeuomorphic — outlined glyph, flat fill for palette colors only, no gradients/bevels per TR-ui-003), hotkey number badge (1-6), AP cost in combat face tabular figures. Default actions:
  | Hotkey | Action | AP Cost |
  |--------|--------|---------|
  | 1 | Move | 1 |
  | 2 | Attack | 2 |
  | 3 | Abilities | 1-3 (varies) |
  | 4 | Items | 1 |
  | 5 | Defend | 1 |
  | 6 | Wait | 0 |
- **Greyed out state**: When `currentAP < abilityCost` → button gets CSS class `disabled`: desaturated icon, muted text, non-interactive (pointer-events: none or click handler returns early). Hotkeys for disabled abilities are ignored. Attack (2 AP) at 1 AP remaining → greyed out. Abilities submenu: each ability shows its specific cost; grey individual abilities the player can't afford.
- **Ability selection**: Click or hotkey → play soft select sound (via Audio System event), set selected ability, emit `AbilitySelected` event → Story 005's targeting system activates. If the same ability is clicked again, deselect it. Only one ability selected at a time.
- **End Turn button**: Bottom-right. Amber fill on umber, prominent — larger than ability buttons. Pen-and-ink hourglass or checkmark icon. Click → emit `EndTurn` event → combat engine advances initiative. Button should require a confirmation step? Per GDD: no confirmation mentioned — direct click ends turn. Consider a brief disable window (300ms) after click to prevent double-trigger.
- **Keyboard-only operation**: Hotkeys 1-6 must work regardless of focus state (global keydown listener during combat). End Turn: bind to a dedicated key (E or Enter) in addition to click. Tab order: AP meter (read-only) → ability buttons (1-6) → End Turn → back to portrait bar (Story 001).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Turn order bar, combat start/end slide animations, turn change highlight, hover tooltips, Field register layout container
- Story 003: World-anchored health bars
- Story 004: Floating damage numbers, crit flash, screen shake, combat audio (except select sound which is part of the ability click flow)
- Story 005: Targeting reticle, range line, valid/invalid target highlighting, hit chance, flanking/cover indicators
- Story 006: Victory/defeat screens

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1** (AP bar depletes in real-time):
  - Setup: In combat, select a character with 7 AP. Use an attack (cost: 2 AP).
  - Verify: AP bar fill shrinks smoothly from 7/8 to 5/8 width over ~150ms. Numeric display counts down: "7 / 8" → "5 / 8". Amber fill recedes from right to left.
  - Pass condition: Bar fill and number update smoothly, no teleport, final values correct

- **AC-2** (ability menu displays all 6 actions):
  - Setup: Enter combat, wait for a player character's turn
  - Verify: Six ability buttons visible in bottom-center area: Move(1), Attack(2), Abilities(3), Items(4), Defend(5), Wait(6). Each has pen-and-ink icon, hotkey number, AP cost. End Turn button visible bottom-right, amber, prominent.
  - Pass condition: All 6 abilities + End Turn visible, correct hotkey labels, pen-and-ink styling

- **AC-3** (abilities greyed out when insufficient AP):
  - Setup: Character with 2 AP remaining (after spending)
  - Verify: Attack (2 AP) — active (cost = AP). Abilities costing 3 — greyed out (desaturated, non-interactive). Move (1 AP), Items (1), Defend (1), Wait (0) — all active. Spend 1 more AP (1 remaining): Attack greys out, only Move/Items/Defend/Wait remain active.
  - Pass condition: Grey state matches `currentAP < cost` rule, disabled buttons don't respond to click/hotkey

- **AC-4** (ability selection and End Turn):
  - Setup: Active turn, click Attack button or press hotkey 2
  - Verify: Soft select sound plays. Attack button shows selected state (amber highlight outline). Targeting mode activates. Click End Turn → turn ends, next combatant highlight appears.
  - Pass condition: Sound plays on select, button shows selected state, End Turn advances initiative

- **AC-5** (AP refill on new turn):
  - Setup: End a turn with 3 AP remaining out of 8 max. Observe next time this character's turn begins.
  - Verify: AP bar animates from 3/8 to 8/8 over 200ms. At completion, brief amber pulse (brightness spike ~150ms). Numeric display counts up to "8 / 8".
  - Pass condition: Smooth fill to max, amber pulse at end, numeric display correct

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `production/qa/evidence/combat-ui-ap-ability-evidence.md` or interaction test

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Combat HUD Framework — layout container and turn order bar must exist; AP bar, ability menu, and End Turn are children of the bottom bar)
- Unlocks: Story 005 (Targeting — ability selection triggers targeting mode)
