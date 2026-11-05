package combat

import (
	"errors"

	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// Action point costs and turn budget per ADR-0009 and TR-combat-002 (Feature
// Layer control manifest: "AP system: 4 + DEX_mod (min 2, max 8), costs:
// Move 1, Attack 2, Ability 1-3, Interact 1"). The ADR fixes these values;
// a future balance story can move them into combat.Config (data-driven
// pattern: config.go).
const (
	// BaseAP is the AP every combatant receives before the DEX modifier.
	BaseAP = 4
	// MinAP is the per-turn AP floor (DEX 6 and below).
	MinAP = 2
	// MaxAP is the per-turn AP ceiling (DEX 18, mod +4).
	MaxAP = 8
	// MoveCost is the AP cost of moving one tile — orthogonal or diagonal
	// (ADR-0009, AC-1).
	MoveCost = 1
	// ClimbCost is the AP cost of a move step that ascends one elevation
	// level (Story 004, ADR-0009: "Climbing 1 level: 2 AP"). GridMover
	// charges MoveCost + (ClimbCost - MoveCost) per level climbed — a
	// one-level climb costs 2 AP total. Descending and level steps cost
	// MoveCost.
	ClimbCost = 2
	// AttackCost is the AP cost of a standard attack (Story 003 resolves the
	// attack itself; this story only defines the cost, AC-5).
	AttackCost = 2
	// StabilizeCost is the AP cost of stabilizing an adjacent dying ally
	// (Story 005, ADR-0009: "Stabilize dying ally | 2 (requires adjacency)").
	StabilizeCost = 2
	// AbilityCostMin is the minimum AP cost of an ability.
	AbilityCostMin = 1
	// AbilityCostMax is the maximum AP cost of an ability.
	AbilityCostMax = 3
	// InteractCost is the AP cost of interacting with an object (door,
	// lever, item).
	InteractCost = 1
	// DefendCost is the AP cost of the Defend action (Combat UI epic,
	// Story 002 action table; the defend effect itself is a later combat
	// story).
	DefendCost = 1
)

// APFor returns a combatant's per-turn AP: 4 + DEX_mod, clamped to [2, 8]
// (ADR-0006, TR-combat-002). DEX is read from the Character Stats component
// (entity.Stats.Dexterity) and the modifier uses the BG1-compatible formula
// floor((DEX-10)/2) clamped to [-4, +4] (stats.Modifier).
//
// Examples: DEX 16 → 7 AP, DEX 10 → 4 AP, DEX 6 → 2 AP (floor), DEX 18 → 8
// AP (ceiling).
func APFor(s entity.Stats) int {
	ap := BaseAP + stats.Modifier(s.Dexterity)
	if ap < MinAP {
		return MinAP
	}
	if ap > MaxAP {
		return MaxAP
	}
	return ap
}

// Sentinel errors returned by APPool.Spend.
var (
	// ErrInsufficientAP is returned when the pool cannot cover the cost.
	// The pool is left unchanged.
	ErrInsufficientAP = errors.New("combat: insufficient action points")
	// ErrInvalidAPCost is returned for a non-positive cost.
	ErrInvalidAPCost = errors.New("combat: invalid action point cost")
)

// APPool is a combatant's action point balance for the current turn
// (TR-combat-002, ADR-0009). The CombatManager resets it at every turn start
// — AP never carries over — and clears it when combat finalizes. Actions
// spend from it: movement (GridMover, 1 AP per tile), attacks (Story 003,
// 2 AP), abilities (1-3 AP), and interactions (1 AP). The same pool is
// shared between the manager and the action systems through dependency
// injection; it is never shared between combatants.
//
// Not safe for concurrent use — the game drives combat from a single thread
// (same contract as CombatManager).
type APPool struct {
	current int
	max     int
}

// NewAPPool creates an empty pool; Reset sets the turn budget.
func NewAPPool() *APPool {
	return &APPool{}
}

// Reset refills the pool for a new turn with the given budget. Negative
// budgets clamp to 0. AP does not carry over between turns — every turn
// start calls Reset (TR-combat-002: "Unused AP is lost at end of turn").
func (p *APPool) Reset(max int) {
	if max < 0 {
		max = 0
	}
	p.max = max
	p.current = max
}

// Current returns the remaining AP.
func (p *APPool) Current() int {
	return p.current
}

// Max returns the turn budget set by the most recent Reset.
func (p *APPool) Max() int {
	return p.max
}

// Spend consumes cost AP from the pool. Returns ErrInsufficientAP when the
// pool cannot cover the cost (the pool is left unchanged) and
// ErrInvalidAPCost for a non-positive cost.
func (p *APPool) Spend(cost int) error {
	if cost <= 0 {
		return ErrInvalidAPCost
	}
	if cost > p.current {
		return ErrInsufficientAP
	}
	p.current -= cost
	return nil
}

// Empty reports whether the pool is fully spent (Current() == 0). A turn
// whose pool is empty has no actions left; the turn ends (enemy turns end
// after the AI acts; the input layer ends player turns via EndTurn).
func (p *APPool) Empty() bool {
	return p.current == 0
}
