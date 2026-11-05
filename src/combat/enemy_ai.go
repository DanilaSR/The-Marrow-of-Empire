package combat

import (
	"math"

	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
)

// Positioner reads entity positions in world units. The Kaiju Transform is
// the authoritative position source (ADR-0001): combat never mutates
// transforms directly — movement goes through Mover. The game wiring supplies
// a Positioner backed by the Kaiju host; tests inject a stub.
type Positioner interface {
	// Position returns the entity's world position. Returns (zero, false)
	// for unknown or destroyed entities.
	Position(id entity.GameEntityID) (matrix.Vec3, bool)
}

// AttackResult is the outcome of an attack resolution. Story 003 (the d20
// resolution: accuracy vs. evasion, crits, damage formula) fills every
// field; Story 001 only needs Hit and Damage so the basic enemy AI can act.
type AttackResult struct {
	// Hit reports whether the attack connected.
	Hit bool
	// Damage is the damage dealt before death handling. Story 005 (death and
	// dying) owns the 0 HP transition; defeated combatants are removed via
	// CombatManager.RemoveCombatant.
	Damage int
	// Crit reports whether the attack was a critical hit — a natural 20,
	// doubling the weapon's base damage (ADR-0009, TR-combat-003).
	Crit bool
	// Roll is the d20 roll that resolved the attack, clamped to [1, 20]
	// (Story 003). Reported for the combat log and UI.
	Roll int
}

// Attacker resolves attacks between combatants. Story 003 (Attack Resolution,
// TR-combat-003) provides the d20 implementation (attack_resolution.go); the
// basic enemy AI calls this interface and never performs attack math itself.
type Attacker interface {
	// ResolveAttack resolves an attack from attacker against target and
	// returns the outcome.
	ResolveAttack(attacker, target entity.GameEntityID) AttackResult
}

// Mover moves a combatant toward a target. Story 002 (Grid Movement & AP)
// replaces this placeholder with tile-based, AP-costed movement; Story 001
// only needs "move toward the nearest target" (AC-6).
type Mover interface {
	// MoveToward moves combatant one step toward target. Returns an error
	// when no path exists; the enemy then ends its turn with no action
	// (AC-6 edge: "enemy has no path to any target").
	MoveToward(combatant, target entity.GameEntityID) error
}

// EnemyAI decides an enemy combatant's action when its turn begins. Story 001
// implements the basic behavior: attack the nearest valid target if within
// attack range, otherwise move toward it (AC-6). Sophisticated AI (flanking,
// low-HP targeting, threat assessment) is deferred to a future story (Out of
// Scope).
type EnemyAI interface {
	// TakeTurn performs the enemy's action given its valid targets — the
	// alive combatants hostile to it. The call is synchronous and completes
	// the enemy's action; the state machine advances afterwards. An empty
	// target list means the enemy takes no action and its turn ends (AC-6
	// edge: all player characters dead).
	TakeTurn(enemy entity.GameEntityID, validTargets []entity.GameEntityID)
}

// basicEnemyAI implements EnemyAI with the Story 001 behavior (AC-6): pick
// the valid target closest by world distance, resolve an attack when it is
// within attack range, otherwise move one step toward it. No action is taken
// when there are no targets or no path.
type basicEnemyAI struct {
	position    Positioner
	attacker    Attacker
	mover       Mover
	attackRange float64
}

// NewBasicEnemyAI creates the Story 001 enemy AI. An attackRange <= 0 falls
// back to DefaultAttackRange.
//
// Example usage:
//
//	ai := combat.NewBasicEnemyAI(positioner, attacker, mover, combat.DefaultAttackRange)
func NewBasicEnemyAI(position Positioner, attacker Attacker, mover Mover, attackRange float64) EnemyAI {
	if attackRange <= 0 {
		attackRange = DefaultAttackRange
	}
	return &basicEnemyAI{
		position:    position,
		attacker:    attacker,
		mover:       mover,
		attackRange: attackRange,
	}
}

// TakeTurn implements EnemyAI: attack the nearest valid target when in range,
// otherwise move toward it (AC-6). With no targets or no path, the enemy
// takes no action and its turn ends.
func (a *basicEnemyAI) TakeTurn(enemy entity.GameEntityID, validTargets []entity.GameEntityID) {
	target := a.nearestTarget(enemy, validTargets)
	if target == 0 {
		return // no valid action — the turn ends without acting (AC-6 edge)
	}
	if a.distanceTo(enemy, target) <= a.attackRange {
		a.attacker.ResolveAttack(enemy, target)
		return
	}
	if err := a.mover.MoveToward(enemy, target); err != nil {
		return // no path to the target — the turn ends without acting (AC-6 edge)
	}
}

// nearestTarget returns the valid target closest to the enemy; the first
// target in slice order wins ties (the manager passes targets in turn order,
// so ties are deterministic). Returns 0 when no targets exist.
func (a *basicEnemyAI) nearestTarget(enemy entity.GameEntityID, targets []entity.GameEntityID) entity.GameEntityID {
	var best entity.GameEntityID
	bestDist := math.MaxFloat64
	for _, t := range targets {
		if d := a.distanceTo(enemy, t); d < bestDist {
			best, bestDist = t, d
		}
	}
	return best
}

// distanceTo returns the world distance between two entities;
// math.MaxFloat64 when either position is unavailable (an unavailable
// position is never "in range").
func (a *basicEnemyAI) distanceTo(from, to entity.GameEntityID) float64 {
	fromPos, ok := a.position.Position(from)
	if !ok {
		return math.MaxFloat64
	}
	toPos, ok := a.position.Position(to)
	if !ok {
		return math.MaxFloat64
	}
	return float64(fromPos.Distance(toPos))
}
