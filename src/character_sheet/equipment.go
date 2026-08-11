// Equipment slots and item data model for Story 003 (Epic: Character &
// Inventory UI): the eight equipment slot definitions, the slot
// compatibility rules, the Item component carrying per-item data, and the
// equip/unequip operations that read and write the Inventory component on
// the viewed entity through the Entity Framework (ADR-0001).
//
// The eight slots come from the GDD equipment registry (Head, Chest, Hands,
// Legs, Main Hand, Off Hand, Accessory 1, Accessory 2 — tuning knob
// equipment_slots = 8). Equipment lives in the entity.Inventory component
// (EquipmentSlots map, keyed by SlotType.Key()); the EquipmentSlots array in
// this file is the UI-side read model of that map. EquipItem and
// UnequipSlot write the Inventory component back via entity.SetComponent,
// and the character sheet recomputes effective attributes and derived stats
// from the equipped items' Item components in the same frame (AC-6, AC-9).
//
// Item types follow the loot-economy GDD CR-1 record ("type"/"subtype")
// encoded as "type.subtype" strings (weapon.sword, armor.head, accessory).
// SlotCompatible maps item types to the slots that accept them — AC-7: a
// helmet (armor.head) does not fit Hands.
//
// Everything here is pure Go except the Item component registration (the
// stats.StatsComponent pattern: components defined outside the entity
// package register with entity.RegisterComponent at init), so the equip
// logic is unit- and integration-testable without the engine running.
package character_sheet

import (
	"fmt"
	"log/slog"
	"strings"

	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// SlotType identifies one of the eight equipment slots (GDD equipment
// registry; tuning knob equipment_slots = 8).
type SlotType int

const (
	// SlotHead is the head armor slot.
	SlotHead SlotType = iota
	// SlotChest is the chest armor slot.
	SlotChest
	// SlotHands is the hand armor (gloves) slot.
	SlotHands
	// SlotLegs is the leg armor slot.
	SlotLegs
	// SlotMainHand is the main-hand weapon slot.
	SlotMainHand
	// SlotOffHand is the off-hand weapon slot.
	SlotOffHand
	// SlotAccessory1 is the first accessory (ring/amulet) slot.
	SlotAccessory1
	// SlotAccessory2 is the second accessory (ring/amulet) slot.
	SlotAccessory2
	// NumSlots is the number of equipment slots (eight).
	NumSlots
)

// InvalidSlot marks "no slot": drag hover over no equipment slot, or a drag
// with no rejection state.
const InvalidSlot SlotType = -1

// slotKeys are the canonical slot key strings: the markup id suffix
// ("equip-{key}") and the entity.Inventory component EquipmentSlots map key
// (ADR-0001; the Inventory doc comment: slot names map to equipped item IDs).
var slotKeys = [NumSlots]string{
	"head", "chest", "hands", "legs",
	"mainhand", "offhand", "accessory1", "accessory2",
}

// SlotTypes is the canonical order of the eight equipment slots, parallel
// to slotKeys: iteration over the slots (populate, bonuses, drag) uses this.
var SlotTypes = [NumSlots]SlotType{
	SlotHead, SlotChest, SlotHands, SlotLegs,
	SlotMainHand, SlotOffHand, SlotAccessory1, SlotAccessory2,
}

// String returns the display name of the slot ("Main Hand", "Accessory 1",
// ...). Unknown values render as "SlotType(N)".
func (s SlotType) String() string {
	switch s {
	case SlotHead:
		return "Head"
	case SlotChest:
		return "Chest"
	case SlotHands:
		return "Hands"
	case SlotLegs:
		return "Legs"
	case SlotMainHand:
		return "Main Hand"
	case SlotOffHand:
		return "Off Hand"
	case SlotAccessory1:
		return "Accessory 1"
	case SlotAccessory2:
		return "Accessory 2"
	default:
		return fmt.Sprintf("SlotType(%d)", int(s))
	}
}

// Key returns the canonical slot key string ("mainhand", "accessory1", ...):
// the markup id suffix and the Inventory component EquipmentSlots map key.
// An invalid slot returns "".
func (s SlotType) Key() string {
	if !s.Valid() {
		return ""
	}
	return slotKeys[s]
}

// Valid reports whether s is one of the eight equipment slots.
func (s SlotType) Valid() bool { return s >= 0 && int(s) < int(NumSlots) }

// Item type strings (loot-economy GDD CR-1 type/subtype encoding). Only the
// equipment-relevant types are defined here; consumables, quest items, and
// crafting materials are never equipment (SlotCompatible rejects them).
const (
	// ItemTypeAccessory is an accessory (ring, amulet) — fits the two
	// Accessory slots.
	ItemTypeAccessory = "accessory"
	// ItemTypeWeaponPrefix prefixes weapon item types ("weapon.sword").
	ItemTypeWeaponPrefix = "weapon."
	// ItemTypeArmorPrefix prefixes armor item types ("armor.head").
	ItemTypeArmorPrefix = "armor."
	// ItemTypeWeaponSword is a sword (weapon subtype, GDD CR-1).
	ItemTypeWeaponSword = "weapon.sword"
	// ItemTypeWeaponAxe is an axe (weapon subtype, GDD CR-1).
	ItemTypeWeaponAxe = "weapon.axe"
	// ItemTypeWeaponDagger is a dagger (weapon subtype, GDD CR-1).
	ItemTypeWeaponDagger = "weapon.dagger"
	// ItemTypeWeaponBow is a bow (weapon subtype, GDD CR-1).
	ItemTypeWeaponBow = "weapon.bow"
	// ItemTypeWeaponSpear is a spear (weapon subtype, GDD CR-1).
	ItemTypeWeaponSpear = "weapon.spear"
	// ItemTypeArmorHead is head armor (armor subtype, GDD CR-1).
	ItemTypeArmorHead = "armor.head"
	// ItemTypeArmorChest is chest armor (armor subtype, GDD CR-1).
	ItemTypeArmorChest = "armor.chest"
	// ItemTypeArmorHands is hand armor — gloves (armor subtype, GDD CR-1).
	ItemTypeArmorHands = "armor.hands"
	// ItemTypeArmorLegs is leg armor (armor subtype, GDD CR-1).
	ItemTypeArmorLegs = "armor.legs"
	// ItemTypeConsumablePrefix prefixes consumable item types
	// ("consumable.potion").
	ItemTypeConsumablePrefix = "consumable."
	// ItemTypeQuestPrefix prefixes quest item types ("quest.key").
	ItemTypeQuestPrefix = "quest."
	// ItemTypeCraftingPrefix prefixes crafting material types
	// ("crafting.ore").
	ItemTypeCraftingPrefix = "crafting."
)

// SlotCompatible reports whether an item of the given type fits the slot:
// each armor subtype fits its named slot, weapons fit Main Hand and Off
// Hand (the GDD defines no two-handed restriction or shield subtype yet —
// a later loot story may restrict the off hand), accessories fit the two
// Accessory slots, and everything else fits nothing (AC-7: armor.head does
// not fit Hands).
func SlotCompatible(itemType string, slot SlotType) bool {
	switch slot {
	case SlotHead:
		return itemType == ItemTypeArmorHead
	case SlotChest:
		return itemType == ItemTypeArmorChest
	case SlotHands:
		return itemType == ItemTypeArmorHands
	case SlotLegs:
		return itemType == ItemTypeArmorLegs
	case SlotMainHand, SlotOffHand:
		return strings.HasPrefix(itemType, ItemTypeWeaponPrefix)
	case SlotAccessory1, SlotAccessory2:
		return itemType == ItemTypeAccessory
	default:
		return false
	}
}

// Item is the equipment item data model for the character sheet (Story
// 003): the minimal per-item data the equip logic and the display need —
// the item type string (SlotCompatible), the attribute bonuses applied
// while equipped (AC-9), and the icon reference (pen-and-ink icon asset,
// ADR-0012). Story 005 extends it with the inventory display fields
// (Description, Weight, Value, Quantity) that the weight bar, stack
// counts, and item detail panel read.
//
// Items are entities (EntityTypeItem) referenced by GameEntityID from the
// entity.Inventory component (ADR-0001, GDD CR-2 #3). The component
// registers with the Entity Framework here (the stats.StatsComponent
// pattern — entity.RegisterComponent at init), because the Inventory/Loot
// system that owns the full loot-economy item record does not exist yet;
// when it lands, that record's type/subtype/stats fields should drive this
// component.
type Item struct {
	// Name is the display name ("Steel Broadsword").
	Name string
	// Type is the item type string ("weapon.sword", "armor.head",
	// "accessory", ...).
	Type string
	// IconRef is the pen-and-ink icon asset reference; empty until icon
	// assets exist.
	IconRef string
	// Description is the item detail panel body text (Story 005); empty
	// until the loot-economy record lands.
	Description string
	// Weight is the item's weight in whole pounds (GDD CR-1: weight in
	// lbs, contributes to carry capacity). Fractional weights arrive with
	// the full loot-economy record; the Story 005 weight bar sums
	// Weight × Quantity per item.
	Weight int
	// Value is the item's base gold value (GDD CR-1: value, before
	// faction/CHA modifiers).
	Value int
	// Quantity is the number of copies carried in one inventory cell (GDD
	// CR-1: stack_size — 1 for equipment, >1 for consumables and
	// materials). The grid cell shows the stack count for Quantity > 1
	// (Story 005) and the weight bar counts Quantity copies.
	Quantity int
	// AttributeBonuses are the attribute bonuses applied while the item is
	// equipped, indexed by stats attribute (STR=0 ... CHA=5): a +2 STR ring
	// sets AttributeBonuses[stats.STR] = 2 (AC-9).
	AttributeBonuses [stats.NumAttributes]int
}

func init() {
	entity.RegisterComponent[Item]()
}

// EquipmentSlots is the read model of the eight equipment slots of one
// entity (GDD tuning knob equipment_slots = 8): each slot holds the
// GameEntityID of the equipped item, or 0 when empty. It is a fixed-size
// array indexed by SlotType — no map, no allocation.
//
// The authoritative equipment state lives in the entity.Inventory component
// (EquipmentSlots map keyed by SlotType.Key()); ReadFromInventory
// materializes it here, and EquipItem/UnequipSlot write it back through
// WriteToInventory.
type EquipmentSlots [NumSlots]entity.GameEntityID

// Item returns the equipped item ID in the slot; 0 when the slot is empty.
// Invalid slots return 0.
func (slots EquipmentSlots) Item(slot SlotType) entity.GameEntityID {
	if !slot.Valid() {
		return 0
	}
	return slots[slot]
}

// Set equips itemID into the slot. Invalid slots are ignored.
func (slots *EquipmentSlots) Set(slot SlotType, itemID entity.GameEntityID) {
	if slot.Valid() {
		slots[slot] = itemID
	}
}

// Clear empties the slot. Invalid slots are ignored.
func (slots *EquipmentSlots) Clear(slot SlotType) {
	if slot.Valid() {
		slots[slot] = 0
	}
}

// SlotHolding returns the slot that currently holds itemID, and false when
// the item is not equipped in any slot.
func (slots EquipmentSlots) SlotHolding(itemID entity.GameEntityID) (SlotType, bool) {
	if itemID == 0 {
		return InvalidSlot, false
	}
	for i, id := range slots {
		if id == itemID {
			return SlotType(i), true
		}
	}
	return InvalidSlot, false
}

// ReadFromInventory populates the slots from an Inventory component's
// EquipmentSlots map; unknown slot keys are ignored and slots absent from
// the map end up empty.
func (slots *EquipmentSlots) ReadFromInventory(inv entity.Inventory) {
	for i, slot := range SlotTypes {
		slots[i] = inv.EquipmentSlots[slot.Key()]
	}
}

// WriteToInventory writes the slots into an Inventory component's
// EquipmentSlots map, rebuilding the map wholesale from the array so slots
// that were cleared drop out of the component.
func (slots EquipmentSlots) WriteToInventory(inv *entity.Inventory) {
	inv.EquipmentSlots = make(map[string]entity.GameEntityID, NumSlots)
	for _, slot := range SlotTypes {
		if id := slots.Item(slot); id != 0 {
			inv.EquipmentSlots[slot.Key()] = id
		}
	}
}

// LoadEquipment reads the Inventory component of the owner entity and
// returns its equipment slots. The second return is false when the entity
// has no Inventory component.
func LoadEquipment(store entity.EntityStore, owner entity.GameEntityID) (EquipmentSlots, bool) {
	var slots EquipmentSlots
	inv, ok := entity.GetComponent[entity.Inventory](store, owner)
	if !ok {
		return slots, false
	}
	slots.ReadFromInventory(inv)
	return slots, true
}

// EquipItem equips the item entity into the slot and writes the change to
// the owner's Inventory component (ADR-0001: equipment changes are
// persisted in the component). It returns an error without writing when the
// owner has no Inventory component, the item has no Item component, or the
// item type is incompatible with the slot (AC-7 — the caller shows the grey
// X and the item stays in inventory).
//
// Occupied-slot semantics (AC-6 QA edge case): the occupant moves to the
// first empty inventory slot. An item already equipped elsewhere is moved
// out of its old slot first (slot-to-slot drags); equipping the item
// already in the slot is a no-op.
func EquipItem(store entity.EntityStore, owner, itemID entity.GameEntityID, slot SlotType) error {
	if !slot.Valid() {
		return fmt.Errorf("character_sheet: invalid equipment slot %d", int(slot))
	}
	inv, ok := entity.GetComponent[entity.Inventory](store, owner)
	if !ok {
		return fmt.Errorf("character_sheet: entity %d has no Inventory component", owner)
	}
	item, ok := entity.GetComponent[Item](store, itemID)
	if !ok {
		return fmt.Errorf("character_sheet: item %d has no Item component", itemID)
	}
	if !SlotCompatible(item.Type, slot) {
		return fmt.Errorf("character_sheet: item %q (type %q) does not fit slot %s",
			item.Name, item.Type, slot)
	}
	var slots EquipmentSlots
	slots.ReadFromInventory(inv)
	if slots.Item(slot) == itemID {
		return nil // already equipped there — no write
	}
	// An item already equipped elsewhere moves out of its old slot.
	if cur, ok := slots.SlotHolding(itemID); ok {
		slots.Clear(cur)
	}
	// The slot's occupant (if any) moves to the first empty inventory slot.
	if occupant := slots.Item(slot); occupant != 0 {
		slots.Clear(slot)
		moveToInventory(&inv, occupant)
	}
	// The item leaves the carried list (it was dragged from inventory) and
	// enters the slot.
	slots.Set(slot, itemID)
	removeFromInventory(&inv, itemID)
	slots.WriteToInventory(&inv)
	if err := entity.SetComponent(store, owner, inv); err != nil {
		return fmt.Errorf("character_sheet: failed to write Inventory for entity %d: %w", owner, err)
	}
	return nil
}

// UnequipSlot moves the equipped item out of the slot into the first empty
// inventory slot and writes the change to the Inventory component
// (right-click unequip, GDD: "Right-click equipped item to unequip to first
// empty inventory slot"). It returns an error without writing when the
// owner has no Inventory component or the slot is empty. The carried list
// has no capacity cap in this story (the GDD slot-cap question is open), so
// an unequip always finds a position.
func UnequipSlot(store entity.EntityStore, owner entity.GameEntityID, slot SlotType) error {
	if !slot.Valid() {
		return fmt.Errorf("character_sheet: invalid equipment slot %d", int(slot))
	}
	inv, ok := entity.GetComponent[entity.Inventory](store, owner)
	if !ok {
		return fmt.Errorf("character_sheet: entity %d has no Inventory component", owner)
	}
	var slots EquipmentSlots
	slots.ReadFromInventory(inv)
	itemID := slots.Item(slot)
	if itemID == 0 {
		return fmt.Errorf("character_sheet: slot %s is empty", slot)
	}
	slots.Clear(slot)
	moveToInventory(&inv, itemID)
	slots.WriteToInventory(&inv)
	if err := entity.SetComponent(store, owner, inv); err != nil {
		return fmt.Errorf("character_sheet: failed to write Inventory for entity %d: %w", owner, err)
	}
	return nil
}

// TotalAttributeBonuses sums the attribute bonuses of every equipped item,
// indexed by stats attribute (AC-9: a +2 STR ring contributes +2 to
// stats.STR). Items without an Item component are skipped with a warning.
func TotalAttributeBonuses(store entity.EntityStore, slots EquipmentSlots) [stats.NumAttributes]int {
	var bonuses [stats.NumAttributes]int
	for _, slot := range SlotTypes {
		id := slots.Item(slot)
		if id == 0 {
			continue
		}
		item, ok := entity.GetComponent[Item](store, id)
		if !ok {
			slog.Warn("character sheet: equipped item has no Item component",
				"item", id, "slot", slot)
			continue
		}
		for i := 0; i < stats.NumAttributes; i++ {
			bonuses[i] += item.AttributeBonuses[i]
		}
	}
	return bonuses
}

// EffectiveAttributes returns a copy of the Stats component with the
// equipment attribute bonuses applied to the base attributes (AC-9: base
// STR 14 + ring +2 → effective STR 16). The base component in the store is
// never mutated — the bonus is a display/derived-stat concern; the combat
// system reads item bonuses from the Inventory component independently
// (GDD: Interactions with Other Systems).
func EffectiveAttributes(base stats.StatsComponent, bonuses [stats.NumAttributes]int) stats.StatsComponent {
	eff := base
	for i := 0; i < stats.NumAttributes; i++ {
		eff.Attributes[i] += bonuses[i]
	}
	return eff
}

// BonusDisplay renders the attribute bonus note per AC-9: "(+2)", "(−1)"
// (proper U+2212 minus), and "" for a zero bonus. The sheet shows the note
// in green ink next to the base value; the sign always shows so the note is
// never hue-only (ADR-0012).
func BonusDisplay(bonus int) string {
	if bonus == 0 {
		return ""
	}
	return fmt.Sprintf("(%s)", ModifierDisplay(bonus))
}

// AttributeBaseDisplay renders the attribute's main value label: the base
// value alone when an equipment bonus is present ("14" — the bonus note is
// the separate green label), or the standard "{value} ({sign}{modifier})"
// display when there is no bonus ("16 (+3)", AC-1).
func AttributeBaseDisplay(base, bonus int) string {
	if bonus != 0 {
		return fmt.Sprintf("%d", base)
	}
	return AttributeDisplay(base)
}

// AttributeDisplayWithBonus renders the full attribute display with the
// equipment bonus note appended per AC-9: base STR 14 with a +2 ring shows
// "14 (+2)". With no bonus it matches AttributeDisplay ("16 (+3)").
func AttributeDisplayWithBonus(base, bonus int) string {
	if bonus == 0 {
		return AttributeDisplay(base)
	}
	return fmt.Sprintf("%d %s", base, BonusDisplay(bonus))
}

// removeFromInventory removes the item from the carried list, compacting
// the list (the list never holds zero IDs after this story's operations).
func removeFromInventory(inv *entity.Inventory, itemID entity.GameEntityID) {
	for i, id := range inv.Items {
		if id == itemID {
			inv.Items = append(inv.Items[:i], inv.Items[i+1:]...)
			return
		}
	}
}

// moveToInventory places the item in the first empty inventory position (a
// zero-ID hole left by a removal; the list is compact in this story, so
// this appends). The item is removed from the list first when it is already
// present — defensive: an equipped item should not also be carried.
func moveToInventory(inv *entity.Inventory, itemID entity.GameEntityID) {
	removeFromInventory(inv, itemID)
	for i, id := range inv.Items {
		if id == 0 {
			inv.Items[i] = itemID
			return
		}
	}
	inv.Items = append(inv.Items, itemID)
}
