# Story 003: Equipment Slots & Drag-and-Drop

> **Epic**: Character & Inventory UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: T-shirt L
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/character-inventory-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-003`, `TR-ui-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework; ADR-0001: Entity Architecture (reads/writes Inventory component)
**ADR Decision Summary**: UI renders equipment slots as part of the Desk-register character sheet. Equipment changes write to the `Inventory` component on the selected entity via Entity Framework. Derived stats recalculate immediately on equip/unequip.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju UI drag-and-drop support — verify against `kaiju/src/`. Post-cutoff APIs: input routing for drag operations.

**Control Manifest Rules (this layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space, Desk register (parchment overlay), Pen-and-ink icons (single-stroke, 2px at 24px, flat fills only for palette colors)
- Forbidden: Never use 3D UI or diegetic HUD, Never use skeuomorphic buttons
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/character-inventory-ui.md`, scoped to this story:*

- [x] **AC-6**: GIVEN an item dragged from inventory to an equipment slot, WHEN the slot is compatible, THEN the item equips and derived stats update immediately.
- [x] **AC-7 — Incompatible slot rejection**: GIVEN a helmet dragged to the Hands equipment slot, WHEN the item is dropped, THEN the slot shows a greyed-out X indicator and the item returns to its original inventory position with no equipment change.
- [x] **AC-9 — Attribute-changing equipment**: GIVEN a character with STR 14 equipping a ring that grants +2 STR, WHEN the ring is equipped, THEN STR displays as "14 (+2)" and derived stats (carry capacity, melee damage) recalculate within one frame.

---

## Implementation Notes

*Derived from ADR-0012 and ADR-0001:*

- **Equipment slots** (8 total): Head, Chest, Hands, Legs, Main Hand, Off Hand, Accessory 1, Accessory 2.
  - Arranged horizontally in a bottom strip of the character sheet.
  - Each slot is a 72×72px square cell.
  - Three visual states per slot: Empty (silhouette outline), Filled (item icon), Incompatible (greyed out with X, shown during drag).
- **Drag-and-drop**:
  - Drag item from inventory grid → equipment slot to equip.
  - Drag from equipment slot → inventory grid to unequip.
  - During drag over a slot: if item type is compatible, slot highlights amber; if incompatible, slot shows grey X.
  - Dropping on incompatible slot: item returns to original inventory position, no equipment change, no error sound.
  - Right-click equipped item: unequip to first empty inventory slot.
  - Click empty equipment slot with empty cursor: highlight compatible inventory items.
- **Equipment writes**: On equip/unequip, write the change to the `Inventory` component on the selected entity via Entity Framework (ADR-0001).
- **Derived stat recalculation**: After any equipment change, trigger recalculation of all derived stats (Max Health, AP, Initiative, Evasion, Carry Capacity). The UI re-reads updated values from the `Stats` component.
- **Attribute-changing equipment**: When an item grants an attribute bonus (e.g., +2 STR ring), the affected attribute displays the modified total with a "(+2)" note in green next to the base value. All dependent derived stats update on-screen within one frame.
- Pen-and-ink icons for equipment slots: single-stroke outlines, 2px at 24px reference. Empty slots use silhouette outlines.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- [Story 004]: Equipment stat comparison panel (comparing equipped vs. inventory item stats)
- [Story 005]: Inventory grid display and weight bar (grid is rendered, this story handles drag FROM grid to slots)
- [Story 008]: Dragging item to sidebar party portrait to transfer item between characters

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-6**: GIVEN an item dragged from inventory to an equipment slot, WHEN the slot is compatible, THEN the item equips and derived stats update immediately
  - Given: A character with a sword in inventory and an empty Main Hand equipment slot. Character has known STR and derived stats.
  - When: Player drags the sword from inventory grid and drops it on the Main Hand slot.
  - Then: Sword icon appears in the Main Hand slot (Filled state). Sword is removed from inventory grid (or marked as equipped). If the sword grants stat bonuses, derived stats in the right panel update to reflect new values within one frame. The `Inventory` component on the entity reflects the equipment change.
  - Edge cases: Swapping — dragging a sword to Main Hand when a dagger is already equipped: dagger moves to inventory, sword equips. Dragging from equipment slot to inventory to unequip. Right-click unequip to first empty slot.

- **AC-7**: GIVEN a helmet dragged to the Hands equipment slot, WHEN the item is dropped, THEN the slot shows a greyed-out X and item returns to original position
  - Given: A helmet in inventory. Hands slot is empty or filled (with gloves).
  - When: Player drags the helmet over the Hands slot (incompatible — helmet type ≠ hands slot type). Slot shows greyed-out X during hover. Player drops the item.
  - Then: Helmet returns to its original inventory grid position. Hands slot returns to its previous state (Empty or Filled with gloves). No equipment change is written to the `Inventory` component. No error sound plays.
  - Edge cases: Dropping incompatible item on a filled slot (existing item stays equipped). Dropping on slot border/edge (rejection zone extends to slot bounds).

- **AC-9**: GIVEN a character with STR 14 equipping a ring that grants +2 STR, WHEN the ring is equipped, THEN STR displays as "14 (+2)" and derived stats recalculate within one frame
  - Given: Character with base STR 14 (modifier +2). A ring with +2 STR enchantment in inventory. Accessory 1 slot empty.
  - When: Player equips the ring to Accessory 1.
  - Then: STR display shows "14 (+2)" — base value 14 with green "(+2)" bonus note. Carry capacity recalculates: was (14×10)+30 = 170, now (16×10)+30 = 190. Any other STR-derived stats update. All updates render within one frame (16.6ms).
  - Edge cases: Unequipping the ring reverts the display. Multiple attribute-changing items stacking (ring +2 STR, belt +1 STR → "14 (+3)"). Item that changes multiple attributes (e.g., +1 STR, +1 CON). Negative attribute modifiers from cursed items.
