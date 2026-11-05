package combat

// Config carries the Story 001 tuning knobs for the combat state machine and
// the basic enemy AI. Zero-valued fields fall back to the GDD/ADR defaults
// (pattern: entity.EntityStoreConfig — gameplay values stay data-driven; a
// later story wires configs to external config files).
type Config struct {
	// AttackRange is the distance in world units at which an enemy resolves
	// an attack instead of moving (GDD CR-3: 1 tile = 1 meter). Default 1.0
	// (adjacent tile).
	AttackRange float64
	// MaxCombatants caps the encounter size (QA performance case: 8 friendly
	// + 8 enemy = 16 total). StartCombat rejects larger encounters. Default
	// 16.
	MaxCombatants int
	// DyingRounds is the death countdown (Story 005, GDD tuning knob
	// `dying_rounds`): the number of the dying character's own turn ends
	// before permanent death without stabilization. Default 3.
	DyingRounds int
}

// Default tuning values from the Tactical Combat Engine GDD and ADR-0009.
const (
	// DefaultAttackRange is the melee reach in world units: 1 tile = 1 meter
	// (GDD CR-3; opportunity attacks trigger at melee range).
	DefaultAttackRange float64 = 1.0
	// DefaultMaxCombatants is the encounter cap (QA performance case:
	// "Maximum combatants per side (8 friendly + 8 enemy = 16 total)").
	DefaultMaxCombatants int = 16
	// DefaultDyingRounds is the death countdown default: 3 of the dying
	// character's own turn ends before permanent death without stabilization
	// (GDD `dying_rounds`, ADR-0009: "3 rounds to stabilize").
	DefaultDyingRounds int = 3
)
