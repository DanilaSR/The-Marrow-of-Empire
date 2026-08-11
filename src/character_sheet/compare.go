// Equipment stat comparison display logic (Story 004, Epic: Character &
// Inventory UI): comparing the equipped item's stats against a selected
// inventory item of the same slot per the GDD formula
// (design/gdd/character-inventory-ui.md, "Equipment Stat Comparison"):
//
//	stat_diff = equipped_item_stat − compared_item_stat
//
// Positive diff means the equipped item is better (shown in amber with "+"),
// negative means the compared item is better (shown in green with "−",
// U+2212), zero means equal (shown in ink with no sign). Color is never the
// sole differentiator — the sign and numeric value always convey the
// direction independently of color (ADR-0012). The amber diff text is a
// discrete accent indicator, not body text, so it stays clear of the
// control-manifest amber-on-parchment prohibition (which covers body text).
//
// ItemStats is the display view of an item's numeric combat stats (damage,
// armor, attribute bonuses); the item data model itself (Story 003's Item
// component in equipment.go) is out of scope here, as are non-numeric
// "special properties" (GDD) — Story 005's item detail panel lists those
// textually. This file only composes the diff math for display; equipment
// slots come from equipment.go (Story 003).
//
// Like the rest of the sheet's display helpers (AttributeDisplay,
// DerivedStats) these functions are pure Go with no engine dependency, so
// they can be unit-tested without the engine running
// (tests/unit/character_sheet/).
//
// Story: production/epics/character-inventory-ui/
// story-004-equipment-stat-comparison.md
package character_sheet

import (
	"fmt"

	"marrow-of-empire/stats"
)

// Color tokens for the comparison diff text, returned by StatDiff and
// carried on StatComparison. They are semantic tokens — the UI layer
// (Story 005) maps them to palette colors (amber accent, green gain, ink =
// parchment-dark).
const (
	// ColorAmber marks a positive diff: the equipped item is better. Amber
	// is an accent indicator for the diff text, never body text
	// (control-manifest: no amber-on-parchment body text).
	ColorAmber = "amber"
	// ColorGreen marks a negative diff: the compared item is better.
	ColorGreen = "green"
	// ColorInk marks an equal diff (parchment-dark ink).
	ColorInk = "ink"
)

// ItemStats is the display view of an item's numeric combat-relevant stats
// for the comparison panel: the equipment slot it occupies (SlotType from
// equipment.go, Story 003), Damage for weapons, Armor for armor pieces, and
// the attribute bonuses (accessories and enchanted gear) in the same
// [stats.NumAttributes]int shape as the Story 003 Item component. The full
// item data model (name, description, weight, value, special properties)
// belongs to Story 003's item model and Story 005's detail panel; this
// struct carries only what the comparison math needs.
type ItemStats struct {
	Slot             SlotType                 // equipment slot the item occupies (equipment.go)
	Damage           int                      // weapon damage (Main Hand / Off Hand)
	Armor            int                      // defense (Head, Chest, Hands, Legs)
	AttributeBonuses [stats.NumAttributes]int // attribute bonuses, indexed by stats attribute
}

// StatComparison is one stat row of the equipment comparison panel: the
// stat name, the equipped item's value (Se), and the diff of that stat
// between the two items. FormatComparison renders it.
type StatComparison struct {
	// Name is the display label: "Damage", "Armor", "STR"….
	Name string
	// Equipped is the equipped item's stat value (Se).
	Equipped int
	// Delta is Se − Sc per the GDD formula.
	Delta int
	// Sign is "+" (equipped better), "−" (U+2212, compared better), or ""
	// (equal). The sign is always present for nonzero deltas — color is
	// never the sole differentiator (ADR-0012).
	Sign string
	// Color is ColorAmber | ColorGreen | ColorInk.
	Color string
}

// StatDiff computes the stat difference between the equipped item and the
// compared item per the GDD formula: stat_diff = equipped_item_stat −
// compared_item_stat. It returns the delta, its display sign, and the diff
// color token:
//
//   - delta > 0: equipped is better → "+", ColorAmber
//   - delta < 0: compared is better → "−" (U+2212), ColorGreen
//   - delta == 0: equal → no sign, ColorInk
//
// Example (AC-8): equipped sword +3 damage vs compared axe +5 damage →
// (−2, "−", "green").
func StatDiff(equipped, compared int) (delta int, sign string, color string) {
	delta = equipped - compared
	switch {
	case delta > 0:
		return delta, "+", ColorAmber
	case delta < 0:
		return delta, minusSign, ColorGreen
	}
	return 0, "", ColorInk
}

// attrLabels are the display labels of the six attributes in canonical
// order [STR, DEX, CON, INT, WIS, CHA] (ADR-0006), used as the comparison
// row names for attribute bonuses.
var attrLabels = [stats.NumAttributes]string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}

// CompareItems returns the stat comparisons between an equipped item and an
// inventory item of the same slot, one StatComparison per stat row in
// canonical order: Damage, Armor, then the six attribute bonuses. A stat row
// appears only when at least one item has a nonzero value for it (a sword vs
// an axe shows Damage, not "Armor: 0 → 0" noise). It returns nil when the
// items occupy different slots — only same-slot items trigger a comparison
// (GDD CompareMode; QA: different slot type → no comparison shown).
func CompareItems(equipped, compared ItemStats) []StatComparison {
	if equipped.Slot != compared.Slot {
		return nil
	}
	rows := make([]StatComparison, 0, 2+stats.NumAttributes)
	appendRow := func(name string, e, c int) {
		if e == 0 && c == 0 {
			return
		}
		delta, sign, color := StatDiff(e, c)
		rows = append(rows, StatComparison{
			Name: name, Equipped: e, Delta: delta, Sign: sign, Color: color,
		})
	}
	appendRow("Damage", equipped.Damage, compared.Damage)
	appendRow("Armor", equipped.Armor, compared.Armor)
	for i := 0; i < stats.NumAttributes; i++ {
		appendRow(attrLabels[i], equipped.AttributeBonuses[i], compared.AttributeBonuses[i])
	}
	return rows
}

// FormatComparison renders one stat comparison row per the GDD display
// format: the stat name, the equipped item's value with its sign, and the
// signed diff, e.g. "Damage: +3 → −2" for an equipped sword (+3 damage)
// compared to an axe (+5 damage) — the axe is better by 2 (AC-8). Equal
// items render without a sign: "Damage: +3 → 0" (GDD: zero shows no sign).
// The direction is carried by the sign, never by color alone (ADR-0012);
// the Color field supplies the diff color token for the UI layer.
func FormatComparison(sc StatComparison) string {
	mag := sc.Delta
	if mag < 0 {
		mag = -mag
	}
	return fmt.Sprintf("%s: %s → %s%d", sc.Name, signedValue(sc.Equipped), sc.Sign, mag)
}

// signedValue renders an item stat value with an explicit sign for nonzero
// values: positive gets "+", negative the proper minus sign (U+2212, not a
// hyphen), zero renders plain "0" (an absent bonus is just 0).
func signedValue(v int) string {
	if v < 0 {
		return fmt.Sprintf("%s%d", minusSign, -v)
	}
	if v > 0 {
		return fmt.Sprintf("+%d", v)
	}
	return "0"
}
