# Story 006: Combat Resolution Screens

> **Epic**: Combat UI
> **Status**: Ready
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: 2h
> **Manifest Version**: 2026-08-09
> **Last Updated**: [set by /dev-story when implementation begins]

## Context

**GDD**: `design/gdd/combat-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-004`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: Desk register (parchment panels, world dims to 30%) for full-screen overlays — victory and defeat screens dim the world. Paper-slide motion for transitions. Desk serif (Garamond class) for narrative text, combat face (grotesque sans, tabular figures) for numbers (XP values). The desk never trembles — immune to any residual screen shake.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI system — verify overlay rendering, world dimming, font loading for mixed serif + grotesque on same screen. Post-cutoff APIs, no LLM training data exists.

**Control Manifest Rules (this layer)**:
- Required: Desk register — parchment panels, world dims to 30%; Desk serif for narrative text; paper-slide motion 200-300ms ease-out; combat face for XP/stat numbers
- Forbidden: Never use amber-on-parchment text (amber accent only); never use bounce/scale/spin/flip; desk never trembles
- Guardrail: UI render (static) <1ms/frame; no frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/combat-ui.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN all enemies are defeated (Victory), WHEN combat ends, THEN all Field register combat UI slides out (200ms), the world dims to 30%, and a Desk register victory overlay appears showing: "Victory" title (Desk serif), party member portraits with +XP earned per character (combat face tabular figures), total XP summary, and a "Continue" button — accompanied by a fanfare sound.
- [ ] **AC-2**: GIVEN all player characters reach 0 HP (Defeat), WHEN the party is wiped, THEN a Desk register defeat overlay appears showing: "Game Over" title, a "Load Save" button that opens the save/load menu, and a "Return to Main Menu" button.
- [ ] **AC-3**: GIVEN the victory or defeat overlay is displayed, WHEN the player clicks Continue (victory) or Load Save / Return to Main Menu (defeat), THEN the overlay slides out (200ms ease-out), the world returns to full brightness, and the game transitions to exploration mode (victory) or the selected screen (defeat).
- [ ] **AC-4**: GIVEN the victory overlay, WHEN XP is awarded, THEN each party member's +XP value counts up from 0 to the earned amount over 500ms (combat face tabular figures, no layout shift during count-up).

---

## Implementation Notes

*Derived from ADR-0012:*

- **Desk register overlay**: The victory/defeat screens use the Desk register — full-screen parchment overlay, world dims to 30% brightness behind it. This is a different register from the Field (combat HUD) and Desk edge (dialogue panel). The Desk register dims the world because the player is no longer in active combat — they're reviewing results.
- **Victory screen layout**:
  - Title: "Victory" in Desk serif (Garamond class), centered at top, large (32px at reference).
  - Party portraits: Horizontal row, each showing portrait thumbnail + character name + "+XP" value. XP counts up from 0 (see AC-4).
  - Total XP: Sum of all earned XP, displayed below the portraits.
  - "Continue" button: Pen-and-ink icon, parchment-toned, centered at bottom. Click → dismiss overlay, transition to exploration.
- **Defeat screen layout**:
  - Title: "Game Over" in Desk serif, centered, large. Somber tone — no amber, oxblood accent may be used sparingly.
  - Two buttons: "Load Save" (primary, opens save/load menu in Desk register) and "Return to Main Menu" (secondary). Pen-and-ink icons. Stacked vertically, centered.
  - No XP display on defeat.
- **Transition sequence** (victory):
  1. Combat engine emits `CombatEnded` with result: Victory and XP awards per entity.
  2. Field register combat UI slides out (200ms) — handled by Story 001.
  3. After slide-out completes (or staggered 100ms after): world dims to 30% (CSS transition on world brightness, 200ms ease-out).
  4. Victory overlay slides in (300ms ease-out — parchment panel enters from center, or fades in + slight scale for paper feel... but ADR-0012 forbids scale. Use fade-in + slide-up from 20px below center: `translateY(20px)` → `translateY(0)` + `opacity: 0` → `opacity: 1` over 300ms).
  5. Fanfare audio plays (emit `VictoryFanfare` event → Audio System).
- **Transition sequence** (defeat):
  1. Combat engine emits `CombatEnded` with result: Defeat.
  2. Field register slides out (200ms).
  3. World dims to 30% (200ms).
  4. Defeat overlay slides in (300ms, same motion as victory).
  5. Somber audio plays (emit `DefeatSting` event → Audio System).
- **XP count-up animation**: Each party member's +XP value animates from 0 to the earned amount over 500ms. Use a Go ticker or requestAnimationFrame loop: increment the displayed number smoothly (ease-out interpolation). Combat face tabular figures ensure the number width doesn't cause layout shift as digits change. For multiple party members, animate all simultaneously (parallel, not sequential — the player shouldn't wait for each member).
- **Continue button**: Emits `VictoryAcknowledged` event → game systems transition to exploration: camera returns to exploration mode, clock resumes (was paused during combat per ADR-0007), party positioned at combat end location. Overlay slides out (200ms, reverse of entrance animation), world brightness returns to 100%.
- **Load Save button** (defeat): Opens the Save/Load menu (Desk register). This is an existing screen from the save-load system — open it via EventBus (`OpenSaveLoadMenu` event). The defeat overlay stays beneath or is dismissed.
- **Return to Main Menu** (defeat): Emits `ReturnToMainMenu` event → game transitions to main menu scene. Confirmation dialog? GDD doesn't specify — add a simple confirmation: "Return to main menu? Unsaved progress will be lost." with Yes/No. This prevents accidental clicks during the emotional low of a party wipe.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Combat HUD slide-out (the Field register exit is triggered by Story 001 on `CombatEnded`; this story triggers the Desk overlay after)
- Story 002: AP meter, ability menu (hidden as part of Field register slide-out)
- Story 003: World health bars (hidden on combat end)
- Story 004: Combat audio (fanfare/defeat sting are new audio events specific to resolution, not part of in-combat feedback)
- Save/Load menu: Implemented by Save/Load system — this story only triggers it via event
- XP calculation: Handled by Tactical Combat Engine — this story only displays the awarded values

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1** (victory overlay with XP and fanfare):
  - Setup: Defeat all enemies in a combat encounter (trigger Victory)
  - Verify: Combat HUD slides out (200ms). World dims to 30%. Victory overlay appears: "Victory" title, each party member portrait with +XP (counting up), total XP, "Continue" button. Fanfare audio plays.
  - Pass condition: Overlay visible, world dimmed, XP values correct per combat result, fanfare played

- **AC-2** (defeat overlay with Load Save and Main Menu):
  - Setup: All party members reduced to 0 HP (trigger Defeat)
  - Verify: Combat HUD slides out. World dims. Defeat overlay: "Game Over" title, "Load Save" button, "Return to Main Menu" button. Somber audio.
  - Pass condition: Overlay visible, both buttons functional, world dimmed

- **AC-3** (overlay dismiss and transition):
  - Setup: Victory overlay displayed. Click "Continue".
  - Verify: Overlay slides out (200ms ease-out). World brightness returns to 100%. Game transitions to exploration mode (camera, clock, controls restored).
  - Setup: Defeat overlay displayed. Click "Load Save" → save/load menu opens. Click "Return to Main Menu" → confirmation dialog, then main menu.
  - Pass condition: Clean transition in all cases, no stuck dimming, correct target screen

- **AC-4** (XP count-up animation):
  - Setup: Victory with party members earning 150 XP, 120 XP, and 0 XP (KO'd member)
  - Verify: Each +XP value starts at 0 and counts up to earned amount over 500ms (ease-out). Combat face tabular figures — no layout shift. KO'd member shows "+0" or is greyed/omitted.
  - Pass condition: Smooth count-up to correct values, no layout jitter, KO'd members handled

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `production/qa/evidence/combat-ui-resolution-evidence.md` or interaction test

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Combat HUD Framework — slide-out transition + `CombatEnded` event handling), Tactical Combat Engine (must emit `CombatEnded` with Victory/Defeat result and XP awards)
- Unlocks: None (last story in epic)
