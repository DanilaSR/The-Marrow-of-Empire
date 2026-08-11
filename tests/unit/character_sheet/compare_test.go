// Unit tests for Story 004: Equipment Stat Comparison.
//
// Evidence path: production/epics/character-inventory-ui/
// story-004-equipment-stat-comparison.md
// Each test maps to one acceptance criterion or QA edge case: AC-8 (sword
// +3 damage vs axe +5 damage → "−2" in green next to Damage), the QA
// reversed scenario (equipped better → "+2" in amber), the equal-items
// scenario ("0" in ink, no sign), the different-slot scenario (no
// comparison), and multi-stat items (damage, armor, attribute bonuses).
// The tests cover the pure display logic only — the side-by-side panel
// layout and real-time update are Story 005's item detail panel and are
// verified by screenshot evidence per the UI story testing standard. No
// engine code runs here.
package character_sheet_test

import (
	"testing"

	"marrow-of-empire/character_sheet"
	"marrow-of-empire/stats"
)

// AC-8: GIVEN an equipped sword (+3 damage) and a compared axe (+5 damage)
// in inventory, WHEN the player selects the axe, THEN the diff is −2 with
// the "−" sign (U+2212) in green (compared is better).
func TestStatDiff_AC8_AxeBetter(t *testing.T) {
	delta, sign, color := character_sheet.StatDiff(3, 5)

	if delta != -2 {
		t.Errorf("StatDiff(3, 5) delta = %d, want -2", delta)
	}
	if sign != "−" {
		t.Errorf("StatDiff(3, 5) sign = %q, want %q (U+2212, not ASCII hyphen)", sign, "−")
	}
	if color != character_sheet.ColorGreen {
		t.Errorf("StatDiff(3, 5) color = %q, want %q (compared better)", color, character_sheet.ColorGreen)
	}
}

// AC-8 end-to-end: GIVEN an equipped sword (+3 damage, Main Hand) and the
// compared axe (+5 damage, Main Hand), WHEN CompareItems runs, THEN a
// single Damage row renders "Damage: +3 → −2" in green — the sign and the
// color both indicate the axe is better (color is never the sole
// differentiator, ADR-0012).
func TestCompareItems_AC8_SwordVsAxe(t *testing.T) {
	sword := character_sheet.ItemStats{Slot: character_sheet.SlotMainHand, Damage: 3}
	axe := character_sheet.ItemStats{Slot: character_sheet.SlotMainHand, Damage: 5}

	rows := character_sheet.CompareItems(sword, axe)

	if len(rows) != 1 {
		t.Fatalf("CompareItems = %d rows, want 1 (Damage only): %+v", len(rows), rows)
	}
	row := rows[0]
	if row.Name != "Damage" {
		t.Errorf("row name = %q, want %q", row.Name, "Damage")
	}
	if got := character_sheet.FormatComparison(row); got != "Damage: +3 → −2" {
		t.Errorf("FormatComparison = %q, want %q", got, "Damage: +3 → −2")
	}
	if row.Color != character_sheet.ColorGreen {
		t.Errorf("row color = %q, want %q (green)", row.Color, character_sheet.ColorGreen)
	}
}

// QA reversed scenario: GIVEN an equipped item that is better (+3 vs +1),
// THEN the diff shows "+2" in amber (equipped is better).
func TestStatDiff_EquippedBetter_Amber(t *testing.T) {
	delta, sign, color := character_sheet.StatDiff(3, 1)

	if delta != 2 {
		t.Errorf("StatDiff(3, 1) delta = %d, want 2", delta)
	}
	if sign != "+" {
		t.Errorf("StatDiff(3, 1) sign = %q, want %q", sign, "+")
	}
	if color != character_sheet.ColorAmber {
		t.Errorf("StatDiff(3, 1) color = %q, want %q (equipped better)", color, character_sheet.ColorAmber)
	}
}

// QA equal-items scenario: GIVEN an equipped ring (+1 STR) and a compared
// ring (+1 STR), THEN the diff shows "0" in ink with no sign — equal items.
func TestStatDiff_Equal_Ink(t *testing.T) {
	delta, sign, color := character_sheet.StatDiff(3, 3)

	if delta != 0 {
		t.Errorf("StatDiff(3, 3) delta = %d, want 0", delta)
	}
	if sign != "" {
		t.Errorf("StatDiff(3, 3) sign = %q, want %q (no sign for equal)", sign, "")
	}
	if color != character_sheet.ColorInk {
		t.Errorf("StatDiff(3, 3) color = %q, want %q (ink, parchment-dark)", color, character_sheet.ColorInk)
	}
}

// QA edge case: GIVEN a hovered item of a different slot type (helmet when
// comparing to an equipped sword), THEN no comparison is produced — only
// same-slot items trigger comparison (GDD CompareMode). The helmet carries
// a higher armor value to prove it is the slot, not the stats, that gates.
func TestCompareItems_DifferentSlots_NoComparison(t *testing.T) {
	sword := character_sheet.ItemStats{Slot: character_sheet.SlotMainHand, Damage: 3}
	helmet := character_sheet.ItemStats{Slot: character_sheet.SlotHead, Armor: 9}

	rows := character_sheet.CompareItems(sword, helmet)

	if rows != nil {
		t.Errorf("CompareItems = %+v, want nil (different slots)", rows)
	}
	if len(rows) != 0 {
		t.Errorf("CompareItems = %d rows, want 0 (different slots)", len(rows))
	}
}

// QA multi-stat scenario: GIVEN two same-slot items with damage and
// attribute bonuses, THEN the comparison returns one row per stat in
// canonical order (Damage, Armor, then attribute bonuses), each with the
// correct delta, sign, and color. Rows where both items are zero (Armor,
// CON, INT, CHA) are omitted — a row must show a real difference or
// presence.
func TestCompareItems_MultipleStats(t *testing.T) {
	equipped := character_sheet.ItemStats{
		Slot:             character_sheet.SlotMainHand,
		Damage:           6,
		AttributeBonuses: [stats.NumAttributes]int{stats.STR: 2, stats.WIS: 1},
	}
	compared := character_sheet.ItemStats{
		Slot:             character_sheet.SlotMainHand,
		Damage:           4,
		AttributeBonuses: [stats.NumAttributes]int{stats.DEX: 3},
	}

	rows := character_sheet.CompareItems(equipped, compared)

	want := []character_sheet.StatComparison{
		{Name: "Damage", Equipped: 6, Delta: 2, Sign: "+", Color: character_sheet.ColorAmber},
		{Name: "STR", Equipped: 2, Delta: 2, Sign: "+", Color: character_sheet.ColorAmber},
		{Name: "DEX", Equipped: 0, Delta: -3, Sign: "−", Color: character_sheet.ColorGreen},
		{Name: "WIS", Equipped: 1, Delta: 1, Sign: "+", Color: character_sheet.ColorAmber},
	}
	if len(rows) != len(want) {
		t.Fatalf("CompareItems = %d rows, want %d: %+v", len(rows), len(want), rows)
	}
	for i, w := range want {
		if rows[i] != w {
			t.Errorf("row %d = %+v, want %+v", i, rows[i], w)
		}
	}
}

// QA edge case: GIVEN equal items, THEN the formatted row carries no sign:
// "Damage: +3 → 0" (GDD: zero shows no sign).
func TestFormatComparison_Equal_NoSign(t *testing.T) {
	sc := character_sheet.StatComparison{
		Name: "Damage", Equipped: 3, Delta: 0, Sign: "", Color: character_sheet.ColorInk,
	}

	if got := character_sheet.FormatComparison(sc); got != "Damage: +3 → 0" {
		t.Errorf("FormatComparison = %q, want %q", got, "Damage: +3 → 0")
	}
}

// QA reversed scenario display: GIVEN the equipped item is better (+3 vs
// +1), THEN the row renders "Damage: +3 → +2" in amber — the diff sign
// matches the color (equipped better).
func TestFormatComparison_EquippedBetter_Amber(t *testing.T) {
	sc := character_sheet.StatComparison{
		Name: "Damage", Equipped: 3, Delta: 2, Sign: "+", Color: character_sheet.ColorAmber,
	}

	if got := character_sheet.FormatComparison(sc); got != "Damage: +3 → +2" {
		t.Errorf("FormatComparison = %q, want %q", got, "Damage: +3 → +2")
	}
	if sc.Color != character_sheet.ColorAmber {
		t.Errorf("Color = %q, want %q", sc.Color, character_sheet.ColorAmber)
	}
}
