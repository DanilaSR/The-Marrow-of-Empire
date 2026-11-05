package combat

import (
	"marrow-of-empire/entity"
)

// FactionReader reports which entities fight alongside or against a combatant
// (Story 004). Positioning reads it for two purposes: flanking needs the
// attacker's allies (an adjacent ally on the opposite side), and opportunity
// attacks need the mover's enemies (an enemy whose melee range was left). The
// game wiring implements it — NewManagerFactions derives it from the combat
// manager's turn order; a future faction system may replace the derivation.
type FactionReader interface {
	// AlliesOf returns the entities fighting alongside the combatant
	// (excluding the combatant itself).
	AlliesOf(id entity.GameEntityID) []entity.GameEntityID
	// EnemiesOf returns the entities hostile to the combatant.
	EnemiesOf(id entity.GameEntityID) []entity.GameEntityID
}

// SideSource exposes the encounter's turn order so faction relationships can
// be derived from combatant sides. *CombatManager satisfies it.
type SideSource interface {
	// TurnOrder returns the combatants still in the encounter, in initiative
	// order.
	TurnOrder() []TurnEntry
}

// managerFactions implements FactionReader over a SideSource's turn order.
// Faction relationships use the camp model: players and allies share a camp,
// enemies are hostile to both. Combatants with 0 HP (or a missing Health
// component — defensive) are excluded from both ally and enemy lists.
type managerFactions struct {
	store entity.EntityStore
	sides SideSource
}

// Compile-time assertion: managerFactions satisfies FactionReader.
var _ FactionReader = (*managerFactions)(nil)

// NewManagerFactions creates a FactionReader derived from the combat
// manager's fixed turn order (sides assigned at StartCombat): an entity's
// allies are the alive combatants of its camp, its enemies the alive
// combatants of the other camp. The store is required for the alive check
// (Health.Current > 0). A combatant absent from the turn order has no faction
// relationships at all (fail-closed: it can neither flank nor trigger
// opportunity attacks).
//
// Example usage:
//
//	positioning := combat.NewGridPositioning(positioner, grid, combat.NewManagerFactions(store, manager))
func NewManagerFactions(store entity.EntityStore, sides SideSource) FactionReader {
	return &managerFactions{store: store, sides: sides}
}

// AlliesOf implements FactionReader: the alive combatants whose side shares
// the combatant's camp (players and allies are friendly to each other;
// enemies are friendly only to enemies), excluding the combatant itself.
func (f *managerFactions) AlliesOf(id entity.GameEntityID) []entity.GameEntityID {
	order := f.sides.TurnOrder()
	side, ok := sideOf(order, id)
	if !ok {
		return nil // not a combatant — no faction relationships (fail-closed)
	}
	var allies []entity.GameEntityID
	for _, e := range order {
		if e.EntityID == id || !sameCamp(e.Side, side) || !f.alive(e.EntityID) {
			continue
		}
		allies = append(allies, e.EntityID)
	}
	return allies
}

// EnemiesOf implements FactionReader: the alive combatants whose side is not
// in the combatant's camp.
func (f *managerFactions) EnemiesOf(id entity.GameEntityID) []entity.GameEntityID {
	order := f.sides.TurnOrder()
	side, ok := sideOf(order, id)
	if !ok {
		return nil // not a combatant — no faction relationships (fail-closed)
	}
	var enemies []entity.GameEntityID
	for _, e := range order {
		if sameCamp(e.Side, side) || !f.alive(e.EntityID) {
			continue
		}
		enemies = append(enemies, e.EntityID)
	}
	return enemies
}

// alive reports whether the combatant has Health.Current > 0. A missing
// Health component counts as not alive (defensive — StartCombat validates
// Health for every combatant).
func (f *managerFactions) alive(id entity.GameEntityID) bool {
	hp, ok := entity.GetComponent[entity.Health](f.store, id)
	return ok && hp.Current > 0
}

// sideOf returns the combatant's side in the turn order. Unknown combatants
// are reported as absent (ok == false).
func sideOf(order []TurnEntry, id entity.GameEntityID) (Side, bool) {
	for _, e := range order {
		if e.EntityID == id {
			return e.Side, true
		}
	}
	return SideEnemy, false
}

// sameCamp reports whether two sides fight alongside each other: players and
// allies share a camp, enemies are hostile to both.
func sameCamp(a, b Side) bool {
	if a == SideEnemy || b == SideEnemy {
		return a == b // both enemies
	}
	return true // player and ally sides are friendly
}
