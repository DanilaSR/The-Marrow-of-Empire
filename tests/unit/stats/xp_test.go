// Unit tests for Story STAT-003: XP & Level Progression.
//
// Evidence path: production/epics/character-stats-progression/story-004-xp-progression.md
// Tests map to the acceptance criteria (AC-1 .. AC-6) and verify TR-stats-005:
// XP_to_next = Level × 300, Total_XP = 150 × N × (N−1), level cap 20, health
// gain max(1, CON_mod), attribute point every 4th level — per ADR-0006.
// Extra tests cover the GDD edge cases (multi-level jumps, sub-threshold
// accumulation, zero/negative awards) and the story-005 bonus utilities
// (LevelForXP, AllocateAttributePoint).
package stats_test

import (
	"slices"
	"testing"

	"marrow-of-empire/stats"
)

// newComponent builds a StatsComponent with the given attributes in
// canonical order [STR, DEX, CON, INT, WIS, CHA] per ADR-0006.
func newComponent(attrs [6]int, level, xp int) stats.StatsComponent {
	return stats.StatsComponent{Attributes: attrs, Level: level, XP: xp}
}

// GIVEN level 1, WHEN XPToNext is queried, THEN 300 XP is required
// (Level × 300) — the threshold behind AC-1.
func TestXPToNext_Level1(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 1}

	// Act
	got := c.XPToNext()

	// Assert
	if got != 300 {
		t.Errorf("XPToNext() = %d, want 300", got)
	}
}

// AC-1: GIVEN level 1 (0 XP), WHEN 300 XP is awarded, THEN the character
// reaches level 2 with 300 total XP.
func TestXPAward_LevelUp(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 1}

	// Act
	gained := c.AwardXP(300)

	// Assert
	if gained != 1 {
		t.Errorf("AwardXP(300) = %d level(s), want 1", gained)
	}
	if c.Level != 2 {
		t.Errorf("Level = %d, want 2", c.Level)
	}
	if c.XP != 300 {
		t.Errorf("XP = %d, want 300 (total accumulated)", c.XP)
	}
}

// AC-2: GIVEN level 4 (1800 XP), WHEN 1200 more XP is awarded (total 3000),
// THEN the character reaches level 5 — exactly on the cumulative threshold,
// no XP lost.
func TestXPAward_ChainLevelUp(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 4, XP: 1800}

	// Act
	gained := c.AwardXP(1200)

	// Assert
	if gained != 1 {
		t.Errorf("AwardXP(1200) = %d level(s), want 1", gained)
	}
	if c.Level != 5 {
		t.Errorf("Level = %d, want 5", c.Level)
	}
	if c.XP != 3000 {
		t.Errorf("XP = %d, want 3000 (total accumulated)", c.XP)
	}
	if c.XPToNext() != 1500 {
		t.Errorf("XPToNext() = %d, want 1500 at level 5", c.XPToNext())
	}
}

// AC-6: GIVEN CON 6 (−2 mod) leveling from 1 to 2, WHEN health is
// recalculated, THEN the health gain is at least +1 (minimum gain rule) —
// the min +1 rule deferred from STAT-002.
func TestLevelUp_HealthGain(t *testing.T) {
	// Arrange — CON 6 gives a -2 modifier.
	c := stats.StatsComponent{Level: 1}
	c.Attributes[stats.CON] = 6

	// Act
	gain := c.HealthGainPerLevel()
	need := c.XPToNext()
	gained := c.AwardXP(need)

	// Assert
	if gain != 1 {
		t.Errorf("HealthGainPerLevel() = %d, want minimum +1 (CON 6, mod -2)", gain)
	}
	if gained != 1 || c.Level != 2 {
		t.Errorf("level-up failed: gained %d level(s), Level = %d", gained, c.Level)
	}
}

// AC-4: GIVEN levels 4, 8, 12, 16, WHEN level-up occurs, THEN the player
// gains +1 attribute point at each.
func TestLevelUp_AttributePoint(t *testing.T) {
	milestones := []struct {
		level int
		want  int
	}{
		{4, 1}, {8, 2}, {12, 3}, {16, 4},
	}

	// Arrange — one character driven through all four milestones; the pool
	// accumulates by exactly one point per milestone.
	prev := 1
	c := stats.StatsComponent{Level: 1}
	for _, m := range milestones {
		// Act — award the exact XP to advance from prev to m.level.
		xp := stats.TotalXPForLevel(m.level) - stats.TotalXPForLevel(prev)
		gained := c.AwardXP(xp)

		// Assert
		if gained != m.level-prev {
			t.Errorf("level %d: AwardXP = %d level(s), want %d", m.level, gained, m.level-prev)
		}
		if c.Level != m.level {
			t.Errorf("level %d: Level = %d, want %d", m.level, c.Level, m.level)
		}
		if c.UnspentPoints != m.want {
			t.Errorf("level %d: UnspentPoints = %d, want %d", m.level, c.UnspentPoints, m.want)
		}
		prev = m.level
	}
}

// AC-5: GIVEN level 20 (cap, 57,000 XP), WHEN XP is awarded, THEN XP
// accumulates but no further level-up occurs, and the cap grants no
// attribute point.
func TestLevelCap(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 20, XP: 57000}

	// Act
	gained := c.AwardXP(9999)

	// Assert
	if gained != 0 {
		t.Errorf("AwardXP(9999) at cap = %d level(s), want 0", gained)
	}
	if c.Level != 20 {
		t.Errorf("Level = %d, want 20 (cap)", c.Level)
	}
	if c.XP != 66999 {
		t.Errorf("XP = %d, want 66999 (accumulates past cap)", c.XP)
	}
	if c.UnspentPoints != 0 {
		t.Errorf("UnspentPoints = %d, want 0 (level 20 grants no point)", c.UnspentPoints)
	}
}

// GDD Edge Cases: GIVEN XP from multiple sources would grant 2+ levels at
// once, WHEN awarded, THEN levels process sequentially — each threshold is
// crossed in order, remaining XP carries into the next threshold, and no XP
// is lost.
func TestXPAward_MultiLevelJump(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 1}

	// Act — 1300 XP crosses L2 (300) and L3 (900), leaving 400 toward L4.
	gained := c.AwardXP(1300)

	// Assert
	if gained != 2 {
		t.Errorf("AwardXP(1300) = %d level(s), want 2", gained)
	}
	if c.Level != 3 {
		t.Errorf("Level = %d, want 3", c.Level)
	}
	if c.XP != 1300 {
		t.Errorf("XP = %d, want 1300 (no XP lost)", c.XP)
	}

	// Act — the carried 400 XP counts toward the next threshold: +500
	// reaches exactly 1800 and crosses level 4 (milestone point granted).
	gained = c.AwardXP(500)

	// Assert
	if gained != 1 {
		t.Errorf("AwardXP(500) = %d level(s), want 1", gained)
	}
	if c.Level != 4 {
		t.Errorf("Level = %d, want 4", c.Level)
	}
	if c.XP != 1800 {
		t.Errorf("XP = %d, want 1800", c.XP)
	}
	if c.UnspentPoints != 1 {
		t.Errorf("UnspentPoints = %d, want 1 (level 4 milestone)", c.UnspentPoints)
	}
}

// GIVEN XP below the next level threshold, WHEN awarded, THEN XP
// accumulates and no level-up occurs.
func TestAwardXP_BelowThreshold_NoLevelUp(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 1}

	// Act
	gained := c.AwardXP(299)

	// Assert
	if gained != 0 {
		t.Errorf("AwardXP(299) = %d level(s), want 0", gained)
	}
	if c.Level != 1 || c.XP != 299 {
		t.Errorf("Level/XP = %d/%d, want 1/299", c.Level, c.XP)
	}
}

// GIVEN a zero or negative XP award, WHEN AwardXP is called, THEN nothing
// changes — the GDD has no XP-loss mechanic.
func TestAwardXP_ZeroAndNegative_NoOp(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 3, XP: 900}

	// Act / Assert
	if gained := c.AwardXP(0); gained != 0 {
		t.Fatalf("AwardXP(0) = %d level(s), want 0", gained)
	}
	if gained := c.AwardXP(-500); gained != 0 {
		t.Fatalf("AwardXP(-500) = %d level(s), want 0", gained)
	}
	if c.Level != 3 || c.XP != 900 {
		t.Errorf("Level/XP = %d/%d, want 3/900", c.Level, c.XP)
	}
}

// GIVEN level 5, WHEN XPToNext is queried, THEN 1500 XP is required (5 × 300).
func TestXPToNext_Level5(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 5}

	// Act
	got := c.XPToNext()

	// Assert
	if got != 1500 {
		t.Errorf("XPToNext() = %d, want 1500", got)
	}
}

// GIVEN level 20 (cap), WHEN XPToNext is queried, THEN 6000 is returned
// (20 × 300 — the GDD formula range is 300-6000). The cap itself is
// enforced by AwardXP, not by XPToNext.
func TestXPToNext_Level20(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 20}

	// Act
	got := c.XPToNext()

	// Assert
	if got != 6000 {
		t.Errorf("XPToNext() = %d, want 6000", got)
	}
}

// AC-3: GIVEN CON 14 (+2 mod), WHEN HealthGainPerLevel is queried, THEN the
// health gain is 2 (CON_mod).
func TestHealthGainPerLevel_Con14(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 1}
	c.Attributes[stats.CON] = 14

	// Act
	got := c.HealthGainPerLevel()

	// Assert
	if got != 2 {
		t.Errorf("HealthGainPerLevel() = %d, want 2 (CON 14, mod +2)", got)
	}
}

// TotalXPForLevel: GIVEN level 5, THEN the cumulative threshold is 3000
// (150 × 5 × 4).
func TestTotalXPForLevel_Level5(t *testing.T) {
	if got := stats.TotalXPForLevel(5); got != 3000 {
		t.Errorf("TotalXPForLevel(5) = %d, want 3000", got)
	}
}

// TotalXPForLevel: GIVEN level 1, THEN the cumulative threshold is 0.
func TestTotalXPForLevel_Level1(t *testing.T) {
	if got := stats.TotalXPForLevel(1); got != 0 {
		t.Errorf("TotalXPForLevel(1) = %d, want 0", got)
	}
}

// TotalXPForLevel: GIVEN level 20 (cap), THEN the cumulative threshold is
// 57,000 (150 × 20 × 19).
func TestTotalXPForLevel_Level20(t *testing.T) {
	if got := stats.TotalXPForLevel(20); got != 57000 {
		t.Errorf("TotalXPForLevel(20) = %d, want 57000", got)
	}
}

// LevelForXP: GIVEN a cumulative XP total, THEN the inverse of
// TotalXPForLevel returns the highest level whose threshold is met.
func TestLevelForXP_Threshold_ReturnsLevel(t *testing.T) {
	if got := stats.LevelForXP(3000); got != 5 {
		t.Errorf("LevelForXP(3000) = %d, want 5", got)
	}
	if got := stats.LevelForXP(299); got != 1 {
		t.Errorf("LevelForXP(299) = %d, want 1", got)
	}
}

// LevelForXP: GIVEN XP beyond the level-20 threshold, THEN the result is
// clamped to the cap.
func TestLevelForXP_BeyondCap_ClampsTo20(t *testing.T) {
	if got := stats.LevelForXP(100000); got != 20 {
		t.Errorf("LevelForXP(100000) = %d, want 20", got)
	}
}

// Full sweep: GIVEN one XP award spanning levels 1-20, THEN exactly the
// milestone levels 4, 8, 12, 16 grant attribute points and the pool holds
// 4 points.
func TestAwardXP_FullSweep_GrantsExactlyMilestones(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 1}

	// Act — award exactly enough XP to reach each next level, one at a time,
	// recording which levels granted a point.
	var granted []int
	for c.Level < stats.LevelCap {
		prev := c.Level
		before := c.UnspentPoints
		gained := c.AwardXP(c.XPToNext())
		if gained != 1 {
			t.Fatalf("level %d: AwardXP = %d level(s), want 1", prev, gained)
		}
		if c.UnspentPoints > before {
			granted = append(granted, c.Level)
		}
	}

	// Assert
	want := []int{4, 8, 12, 16}
	if !slices.Equal(granted, want) {
		t.Errorf("attribute points granted at levels %v, want %v", granted, want)
	}
	if c.UnspentPoints != 4 {
		t.Errorf("UnspentPoints = %d, want 4", c.UnspentPoints)
	}
	if c.Level != 20 || c.XP != 57000 {
		t.Errorf("Level/XP = %d/%d, want 20/57000", c.Level, c.XP)
	}
}

// AC-4 (allocation side): GIVEN one unallocated attribute point, WHEN it is
// spent on CON, THEN CON rises by 1 and the pool empties.
func TestAllocateAttributePoint_SpendsPointAndRaisesAttribute(t *testing.T) {
	// Arrange
	c := newComponent([6]int{10, 10, 12, 10, 10, 10}, 4, 1800)
	c.UnspentPoints = 1

	// Act
	err := c.AllocateAttributePoint(stats.CON)

	// Assert
	if err != nil {
		t.Fatalf("AllocateAttributePoint(CON) returned error: %v", err)
	}
	if c.Attributes[stats.CON] != 13 {
		t.Errorf("CON = %d, want 13", c.Attributes[stats.CON])
	}
	if c.UnspentPoints != 0 {
		t.Errorf("UnspentPoints = %d, want 0", c.UnspentPoints)
	}
}

// GIVEN an empty attribute point pool, WHEN a point is spent, THEN an error
// is returned and the attribute is unchanged.
func TestAllocateAttributePoint_NoPoints_Errors(t *testing.T) {
	// Arrange
	c := newComponent([6]int{10, 10, 10, 10, 10, 10}, 4, 1800)

	// Act
	err := c.AllocateAttributePoint(stats.CON)

	// Assert
	if err == nil {
		t.Fatal("AllocateAttributePoint with 0 points: want error, got nil")
	}
	if c.Attributes[stats.CON] != 10 {
		t.Errorf("CON = %d, want unchanged 10", c.Attributes[stats.CON])
	}
}

// GIVEN an out-of-range attribute index, WHEN a point is spent, THEN an
// error is returned and the pool is untouched.
func TestAllocateAttributePoint_InvalidIndex_Errors(t *testing.T) {
	// Arrange
	c := newComponent([6]int{10, 10, 10, 10, 10, 10}, 4, 1800)
	c.UnspentPoints = 1

	// Act
	err := c.AllocateAttributePoint(stats.NumAttributes)

	// Assert
	if err == nil {
		t.Fatal("AllocateAttributePoint(NumAttributes): want error, got nil")
	}
	if c.UnspentPoints != 1 {
		t.Errorf("UnspentPoints = %d, want unchanged 1", c.UnspentPoints)
	}
}

// GIVEN an attribute at the mortal cap 18, WHEN a point is spent, THEN an
// error is returned (CR-1 range 3-18).
func TestAllocateAttributePoint_AtCap18_Errors(t *testing.T) {
	// Arrange
	c := newComponent([6]int{10, 10, 18, 10, 10, 10}, 8, 8400)
	c.UnspentPoints = 1

	// Act
	err := c.AllocateAttributePoint(stats.CON)

	// Assert
	if err == nil {
		t.Fatal("AllocateAttributePoint on 18 CON: want error, got nil")
	}
	if c.Attributes[stats.CON] != 18 {
		t.Errorf("CON = %d, want unchanged 18", c.Attributes[stats.CON])
	}
	if c.UnspentPoints != 1 {
		t.Errorf("UnspentPoints = %d, want unchanged 1", c.UnspentPoints)
	}
}
