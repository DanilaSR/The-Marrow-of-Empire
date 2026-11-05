package combat

import (
	"marrow-of-empire/entity"
)

// Story 004 (Positioning & Cover, ADR-0009): "Opportunity | Moving out of an
// enemy's melee range triggers free attack (once/turn/enemy)". Melee range is
// adjacency on the square grid: Chebyshev distance 1 (DefaultAttackRange
// 1.0 — 1 tile = 1 meter).

// OpportunityWatcher is notified after every successful movement step (Story
// 004). GridMover reports each step through this interface — the single
// choke point all combat movement flows through (the enemy AI today, player
// move commands later) — so free attacks trigger exactly when a combatant's
// position changes. The returned value is the number of attacks resolved.
type OpportunityWatcher interface {
	// CheckStep resolves the free attacks triggered by the mover's step from
	// tile from to tile to. Returns the number of attacks made.
	CheckStep(mover entity.GameEntityID, from, to Tile) int
}

// opportunityResetPriority is the EventBus priority of the OpportunityAttacker's
// turn-start subscription. No ordering constraint exists with other
// subscribers (the reset only touches internal state; nothing moves
// combatants during event dispatch), so a low priority is used.
const opportunityResetPriority = 100

// OpportunityAttacker implements OpportunityWatcher: a combatant leaving an
// enemy's melee range gives that enemy a free attack (ADR-0009) — resolved
// through the injected Attacker (which includes Story 004 positioning
// bonuses). Each enemy attacks at most once per turn ("once/turn/enemy");
// the per-turn state resets when EventTurnStarted fires, so the once-per-turn
// budget aligns with the combat manager's turn boundaries without any
// CombatManager changes.
//
// The rule is symmetric: any combatant — player, ally, or enemy — moving out
// of any enemy's melee range triggers that enemy's free attack, per the
// ADR/GDD wording ("moving out of an enemy's melee range triggers a free
// attack from that enemy").
//
// The resolved AttackResult is currently dropped (the enemy AI does the
// same — damage application and events are Story 005/006 territory).
type OpportunityAttacker struct {
	position Positioner
	attacker Attacker
	factions FactionReader
	// used tracks the enemies that already made an opportunity attack this
	// turn.
	used map[entity.GameEntityID]bool
}

// Compile-time assertion: OpportunityAttacker satisfies OpportunityWatcher.
var _ OpportunityWatcher = (*OpportunityAttacker)(nil)

// NewOpportunityAttacker creates the opportunity attacker and subscribes it
// to EventTurnStarted so the once-per-turn budget resets at every turn start
// (constructor subscription — the pattern audio.go and trust_system.go use).
// position supplies combatant tiles; attacker resolves the free attacks;
// factions supplies the mover's enemies. The bus must be the same bus wired
// into the combat manager.
//
// Example usage:
//
//	opportunity := combat.NewOpportunityAttacker(bus, positioner, attacker, factions)
//	mover := combat.NewGridMover(positioner, setter, blocker, pool, grid, opportunity)
func NewOpportunityAttacker(bus entity.EventBus, position Positioner, attacker Attacker, factions FactionReader) *OpportunityAttacker {
	o := &OpportunityAttacker{
		position: position,
		attacker: attacker,
		factions: factions,
		used:     make(map[entity.GameEntityID]bool),
	}
	bus.Subscribe(EventTurnStarted, opportunityResetPriority, func(entity.Event) {
		o.ResetTurn()
	})
	return o
}

// CheckStep implements OpportunityWatcher: for every enemy of the mover (per
// the FactionReader) that was within melee range (Chebyshev <= 1) of the
// mover's previous tile and is no longer within melee range of the new tile,
// the enemy attacks the mover — unless it already attacked this turn. Returns
// the number of attacks resolved. Enemies with unavailable positions are
// skipped (a positionless enemy cannot be judged in range — defensive).
func (o *OpportunityAttacker) CheckStep(mover entity.GameEntityID, from, to Tile) int {
	attacks := 0
	for _, enemy := range o.factions.EnemiesOf(mover) {
		if o.used[enemy] {
			continue // once per turn per enemy (ADR-0009)
		}
		pos, ok := o.position.Position(enemy)
		if !ok {
			continue // unknown position — cannot judge melee range (defensive)
		}
		enemyTile := WorldToTile(pos)
		if Chebyshev(enemyTile, from) <= 1 && Chebyshev(enemyTile, to) > 1 {
			// The mover left the enemy's melee range: the enemy takes its
			// free attack. The result is dropped — damage application and
			// death handling are Story 005/006.
			o.attacker.ResolveAttack(enemy, mover)
			o.used[enemy] = true
			attacks++
		}
	}
	return attacks
}

// ResetTurn clears the once-per-turn budget: every enemy may make one
// opportunity attack again. Called automatically at every EventTurnStarted;
// exposed for tests and for direct wiring without a bus.
func (o *OpportunityAttacker) ResetTurn() {
	clear(o.used)
}
