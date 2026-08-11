# Story 004: Equipment Stat Comparison

> **Epic**: Character & Inventory UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: T-shirt S
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/character-inventory-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: The item detail panel on the inventory screen includes a comparison mode that shows equipped item vs. selected inventory item with stat differences.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI rendering API — verify dynamic text coloring support against `kaiju/src/`.

**Control Manifest Rules (this layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space, Desk serif for text, Never encode game-critical information by hue alone — stat diffs use both color AND sign (+/−)
- Forbidden: Never use amber-on-parchment text — amber is accent/fill only
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/character-inventory-ui.md`, scoped to this story:*

- [ ] **AC-8 — Equipment stat comparison**: GIVEN an equipped sword (+3 damage) and a compared axe (+5 damage) in inventory, WHEN the player selects the axe, THEN the comparison panel shows "−2" in green next to Damage.

---

## Implementation Notes

*Derived from ADR-0012:*

- The item detail panel (right side of inventory screen) includes a comparison section when the selected item is equipment and the same slot is currently occupied.
- Layout: equipped item on the left, selected/hovered item on the right, stat differences between them with +/− coloring.
- Formula: `stat_diff = equipped_item_stat − compared_item_stat`
  - Positive diff: equipped is better → displayed in amber with "+" sign
  - Negative diff: compared is better → displayed in green with "−" sign
  - Zero: equal → displayed in ink (parchment-dark) with no sign
  - Diff formatted with sign: "+2" or "−3"
- Example from GDD: Equipped sword gives +3 damage. Compared axe gives +5 damage. Diff = 3 − 5 = −2 → displayed as "−2" in green (axe is better by 2 damage).
- The comparison updates in real-time as the player hovers or selects different items in the inventory grid.
- Accessibility: color is never the sole differentiator — the +/− sign and numeric value convey the information independently of color (per ADR-0012 forbidden: never encode game-critical information by hue alone).
- Compare all relevant stats for the item type: Damage for weapons, Armor/Defense for armor pieces, attribute bonuses, special properties.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- [Story 003]: The actual equip/unequip action (this story only handles the comparison display)
- [Story 005]: Inventory grid rendering and item detail panel structure (this story adds the comparison mode to the detail panel)

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-8**: GIVEN an equipped sword (+3 damage) and a compared axe (+5 damage) in inventory, WHEN the player selects the axe, THEN the comparison panel shows "−2" in green next to Damage
  - Setup: Equip a sword with +3 damage to Main Hand. Place an axe with +5 damage in inventory. Open the inventory tab.
  - Verify: Click/select the axe in the inventory grid. The item detail panel shows both items side by side (equipped sword on left, selected axe on right). Damage stat shows "−2" in green next to the equipped item's +3 and the axe's +5.
  - Pass condition: The comparison clearly communicates that the axe has +2 more damage than the sword. Green color AND minus sign both indicate the axe is better. Test with reversed scenario (equipped item is better → positive diff in amber).

  - **Additional scenarios** (manual check):
    - Setup: Equip armor with +5 defense. Hover over weaker armor (+2 defense) in inventory.
    - Verify: Diff shows "+3" in amber (equipped is better).
    - Setup: Equip ring with +1 STR. Hover over another ring with +1 STR.
    - Verify: Diff shows "0" in ink — equal items.
    - Setup: Hover over an item of a different slot type (e.g., helmet when comparing to equipped sword).
    - Verify: No comparison shown — only same-slot items trigger comparison.
