// XP & level progression per ADR-0006 and TR-stats-005: linear level
// thresholds (XP_to_next = Level × 300), cumulative total XP
// (Total_XP = 150 × N × (N−1)), level cap 20, and level-up benefits
// (health +max(1, CON_mod), one attribute point every 4th level).
//
// All core math is pure integer arithmetic with zero heap allocations, so it
// stays well inside the frame budget (ADR-0006 Performance Implications:
// XP threshold check < 0.01ms).
//
// Story: STAT-003 (production/epics/character-stats-progression/
// story-004-xp-progression.md).
package stats

import (
	"errors"
	"fmt"
)

// LevelCap is the maximum character level per ADR-0006 and TR-stats-005.
// At the cap, XP continues to accumulate but no further level-ups occur
// (GDD CR-5, Edge Cases).
const LevelCap = 20

// xpPerLevel is the XP-to-next-level multiplier (Level × 300, TR-stats-005).
// TotalXPForLevel derives its 150 from this (300/2) so the two formulas can
// never drift apart.
const xpPerLevel = 300

// attributePointInterval is how often a level-up grants +1 attribute point:
// levels 4, 8, 12, 16 per STAT-003 AC-4 (level 20 excluded).
const attributePointInterval = 4

// maxAttributeValue is the mortal attribute cap per GDD CR-1 (range 3-18);
// level-up allocation cannot raise an attribute beyond it.
const maxAttributeValue = 18

// XPToNext returns the XP required to advance from the current level to the
// next per TR-stats-005: Level × 300 (300 at level 1, 6000 at level 20 —
// the GDD formula range). At the level cap the value is informational only;
// AwardXP never levels past LevelCap.
//
// Example:
//
//	c := stats.StatsComponent{Level: 5}
//	c.XPToNext() // 1500
func (s StatsComponent) XPToNext() int {
	return s.Level * xpPerLevel
}

// TotalXPForLevel returns the cumulative XP required to reach level n per
// TR-stats-005: 150 × n × (n−1). Level 1 requires 0 XP; level 20 requires
// 57,000 XP.
//
// Example:
//
//	stats.TotalXPForLevel(5) // 3000
func TotalXPForLevel(n int) int {
	return n * (n - 1) * xpPerLevel / 2
}

// LevelForXP returns the highest level whose cumulative threshold
// TotalXPForLevel(level) is ≤ totalXP, clamped to LevelCap. It is the
// inverse of TotalXPForLevel; XP beyond the level-20 threshold still maps
// to level 20 (GDD Edge Cases: post-cap XP is preserved for display).
//
// Example:
//
//	stats.LevelForXP(3000) // 5
func LevelForXP(totalXP int) int {
	level := 1
	for level < LevelCap && totalXP >= TotalXPForLevel(level+1) {
		level++
	}
	return level
}

// AwardXP adds xp to the character's total accumulated XP and processes any
// resulting level-ups sequentially per ADR-0006 and TR-stats-005: levels
// advance while total XP meets TotalXPForLevel(level+1), one level at a
// time — XP is never reset or lost across a multi-level jump (GDD Edge
// Cases). Returns the number of levels gained.
//
// Each level-up at levels 4, 8, 12, 16 grants +1 attribute point to
// UnspentPoints (STAT-003 AC-4); the cap level 20 grants none. Health gain
// per level is max(1, CON_mod) via HealthGainPerLevel; MaxHealth itself
// remains the derived closed-form formula from STAT-002. At LevelCap, XP
// keeps accumulating but no further level-ups occur.
//
// xp ≤ 0 is a no-op: the GDD has no XP-loss mechanic, so negative awards
// are ignored rather than deducted.
//
// Example:
//
//	c := stats.StatsComponent{Level: 4, XP: 1800}
//	gained := c.AwardXP(1200) // gained == 1, c.Level == 5
//
// No heap allocations on any path.
func (s *StatsComponent) AwardXP(xp int) int {
	if xp <= 0 {
		return 0
	}
	s.XP += xp
	levels := 0
	for s.Level < LevelCap {
		if s.XP < TotalXPForLevel(s.Level+1) {
			break
		}
		s.Level++
		levels++
		if s.Level%attributePointInterval == 0 && s.Level < LevelCap {
			s.UnspentPoints++
		}
	}
	return levels
}

// HealthGainPerLevel returns the hit points gained on level-up per ADR-0006
// and TR-stats-005: max(1, CON_mod). This is the minimum +1 rule deferred
// from STAT-002 — a character with 6 CON (−2 modifier) gains 1 HP per level
// and never loses health on level-up (GDD Edge Cases).
//
// Example:
//
//	c := stats.StatsComponent{}
//	c.Attributes[stats.CON] = 6
//	c.HealthGainPerLevel() // 1
func (s StatsComponent) HealthGainPerLevel() int {
	g := Modifier(s.Attributes[CON])
	if g < 1 {
		return 1
	}
	return g
}

// AllocateAttributePoint spends one unallocated attribute point to raise
// the given attribute by 1 (STAT-003 AC-4, GDD CR-5: "+1 to one attribute
// of the player's choice" every 4th level). It fails when no points remain,
// the attribute index is out of range, or the attribute is already at the
// mortal maximum of 18 (CR-1).
//
// This utility is kept from the story-004 draft for the story-005
// integration (level-up UI consumes UnspentPoints); the story-003 point-buy
// creation flow is a separate mechanic.
//
// Example:
//
//	c := stats.StatsComponent{UnspentPoints: 1}
//	err := c.AllocateAttributePoint(stats.CON) // CON raised by 1
func (s *StatsComponent) AllocateAttributePoint(attr int) error {
	if attr < 0 || attr >= NumAttributes {
		return fmt.Errorf("stats: attribute index %d out of range", attr)
	}
	if s.UnspentPoints <= 0 {
		return errors.New("stats: no attribute points available")
	}
	if s.Attributes[attr] >= maxAttributeValue {
		return fmt.Errorf("stats: attribute %d already at maximum %d", attr, maxAttributeValue)
	}
	s.UnspentPoints--
	s.Attributes[attr]++
	return nil
}
