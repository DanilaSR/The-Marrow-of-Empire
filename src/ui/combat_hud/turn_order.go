// TurnOrder is the engine-free state model behind the turn order bar.
//
// It tracks the combatant order, the current combatant, and the
// acted-this-round marks purely as data — no Kaiju imports — so the
// HUD's event wiring and state transitions are unit-testable without a
// rendering context (tests/integration/combat_ui/hud_framework_test.go).
// The hudRenderer translates this model into document operations.
//
// Event contract (src/combat/events.go): the combat system is the sole
// producer and guarantees ordering. CombatStarted carries the fixed turn
// order (descending initiative, rolled by rollInitiative); TurnStarted
// always follows CombatStarted synchronously; removals arrive as
// CombatantRemoved; the encounter ends with CombatEnded. This model
// assumes that contract and does not re-sort or re-derive anything.
package combat_hud

import (
	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
)

// TurnOrder is the turn order state for one encounter.
type TurnOrder struct {
	// order is the fixed encounter order minus removed combatants, in
	// descending initiative order as provided by the combat system.
	order []combat.TurnEntry
	// current is the combatant whose turn is active (0 when no turn is
	// active). A removed combatant keeps its current slot until the next
	// TurnStarted — the combat system completes the active turn before
	// removal takes effect.
	current entity.GameEntityID
	// acted marks combatants that have taken their turn this round. The
	// marks reset when the round increments (the turn order wraps).
	acted map[entity.GameEntityID]bool
	// active is true from BeginCombat until EndCombat.
	active bool
	// round is the current round number, 1-based (TurnStartedData.Round).
	round int
}

// NewTurnOrder creates an empty, inactive turn order.
func NewTurnOrder() *TurnOrder {
	return &TurnOrder{acted: make(map[entity.GameEntityID]bool)}
}

// BeginCombat resets all state and populates the order from a
// CombatStarted payload. The entries must already be in descending
// initiative order — the combat system rolls the order (combat.rollInitiative)
// and this model trusts it.
func (t *TurnOrder) BeginCombat(entries []combat.TurnEntry) {
	t.clear()
	t.order = append([]combat.TurnEntry(nil), entries...)
	t.active = true
	t.round = 1
}

// OnTurnStarted advances the current combatant to id and the round to
// round (TurnStartedData.Round). When the round increments — the turn
// order wrapped — every acted mark is cleared: "acted" means acted this
// round.
func (t *TurnOrder) OnTurnStarted(id entity.GameEntityID, round int) {
	if round != t.round {
		t.acted = make(map[entity.GameEntityID]bool)
		t.round = round
	}
	t.current = id
}

// OnTurnEnded marks the combatant as acted this round. The combat system
// guarantees TurnEnded precedes the next TurnStarted, so a combatant that
// just acted can never be current at the same time.
func (t *TurnOrder) OnTurnEnded(id entity.GameEntityID) {
	t.acted[id] = true
}

// Remove drops the combatant from the order (CombatantRemoved payload:
// defeated, fled, or surrendered). The current slot is untouched — the
// combat system completes the active turn and the next TurnStarted
// advances the highlight. Removing a combatant clears its acted mark.
func (t *TurnOrder) Remove(id entity.GameEntityID) {
	for i, e := range t.order {
		if e.EntityID == id {
			t.order = append(t.order[:i], t.order[i+1:]...)
			break
		}
	}
	delete(t.acted, id)
}

// EndCombat clears all encounter state (CombatEnded).
func (t *TurnOrder) EndCombat() { t.clear() }

// clear resets every field to the inactive zero state.
func (t *TurnOrder) clear() {
	t.order = nil
	t.current = 0
	t.acted = make(map[entity.GameEntityID]bool)
	t.active = false
	t.round = 0
}

// Active reports whether an encounter is running.
func (t *TurnOrder) Active() bool { return t.active }

// Entries returns a copy of the turn order — the combatants still in the
// encounter, in descending initiative order.
func (t *TurnOrder) Entries() []combat.TurnEntry {
	return append([]combat.TurnEntry(nil), t.order...)
}

// Current returns the combatant whose turn is active, or (0, false) when
// no turn is active.
func (t *TurnOrder) Current() (entity.GameEntityID, bool) {
	if t.current == 0 {
		return 0, false
	}
	return t.current, true
}

// Round returns the current round number, 1-based.
func (t *TurnOrder) Round() int { return t.round }

// IsActed reports whether the combatant has taken its turn this round.
func (t *TurnOrder) IsActed(id entity.GameEntityID) bool {
	return t.acted[id]
}
