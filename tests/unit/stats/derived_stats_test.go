// Unit tests for Story STAT-002: Derived Stats.
//
// Evidence path: production/epics/character-stats-progression/story-002-derived-stats.md
// Each test maps to one acceptance criterion (AC-1 .. AC-6) and verifies
// TR-stats-003: derived stats computed from attributes + modifiers per
// ADR-0006. All stats reuse the existing Modifier() formula — none are
// reimplemented here.
package stats_test

import (
	"testing"

	"marrow-of-empire/stats"
)

// AC-1: GIVEN CON 14 (+2) at level 3, WHEN MaxHealth is calculated, THEN
// result is 20 + (14×2) + (3×2) = 54.
func TestMaxHealth(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{Level: 3}
	c.Attributes[stats.CON] = 14

	// Act
	got := c.MaxHealth()

	// Assert
	if got != 54 {
		t.Errorf("MaxHealth() = %d, want 54", got)
	}
}

// AC-2: GIVEN DEX 16 (+3), WHEN AP is calculated, THEN result is 4 + 3 = 7.
func TestAP(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{}
	c.Attributes[stats.DEX] = 16

	// Act
	got := c.AP()

	// Assert
	if got != 7 {
		t.Errorf("AP() = %d, want 7", got)
	}
}

// AC-3: GIVEN DEX 6 (-2), WHEN AP is calculated, THEN result is
// max(2, 4-2) = 2 (minimum 2).
func TestAP_Minimum(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{}
	c.Attributes[stats.DEX] = 6

	// Act
	got := c.AP()

	// Assert
	if got != 2 {
		t.Errorf("AP() = %d, want minimum 2", got)
	}
}

// AC-4: GIVEN DEX 14 (+2) and WIS 12 (+1), WHEN Initiative is calculated,
// THEN result is 2 + 1 = 3.
func TestInitiative(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{}
	c.Attributes[stats.DEX] = 14
	c.Attributes[stats.WIS] = 12

	// Act
	got := c.Initiative()

	// Assert
	if got != 3 {
		t.Errorf("Initiative() = %d, want 3", got)
	}
}

// AC-5: GIVEN DEX 12 (+1), WHEN Evasion is calculated, THEN result is
// 10 + 1 = 11.
func TestEvasion(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{}
	c.Attributes[stats.DEX] = 12

	// Act
	got := c.Evasion()

	// Assert
	if got != 11 {
		t.Errorf("Evasion() = %d, want 11", got)
	}
}

// AC-6: GIVEN STR 16 (+3), WHEN CarryCapacity is calculated, THEN result is
// (16×10) + 30 = 190 lbs.
func TestCarryCapacity(t *testing.T) {
	// Arrange
	c := stats.StatsComponent{}
	c.Attributes[stats.STR] = 16

	// Act
	got := c.CarryCapacity()

	// Assert
	if got != 190 {
		t.Errorf("CarryCapacity() = %d, want 190", got)
	}
}
