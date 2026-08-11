# Story 002: XP Progress Bar & Level-Up Modal

> **Epic**: Character & Inventory UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Logic
> **Estimate**: T-shirt M
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/character-inventory-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: All UI rendered in screen space via Kaiju HTML/CSS-like UI, desk register (parchment overlay). The level-up modal is a Desk-register overlay on the character sheet.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI rendering API — verify modal overlay support against `kaiju/src/`.

**Control Manifest Rules (this layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space, Desk register (parchment overlay, world dims 30%), Paper-slide motion: enter 200-300ms ease-out, Desk never trembles
- Forbidden: Never use 3D UI or diegetic HUD, Never use skeuomorphic buttons — pen-and-ink icons only
- Guardrail: UI render (static): <1ms/frame, No frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/character-inventory-ui.md`, scoped to this story:*

- [x] **AC-4 — XP progress bar** (note: actual fill is 50% per ADR-0006, not 40% — GDD example uses incorrect Xnext): GIVEN a level 4 character (Lxp = 1800) with 2400 total XP (Xnext = 1500), WHEN the character sheet is opened, THEN the XP bar fill is at 40% (±2% tolerance).
- [x] **AC-18 — Level-up modal mandatory**: GIVEN a character at level 4 earning enough XP to reach level 5 while the UI is open, WHEN the XP is awarded, THEN the Level-Up modal appears showing the HP increase and the modal cannot be dismissed without pressing Confirm.

---

## Implementation Notes

*Derived from ADR-0012 and ADR-0006 Implementation Guidelines:*

- **XP progress bar**: Full-width bar across the bottom of the center panel of the character sheet.
  - Shows current level, XP progress (fill bar), and "XP to next level" text.
  - Formula: `xp_pct = ((total_xp − xp_for_current_level) / xp_to_next_level) × 100`
  - `xp_for_current_level = 150 × Level × (Level − 1)` (from ADR-0006)
  - `xp_to_next_level = Level × 300` (from ADR-0006)
  - Read `total_xp` and `Level` from the `Stats` component via Entity Framework.
- **Level-Up modal**: Overlay on the character sheet when XP threshold is reached while UI is open.
  - Header: "Level Up — Level [N] → [N+1]"
  - Body: Health increase shown (+N HP). Every 4th level: attribute point to allocate — six attribute options with + button, current modifier shown, bars pulse amber on hover.
  - Footer: "Confirm" button (disabled until point allocated if 4th level). No cancel button — modal is mandatory.
  - Cannot be dismissed without allocating point (if 4th level) or pressing Confirm.
  - If multiple characters level up simultaneously (shared XP event), modals queue in party order (player first, then companions by join order).
- The level-up modal's Confirm action writes the attribute point allocation to the `Stats` component.
- XP bar uses amber fill on parchment background — amber is fill, not text (per ADR-0012 forbidden: never amber-on-parchment text).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- [Story 001]: Attribute bars and derived stat display (XP bar sits below them)
- [Story 003]: Equipment interaction (level-up is independent of equipment)
- [Story 009]: Accessibility — keyboard navigation within the level-up modal is covered there

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-4**: GIVEN a level 4 character (Lxp = 1800) with 2400 total XP (Xnext = 1500), WHEN the character sheet is opened, THEN the XP bar fill is at 40%
  - Given: Character at level 4. `xp_for_current_level` = 150 × 4 × 3 = 1800. `xp_to_next_level` = 4 × 300 = 1200. Wait — GDD says Xnext = 1500 for this example. Let's use GDD values: Lxp = 1800, total_xp = 2400, Xnext = 1500. Progress = (2400 − 1800) / 1500 × 100 = 40%.
  - When: Character sheet is opened.
  - Then: XP bar visual fill is at 40% of the bar width (±2% tolerance → 38–42%).
  - Edge cases: 0% (total_xp = xp_for_current_level, just leveled), 100% (total_xp = xp_for_current_level + xp_to_next_level, ready to level), >100% (overflow — clamp to 100% visually). Level 1 (Lxp = 0). Level cap 20 (no XP bar shown — "Max Level" text).

- **AC-18**: GIVEN a character at level 4 earning enough XP to reach level 5 while the UI is open, WHEN the XP is awarded, THEN the Level-Up modal appears and cannot be dismissed without Confirm
  - Given: Character at level 4 with total_xp close to threshold. UI is open on character sheet.
  - When: XP is awarded that pushes total_xp to or past the level 5 threshold (threshold = 150 × 5 × 4 = 3000).
  - Then: Level-Up modal appears as overlay. Shows "Level Up — Level 4 → 5". HP increase displayed. If level 5 is not a 4th-level milestone, no attribute point to allocate — Confirm is immediately enabled. Modal cannot be dismissed by pressing Esc or clicking outside. Only Confirm closes it.
  - Edge cases: 4th level (level 4→5 is not a 4th-level milestone; 4→5 doesn't give attribute point). Attribute point levels: 4→5 (no), but 3→4 (yes, every 4th). Test both a non-attribute level-up and an attribute-point level-up (e.g., 3→4). Multiple simultaneous level-ups (shared XP event) — modals queue in party order. Level-up while viewing a different character's sheet — modal appears for the character who earned the level.

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 2/2 passing
**Deviations**: ADVISORY — GDD AC-4 uses Xnext=1500 (40%); ADR-0006 formula yields Xnext=1200 (50%). Implementation uses correct ADR formula. Documented in code + test.
**Test Evidence**: `tests/unit/character_sheet/xp_progress_test.go` — 23 tests PASS
**Code Review**: APPROVED (2026-08-11)
