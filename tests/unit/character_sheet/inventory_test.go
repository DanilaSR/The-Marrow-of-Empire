// Unit tests for Story 005: Inventory Grid & Weight Bar.
//
// Evidence path: production/epics/character-inventory-ui/
// story-005-inventory-grid-weight-bar.md
// Each test maps to one acceptance criterion (AC-10, AC-11, AC-12, AC-14),
// the weight-color threshold table from the GDD "Weight Percentage"
// formula, or the gold display cap. The tests cover the pure display logic
// (percentage, color tokens, pulse predicate, formatting, filtering,
// sorting, weight sums) only — grid rendering, the pulse animation, and
// the markup layout are verified by screenshot evidence per the UI story
// testing standard. No engine code runs here.
package character_sheet_test

import (
	"strings"
	"testing"

	"marrow-of-empire/character_sheet"
)

// AC-10: GIVEN inventory weight at 95% of max, WHEN the inventory tab is
// open, THEN the weight bar pulses oxblood. Per the GDD threshold table
// the 90-100% fill is oxblood; per AC-10/QA the pulse runs from 90% — the
// two concerns are split (WeightColor reports the band, WeightPulse the
// animation state) so 95% is oxblood AND pulsing.
func TestWeightColor_AC10_PulsesOxbloodAt95Percent(t *testing.T) {
	pct := character_sheet.WeightPct(95, 100)
	if pct != 95.0 {
		t.Fatalf("WeightPct(95, 100) = %f, want 95", pct)
	}
	if got := character_sheet.WeightColor(pct); got != character_sheet.WeightColorOxblood {
		t.Errorf("WeightColor(95%%) = %q, want %q (oxblood near limit)", got, character_sheet.WeightColorOxblood)
	}
	if !character_sheet.WeightPulse(pct) {
		t.Error("WeightPulse(95%) = false, want true (bar pulses oxblood, AC-10)")
	}
}

// AC-10 end-to-end: the view model composes the same values — fill 95%,
// oxblood token, and a label that always carries the percentage text
// (ADR-0012: never hue-only; QA: percentage text shows "95%").
func TestWeightBar_AC10_ComposesOxbloodAndPct(t *testing.T) {
	bar := character_sheet.WeightBar{Current: 161, Max: 170}
	if got := bar.Pct(); got < 94 || got > 96 {
		t.Fatalf("WeightBar{161, 170}.Pct() = %f, want ~95", got)
	}
	if got := bar.Color(); got != character_sheet.WeightColorOxblood {
		t.Errorf("WeightBar{161, 170}.Color() = %q, want %q", got, character_sheet.WeightColorOxblood)
	}
	if !strings.Contains(bar.Label(), "95%") {
		t.Errorf("WeightBar.Label() = %q, want it to contain the percentage text", bar.Label())
	}
}

// AC-11: GIVEN inventory weight at 30% of max, WHEN the inventory tab is
// open, THEN the weight bar is solid amber with no pulse.
func TestWeightColor_AC11_AmberAt30Percent(t *testing.T) {
	pct := character_sheet.WeightPct(30, 100)
	if got := character_sheet.WeightColor(pct); got != character_sheet.WeightColorAmber {
		t.Errorf("WeightColor(30%%) = %q, want %q (solid amber)", got, character_sheet.WeightColorAmber)
	}
	if character_sheet.WeightPulse(pct) {
		t.Error("WeightPulse(30%) = true, want false (no pulse at normal weight, AC-11)")
	}
}

// AC-12: GIVEN inventory weight at 70% of max, WHEN the inventory tab is
// open, THEN the weight bar shows the amber-oxblood transition color — a
// blend, never purely one or the other (QA AC-12), with no pulse (QA:
// pulse only at ≥90%).
func TestWeightColor_AC12_TransitionAt70Percent(t *testing.T) {
	pct := character_sheet.WeightPct(70, 100)
	if got := character_sheet.WeightColor(pct); got != character_sheet.WeightColorAmberOxblood {
		t.Errorf("WeightColor(70%%) = %q, want %q (amber-oxblood transition)", got, character_sheet.WeightColorAmberOxblood)
	}
	if character_sheet.WeightPulse(pct) {
		t.Error("WeightPulse(70%) = true, want false (no pulse below 90%, QA AC-12)")
	}
}

// AC-14: GIVEN a character with no items in inventory, WHEN the inventory
// tab is opened, THEN the grid shows "Your pack is empty." in subdued ink
// and the gold display remains visible — even at 0 gold.
func TestEmptyInventory_AC14_MessageAndGoldVisible(t *testing.T) {
	if got := character_sheet.EmptyInventoryMessage(); got != "Your pack is empty." {
		t.Errorf("EmptyInventoryMessage() = %q, want %q", got, "Your pack is empty.")
	}
	if got := character_sheet.FormatGold(0); got != "0" {
		t.Errorf("FormatGold(0) = %q, want %q (gold display remains visible, AC-14)", got, "0")
	}
}

// GDD weight-color thresholds: 0-49% amber, 50-89% amber-oxblood, 90-100%
// oxblood, >100% oxblood-pulse. The boundary values 49/50 and 89/90 pin
// the band edges exactly.
func TestWeightColorThresholds(t *testing.T) {
	tests := []struct {
		pct  float64
		want string
	}{
		{pct: 0, want: character_sheet.WeightColorAmber},
		{pct: 49, want: character_sheet.WeightColorAmber},
		{pct: 50, want: character_sheet.WeightColorAmberOxblood}, // warning starts
		{pct: 70, want: character_sheet.WeightColorAmberOxblood},
		{pct: 89, want: character_sheet.WeightColorAmberOxblood},
		{pct: 90, want: character_sheet.WeightColorOxblood}, // critical starts
		{pct: 95, want: character_sheet.WeightColorOxblood},
		{pct: 100, want: character_sheet.WeightColorOxblood},
		{pct: 101, want: character_sheet.WeightColorOxbloodPulse}, // encumbered
		{pct: 150, want: character_sheet.WeightColorOxbloodPulse},
	}
	for _, tt := range tests {
		if got := character_sheet.WeightColor(tt.pct); got != tt.want {
			t.Errorf("WeightColor(%.0f%%) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

// Pulse predicate: starts at 90% (AC-10 95% pulses; QA AC-12 "pulse only
// at ≥90%"), off below it.
func TestWeightPulseThresholds(t *testing.T) {
	tests := []struct {
		pct  float64
		want bool
	}{
		{pct: 0, want: false},
		{pct: 49, want: false},
		{pct: 89, want: false},
		{pct: 90, want: true},
		{pct: 95, want: true},  // AC-10
		{pct: 100, want: true},
		{pct: 101, want: true},
	}
	for _, tt := range tests {
		if got := character_sheet.WeightPulse(tt.pct); got != tt.want {
			t.Errorf("WeightPulse(%.0f%%) = %v, want %v", tt.pct, got, tt.want)
		}
	}
}

// WeightPct clamps: over-capacity fills the bar fully (encumbrance is the
// pulse/color's job, not the fill's), negative and zero-capacity inputs
// are defensive zero.
func TestWeightPctClamping(t *testing.T) {
	tests := []struct {
		current, max int
		want         float64
	}{
		{current: 0, max: 170, want: 0},
		{current: 51, max: 170, want: 30},
		{current: 119, max: 170, want: 70},   // GDD example: STR 14, 119 lbs
		{current: 170, max: 170, want: 100},
		{current: 200, max: 170, want: 100},  // clamped
		{current: -5, max: 170, want: 0},     // defensive
		{current: 10, max: 0, want: 0},       // defensive: no Stats component
		{current: 10, max: -1, want: 0},      // defensive
	}
	for _, tt := range tests {
		got := character_sheet.WeightPct(tt.current, tt.max)
		if got != tt.want {
			t.Errorf("WeightPct(%d, %d) = %f, want %f", tt.current, tt.max, got, tt.want)
		}
	}
}

// Gold display: comma-grouped up to the cosmetic cap (GDD
// gold_display_cap = 999999); at and beyond the cap the display shows
// "999,999+" while the gold itself keeps accumulating (the cap is purely
// cosmetic).
func TestFormatGold(t *testing.T) {
	tests := []struct {
		amount int
		want   string
	}{
		{amount: 0, want: "0"},
		{amount: 999, want: "999"},
		{amount: 1000, want: "1,000"},
		{amount: 1234, want: "1,234"},
		{amount: 999999, want: "999,999"},
		{amount: 1000000, want: "999,999+"},  // at the cap
		{amount: 2500000, want: "999,999+"},  // beyond the cap
		{amount: -1, want: "0"},              // defensive: gold is never negative
	}
	for _, tt := range tests {
		if got := character_sheet.FormatGold(tt.amount); got != tt.want {
			t.Errorf("FormatGold(%d) = %q, want %q", tt.amount, got, tt.want)
		}
	}
}

// GoldDisplay view model composes FormatGold.
func TestGoldDisplayText(t *testing.T) {
	if got := (character_sheet.GoldDisplay{Amount: 1234}).Text(); got != "1,234" {
		t.Errorf("GoldDisplay{1234}.Text() = %q, want %q", got, "1,234")
	}
	if got := (character_sheet.GoldDisplay{Amount: 1000000}).Text(); got != "999,999+" {
		t.Errorf("GoldDisplay{1000000}.Text() = %q, want %q", got, "999,999+")
	}
}

// Stack counts: single items (Quantity 0/1 — equipment) show no count;
// stacks (consumables, materials) show the U+00D7 multiplication sign.
func TestStackCountLabel(t *testing.T) {
	tests := []struct {
		quantity int
		want     string
	}{
		{quantity: 0, want: ""},
		{quantity: 1, want: ""},
		{quantity: 2, want: "×2"},
		{quantity: 10, want: "×10"},
	}
	for _, tt := range tests {
		if got := character_sheet.StackCountLabel(tt.quantity); got != tt.want {
			t.Errorf("StackCountLabel(%d) = %q, want %q", tt.quantity, got, tt.want)
		}
	}
}

// Category mapping from item type strings: the five filter tabs by
// type.subtype prefix; types without a tab (accessory, readable, empty)
// appear under All only.
func TestCategoryFromType(t *testing.T) {
	tests := []struct {
		itemType string
		want     character_sheet.Category
	}{
		{itemType: "weapon.sword", want: character_sheet.CategoryWeapons},
		{itemType: "weapon.bow", want: character_sheet.CategoryWeapons},
		{itemType: "armor.head", want: character_sheet.CategoryArmor},
		{itemType: "consumable.potion", want: character_sheet.CategoryConsumables},
		{itemType: "quest.key", want: character_sheet.CategoryQuestItems},
		{itemType: "crafting.ore", want: character_sheet.CategoryCrafting},
		{itemType: "accessory", want: character_sheet.CategoryAll},
		{itemType: "readable", want: character_sheet.CategoryAll},
		{itemType: "", want: character_sheet.CategoryAll},
	}
	for _, tt := range tests {
		if got := character_sheet.CategoryFromType(tt.itemType); got != tt.want {
			t.Errorf("CategoryFromType(%q) = %v, want %v", tt.itemType, got, tt.want)
		}
	}
}

// Filter keys round-trip through the markup data-cat values; unknown keys
// are rejected.
func TestCategoryFromKey(t *testing.T) {
	for _, c := range []character_sheet.Category{
		character_sheet.CategoryAll, character_sheet.CategoryWeapons,
		character_sheet.CategoryArmor, character_sheet.CategoryConsumables,
		character_sheet.CategoryQuestItems, character_sheet.CategoryCrafting,
	} {
		got, ok := character_sheet.CategoryFromKey(c.Key())
		if !ok || got != c {
			t.Errorf("CategoryFromKey(%q) = (%v, %v), want (%v, true)", c.Key(), got, ok, c)
		}
	}
	if _, ok := character_sheet.CategoryFromKey("nope"); ok {
		t.Error("CategoryFromKey(\"nope\") = ok, want false")
	}
}

// CategoryFilter.Matches: category tab + case-insensitive name query
// compose; All matches every category; an empty query matches every name.
func TestCategoryFilterMatches(t *testing.T) {
	sword := character_sheet.Item{Name: "Steel Broadsword", Type: "weapon.sword"}
	potion := character_sheet.Item{Name: "Minor Healing Potion", Type: "consumable.potion"}
	query := character_sheet.CategoryFilter{Category: character_sheet.CategoryWeapons, Query: "steel"}
	if !query.Matches(sword) {
		t.Error("Weapons+steel should match Steel Broadsword")
	}
	if query.Matches(potion) {
		t.Error("Weapons+steel should not match a potion")
	}
	all := character_sheet.CategoryFilter{Category: character_sheet.CategoryAll, Query: ""}
	if !all.Matches(sword) || !all.Matches(potion) {
		t.Error("All+empty query should match every item")
	}
	caseInsensitive := character_sheet.CategoryFilter{Category: character_sheet.CategoryAll, Query: "BROAD"}
	if !caseInsensitive.Matches(sword) {
		t.Error("Query matching must be case-insensitive")
	}
	noMatch := character_sheet.CategoryFilter{Category: character_sheet.CategoryAll, Query: "zzz"}
	if noMatch.Matches(sword) {
		t.Error("Non-matching query should reject the item")
	}
}

// Grid sort: category order (weapons → armor → consumables → quest items →
// crafting), then alphabetically within a category; unknown types sort
// after the five categories.
func TestSortItemViews(t *testing.T) {
	views := []character_sheet.ItemView{
		{Item: character_sheet.Item{Name: "Iron Helm", Type: "armor.head"}},
		{Item: character_sheet.Item{Name: "Ancient Amulet", Type: "accessory"}},
		{Item: character_sheet.Item{Name: "Axe", Type: "weapon.axe"}},
		{Item: character_sheet.Item{Name: "Iron Ore", Type: "crafting.ore"}},
		{Item: character_sheet.Item{Name: "Bread", Type: "consumable.food"}},
	}
	character_sheet.SortItemViews(views)
	var order []string
	for _, v := range views {
		order = append(order, v.Item.Name)
	}
	want := []string{"Axe", "Iron Helm", "Bread", "Iron Ore", "Ancient Amulet"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("SortItemViews order = %v, want %v", order, want)
		}
	}
}

// TotalWeightOf: weight × quantity per item (a stack of three 2-lb
// potions weighs 6 lbs); Quantity ≤ 1 counts as one.
func TestTotalWeightOf(t *testing.T) {
	views := []character_sheet.ItemView{
		{Item: character_sheet.Item{Name: "Potion", Weight: 2, Quantity: 3}},
		{Item: character_sheet.Item{Name: "Sword", Weight: 5, Quantity: 0}}, // single
		{Item: character_sheet.Item{Name: "Ore", Weight: 1, Quantity: 10}},
	}
	if got := character_sheet.TotalWeightOf(views); got != 21 { // 6 + 5 + 10
		t.Errorf("TotalWeightOf = %d, want 21 (6 + 5 + 10)", got)
	}
}

// WeightBar label always pairs the percentage with the numbers (ADR-0012:
// never hue-only) — the label format is "current / max lbs (pct%)".
func TestWeightBarLabelPairsPctWithNumbers(t *testing.T) {
	label := (character_sheet.WeightBar{Current: 51, Max: 170}).Label()
	if label != "51 / 170 lbs (30%)" {
		t.Errorf("WeightBar{51, 170}.Label() = %q, want %q", label, "51 / 170 lbs (30%)")
	}
}
