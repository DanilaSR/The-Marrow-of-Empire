// Unit tests for Story 001: Character Sheet Stat Display.
//
// Evidence path: production/epics/character-inventory-ui/story-001-character-sheet-display.md
// Each test maps to one acceptance criterion (AC-1, AC-2, AC-3, AC-5). The
// tests cover the pure display logic (formatting and computation) only —
// rendering itself is verified by screenshot evidence per the UI story
// testing standard. No engine code runs here.
package character_sheet_test

import (
	"testing"

	"marrow-of-empire/character_sheet"
	"marrow-of-empire/stats"
)

// AC-1: GIVEN a character with STR 16 (+3), WHEN the character sheet is
// opened, THEN STR displays as "16 (+3)" with the modifier correctly
// calculated.
func TestAttributeModifierDisplay(t *testing.T) {
	got := character_sheet.AttributeDisplay(16)
	if got != "16 (+3)" {
		t.Errorf("AttributeDisplay(16) = %q, want %q", got, "16 (+3)")
	}
}

// AC-2: GIVEN a character with CON 14 (+2) at level 3, WHEN the character
// sheet is opened, THEN Max Health displays as "54 HP"
// (20 + 14×2 + 3×2 = 20 + 28 + 6 = 54).
func TestDerivedStatDisplay(t *testing.T) {
	c := stats.StatsComponent{}
	c.Attributes[stats.CON] = 14
	c.Level = 3

	rows := character_sheet.DerivedStats(c)
	if rows[0].Name != "Max Health" || rows[0].Value != 54 {
		t.Fatalf("Max Health row = %+v, want value 54", rows[0])
	}
	if got := character_sheet.DerivedValueDisplay(rows[0]); got != "54 HP" {
		t.Errorf("DerivedValueDisplay(Max Health) = %q, want %q", got, "54 HP")
	}
}

// AC-3: GIVEN a character with STR 16 and CHA 8, WHEN the character sheet is
// displayed, THEN the STR bar is approximately 2× the width of the CHA bar
// (±5% tolerance: ratio between 1.9 and 2.1).
func TestBarProportionalWidth(t *testing.T) {
	str := character_sheet.BarFillFraction(16)
	cha := character_sheet.BarFillFraction(8)
	if cha <= 0 {
		t.Fatalf("CHA bar fill = %f, want > 0", cha)
	}
	ratio := str / cha
	if ratio < 1.9 || ratio > 2.1 {
		t.Errorf("STR/CHA bar width ratio = %.4f, want 2.0 (±5%%)", ratio)
	}
}

// AC-5: GIVEN characters with attribute values 7, 10, and 16, WHEN their
// sheets are displayed, THEN modifiers appear as "−2", "+0", and "+3"
// respectively — sign always shown, minus is the proper U+2212 character,
// never the ASCII hyphen. Range boundaries 3 and 18 are included to pin the
// clamp bounds.
func TestModifierSignDisplay(t *testing.T) {
	tests := []struct {
		value int
		want  string
	}{
		{value: 7, want: "7 (−2)"},
		{value: 10, want: "10 (+0)"},
		{value: 16, want: "16 (+3)"},
		{value: 3, want: "3 (−4)"},   // range floor: modifier clamps at -4
		{value: 18, want: "18 (+4)"}, // range ceiling: modifier clamps at +4
	}
	for _, tt := range tests {
		if got := character_sheet.AttributeDisplay(tt.value); got != tt.want {
			t.Errorf("AttributeDisplay(%d) = %q, want %q", tt.value, got, tt.want)
		}
	}
}
