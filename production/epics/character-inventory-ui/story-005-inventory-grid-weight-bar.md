# Story 005: Inventory Grid & Weight Bar

> **Epic**: Character & Inventory UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: T-shirt L
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/character-inventory-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-003`, `TR-ui-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: The inventory tab renders a scrollable grid of items sorted by category with pen-and-ink icons, an item detail panel, a weight bar, and a gold display. All within the Desk register.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI rendering API — verify scrollable grid and dynamic bar fill support against `kaiju/src/`.

**Control Manifest Rules (this layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space, Desk register (parchment overlay), Pen-and-ink icons (single-stroke, 2px at 24px, flat fills only for palette colors), Desk serif for text
- Forbidden: Never use amber-on-parchment text — amber is accent/fill only (weight bar uses amber FILL, percentage text uses ink), Never encode game-critical information by hue alone — weight bar color changes accompanied by percentage text
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/character-inventory-ui.md`, scoped to this story:*

- [ ] **AC-10**: GIVEN inventory weight at 95% of max, WHEN the inventory tab is open, THEN the weight bar pulses oxblood.
- [ ] **AC-11 — Weight bar color at normal**: GIVEN inventory weight at 30% of max, WHEN the inventory tab is open, THEN the weight bar is solid amber with no pulse.
- [ ] **AC-12 — Weight bar color at warning**: GIVEN inventory weight at 70% of max, WHEN the inventory tab is open, THEN the weight bar shows the amber-oxblood transition color.
- [ ] **AC-14 — Empty inventory**: GIVEN a character with no items in inventory, WHEN the inventory tab is opened, THEN the grid shows "Your pack is empty." in subdued ink and the gold display remains visible.

---

## Implementation Notes

*Derived from ADR-0012:*

- **Inventory grid**: Scrollable grid, default 6 columns × N rows. Each cell 64×64px.
  - Each cell: item icon + stack count (bottom-right corner, amber text).
  - Empty cells: subtle grid lines.
  - Sorted by category (weapons → armor → consumables → quest items → crafting materials), then alphabetically within category.
- **Category filter** bar at top of grid: All, Weapons, Armor, Consumables, Quest Items, Crafting. Amber underline on active filter.
- **Search bar** at top of grid: type to filter items by name.
- **Item detail panel** (right): Selected item's name, icon (large), description text, stats list, weight, value. Equipment comparison (Story 004) extends this panel.
- **Weight bar** (below grid): Horizontal bar showing `current_weight / max_capacity` with percentage text.
  - Formula: `weight_pct = (current_carry_weight / max_carry_capacity) × 100`
  - `max_carry_capacity = (STR × 10) + 30` (from ADR-0006)
  - Color thresholds:
    - 0–49%: solid amber (normal)
    - 50–89%: amber shifting toward oxblood (warning transition)
    - 90–100%: oxblood (near limit)
    - \>100%: oxblood pulsing (encumbered — movement penalty applies per Character Stats)
  - Percentage text always visible next to or on the bar (ensures color is not the sole information channel).
- **Gold display**: Top-right corner of inventory panel. Coin icon + gold amount. Cosmetic cap at 999,999 (displays "999,999+" beyond that).
- **Empty inventory state**: When inventory has zero items, grid area shows "Your pack is empty." in subdued ink (centered). Weight bar hidden (0 lbs). Gold display remains visible.
- Read inventory data from the `Inventory` component via Entity Framework. Gold amount also from `Inventory` component.
- Interaction: Click item to select → detail panel updates. Double-click to equip/use. Right-click for context menu (Equip / Use / Drop / Examine / Split Stack). Shift+click to split stack (modal with quantity slider).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- [Story 003]: Drag-and-drop from inventory grid to equipment slots
- [Story 004]: Equipment stat comparison in the detail panel (this story builds the detail panel structure; Story 004 adds comparison mode)
- [Story 006]: Inventory full toast notification
- [Story 008]: Dragging item to sidebar portrait for character-to-character transfer

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-10**: GIVEN inventory weight at 95% of max, WHEN the inventory tab is open, THEN the weight bar pulses oxblood
  - Setup: Fill a character's inventory until current weight equals 95% of max carry capacity. Open inventory tab.
  - Verify: Weight bar fill covers 95% of the bar width. Bar color is oxblood. Bar has a visible pulse animation (not static). Percentage text shows "95%".
  - Pass condition: Oxblood color + pulse animation both present. Percentage text confirms 95% value.

- **AC-11**: GIVEN inventory weight at 30% of max, WHEN the inventory tab is open, THEN the weight bar is solid amber with no pulse
  - Setup: Fill inventory to 30% of max carry capacity. Open inventory tab.
  - Verify: Weight bar fill at 30%. Color is amber. No pulse animation. Percentage text shows "30%".
  - Pass condition: Solid amber fill, no animation. Percentage matches.

- **AC-12**: GIVEN inventory weight at 70% of max, WHEN the inventory tab is open, THEN the weight bar shows amber-oxblood transition color
  - Setup: Fill inventory to 70% of max carry capacity. Open inventory tab.
  - Verify: Weight bar fill at 70%. Color is a blend between amber and oxblood (not purely one or the other). No pulse (pulse only at ≥90%). Percentage text shows "70%".
  - Pass condition: Transition color visually distinct from both pure amber (30%) and pure oxblood (95%).

- **AC-14**: GIVEN a character with no items in inventory, WHEN the inventory tab is opened, THEN the grid shows "Your pack is empty."
  - Setup: Create or use a character with an empty inventory (no items, 0 weight). Open inventory tab.
  - Verify: Grid area displays the text "Your pack is empty." centered in subdued ink (not bright white or amber). No weight bar is shown. Gold display is visible in top-right corner showing current gold (even if 0).
  - Pass condition: Empty state message visible, weight bar absent, gold display present.
