// Integration tests for Story 003: Equipment Slots & Drag-and-Drop.
//
// Evidence path: production/epics/character-inventory-ui/
// story-003-equipment-slots-drag-drop.md
// Required evidence: this file must exist and pass (Integration story type —
// BLOCKING gate).
//
// The acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-6  test_equip_sword_to_main_hand_updates_inventory_and_derived_stats -> TestEquipSwordToMainHandUpdatesInventoryAndDerivedStats
//	AC-7  test_reject_helmet_on_hands_leaves_no_change                       -> TestRejectHelmetOnHandsLeavesNoChange
//	AC-9  test_equip_str_ring_updates_display_and_derived_stats              -> TestEquipStrRingUpdatesDisplayAndDerivedStats
//
// QA edge cases from the story are covered alongside: swapping (dragging
// onto an occupied slot moves the occupant to inventory), incompatible
// drops on filled slots (the existing item stays equipped), drags from an
// equipment slot (cancel keeps the item; compatible drop moves it),
// right-click unequip to the first empty inventory slot, and stacking
// attribute bonuses (ring +2 STR, belt +1 STR → "14 (+3)").
//
// The tests drive the DragDrop state machine and the equip logic against a
// real EntityStoreImpl backed by the mock Kaiju host (mirrored from
// tests/integration/entity per the shared-helper convention — Go cannot
// import test helpers across packages). No engine UI runs: the markup layer
// (sheet.go element binding and rendering) is verified by screenshot
// evidence per the UI story testing standard; everything data-bearing is
// exercised here against the entity store.
package character_sheet_test

import (
	"testing"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"

	"marrow-of-empire/character_sheet"
	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// mockKaijuHost is a test double for entity.KaijuHost (mirrored from
// tests/integration/entity). It registers engine entities by EntityId and
// records native destroy calls.
type mockKaijuHost struct {
	entities  map[engine.EntityId]*engine.Entity
	destroyed []engine.EntityId
	workGroup concurrent.WorkGroup
}

func (m *mockKaijuHost) EntityById(id engine.EntityId) *engine.Entity {
	return m.entities[id]
}

func (m *mockKaijuHost) SetEntityId(entity *engine.Entity, id engine.EntityId) bool {
	if id == "" {
		return false
	}
	if _, exists := m.entities[id]; exists {
		return false
	}
	m.entities[id] = entity
	return true
}

func (m *mockKaijuHost) DestroyEntity(entity *engine.Entity) {
	for id, e := range m.entities {
		if e == entity {
			delete(m.entities, id)
			m.destroyed = append(m.destroyed, id)
			return
		}
	}
}

func (m *mockKaijuHost) WorkGroup() *concurrent.WorkGroup { return &m.workGroup }

func newMockHost() *mockKaijuHost {
	m := &mockKaijuHost{entities: make(map[engine.EntityId]*engine.Entity)}
	m.workGroup.Init()
	return m
}

// newTestStore wires a store to a mock host and a fresh event bus (mirrored
// from tests/integration/entity). Config zero values fall back to the GDD
// defaults.
func newTestStore() (*entity.EntityStoreImpl, *mockKaijuHost) {
	host := newMockHost()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(host, bus, entity.EntityStoreConfig{})
	return store, host
}

// createCharacter creates a level-1 character entity with the given base
// attributes and an empty inventory.
func createCharacter(t *testing.T, store *entity.EntityStoreImpl, attrs [stats.NumAttributes]int) entity.GameEntityID {
	t.Helper()
	id, err := store.CreateEntity(entity.EntityTypeCharacter, "Player", matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity(character) error: %v", err)
	}
	if err := entity.SetComponent(store, id, stats.StatsComponent{Attributes: attrs, Level: 1}); err != nil {
		t.Fatalf("SetComponent(Stats) error: %v", err)
	}
	if err := entity.SetComponent(store, id, entity.Inventory{
		Items:          []entity.GameEntityID{},
		EquipmentSlots: map[string]entity.GameEntityID{},
	}); err != nil {
		t.Fatalf("SetComponent(Inventory) error: %v", err)
	}
	return id
}

// createItem creates an item entity with the given name, item type string,
// and attribute bonuses (Story 003 Item component).
func createItem(t *testing.T, store *entity.EntityStoreImpl, name, itemType string, bonuses [stats.NumAttributes]int) entity.GameEntityID {
	t.Helper()
	id, err := store.CreateEntity(entity.EntityTypeItem, name, matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity(item %q) error: %v", name, err)
	}
	if err := entity.SetComponent(store, id, character_sheet.Item{
		Name:             name,
		Type:             itemType,
		AttributeBonuses: bonuses,
	}); err != nil {
		t.Fatalf("SetComponent(Item) error: %v", err)
	}
	return id
}

// setInventory replaces the character's Inventory component (test setup).
func setInventory(t *testing.T, store *entity.EntityStoreImpl, owner entity.GameEntityID, inv entity.Inventory) {
	t.Helper()
	if err := entity.SetComponent(store, owner, inv); err != nil {
		t.Fatalf("SetComponent(Inventory) error: %v", err)
	}
}

// inventory returns the character's current Inventory component.
func inventory(t *testing.T, store *entity.EntityStoreImpl, owner entity.GameEntityID) entity.Inventory {
	t.Helper()
	inv, ok := entity.GetComponent[entity.Inventory](store, owner)
	if !ok {
		t.Fatalf("entity %d has no Inventory component", owner)
	}
	return inv
}

// carries reports whether the inventory list contains the item.
func carries(inv entity.Inventory, itemID entity.GameEntityID) bool {
	for _, id := range inv.Items {
		if id == itemID {
			return true
		}
	}
	return false
}

// equipped reports whether the slot holds the item.
func equipped(inv entity.Inventory, slot character_sheet.SlotType, itemID entity.GameEntityID) bool {
	return inv.EquipmentSlots[slot.Key()] == itemID
}

// derivedCarry reads the entity's Stats and equipment and returns the carry
// capacity displayed with equipment bonuses applied (AC-6/AC-9: effective
// STR ×10 + 30, DerivedStats index 4).
func derivedCarry(t *testing.T, store *entity.EntityStoreImpl, owner entity.GameEntityID) int {
	t.Helper()
	base, ok := entity.GetComponent[stats.StatsComponent](store, owner)
	if !ok {
		t.Fatalf("entity %d has no Stats component", owner)
	}
	slots, ok := character_sheet.LoadEquipment(store, owner)
	if !ok {
		t.Fatalf("entity %d has no Inventory component", owner)
	}
	eff := character_sheet.EffectiveAttributes(base, character_sheet.TotalAttributeBonuses(store, slots))
	return character_sheet.DerivedStats(eff)[4].Value
}

// AC-6: GIVEN an item dragged from inventory to an equipment slot, WHEN the
// slot is compatible, THEN the item equips and derived stats update
// immediately.
//
// Setup: STR 14 character (carry 14×10+30 = 170) with a sword in inventory
// and an empty Main Hand slot. The sword grants no attribute bonuses, so
// the derived stats are unchanged by the equip itself — the update is
// verified through the equip callback (fired synchronously inside Drop:
// same-frame repopulation) and through the Inventory component write.
func TestEquipSwordToMainHandUpdatesInventoryAndDerivedStats(t *testing.T) {
	store, _ := newTestStore()
	attrs := [stats.NumAttributes]int{stats.STR: 14, stats.DEX: 10, stats.CON: 10}
	hero := createCharacter(t, store, attrs)
	sword := createItem(t, store, "Iron Sword", character_sheet.ItemTypeWeaponSword, [stats.NumAttributes]int{})
	setInventory(t, store, hero, entity.Inventory{
		Items:          []entity.GameEntityID{sword},
		EquipmentSlots: map[string]entity.GameEntityID{},
	})

	dd := character_sheet.NewDragDrop(store)
	dd.SetOwner(hero)
	equips := 0
	carryDuringCallback := 0
	dd.OnEquip(func() {
		equips++
		carryDuringCallback = derivedCarry(t, store, hero)
	})

	dd.DragFromInventory(sword, 0)
	if !dd.Dragging() {
		t.Fatal("drag did not start from inventory")
	}
	if dd.DraggedItem() != sword || !dd.FromInventory() {
		t.Fatalf("drag state: item = %d (want %d), fromInventory = %v (want true)",
			dd.DraggedItem(), sword, dd.FromInventory())
	}
	dd.HoverSlot(character_sheet.SlotMainHand)
	if dd.HoverTarget() != character_sheet.SlotMainHand {
		t.Errorf("hover target = %v, want Main Hand", dd.HoverTarget())
	}
	if dd.RejectedSlot() != character_sheet.InvalidSlot {
		t.Errorf("compatible hover must not reject; rejected slot = %v", dd.RejectedSlot())
	}
	dd.Drop()

	if equips != 1 {
		t.Errorf("equip callback fired %d times, want exactly 1 (same-frame update)", equips)
	}
	if carryDuringCallback != 170 {
		t.Errorf("carry capacity in the equip callback = %d, want 170 (STR 14: 14×10+30)", carryDuringCallback)
	}
	inv := inventory(t, store, hero)
	if !equipped(inv, character_sheet.SlotMainHand, sword) {
		t.Errorf("Main Hand slot does not hold the sword: %v", inv.EquipmentSlots)
	}
	if carries(inv, sword) {
		t.Error("sword still in the carried list after equipping")
	}
	if len(inv.EquipmentSlots) != 1 {
		t.Errorf("equipment map has %d entries, want exactly 1 (no stale slots)", len(inv.EquipmentSlots))
	}
	if dd.Dragging() {
		t.Error("drag still active after drop")
	}
}

// AC-6 QA edge case: swapping — dragging a sword to Main Hand when a dagger
// is already equipped moves the dagger back to the first empty inventory
// slot and the sword equips.
func TestEquipToOccupiedSlotSwapsOccupantToInventory(t *testing.T) {
	store, _ := newTestStore()
	hero := createCharacter(t, store, [stats.NumAttributes]int{stats.STR: 14})
	dagger := createItem(t, store, "Bone Dagger", character_sheet.ItemTypeWeaponDagger, [stats.NumAttributes]int{})
	sword := createItem(t, store, "Iron Sword", character_sheet.ItemTypeWeaponSword, [stats.NumAttributes]int{})
	setInventory(t, store, hero, entity.Inventory{
		Items:          []entity.GameEntityID{sword},
		EquipmentSlots: map[string]entity.GameEntityID{character_sheet.SlotMainHand.Key(): dagger},
	})

	dd := character_sheet.NewDragDrop(store)
	dd.SetOwner(hero)
	dd.DragFromInventory(sword, 0)
	dd.HoverSlot(character_sheet.SlotMainHand)
	dd.Drop()

	inv := inventory(t, store, hero)
	if !equipped(inv, character_sheet.SlotMainHand, sword) {
		t.Errorf("Main Hand holds %d, want the sword %d", inv.EquipmentSlots[character_sheet.SlotMainHand.Key()], sword)
	}
	if carries(inv, sword) {
		t.Error("sword still in the carried list after equipping")
	}
	if !carries(inv, dagger) {
		t.Error("displaced dagger did not return to the carried list")
	}
}

// AC-7: GIVEN a helmet dragged to the Hands equipment slot, WHEN the item
// is dropped, THEN the slot shows a greyed-out X indicator and the item
// returns to its original inventory position with no equipment change.
//
// The grey X is asserted through the rejected-slot state: it appears while
// hovering the incompatible slot, persists after the drop (AC-7's "slot
// shows a greyed-out X"), and clears when the pointer leaves (HoverNone).
// No equip callback fires and the Inventory component is untouched — no
// write, no error sound.
func TestRejectHelmetOnHandsLeavesNoChange(t *testing.T) {
	store, _ := newTestStore()
	hero := createCharacter(t, store, [stats.NumAttributes]int{stats.STR: 14})
	helmet := createItem(t, store, "Iron Helmet", character_sheet.ItemTypeArmorHead, [stats.NumAttributes]int{})
	setInventory(t, store, hero, entity.Inventory{
		Items:          []entity.GameEntityID{helmet},
		EquipmentSlots: map[string]entity.GameEntityID{},
	})

	dd := character_sheet.NewDragDrop(store)
	dd.SetOwner(hero)
	equips := 0
	dd.OnEquip(func() { equips++ })

	dd.DragFromInventory(helmet, 0)
	dd.HoverSlot(character_sheet.SlotHands)
	if dd.RejectedSlot() != character_sheet.SlotHands {
		t.Errorf("hovering an incompatible slot: rejected slot = %v, want Hands (grey X shown)", dd.RejectedSlot())
	}
	dd.Drop()

	if dd.RejectedSlot() != character_sheet.SlotHands {
		t.Errorf("after a rejected drop: rejected slot = %v, want Hands (grey X persists)", dd.RejectedSlot())
	}
	if dd.Dragging() {
		t.Error("drag still active after a rejected drop")
	}
	if equips != 0 {
		t.Errorf("equip callback fired %d times, want 0 (no equipment change)", equips)
	}
	inv := inventory(t, store, hero)
	if _, ok := inv.EquipmentSlots[character_sheet.SlotHands.Key()]; ok {
		t.Errorf("Hands slot was written: %v", inv.EquipmentSlots)
	}
	if !carries(inv, helmet) {
		t.Error("helmet did not return to its original inventory position")
	}

	dd.HoverNone()
	if dd.RejectedSlot() != character_sheet.InvalidSlot {
		t.Errorf("pointer leaving the slot must clear the grey X; rejected slot = %v", dd.RejectedSlot())
	}
}

// AC-7 QA edge case: dropping an incompatible item on a filled slot leaves
// the existing item equipped — the rejection never touches the slot.
func TestRejectHelmetOnFilledHandsKeepsGlovesEquipped(t *testing.T) {
	store, _ := newTestStore()
	hero := createCharacter(t, store, [stats.NumAttributes]int{stats.STR: 14})
	gloves := createItem(t, store, "Leather Gloves", character_sheet.ItemTypeArmorHands, [stats.NumAttributes]int{})
	helmet := createItem(t, store, "Iron Helmet", character_sheet.ItemTypeArmorHead, [stats.NumAttributes]int{})
	setInventory(t, store, hero, entity.Inventory{
		Items:          []entity.GameEntityID{helmet},
		EquipmentSlots: map[string]entity.GameEntityID{character_sheet.SlotHands.Key(): gloves},
	})

	dd := character_sheet.NewDragDrop(store)
	dd.SetOwner(hero)
	dd.DragFromInventory(helmet, 0)
	dd.HoverSlot(character_sheet.SlotHands)
	if dd.RejectedSlot() != character_sheet.SlotHands {
		t.Fatalf("hovering the filled Hands slot with a helmet: rejected slot = %v, want Hands", dd.RejectedSlot())
	}
	dd.Drop()

	inv := inventory(t, store, hero)
	if !equipped(inv, character_sheet.SlotHands, gloves) {
		t.Errorf("gloves no longer equipped after the rejected drop: %v", inv.EquipmentSlots)
	}
	if !carries(inv, helmet) {
		t.Error("helmet did not return to the carried list")
	}
}

// AC-9: GIVEN a character with STR 14 equipping a ring that grants +2 STR,
// WHEN the ring is equipped, THEN STR displays as "14 (+2)" and derived
// stats (carry capacity) recalculate within one frame.
//
// Carry capacity: (14×10)+30 = 170 before, (16×10)+30 = 190 after. The
// recalculation happens synchronously inside Drop (the equip callback
// re-reads the store), so the updated values are visible in the same frame.
// Unequipping (right-click) reverts the display and the derived stats.
func TestEquipStrRingUpdatesDisplayAndDerivedStats(t *testing.T) {
	store, _ := newTestStore()
	attrs := [stats.NumAttributes]int{stats.STR: 14, stats.DEX: 12, stats.CON: 10, stats.WIS: 10}
	hero := createCharacter(t, store, attrs)
	ring := createItem(t, store, "Ring of Might", character_sheet.ItemTypeAccessory,
		[stats.NumAttributes]int{stats.STR: 2})
	setInventory(t, store, hero, entity.Inventory{
		Items:          []entity.GameEntityID{ring},
		EquipmentSlots: map[string]entity.GameEntityID{},
	})

	if c := derivedCarry(t, store, hero); c != 170 {
		t.Fatalf("pre-equip carry capacity = %d, want 170 (STR 14)", c)
	}

	dd := character_sheet.NewDragDrop(store)
	dd.SetOwner(hero)
	equips := 0
	carryDuringCallback := 0
	dd.OnEquip(func() {
		equips++
		carryDuringCallback = derivedCarry(t, store, hero)
	})

	dd.DragFromInventory(ring, 0)
	dd.HoverSlot(character_sheet.SlotAccessory1)
	if dd.RejectedSlot() != character_sheet.InvalidSlot {
		t.Fatalf("accessory must accept the ring; rejected slot = %v", dd.RejectedSlot())
	}
	dd.Drop()

	if equips != 1 {
		t.Errorf("equip callback fired %d times, want exactly 1", equips)
	}
	if carryDuringCallback != 190 {
		t.Errorf("carry capacity in the equip callback = %d, want 190 (effective STR 16: 16×10+30)", carryDuringCallback)
	}
	inv := inventory(t, store, hero)
	if !equipped(inv, character_sheet.SlotAccessory1, ring) {
		t.Errorf("Accessory 1 does not hold the ring: %v", inv.EquipmentSlots)
	}
	if carries(inv, ring) {
		t.Error("ring still in the carried list after equipping")
	}

	// Display format per AC-9: base 14 with the green "(+2)" bonus note.
	if got := character_sheet.AttributeDisplayWithBonus(14, 2); got != "14 (+2)" {
		t.Errorf("AttributeDisplayWithBonus(14, 2) = %q, want %q", got, "14 (+2)")
	}
	if got := character_sheet.BonusDisplay(2); got != "(+2)" {
		t.Errorf("BonusDisplay(2) = %q, want %q", got, "(+2)")
	}
	if got := character_sheet.AttributeBaseDisplay(14, 2); got != "14" {
		t.Errorf("AttributeBaseDisplay(14, 2) = %q, want %q (base value alone)", got, "14")
	}
	// No bonus: the standard "{value} ({sign}{modifier})" display stays.
	if got := character_sheet.AttributeDisplayWithBonus(14, 0); got != "14 (+2)" {
		t.Errorf("AttributeDisplayWithBonus(14, 0) = %q, want %q (modifier display, AC-1)", got, "14 (+2)")
	}

	// Unequipping the ring reverts the display and the derived stats.
	dd.RightClickUnequip(character_sheet.SlotAccessory1)
	inv = inventory(t, store, hero)
	if _, ok := inv.EquipmentSlots[character_sheet.SlotAccessory1.Key()]; ok {
		t.Errorf("Accessory 1 still occupied after unequip: %v", inv.EquipmentSlots)
	}
	if !carries(inv, ring) {
		t.Error("ring did not return to the carried list on unequip")
	}
	if c := derivedCarry(t, store, hero); c != 170 {
		t.Errorf("carry capacity after unequip = %d, want 170 (bonus reverted)", c)
	}
	if got := character_sheet.BonusDisplay(0); got != "" {
		t.Errorf("BonusDisplay(0) = %q, want empty", got)
	}
}

// AC-9 QA edge case: multiple attribute-changing items stack (ring +2 STR,
// belt +1 STR → "14 (+3)") and unequipping one reverts to the remaining
// bonus.
func TestStackingAttributeBonuses(t *testing.T) {
	store, _ := newTestStore()
	hero := createCharacter(t, store, [stats.NumAttributes]int{stats.STR: 14})
	ring := createItem(t, store, "Ring of Might", character_sheet.ItemTypeAccessory,
		[stats.NumAttributes]int{stats.STR: 2})
	belt := createItem(t, store, "Belt of Vigor", character_sheet.ItemTypeArmorChest,
		[stats.NumAttributes]int{stats.STR: 1})
	setInventory(t, store, hero, entity.Inventory{
		Items:          []entity.GameEntityID{ring, belt},
		EquipmentSlots: map[string]entity.GameEntityID{},
	})

	dd := character_sheet.NewDragDrop(store)
	dd.SetOwner(hero)
	dd.DragFromInventory(ring, 0)
	dd.HoverSlot(character_sheet.SlotAccessory1)
	dd.Drop()
	dd.DragFromInventory(belt, 0)
	dd.HoverSlot(character_sheet.SlotChest)
	dd.Drop()

	if got := character_sheet.AttributeDisplayWithBonus(14, 3); got != "14 (+3)" {
		t.Errorf("stacked bonuses display = %q, want %q", got, "14 (+3)")
	}
	if c := derivedCarry(t, store, hero); c != 200 {
		t.Errorf("carry capacity with stacked bonuses = %d, want 200 (effective STR 17: 17×10+30)", c)
	}

	// Unequipping the belt leaves only the ring's +2.
	dd.RightClickUnequip(character_sheet.SlotChest)
	if got := character_sheet.AttributeDisplayWithBonus(14, 2); got != "14 (+2)" {
		t.Errorf("display after unequipping the belt = %q, want %q", got, "14 (+2)")
	}
	if c := derivedCarry(t, store, hero); c != 190 {
		t.Errorf("carry capacity after unequipping the belt = %d, want 190 (effective STR 16)", c)
	}
}

// QA edge case: dragging from an equipment slot — dropping outside any slot
// cancels and the item stays equipped; dropping on a compatible slot moves
// the item (EquipItem clears the old slot). Right-click on an empty slot is
// a silent no-op.
func TestDragFromSlotCancelAndMove(t *testing.T) {
	store, _ := newTestStore()
	hero := createCharacter(t, store, [stats.NumAttributes]int{stats.STR: 14})
	sword := createItem(t, store, "Iron Sword", character_sheet.ItemTypeWeaponSword, [stats.NumAttributes]int{})
	setInventory(t, store, hero, entity.Inventory{
		Items:          []entity.GameEntityID{},
		EquipmentSlots: map[string]entity.GameEntityID{character_sheet.SlotMainHand.Key(): sword},
	})

	dd := character_sheet.NewDragDrop(store)
	dd.SetOwner(hero)

	// Cancel: drag off the slot, drop outside — nothing moves.
	dd.DragFromSlot(character_sheet.SlotMainHand)
	if !dd.Dragging() || dd.FromInventory() {
		t.Fatal("slot drag did not start (must not be an inventory drag)")
	}
	if dd.SourceSlot() != character_sheet.SlotMainHand || dd.DraggedItem() != sword {
		t.Fatalf("slot drag state: source = %v, item = %d (want Main Hand, %d)",
			dd.SourceSlot(), dd.DraggedItem(), sword)
	}
	dd.HoverNone()
	dd.Drop()
	if !equipped(inventory(t, store, hero), character_sheet.SlotMainHand, sword) {
		t.Error("cancelled slot drag must leave the item equipped")
	}

	// Move: Main Hand → Off Hand.
	dd.DragFromSlot(character_sheet.SlotMainHand)
	dd.HoverSlot(character_sheet.SlotOffHand)
	dd.Drop()
	inv := inventory(t, store, hero)
	if !equipped(inv, character_sheet.SlotOffHand, sword) {
		t.Errorf("sword not in Off Hand after the move: %v", inv.EquipmentSlots)
	}
	if _, ok := inv.EquipmentSlots[character_sheet.SlotMainHand.Key()]; ok {
		t.Error("Main Hand still occupied after the move")
	}

	// Dragging from an empty slot starts no drag; right-click on it is a
	// silent no-op.
	dd.DragFromSlot(character_sheet.SlotHead)
	if dd.Dragging() {
		t.Error("drag started from an empty slot")
	}
	dd.RightClickUnequip(character_sheet.SlotHead)
	if len(inventory(t, store, hero).EquipmentSlots) != 1 {
		t.Error("right-click on an empty slot changed the equipment")
	}
}
