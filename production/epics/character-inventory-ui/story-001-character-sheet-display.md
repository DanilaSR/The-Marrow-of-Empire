# Story 001: Character Sheet Stat Display

> **Epic**: Character & Inventory UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: T-shirt M
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/character-inventory-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-005`, `TR-ui-006`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: All UI rendered in screen space via Kaiju HTML/CSS-like UI, desk register (parchment overlay, world dims to 30%), desk serif typography, minimum 1280×720 with height-based scaling.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI rendering API, font loading — verify against `kaiju/src/` before implementing. Post-cutoff APIs: Kaiju UI system.

**Control Manifest Rules (this layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space, Desk register (parchment overlay, world dims 30%), Desk serif (Garamond class) for journal/dialogue/menus, Minimum 1280×720, reference 1920×1080 with height-based scaling, Body text 16px at reference
- Forbidden: Never use 3D UI or diegetic HUD — the desk is flat by law, Never use amber-on-parchment text — amber is accent/fill only, Never encode game-critical information by hue alone
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/character-inventory-ui.md`, scoped to this story:*

- [x] **AC-1**: GIVEN a character with STR 16 (+3), WHEN the character sheet is opened, THEN STR displays as "16 (+3)" with the modifier correctly calculated.
- [x] **AC-2 — Derived stat display**: GIVEN a character with CON 14 (+2) at level 3, WHEN the character sheet is opened, THEN Max Health displays as "54 HP" (20 + 28 + 6).
- [x] **AC-3 — Attribute bar proportional fill**: GIVEN a character with STR 16 and CHA 8, WHEN the character sheet is displayed, THEN the STR bar is approximately 2× the width of the CHA bar (±5% tolerance).
- [x] **AC-5 — Modifier sign display**: GIVEN characters with attribute values 7, 10, and 16, WHEN their sheets are displayed, THEN modifiers appear as "−2", "+0", and "+3" respectively.

---

## Implementation Notes

*Derived from ADR-0012 Implementation Guidelines:*

- Render the character sheet as a Desk-register full-screen parchment overlay using Kaiju's HTML/CSS-like UI system.
- Six attribute bars: each shows name (left), numeric value with modifier (right), and a horizontal bar proportional to the value (range 3–18).
- Derived stats panel (right 30% width): Max Health, Action Points, Initiative, Evasion, Carry Capacity — each with computed value and formula breakdown on hover/tooltip.
- Center panel (50% width) for attributes; left sidebar (20% width) for party portraits (see Story 008).
- Typography: desk serif (Garamond class) for all text on this screen.
- Attribute modifier display format: `"{value} ({sign}{modifier})"` — e.g., STR 16 → "16 (+3)", STR 7 → "7 (−2)", STR 10 → "10 (+0)".
- Read attributes and derived stats from the `Stats` component via Entity Framework (ADR-0001). The UI reads only — does not write stats.
- Derived stat formulas (from ADR-0006): MaxHealth = 20 + CON×2 + Level×CON_mod; AP = 4 + DEX_mod (min 2); Initiative = DEX_mod + WIS_mod; Evasion = 10 + DEX_mod; Carry Capacity = (STR × 10) + 30.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- [Story 002]: XP progress bar and level-up modal
- [Story 003]: Equipment slots and drag-and-drop interaction
- [Story 004]: Equipment stat comparison panel
- [Story 008]: Party sidebar and companion inspect
- [Story 009]: Keyboard navigation, reduce-motion, resolution floor

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1**: GIVEN a character with STR 16 (+3), WHEN the character sheet is opened, THEN STR displays as "16 (+3)"
  - Setup: Create a character with STR attribute set to 16. Open the character sheet.
  - Verify: STR row shows "16" as the value and "(+3)" as the modifier, displayed together or adjacently.
  - Pass condition: The text "16 (+3)" is visible in the STR row. No other attribute rows show incorrect modifiers.

- **AC-2**: GIVEN a character with CON 14 (+2) at level 3, WHEN the character sheet is opened, THEN Max Health displays as "54 HP"
  - Setup: Create a character with CON 14 (modifier +2) at level 3. Expected: 20 + (14×2) + (3×2) = 20 + 28 + 6 = 54.
  - Verify: Derived stats panel shows Max Health as "54 HP" or "54".
  - Pass condition: The displayed Max Health value equals 54. Test with additional CON/level combinations to verify formula correctness.

- **AC-3**: GIVEN a character with STR 16 and CHA 8, WHEN the character sheet is displayed, THEN the STR bar is approximately 2× the width of the CHA bar
  - Setup: Create a character with STR 16 and CHA 8.
  - Verify: STR bar fill width is approximately double the CHA bar fill width (within ±5% tolerance). Bar width should be proportional to attribute value in the 3–18 range.
  - Pass condition: STR bar width / CHA bar width ≈ 16/8 = 2.0 (±5% → between 1.9 and 2.1).

- **AC-5**: GIVEN characters with attribute values 7, 10, and 16, WHEN their sheets are displayed, THEN modifiers appear as "−2", "+0", and "+3"
  - Setup: Create three characters or modify one character to have attributes at values 7, 10, and 16.
  - Verify: Modifier text for value 7 shows "−2" (minus sign, not hyphen). Value 10 shows "+0". Value 16 shows "+3".
  - Pass condition: All three modifier displays are correct. Sign is always shown (including + for positive and 0). Minus uses the proper minus sign character.

---

## Test Evidence

**Story Type**: UI
**Required evidence**: `production/qa/evidence/story-001-character-sheet-display-evidence.md` or interaction test
**Status**: [ ] Not yet created

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 4/4 passing (AC-1/2/3/5; AC-4 is Story 002)
**Deviations**: None
**Test Evidence**: `tests/unit/character_sheet/sheet_test.go` — 4 tests PASS. Manual screenshot at `production/qa/evidence/story-001-character-sheet-display-evidence.md` pending.
**Code Review**: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester, 2026-08-11)

---

## Dependencies

- Depends on: Character Stats & Progression system (Stats component must exist and provide attribute/derived stat values)
- Unlocks: Story 003 (Equipment slots — needs character sheet to display on), Story 008 (Party sidebar — shares the character sheet layout)
