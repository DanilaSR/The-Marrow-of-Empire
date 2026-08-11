// Go-driven drag-and-drop for the equipment slots (Story 003, AC-6/7/9).
// Kaiju's CSS has no drag events (Story 003 Engine Notes), so dragging is a
// Go state machine driven by the markup mouse handlers (equipmentFuncMap in
// this file): onmousedown on a filled slot starts a drag of its equipped
// item, onmouseenter/onmouseleave track the hover target (amber highlight
// on compatible slots, grey X on incompatible ones — AC-7), and onmouseup
// drops. Drags from the inventory grid (Story 005) start through
// DragFromInventory with the item's grid position.
//
// Dropping an item on a compatible slot equips it (EquipItem — writes the
// Inventory component and, via the equip callback, repopulates the sheet in
// the same frame). Dropping on an incompatible slot marks the slot rejected
// (grey X) and leaves the item where it was — no write, no error sound
// (AC-7). Dropping outside any slot cancels the drag. Right-click on a
// filled slot unequips the item to the first empty inventory slot (GDD).
//
// The controller is engine-free: it talks to the entity store and fires
// callbacks. The sheet wires the callbacks to repopulation and visuals;
// integration tests drive the state machine directly.
package character_sheet

import (
	"log/slog"

	"kaijuengine.com/engine/ui/markup/document"

	"marrow-of-empire/entity"
)

// DragDrop is the equipment drag-and-drop state machine (Story 003). One
// instance per CharacterSheet, owned by the sheet. It holds the owner
// entity (the viewed character), the active drag (source, dragged item,
// hover target), and the grey-X rejection slot.
type DragDrop struct {
	store entity.EntityStore
	owner entity.GameEntityID

	// onEquip fires after any equipment change (equip, unequip, move) — the
	// sheet re-reads the Stats and Inventory components and repopulates
	// synchronously, so displayed attributes and derived stats update within
	// the same frame (AC-9).
	onEquip func()
	// onVisual fires after any drag-state change (start, hover, reject,
	// drop, cancel) — the sheet refreshes the slot hover/reject classes.
	onVisual func()

	// Active drag state. dragging is true between a drag start and the
	// resolving drop/cancel; fromInventory distinguishes grid drags (Story
	// 005) from slot drags; sourceSlot/sourceGrid record the origin.
	dragging      bool
	fromInventory bool
	sourceSlot    SlotType
	sourceGrid    int
	itemID        entity.GameEntityID

	// hover is the slot under the pointer during a drag; rejected is the
	// slot showing the grey X (incompatible hover or a rejected drop),
	// InvalidSlot when none.
	hover    SlotType
	rejected SlotType
}

// NewDragDrop creates the drag state machine for the given entity store.
func NewDragDrop(store entity.EntityStore) *DragDrop {
	return &DragDrop{
		store:    store,
		hover:    InvalidSlot,
		rejected: InvalidSlot,
	}
}

// SetOwner sets the character whose equipment the drags act on (the sheet
// sets it on Open).
func (d *DragDrop) SetOwner(owner entity.GameEntityID) { d.owner = owner }

// Owner returns the current drag target entity (0 when the sheet is closed).
func (d *DragDrop) Owner() entity.GameEntityID { return d.owner }

// OnEquip sets the equipment-change callback (called synchronously inside
// the resolving Drop/RightClickUnequip, so the caller's repopulation lands
// in the same frame).
func (d *DragDrop) OnEquip(fn func()) { d.onEquip = fn }

// OnVisual sets the drag-visual callback (hover highlight, grey X).
func (d *DragDrop) OnVisual(fn func()) { d.onVisual = fn }

// Dragging reports whether a drag is in progress.
func (d *DragDrop) Dragging() bool { return d.dragging }

// FromInventory reports whether the active drag started in the inventory
// grid (Story 005) rather than on an equipment slot.
func (d *DragDrop) FromInventory() bool { return d.fromInventory }

// SourceSlot returns the slot the active drag started from; valid only when
// FromInventory is false.
func (d *DragDrop) SourceSlot() SlotType { return d.sourceSlot }

// DraggedItem returns the item entity currently being dragged; 0 when idle.
func (d *DragDrop) DraggedItem() entity.GameEntityID { return d.itemID }

// HoverTarget returns the slot currently under the pointer during a drag;
// InvalidSlot when none.
func (d *DragDrop) HoverTarget() SlotType { return d.hover }

// RejectedSlot returns the slot currently showing the grey X (incompatible
// hover or rejected drop); InvalidSlot when none.
func (d *DragDrop) RejectedSlot() SlotType { return d.rejected }

// DragFromInventory starts a drag of the inventory item (Story 005's grid
// calls this with the item's grid position). HoverSlot then tracks the
// pointer and Drop resolves the equip. No-op while no owner is set or for a
// zero item ID.
func (d *DragDrop) DragFromInventory(itemID entity.GameEntityID, gridPos int) {
	if itemID == 0 || d.owner == 0 {
		return
	}
	d.dragging = true
	d.fromInventory = true
	d.sourceSlot = InvalidSlot
	d.sourceGrid = gridPos
	d.itemID = itemID
	d.hover = InvalidSlot
	d.rejected = InvalidSlot
	d.fireVisual()
}

// DragFromSlot starts a drag of the item equipped in the slot (onmousedown
// on a filled slot). Dropping on another compatible slot moves the item
// (EquipItem clears the old slot); dropping elsewhere cancels and the item
// stays equipped. No-op for empty or invalid slots.
func (d *DragDrop) DragFromSlot(slot SlotType) {
	if d.owner == 0 || !slot.Valid() {
		return
	}
	slots, ok := LoadEquipment(d.store, d.owner)
	if !ok || slots.Item(slot) == 0 {
		return
	}
	d.dragging = true
	d.fromInventory = false
	d.sourceSlot = slot
	d.sourceGrid = 0
	d.itemID = slots.Item(slot)
	d.hover = InvalidSlot
	d.rejected = InvalidSlot
	d.fireVisual()
}

// HoverSlot tracks the pointer over a slot during a drag: compatible items
// keep the slot as the hover target (the sheet's amber highlight), and
// incompatible items mark the slot rejected (grey X — AC-7's in-drag state).
func (d *DragDrop) HoverSlot(slot SlotType) {
	if !d.dragging || !slot.Valid() {
		return
	}
	d.hover = slot
	if d.compatible(slot) {
		d.rejected = InvalidSlot
	} else {
		d.rejected = slot
	}
	d.fireVisual()
}

// HoverNone clears the hover target and the grey-X rejection (the pointer
// left the slot). Not gated on the drag: after a rejected drop the X stays
// visible until the pointer moves off the slot.
func (d *DragDrop) HoverNone() {
	d.hover = InvalidSlot
	d.rejected = InvalidSlot
	d.fireVisual()
}

// Drop resolves the drag at the current hover target:
//
//   - a compatible slot equips the item (EquipItem writes the Inventory
//     component; onEquip fires, so derived stats update in the same frame —
//     AC-6, AC-9);
//   - an incompatible slot keeps the rejected grey X and the item stays
//     where it was (AC-7 — no write, no error sound);
//   - no hover cancels the drag (item stays; dropping back on a slot drag's
//     source slot is a no-op move).
func (d *DragDrop) Drop() {
	if !d.dragging {
		return
	}
	item := d.itemID
	target := d.hover
	d.dragging = false
	if target.Valid() && !d.compatible(target) {
		// Rejected drop: the grey X stays visible, the item returns to its
		// origin (the store was never touched).
		d.rejected = target
		d.hover = InvalidSlot
		d.itemID = 0
		d.fireVisual()
		return
	}
	if !target.Valid() {
		// Dropped outside any slot — cancel.
		d.clearDrag()
		return
	}
	if err := EquipItem(d.store, d.owner, item, target); err != nil {
		// Defensive: HoverSlot already rejected incompatible targets, so
		// this only fires for store errors (missing components).
		slog.Warn("character sheet: drop rejected", "item", item, "slot", target, "err", err)
		d.rejected = target
		d.hover = InvalidSlot
		d.itemID = 0
		d.fireVisual()
		return
	}
	d.itemID = 0
	d.fireVisual()
	if d.onEquip != nil {
		d.onEquip()
	}
}

// Cancel ends the drag without any equipment change (mouseup outside any
// slot, or the sheet closing mid-drag). No-op when no drag is active.
func (d *DragDrop) Cancel() {
	if !d.dragging {
		return
	}
	d.clearDrag()
}

// Reset clears all drag state unconditionally (the sheet calls it on Open
// and Close so a half-finished or rejected drag never carries into the next
// session). It does not touch the owner.
func (d *DragDrop) Reset() {
	d.dragging = false
	d.fromInventory = false
	d.sourceSlot = InvalidSlot
	d.sourceGrid = 0
	d.itemID = 0
	d.hover = InvalidSlot
	d.rejected = InvalidSlot
	d.fireVisual()
}

// RightClickUnequip unequips the slot's item to the first empty inventory
// slot (right-click on a filled slot, GDD) and fires the equip callback so
// the display and derived stats update in the same frame. Errors (no
// Inventory component, empty slot) are logged, not played — mirroring AC-7's
// silent rejection (no error sound).
func (d *DragDrop) RightClickUnequip(slot SlotType) {
	if d.owner == 0 {
		return
	}
	if err := UnequipSlot(d.store, d.owner, slot); err != nil {
		slog.Warn("character sheet: unequip rejected", "slot", slot, "err", err)
		return
	}
	if d.onEquip != nil {
		d.onEquip()
	}
}

// compatible reports whether the dragged item fits the slot, reading the
// item's Item component (items without one are incompatible).
func (d *DragDrop) compatible(slot SlotType) bool {
	item, ok := entity.GetComponent[Item](d.store, d.itemID)
	if !ok {
		return false
	}
	return SlotCompatible(item.Type, slot)
}

// clearDrag ends the drag without any equipment change.
func (d *DragDrop) clearDrag() {
	d.dragging = false
	d.fromInventory = false
	d.sourceSlot = InvalidSlot
	d.sourceGrid = 0
	d.itemID = 0
	d.hover = InvalidSlot
	d.rejected = InvalidSlot
	d.fireVisual()
}

// fireVisual notifies the sheet that the slot visuals need refreshing.
func (d *DragDrop) fireVisual() {
	if d.onVisual != nil {
		d.onVisual()
	}
}

// equipmentFuncMap returns the mouse handler map for the equipment slots
// (Story 003), dispatched through the markup loader's funcMap. Kaiju CSS
// has no drag events, so the handlers drive the Go DragDrop state machine.
// The handlers dispatch to the sheet's drag controller, which is created
// before the document loads; the closures read s.drag lazily and no-op
// while it is nil. Markup contract:
//
//	onmousedown="equipDragStart"   — slot cell, data-slot="{key}" (drag from slot)
//	onmouseenter="equipHover"      — slot cell, data-slot="{key}"
//	onmouseleave="equipHoverNone"  — slot cell
//	onmouseup="equipDrop"          — slot cell, data-slot="{key}"
//	onmouseup="equipDropOutside"   — sheet-root (cancel on drop outside)
//	onrightclick="equipUnequip"    — slot cell, data-slot="{key}"
func equipmentFuncMap(s *CharacterSheet) map[string]func(*document.Element) {
	return map[string]func(*document.Element){
		"equipDragStart": func(elm *document.Element) {
			if s.drag == nil {
				return
			}
			if slot, ok := slotFromData(elm); ok {
				s.drag.DragFromSlot(slot)
			}
		},
		"equipHover": func(elm *document.Element) {
			if s.drag == nil {
				return
			}
			if slot, ok := slotFromData(elm); ok {
				s.drag.HoverSlot(slot)
			}
		},
		"equipHoverNone": func(*document.Element) {
			if s.drag != nil {
				s.drag.HoverNone()
			}
		},
		"equipDrop": func(elm *document.Element) {
			if s.drag == nil {
				return
			}
			if slot, ok := slotFromData(elm); ok {
				s.drag.HoverSlot(slot) // drop lands on the hovered slot
				s.drag.Drop()
			}
		},
		"equipDropOutside": func(*document.Element) {
			if s.drag != nil {
				s.drag.Cancel()
			}
		},
		"equipUnequip": func(elm *document.Element) {
			if s.drag == nil {
				return
			}
			if slot, ok := slotFromData(elm); ok {
				s.drag.RightClickUnequip(slot)
			}
		},
	}
}

// slotFromData maps a slot element's data-slot value to a SlotType.
// Unknown values are logged and rejected.
func slotFromData(elm *document.Element) (SlotType, bool) {
	key := elm.Attribute("data-slot")
	for _, slot := range SlotTypes {
		if slot.Key() == key {
			return slot, true
		}
	}
	slog.Warn("character sheet: unknown data-slot", "slot", key)
	return InvalidSlot, false
}
