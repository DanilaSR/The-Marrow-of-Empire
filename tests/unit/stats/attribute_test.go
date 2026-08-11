// Unit tests for Story STAT-001: Attribute Model & Modifiers.
//
// Evidence path: production/epics/character-stats-progression/story-001-attribute-model.md
// Each test maps to one acceptance criterion (AC-1 .. AC-7). The Modifier
// tests verify TR-stats-002 (floor((A-10)/2) clamped [-4, +4]), including
// both clamp bounds and the zero-truncation fix for negative odd values.
package stats_test

import (
	"testing"

	"marrow-of-empire/stats"
)

// AC-1: GIVEN a Stats component, WHEN attributes are set to STR 16, DEX 12,
// CON 14, INT 10, WIS 10, CHA 8, THEN all six are stored and retrievable.
func TestAttributeStorage(t *testing.T) {
	c := stats.StatsComponent{}
	c.Attributes[stats.STR] = 16
	c.Attributes[stats.DEX] = 12
	c.Attributes[stats.CON] = 14
	c.Attributes[stats.INT] = 10
	c.Attributes[stats.WIS] = 10
	c.Attributes[stats.CHA] = 8

	want := [...]int{16, 12, 14, 10, 10, 8}
	if c.Attributes != want {
		t.Errorf("attributes = %v, want %v", c.Attributes, want)
	}
}

// AC-2: GIVEN attribute 14, WHEN modifier is calculated, THEN result is +2
// (floor((14-10)/2)).
func TestModifier_14_ReturnsPlus2(t *testing.T) {
	if got := stats.Modifier(14); got != 2 {
		t.Errorf("Modifier(14) = %d, want +2", got)
	}
}

// AC-3: GIVEN attribute 18, WHEN modifier is calculated, THEN result is +4.
func TestModifier_18_ReturnsPlus4(t *testing.T) {
	if got := stats.Modifier(18); got != 4 {
		t.Errorf("Modifier(18) = %d, want +4", got)
	}
}

// AC-4: GIVEN attribute 3, WHEN modifier is calculated, THEN result is -4.
func TestModifier_3_ReturnsMinus4(t *testing.T) {
	if got := stats.Modifier(3); got != -4 {
		t.Errorf("Modifier(3) = %d, want -4", got)
	}
}

// AC-5: GIVEN attribute 20 (above cap), WHEN modifier is calculated, THEN
// result is clamped to +4 (not +5).
func TestModifier_20_ClampedToPlus4(t *testing.T) {
	if got := stats.Modifier(20); got != 4 {
		t.Errorf("Modifier(20) = %d, want clamped +4", got)
	}
}

// AC-6: GIVEN attribute 1 (below floor), WHEN modifier is calculated, THEN
// result is clamped to -4.
func TestModifier_1_ClampedToMinus4(t *testing.T) {
	if got := stats.Modifier(1); got != -4 {
		t.Errorf("Modifier(1) = %d, want clamped -4", got)
	}
}

// AC-7: GIVEN attribute 10, WHEN modifier is calculated, THEN result is 0.
func TestModifier_10_ReturnsZero(t *testing.T) {
	if got := stats.Modifier(10); got != 0 {
		t.Errorf("Modifier(10) = %d, want 0", got)
	}
}
