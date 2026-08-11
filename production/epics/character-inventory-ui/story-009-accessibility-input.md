# Story 009: Accessibility & Input

> **Epic**: Character & Inventory UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: T-shirt M
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/character-inventory-ui.md`
**Requirement**: `TR-ui-005`, `TR-ui-006`, `TR-ui-007`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: Accessibility is a first-class requirement: tab order + focus ring (≥3:1 contrast), arrow-key navigation, reduce-motion setting, text scaling, minimum resolution enforcement. Kaiju HTML/CSS-like UI must support these patterns.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju UI accessibility hooks (AccessKit screen reader support), focus management, input routing — verify against `kaiju/src/`. Post-cutoff APIs: Kaiju accessibility integration.

**Control Manifest Rules (this layer)**:
- Required: Tab order + focus ring (≥3:1), arrow-key navigation, typewriter speed setting, Esc mapping, OFL-licensed fonts shipped with game, Minimum 1280×720, reference 1920×1080 with height-based scaling, Body text 16px at reference, minimum 12px effective
- Forbidden: Never encode game-critical information by hue alone — all colors have shape/icon/value/sound backup
- Guardrail: No frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/character-inventory-ui.md`, scoped to this story:*

- [ ] **AC-23 — Keyboard navigation**: GIVEN the C&I UI is open and focused, WHEN the player presses Tab repeatedly, THEN focus moves through sidebar → attributes → equipment slots → tabs and wraps back to the first element.
- [ ] **AC-24 — Reduce-motion**: GIVEN the `reduce_motion` setting is enabled, WHEN the C&I UI is opened, THEN the parchment appears instantly with no slide animation and tab switches have no crossfade.
- [ ] **AC-25 — Resolution floor**: GIVEN the game window is resized below 1280×720, WHEN the C&I UI is open, THEN the UI displays "Minimum resolution: 1280×720" and prevents interaction.

---

## Implementation Notes

*Derived from ADR-0012:*

- **Keyboard navigation** (AC-23):
  - Tab order: sidebar portraits → attribute bars → equipment slots → tabs (Character/Inventory) → close button → wrap to sidebar.
  - Shift+Tab: reverse order.
  - Focus ring: oxblood/umber outline, ≥3:1 contrast ratio against parchment background. 3px width.
  - Arrow keys: within a focus group (e.g., attributes), ↑/↓ to move between individual attribute bars. Within equipment slots, ←/→ to move between slots. Within sidebar, ↑/↓ to cycle party members.
  - Enter/Space: activate focused element (select party member, open attribute breakdown, equip/unequip slot, switch tab).
  - Esc: close the UI (same as clicking close button or pressing the toggle key).
- **Reduce-motion** (AC-24):
  - Check the `reduce_motion` accessibility setting (from Settings system).
  - When enabled: all UI animations are disabled.
    - Parchment appears instantly on open (0ms, no slide).
    - Tab switches are instant (0ms, no crossfade).
    - Weight bar does not pulse (static oxblood for >90%).
    - Toast notifications appear/disappear instantly (no fade).
  - When disabled (default): all animations play normally.
- **Resolution floor** (AC-25):
  - Check window dimensions each frame (or on resize event).
  - Minimum supported: 1280×720.
  - Below minimum while C&I UI is open:
    - Display centered message: "Minimum resolution: 1280×720" in the parchment area.
    - Block all interactive elements — clicks, keyboard navigation, drag-and-drop do nothing.
    - UI does not close — it remains open but locked.
  - When window is resized back above threshold: message disappears, interaction restored immediately.
- **Screen reader support**:
  - Parchment screen announced as "Character Sheet — [character name]" or "Inventory — [character name]".
  - Attribute bars announced with name, value, and modifier.
  - All interactive elements have accessible labels.
- **Text scaling**: All text respects `ui_text_scale` setting (0.75× to 1.5×). Body text floor: 12px effective (16px × 0.75).
- All stat values conveyed by both position (bar fill) AND numeric text — position is never the sole information channel.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- [Story 001]: The attribute bars themselves (this story adds keyboard navigation TO them)
- [Story 007]: Screen state machine (this story adds reduce-motion TO the transitions)
- Typewriter speed setting — belongs to Dialogue UI system, not C&I UI
- Colorblind modes — handled by the general accessibility system, not C&I UI specifically
- Gamepad navigation implementation — covered in the individual stories (sidebar, grid, equipment)

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-23**: GIVEN the C&I UI is open and focused, WHEN the player presses Tab repeatedly, THEN focus moves through the correct order and wraps
  - Setup: Open the C&I UI. Ensure keyboard is the primary input device.
  - Verify: Press Tab. Focus moves to the first sidebar portrait. Press Tab again → focus moves to first attribute bar. Continue pressing Tab and verify the order: sidebar → attributes → equipment slots → tabs → close button → back to sidebar.
  - Pass condition: Focus ring is visible (≥3:1 contrast) at each stop. Shift+Tab moves in reverse order. Arrow keys work within focus groups (↑/↓ in sidebar and attributes, ←/→ in equipment slots). Enter/Space activates the focused element. Wrap-around works at both ends of the tab order.

- **AC-24**: GIVEN reduce_motion is enabled, WHEN the C&I UI is opened, THEN animations are disabled
  - Setup: Enable the `reduce_motion` setting in the game settings.
  - Verify: Press C to open character sheet. Parchment appears instantly — no slide-in animation. Press I or click Inventory tab — content switches instantly, no crossfade. Weight bar at >90% — static oxblood, no pulse. Press Esc to close — UI disappears instantly, no slide-out.
  - Pass condition: All animations suppressed. UI remains fully functional. Disable reduce_motion and verify animations return.

- **AC-25**: GIVEN the game window is resized below 1280×720, WHEN the C&I UI is open, THEN the UI locks and shows the minimum resolution message
  - Setup: Open the C&I UI. The game window is at a resolution ≥1280×720.
  - Verify: Resize the window to below 1280×720 (e.g., 1024×700). Observe the C&I UI. The message "Minimum resolution: 1280×720" appears centered. Click on various UI elements — nothing responds. Resize back above 1280×720 → message disappears, UI is interactive again.
  - Edge cases: Open the UI when window is already below minimum → UI opens directly to the locked/message state. Resize to exactly 1280×720 → UI is functional (this IS the minimum). Resize below minimum, close UI (via Esc if possible), resize up, reopen → UI works normally. Window at 1280×719 → locked.
