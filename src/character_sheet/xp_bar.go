// XP progress bar display logic (Story 002, Epic: Character & Inventory
// UI): the fill percentage and text of the character sheet's XP bar per the
// GDD formula (design/gdd/character-inventory-ui.md, "XP Progress"):
//
//	xp_pct = ((total_xp − xp_for_current_level) / xp_to_next_level) × 100
//
// where xp_for_current_level = stats.TotalXPForLevel(Level) and
// xp_to_next_level = stats.XPToNext() — the ADR-0006 formulas implemented
// in the stats package. This file only composes them for display; the XP
// math itself lives in stats/xp.go and is never re-implemented here.
//
// Like the rest of the sheet's display helpers (AttributeDisplay,
// DerivedStats) these functions are pure Go with no engine dependency, so
// they can be unit-tested without the engine running
// (tests/unit/character_sheet/).
//
// Story: production/epics/character-inventory-ui/
// story-002-xp-progress-level-up.md
package character_sheet

import (
	"fmt"

	"marrow-of-empire/stats"
)

// XPProgressPct returns the XP bar fill percentage for the character per
// the GDD formula: ((total_xp − xp_for_current_level) / xp_to_next_level)
// × 100, where xp_for_current_level = stats.TotalXPForLevel(Level) and
// xp_to_next_level = stats.XPToNext() (ADR-0006 / TR-stats-005).
//
// The result is clamped to [0, 100]: 0% when the character just leveled
// (total_xp == xp_for_current_level) and 100% when the next level threshold
// is reached or passed (overflowing XP still fills the bar, the level-up
// itself is handled by AwardXP). At the level cap (stats.LevelCap) the bar
// is always full — XP continues to accumulate but no further level-ups
// occur (GDD Edge Cases).
//
// Example:
//
//	c := stats.StatsComponent{Level: 4, XP: 2400}
//	character_sheet.XPProgressPct(c) // 40
func XPProgressPct(c stats.StatsComponent) float64 {
	if c.Level >= stats.LevelCap {
		return 100
	}
	cur := stats.TotalXPForLevel(c.Level)
	next := c.XPToNext()
	if next <= 0 {
		// Defensive: a Level ≤ 0 component has no threshold to progress
		// toward; the stats contract is Level 1-20.
		return 0
	}
	pct := float64(c.XP-cur) / float64(next) * 100
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// XPProgressDisplay renders the XP bar label per the GDD: the character's
// total accumulated XP and the XP required for the next level, comma-
// grouped, e.g. "2,400 / 1,500 XP" for a level 4 character with 2400 total
// XP. At the level cap the label reads "Max Level" (the bar is full and no
// further level-ups occur).
//
// Example:
//
//	c := stats.StatsComponent{Level: 4, XP: 2400}
//	character_sheet.XPProgressDisplay(c) // "2,400 / 1,500 XP"
func XPProgressDisplay(c stats.StatsComponent) string {
	if c.Level >= stats.LevelCap {
		return "Max Level"
	}
	return fmt.Sprintf("%s / %s XP", formatThousands(c.XP), formatThousands(c.XPToNext()))
}

// LevelUpHeader renders the level-up modal header per the GDD: "Level Up —
// Level [N] → [N+1]". The actual previous and new levels are used rather
// than deriving [N+1] from the new level, so a multi-level award renders
// its true range, e.g. "Level Up — Level 3 → 5".
//
// Example:
//
//	character_sheet.LevelUpHeader(4, 5) // "Level Up — Level 4 → 5"
func LevelUpHeader(prevLevel, newLevel int) string {
	return fmt.Sprintf("Level Up — Level %d → %d", prevLevel, newLevel)
}

// formatThousands comma-groups a non-negative integer's digits by threes
// ("2400" → "2,400", "57000" → "57,000"). Go's standard library has no
// thousands formatter and the display values stay small (XP ≤ 57,000 at
// the cap), so a tiny recursive grouping is used instead of a dependency.
// Negative values render with the ASCII hyphen (XP is never negative; the
// branch is purely defensive).
func formatThousands(n int) string {
	if n < 0 {
		return "-" + formatThousands(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", formatThousands(n/1000), n%1000)
}
