// Action bar model (Story 002, AC-2/AC-3/AC-4): the engine-free state
// behind the bottom-center ability menu and the bottom-right End Turn
// button. The hudRenderer (action_render.go) translates it into document
// operations per frame.
//
// The six ability slots are the Story 002 action table (hotkey, AP cost);
// costs reference the combat package constants so a balance change cannot
// drift from the displayed values. The menu and End Turn are visible only
// during player-controlled turns (AC-2); slots grey out when the active
// turn's AP cannot cover their cost (AC-3); selection toggles on click or
// hotkey, one ability at a time, re-selecting deselects (AC-4). The End
// Turn action carries a 300ms double-trigger guard.
//
// Targeting (Story 005) and the select sound (AUDIO-004) consume the
// selection through the EventAbilitySelected event the HUD emits on the
// bus — this model only owns the state and the rules.
package combat_hud

import (
	"fmt"
	"strconv"

	"marrow-of-empire/combat"
)

// endTurnLockoutDuration is the End Turn double-trigger guard (Story 002
// implementation notes: a brief disable window after click, 300ms).
const endTurnLockoutDuration = 0.3 // seconds

// AbilitySlot is one action-bar entry: its hotkey (1-6), its AP cost, and
// its display name. The Abilities slot is a submenu whose contents vary
// between AbilityCostMin and AbilityCostMax AP; it greys out when the
// player cannot afford the cheapest ability.
type AbilitySlot struct {
	// Hotkey is the 1-6 binding (AC-2).
	Hotkey int
	// Name is the player-facing action name (Move, Attack, ...).
	// TODO(localization): user-facing string — route through the
	// localization system when it lands (ui-code rules).
	Name string
	// Cost is the affordability threshold: the ability cannot be selected
	// when the active turn's AP is below it (AC-3). The submenu slot uses
	// the cheapest ability's cost.
	Cost int
	// Submenu marks the Abilities slot: its displayed cost is the range
	// AbilityCostMin-AbilityCostMax rather than a single value.
	Submenu bool
}

// CostLabel renders the displayed AP cost in combat face: the submenu
// slot shows its range, the others their fixed cost.
func (s AbilitySlot) CostLabel() string {
	if s.Submenu {
		return fmt.Sprintf("%d-%d", combat.AbilityCostMin, combat.AbilityCostMax)
	}
	return strconv.Itoa(s.Cost)
}

// defaultAbilitySlots is the Story 002 action table (AC-2): Move [1] 1 AP,
// Attack [2] 2 AP, Abilities [3] 1-3 AP, Items [4] 1 AP, Defend [5] 1 AP,
// Wait [6] 0 AP. Costs come from the combat package's constants (single
// source of truth); a future balance story can move the table into
// combat.Config (data-driven pattern).
var defaultAbilitySlots = []AbilitySlot{
	{Hotkey: 1, Name: "Move", Cost: combat.MoveCost},
	{Hotkey: 2, Name: "Attack", Cost: combat.AttackCost},
	{Hotkey: 3, Name: "Abilities", Cost: combat.AbilityCostMin, Submenu: true},
	{Hotkey: 4, Name: "Items", Cost: combat.InteractCost},
	{Hotkey: 5, Name: "Defend", Cost: combat.DefendCost},
	{Hotkey: 6, Name: "Wait", Cost: 0},
}

// ActionBar holds the action bar state: visibility (player turn), the
// active turn's AP for affordability, the selection, and the End Turn
// lockout. It is not safe for concurrent use (same contract as
// CombatHUD).
type ActionBar struct {
	// active is true while a player-controlled turn runs (SidePlayer):
	// the menu and End Turn are visible and interactive (AC-2).
	active bool
	// ap is the active turn's current AP (read from the manager at turn
	// start, updated via SetAP when AP spending lands).
	ap int
	// selected is the selected slot's hotkey (1-6), 0 when none.
	selected int
	// endTurnLockout is the remaining double-trigger guard time; > 0
	// blocks further End Turn presses.
	endTurnLockout float64
}

// NewActionBar creates an inactive action bar (menu hidden, nothing
// selected).
func NewActionBar() *ActionBar { return &ActionBar{} }

// Slots returns the six ability slots in hotkey order.
func (b *ActionBar) Slots() []AbilitySlot { return defaultAbilitySlots }

// OnTurnStarted applies a turn change: the menu shows for player
// turns only (AC-2), the AP budget refreshes, and any selection from the
// previous turn clears (it belonged to the previous combatant).
func (b *ActionBar) OnTurnStarted(side combat.Side, ap int) {
	b.active = side == combat.SidePlayer
	b.ap = ap
	b.selected = 0
}

// OnTurnEnded hides the menu and clears the selection for the turn
// change.
func (b *ActionBar) OnTurnEnded() {
	b.active = false
	b.selected = 0
}

// Reset clears all state (combat start and combat end).
func (b *ActionBar) Reset() {
	b.active = false
	b.ap = 0
	b.selected = 0
	b.endTurnLockout = 0
}

// Active reports whether the action bar is shown (player turn).
func (b *ActionBar) Active() bool { return b.active }

// Selected returns the selected slot's hotkey, 0 when none (AC-4).
func (b *ActionBar) Selected() int { return b.selected }

// SetAP updates the active turn's AP for affordability (AC-1/AC-3). This
// is the hook the future EventAPChanged wiring calls; no live caller
// exists yet (see ap_meter.go).
func (b *ActionBar) SetAP(ap int) { b.ap = ap }

// CanAfford reports whether the ability with the given hotkey is
// selectable with the active turn's AP (AC-3: cost <= current AP).
// Unknown hotkeys and out-of-turn requests are not affordable.
func (b *ActionBar) CanAfford(hotkey int) bool {
	if !b.active {
		return false
	}
	slot, ok := b.slot(hotkey)
	if !ok {
		return false
	}
	return slot.Cost <= b.ap
}

// Select toggles the ability with the given hotkey (AC-4): selecting the
// already-selected ability deselects it, selecting another moves the
// selection, and only one ability is selected at a time. Returns the
// newly selected hotkey (0 = deselected) and whether the request was
// honored. Requests are ignored — (0, false) — outside a player turn,
// for unknown hotkeys, and for unaffordable abilities (AC-3: disabled
// abilities do not respond to click or hotkey).
func (b *ActionBar) Select(hotkey int) (int, bool) {
	if !b.active {
		return 0, false
	}
	if _, ok := b.slot(hotkey); !ok {
		return 0, false
	}
	if !b.CanAfford(hotkey) {
		return 0, false
	}
	if b.selected == hotkey {
		b.selected = 0
		return 0, true
	}
	b.selected = hotkey
	return hotkey, true
}

// TryEndTurn attempts the End Turn action (AC-4): allowed only during a
// player turn and outside the double-trigger guard. On success the 300ms
// lockout starts and true is returned — the caller (the HUD) ends the
// turn via combat.Manager.EndTurn.
func (b *ActionBar) TryEndTurn() bool {
	if !b.active {
		return false
	}
	if b.endTurnLockout > 0 {
		return false
	}
	b.endTurnLockout = endTurnLockoutDuration
	return true
}

// EndTurnLocked reports whether the double-trigger guard is active.
func (b *ActionBar) EndTurnLocked() bool { return b.endTurnLockout > 0 }

// Update decrements the End Turn lockout by dt seconds.
func (b *ActionBar) Update(dt float64) {
	if b.endTurnLockout > 0 {
		b.endTurnLockout -= dt
		if b.endTurnLockout < 0 {
			b.endTurnLockout = 0
		}
	}
}

// slot returns the slot with the given hotkey, or false.
func (b *ActionBar) slot(hotkey int) (AbilitySlot, bool) {
	for _, s := range defaultAbilitySlots {
		if s.Hotkey == hotkey {
			return s, true
		}
	}
	return AbilitySlot{}, false
}
