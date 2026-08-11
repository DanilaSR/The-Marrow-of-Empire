// Package stats implements the character attribute model, modifier
// formula, and derived stats per ADR-0006 (Character Stats & Component
// Schema).
//
// This package is pure Go math with no Kaiju engine dependency, so it can
// be unit-tested without the engine running. The StatsComponent attaches to
// entities via the Entity Framework (EF-001) component store.
//
// Stories: STAT-001 (production/epics/character-stats-progression/
// story-001-attribute-model.md), STAT-002 (production/epics/
// character-stats-progression/story-002-derived-stats.md), STAT-003
// (production/epics/character-stats-progression/story-004-xp-progression.md)
package stats

// Attribute indices into StatsComponent.Attributes, in canonical order
// [STR, DEX, CON, INT, WIS, CHA] per ADR-0006 and TR-stats-001.
const (
	STR = iota
	DEX
	CON
	INT
	WIS
	CHA
	// NumAttributes is the number of core attributes (six).
	NumAttributes
)

// StatsComponent holds the six core attributes per ADR-0006.
// Attributes are stored in a fixed-size [6]int array for zero heap allocations.
//
// Attributes range 3-18 with human baseline 10 (TR-stats-001). Level ranges
// 1-20 and XP is total accumulated XP; leveling and XP award are handled in
// xp.go (STAT-003).
type StatsComponent struct {
	Attributes [NumAttributes]int
	Level      int
	XP         int
	// UnspentPoints are unallocated attribute points granted by level-ups
	// at levels 4/8/12/16 (STAT-003 AC-4, GDD CR-5). AwardXP increments the
	// pool; AllocateAttributePoint spends one point to raise an attribute.
	UnspentPoints int
}

// Modifier returns the BG1-compatible attribute modifier per TR-stats-002
// and ADR-0006: floor((attr-10)/2) clamped to [-4, +4].
//
// Go integer division truncates toward zero, which matches floor for
// positive and negative-even differences. For negative odd differences
// (e.g., attr=9 → (9-10)/2 = 0 in Go, but floor(-0.5) = -1), a conditional
// decrement corrects the result to floor semantics.
func Modifier(attr int) int {
	v := (attr - 10) / 2
	if (attr-10) < 0 && (attr-10)%2 != 0 {
		v-- // correct truncation toward zero to floor for negative odd values
	}
	if v < -4 {
		return -4
	}
	if v > 4 {
		return 4
	}
	return v
}

// MaxHealth returns the character's maximum hit points per ADR-0006 and
// TR-stats-003: 20 + (CON×2) + (Level×CON_mod).
//
// Story: STAT-002 (production/epics/character-stats-progression/
// story-002-derived-stats.md).
func (s StatsComponent) MaxHealth() int {
	return 20 + s.Attributes[CON]*2 + s.Level*Modifier(s.Attributes[CON])
}

// AP returns the character's action points per ADR-0006 and TR-stats-003:
// 4 + DEX_mod, with a minimum of 2. The ADR's maximum of 8 is unreachable
// given the attribute range 3-18 (max DEX_mod is +4 → AP 8).
//
// Story: STAT-002 (production/epics/character-stats-progression/
// story-002-derived-stats.md).
func (s StatsComponent) AP() int {
	ap := 4 + Modifier(s.Attributes[DEX])
	if ap < 2 {
		return 2
	}
	return ap
}

// Initiative returns the character's initiative per ADR-0006 and
// TR-stats-003: DEX_mod + WIS_mod.
//
// Story: STAT-002 (production/epics/character-stats-progression/
// story-002-derived-stats.md).
func (s StatsComponent) Initiative() int {
	return Modifier(s.Attributes[DEX]) + Modifier(s.Attributes[WIS])
}

// Evasion returns the character's evasion per ADR-0006 and TR-stats-003:
// 10 + DEX_mod.
//
// Story: STAT-002 (production/epics/character-stats-progression/
// story-002-derived-stats.md).
func (s StatsComponent) Evasion() int {
	return 10 + Modifier(s.Attributes[DEX])
}

// CarryCapacity returns the character's carry capacity in pounds per
// ADR-0006 and TR-stats-003: (STR×10) + 30.
//
// Story: STAT-002 (production/epics/character-stats-progression/
// story-002-derived-stats.md).
func (s StatsComponent) CarryCapacity() int {
	return s.Attributes[STR]*10 + 30
}
