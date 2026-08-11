// Inventory tab display logic and grid (Story 005, Epic: Character &
// Inventory UI): the item grid with category filter and search, the weight
// bar, the gold display, and the item detail panel per the GDD
// (design/gdd/character-inventory-ui.md) and ADR-0012.
//
// The file has two halves, matching the sheet's split between pure display
// helpers (xp_bar.go, compare.go) and engine-bound glue (sheet.go,
// drag_drop.go):
//
//   - Pure functions and view models — WeightPct, WeightColor, WeightPulse,
//     FormatGold, EmptyInventoryMessage, StackCountLabel, TotalWeightOf,
//     Category/CategoryFromType, CategoryFilter, WeightBar, GoldDisplay,
//     SortItemViews — are plain Go with no engine dependency, so they can be
//     unit-tested without the engine running (tests/unit/character_sheet/).
//   - InventoryGrid is the engine-bound presenter: it binds the markup
//     elements of the inventory tab, duplicates the hidden cell template
//     (document.DuplicateElementToParent) into a scrollable 6-column grid
//     (GDD tuning knob inventory_grid_columns = 6, cell size 64px), and
//     drives the weight bar fill, gold label, empty state, and detail panel.
//
// Weight bar rules (GDD "Weight Percentage", ADR-0012):
//
//	weight_pct = (current_carry_weight / max_carry_capacity) × 100
//	max_carry_capacity = (STR × 10) + 30 (ADR-0006; the sheet passes the
//	effective STR, so the bar always matches the Carry Capacity row)
//
// Color thresholds: 0-49% amber, 50-89% amber shifting to oxblood, 90-100%
// oxblood, >100% oxblood pulsing (encumbered). The percentage text is
// always visible next to the bar — color is never the sole information
// channel (control manifest: never encode game-critical information by hue
// alone). Amber is fill/accent only; the weight text is ink on parchment.
//
// AC-10 (95% weight pulses oxblood) conflicts with the GDD threshold table
// (pulse only at >100%) and QA AC-12 ("pulse only at ≥90%"): the GDD's own
// acceptance criterion and QA evidence both demand a pulse at 95%. The two
// concerns are split accordingly — WeightColor reports the band (oxblood at
// 95%), WeightPulse reports the pulse animation state (true at ≥90%) — so
// AC-10 (oxblood + pulse at 95%), QA AC-12 (no pulse at 70%), and the GDD
// threshold table (>100% is the "oxblood-pulse" token) all hold.
//
// Story: production/epics/character-inventory-ui/
// story-005-inventory-grid-weight-bar.md
package character_sheet

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"

	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup/document"

	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// Weight bar color tokens, returned by WeightColor and mapped to palette
// hex by the presenter (weightColorHex). Amber is the normal fill; the
// 50-89% band blends amber toward oxblood; 90-100% is oxblood; >100% is
// oxblood with the pulse animation (encumbered). The tokens are semantic —
// the UI layer owns the hex values.
const (
	// WeightColorAmber is the 0-49% fill (normal).
	WeightColorAmber = "amber"
	// WeightColorAmberOxblood is the 50-89% fill (warning transition —
	// a blend of amber and oxblood, never purely one or the other).
	WeightColorAmberOxblood = "amber-oxblood"
	// WeightColorOxblood is the 90-100% fill (near limit).
	WeightColorOxblood = "oxblood"
	// WeightColorOxbloodPulse is the >100% fill (encumbered — the pulse
	// animation runs; movement penalty applies per Character Stats).
	WeightColorOxbloodPulse = "oxblood-pulse"
)

// Weight bar tuning knobs (GDD: weight_bar_warning_pct = 50,
// weight_bar_critical_pct = 90).
const (
	weightBarWarningPct  = 50.0
	weightBarCriticalPct = 90.0
)

// WeightPct returns the weight bar fill percentage per the GDD formula:
// (current / max) × 100, clamped to [0, 100]. Over-capacity weights fill
// the bar fully — the encumbrance state is carried by WeightColor and
// WeightPulse, not by overflowing the fill. A max capacity ≤ 0 (an entity
// without a Stats component) is defensive zero: the bar shows 0%.
//
// Example:
//
//	character_sheet.WeightPct(119, 170) // 70 (GDD example: STR 14, 119 lbs)
func WeightPct(current, max int) float64 {
	if max <= 0 {
		return 0
	}
	pct := float64(current) / float64(max) * 100
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// WeightColor returns the weight bar fill color token for a fill
// percentage per the GDD thresholds: "amber" (0-49%), "amber-oxblood"
// (50-89%), "oxblood" (90-100%), "oxblood-pulse" (>100%). The percentage
// text always accompanies the color (ADR-0012: never hue-only) — the
// presenter renders the label from WeightBar.Label in the same pass.
//
// Example:
//
//	character_sheet.WeightColor(70) // "amber-oxblood" (AC-12)
func WeightColor(pct float64) string {
	switch {
	case pct < weightBarWarningPct:
		return WeightColorAmber
	case pct < weightBarCriticalPct:
		return WeightColorAmberOxblood
	case pct <= 100:
		return WeightColorOxblood
	}
	return WeightColorOxbloodPulse
}

// WeightPulse reports whether the weight bar shows the oxblood pulse
// animation at a fill percentage. The pulse starts at 90% — AC-10 (95%
// pulses) and QA AC-12 ("pulse only at ≥90%") — while WeightColor keeps
// the GDD threshold bands (90-100% is "oxblood", >100% "oxblood-pulse").
// The pulse is a Go-driven opacity tween (Kaiju CSS has no animations);
// reduced motion snaps it off (Story 009 wires the setting here).
//
// Example:
//
//	character_sheet.WeightPulse(95) // true (AC-10)
//	character_sheet.WeightPulse(70) // false (AC-11/12)
func WeightPulse(pct float64) bool {
	return pct >= weightBarCriticalPct
}

// FormatGold renders the gold amount for the inventory panel display:
// comma-grouped digits ("1,234"), capped cosmetically at "999,999+" beyond
// the display cap (GDD: gold_display_cap = 999999 — gold keeps
// accumulating past the cap; the cap is purely cosmetic). Negative amounts
// render as "0" (gold is never negative; the branch is purely defensive).
//
// Example:
//
//	character_sheet.FormatGold(1234)   // "1,234"
//	character_sheet.FormatGold(1000000) // "999,999+"
func FormatGold(amount int) string {
	if amount < 0 {
		return "0"
	}
	if amount >= 1000000 {
		return "999,999+"
	}
	return formatThousands(amount)
}

// EmptyInventoryMessage returns the grid's empty-inventory line per the
// GDD: "Your pack is empty." — shown centered in subdued ink when the
// character carries no items (AC-14). The gold display stays visible.
func EmptyInventoryMessage() string {
	return "Your pack is empty."
}

// StackCountLabel renders the grid cell's stack count (bottom-right corner,
// amber accent text, Story 005 Implementation Notes): "" for a single item
// (Quantity 0 or 1 — equipment), "×2" and up for stacks (consumables and
// crafting materials, GDD CR-1 stack_size). The "×" is the proper U+00D7
// multiplication sign, never the ASCII "x".
func StackCountLabel(quantity int) string {
	if quantity <= 1 {
		return ""
	}
	return fmt.Sprintf("×%d", quantity)
}

// Category is one entry of the inventory grid's category filter bar
// (Story 005 Implementation Notes): All, Weapons, Armor, Consumables, Quest
// Items, Crafting — the GDD CR-1 item categories, in the GDD sort order
// (weapons → armor → consumables → quest items → crafting materials).
type Category int

const (
	// CategoryAll shows every item regardless of category. Item types
	// without a filter tab of their own (accessory, readable, ...) appear
	// under All only.
	CategoryAll Category = iota
	// CategoryWeapons shows weapon.* items.
	CategoryWeapons
	// CategoryArmor shows armor.* items.
	CategoryArmor
	// CategoryConsumables shows consumable.* items.
	CategoryConsumables
	// CategoryQuestItems shows quest.* items.
	CategoryQuestItems
	// CategoryCrafting shows crafting.* items.
	CategoryCrafting
	// numCategories is the number of filter entries including All.
	numCategories
)

// categoryKeys are the canonical filter key strings: the markup id suffix
// ("inv-filter-{key}") and the data-cat attribute value.
var categoryKeys = [numCategories]string{
	"all", "weapons", "armor", "consumables", "questitems", "crafting",
}

// Key returns the canonical filter key ("all", "weapons", "questitems",
// ...): the markup id suffix and the data-cat attribute value.
func (c Category) Key() string {
	if c < CategoryAll || int(c) >= int(numCategories) {
		return ""
	}
	return categoryKeys[c]
}

// Label returns the filter bar display label ("All", "Quest Items", ...).
func (c Category) Label() string {
	switch c {
	case CategoryAll:
		return "All"
	case CategoryWeapons:
		return "Weapons"
	case CategoryArmor:
		return "Armor"
	case CategoryConsumables:
		return "Consumables"
	case CategoryQuestItems:
		return "Quest Items"
	case CategoryCrafting:
		return "Crafting"
	}
	return fmt.Sprintf("Category(%d)", int(c))
}

// CategoryFromKey maps a filter key string ("weapons") back to its
// Category. It reports false for unknown keys (a markup data-cat that does
// not match the filter bar).
func CategoryFromKey(key string) (Category, bool) {
	for c := 0; c < int(numCategories); c++ {
		if Category(c).Key() == key {
			return Category(c), true
		}
	}
	return CategoryAll, false
}

// CategoryFromType maps an item type string to its filter category by the
// type.subtype prefix: "weapon.*" → Weapons, "armor.*" → Armor,
// "consumable.*" → Consumables, "quest.*" → QuestItems, "crafting.*" →
// Crafting. Types without a tab (accessory, readable, and empty strings)
// map to CategoryAll — they appear under All only.
func CategoryFromType(itemType string) Category {
	switch {
	case strings.HasPrefix(itemType, ItemTypeWeaponPrefix):
		return CategoryWeapons
	case strings.HasPrefix(itemType, ItemTypeArmorPrefix):
		return CategoryArmor
	case strings.HasPrefix(itemType, ItemTypeConsumablePrefix):
		return CategoryConsumables
	case strings.HasPrefix(itemType, ItemTypeQuestPrefix):
		return CategoryQuestItems
	case strings.HasPrefix(itemType, ItemTypeCraftingPrefix):
		return CategoryCrafting
	}
	return CategoryAll
}

// CategoryFilter is the grid's active filter: a category tab plus a
// free-text name query from the search bar. Matches is the pure filter
// predicate, shared by the presenter and the unit tests.
type CategoryFilter struct {
	// Category is the active filter tab; CategoryAll disables the category
	// test.
	Category Category
	// Query is the search bar text; matched case-insensitively against the
	// item name. Empty disables the name test.
	Query string
}

// Matches reports whether an item passes the filter: it is in the active
// category (unless All) and its name contains the query (case-insensitive,
// trimmed; empty query matches everything).
func (f CategoryFilter) Matches(item Item) bool {
	if f.Category != CategoryAll && CategoryFromType(item.Type) != f.Category {
		return false
	}
	if q := strings.ToLower(strings.TrimSpace(f.Query)); q != "" {
		if !strings.Contains(strings.ToLower(item.Name), q) {
			return false
		}
	}
	return true
}

// WeightBar is the weight bar view model: the current carried weight and
// the max carry capacity in pounds. Pct and Color compose the GDD formula;
// Label renders the always-visible percentage text that pairs with the
// fill color (ADR-0012: never hue-only).
type WeightBar struct {
	// Current is the total carried weight in lbs (sum of item
	// Weight × Quantity).
	Current int
	// Max is the max carry capacity in lbs: (STR × 10) + 30 with the
	// effective STR (ADR-0006).
	Max int
}

// Pct returns the fill percentage, clamped to [0, 100] (WeightPct).
func (b WeightBar) Pct() float64 { return WeightPct(b.Current, b.Max) }

// Color returns the fill color token for the current percentage
// (WeightColor): amber, amber-oxblood, oxblood, or oxblood-pulse.
func (b WeightBar) Color() string { return WeightColor(b.Pct()) }

// Label renders the weight bar text — "161 / 170 lbs (95%)" — ink on
// parchment, always shown next to the fill so the bar is never read by
// color alone (ADR-0012).
func (b WeightBar) Label() string {
	return fmt.Sprintf("%d / %d lbs (%.0f%%)", b.Current, b.Max, b.Pct())
}

// GoldDisplay is the gold display view model for the inventory panel's
// top-right corner: the coin icon (markup) plus the formatted amount.
type GoldDisplay struct {
	// Amount is the carried gold from the Inventory component.
	Amount int
}

// Text renders the display string (FormatGold): "0", "1,234", "999,999+"
// at and beyond the cosmetic cap.
func (g GoldDisplay) Text() string { return FormatGold(g.Amount) }

// ItemView is one carried item's read model for the grid and the weight
// sum: the item entity ID and its Item component. The grid sorts and
// filters views; the weight bar sums them.
type ItemView struct {
	// ID is the item entity's GameEntityID (entity.Inventory.Items entry).
	ID entity.GameEntityID
	// Item is the item's Item component (Story 003 data model, extended
	// with Story 005 fields: Description, Weight, Value, Quantity).
	Item Item
}

// categoryOrder is the GDD grid sort order: weapons → armor → consumables
// → quest items → crafting materials. Types without a tab (CategoryAll
// from CategoryFromType) sort after the five categories.
var categoryOrder = [...]Category{
	CategoryWeapons, CategoryArmor, CategoryConsumables,
	CategoryQuestItems, CategoryCrafting,
}

// categoryRank maps a category to its sort position; unknown/All sorts
// last.
func categoryRank(c Category) int {
	for i, cat := range categoryOrder {
		if c == cat {
			return i
		}
	}
	return len(categoryOrder)
}

// SortItemViews orders the carried items per the GDD: by category
// (weapons → armor → consumables → quest items → crafting materials), then
// alphabetically by name within a category (case-insensitive). The sort is
// stable, so equal keys keep their inventory order.
func SortItemViews(views []ItemView) {
	sort.SliceStable(views, func(i, j int) bool {
		ri := categoryRank(CategoryFromType(views[i].Item.Type))
		rj := categoryRank(CategoryFromType(views[j].Item.Type))
		if ri != rj {
			return ri < rj
		}
		return strings.ToLower(views[i].Item.Name) < strings.ToLower(views[j].Item.Name)
	})
}

// TotalWeightOf returns the total carried weight of the given item views
// in pounds: the sum over items of Weight × Quantity (GDD CR-1: Wc = sum
// of all inventory item weights; a stack carries Quantity copies, so a
// stack of three 2-lb potions weighs 6 lbs). A Quantity ≤ 1 counts as one.
func TotalWeightOf(views []ItemView) int {
	total := 0
	for _, v := range views {
		qty := v.Item.Quantity
		if qty < 1 {
			qty = 1
		}
		total += v.Item.Weight * qty
	}
	return total
}

// TotalCarryWeight returns the total carried weight of the given inventory
// item IDs in pounds: TotalWeightOf over the items' Item components. Items
// without an Item component are skipped with a warning (matching
// TotalAttributeBonuses). The entity store is pure Go, so this stays
// unit-testable without the engine.
func TotalCarryWeight(store entity.EntityStore, itemIDs []entity.GameEntityID) int {
	views := make([]ItemView, 0, len(itemIDs))
	for _, id := range itemIDs {
		item, ok := entity.GetComponent[Item](store, id)
		if !ok {
			slog.Warn("character sheet: inventory item has no Item component",
				"item", id)
			continue
		}
		views = append(views, ItemView{ID: id, Item: item})
	}
	return TotalWeightOf(views)
}

// itemViewsContain reports whether an item ID is present in the views.
func itemViewsContain(views []ItemView, id entity.GameEntityID) bool {
	for _, v := range views {
		if v.ID == id {
			return true
		}
	}
	return false
}

// itemStatsText renders the detail panel's stats list of an item: one line
// per nonzero attribute bonus ("STR +2", "CON +1"), newline-separated.
// Damage and armor lines arrive with the full loot-economy item record;
// the Story 003 Item component carries attribute bonuses only (the
// comparison's ItemStats in compare.go is a display view, not stored
// state).
func itemStatsText(item Item) string {
	var b strings.Builder
	for i := 0; i < stats.NumAttributes; i++ {
		if bonus := item.AttributeBonuses[i]; bonus != 0 {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "%s %s", attrLabels[i], ModifierDisplay(bonus))
		}
	}
	return b.String()
}

// weightColorHex maps the WeightColor tokens to the art-bible palette hex
// values (ADR-0012 desk register). The amber-oxblood band is the midpoint
// blend of amber #D9A441 and oxblood #7A2E28, so it is visually distinct
// from both pure colors (QA AC-12). Unknown tokens fall back to amber.
func weightColorHex(token string) string {
	switch token {
	case WeightColorAmber:
		return "#D9A441"
	case WeightColorAmberOxblood:
		return "#A96934" // amber → oxblood midpoint blend
	case WeightColorOxblood, WeightColorOxbloodPulse:
		return "#7A2E28"
	}
	return "#D9A441"
}

// cellBinding is one rendered grid cell: the duplicated markup element and
// the item it displays. The element's data-index attribute maps back to
// the binding (markup event handlers carry only the element, no closures).
type cellBinding struct {
	cell   *document.Element
	itemID entity.GameEntityID
}

// InventoryGrid is the inventory tab presenter (Story 005): it binds the
// inventory markup elements, renders the scrollable 6×64px item grid by
// duplicating the hidden cell template (document.DuplicateElementToParent,
// Kaiju's dynamic-list pattern), applies the category filter and search
// query, and drives the weight bar, gold display, empty state, and item
// detail panel. The sheet owns its lifecycle: Populate on every sheet
// populate, Update in the host updater for the weight pulse, Reset on
// Open. It reads the entity's Inventory component and the items' Item
// components only — it never writes entity state (ADR-0001).
type InventoryGrid struct {
	doc   *document.Document
	store entity.EntityStore

	// panel is the inventory tab panel ("inv-panel"); the sheet's showTab
	// toggles it against the character-tab columns.
	panel *document.Element
	// grid is the scrollable 6-column container ("inv-grid").
	grid *document.Element
	// tpl is the hidden grid cell template ("inv-cell-tpl"); every item
	// cell is a duplicated copy of it.
	tpl *document.Element
	// cells are the currently rendered item cells, one per filtered item.
	cells []cellBinding

	// empty is the "Your pack is empty." label ("inv-empty"), shown
	// centered over the grid when the character carries no items (AC-14).
	empty *document.Element

	// search is the search bar input ("inv-search"); typing refilters the
	// grid by item name.
	search *ui.Input

	// filterCells are the six category filter cells ("inv-filter-{key}");
	// the active one carries the amber underline accent class
	// (inv-filter-active — amber is accent, never text, ADR-0012).
	filterCells [numCategories]*document.Element
	// activeFilter is the selected category tab (default All).
	activeFilter Category
	// query is the current search text (case-insensitive name filter).
	query string

	// weightRow is the weight bar row ("inv-weight-row"), hidden when the
	// pack is empty (AC-14); weightFill is the amber/oxblood fill panel
	// ("inv-weight-fill"); weightLabel is the ink percentage text
	// ("inv-weight-label").
	weightRow   *document.Element
	weightFill  *document.Element
	weightLabel *ui.Label

	// goldLabel is the gold amount label ("inv-gold-value"), top-right of
	// the inventory panel.
	goldLabel *ui.Label

	// Detail panel labels (right side): name, description, stats list,
	// weight, value.
	detailName   *ui.Label
	detailDesc   *ui.Label
	detailStats  *ui.Label
	detailWeight *ui.Label
	detailValue  *ui.Label

	// currentWeight is the total carried weight (lbs) of the last Populate;
	// maxCapacity is the effective max carry capacity ((STR × 10) + 30 with
	// equipment bonuses — the same value as the Carry Capacity derived row).
	currentWeight int
	maxCapacity   int
	// views are the full carried item list (sorted, unfiltered); the grid
	// re-renders from it on filter/search changes without re-reading the
	// store.
	views []ItemView
	// selectedID is the item shown in the detail panel; 0 when none.
	selectedID entity.GameEntityID

	// pulseT advances the weight bar's oxblood pulse phase in Update;
	// visible gates the pulse to the open inventory tab; reducedMotion
	// disables it (Story 009 wires the accessibility setting here).
	pulseT        float64
	visible       bool
	reducedMotion bool
}

// NewInventoryGrid binds the inventory tab markup elements for the sheet's
// document. Missing elements are logged and left nil; every update path
// skips nil elements, so a markup mismatch degrades to a no-op inventory
// tab instead of a crash (matching the rest of the sheet).
func NewInventoryGrid(doc *document.Document, store entity.EntityStore) *InventoryGrid {
	g := &InventoryGrid{doc: doc, store: store}
	g.panel = sheetElement(doc, "inv-panel")
	g.grid = sheetElement(doc, "inv-grid")
	g.tpl = sheetElement(doc, "inv-cell-tpl")
	g.empty = sheetElement(doc, "inv-empty")
	if elm, ok := doc.GetElementById("inv-search"); ok && elm.UI != nil && elm.UI.IsType(ui.ElementTypeInput) {
		g.search = elm.UI.ToInput()
	} else {
		slog.Warn("character sheet: markup element not found", "id", "inv-search")
	}
	for c := 0; c < int(numCategories); c++ {
		g.filterCells[c] = sheetElement(doc, "inv-filter-"+Category(c).Key())
	}
	g.weightRow = sheetElement(doc, "inv-weight-row")
	g.weightFill = sheetElement(doc, "inv-weight-fill")
	g.weightLabel = sheetLabel(doc, "inv-weight-label")
	g.goldLabel = sheetLabel(doc, "inv-gold-value")
	g.detailName = sheetLabel(doc, "inv-detail-name")
	g.detailDesc = sheetLabel(doc, "inv-detail-desc")
	g.detailStats = sheetLabel(doc, "inv-detail-stats")
	g.detailWeight = sheetLabel(doc, "inv-detail-weight")
	g.detailValue = sheetLabel(doc, "inv-detail-value")
	return g
}

// Populate re-reads the owner's Inventory component and the items' Item
// components, sorts the pack by category then name, and re-renders the
// grid, weight bar, gold display, and empty state. maxCapacity is the
// effective max carry capacity ((STR × 10) + 30 with equipment bonuses).
// The sheet calls this from its populate on Open and on every Refresh, so
// an equip/unequip lands on screen in the same frame. An entity without an
// Inventory component shows an empty pack (the sheet's populate already
// warns about the missing component).
func (g *InventoryGrid) Populate(owner entity.GameEntityID, maxCapacity int) {
	g.maxCapacity = maxCapacity
	g.views = g.views[:0]
	g.currentWeight = 0
	if inv, ok := entity.GetComponent[entity.Inventory](g.store, owner); ok {
		g.currentWeight = TotalCarryWeight(g.store, inv.Items)
		if g.goldLabel != nil {
			g.goldLabel.SetText(FormatGold(inv.Gold))
		}
		for _, id := range inv.Items {
			item, ok := entity.GetComponent[Item](g.store, id)
			if !ok {
				slog.Warn("character sheet: inventory item has no Item component",
					"item", id)
				continue
			}
			g.views = append(g.views, ItemView{ID: id, Item: item})
		}
	} else if g.goldLabel != nil {
		g.goldLabel.SetText("0")
	}
	SortItemViews(g.views)
	g.applyFilter()
}

// SetCategory selects a filter tab and re-renders the grid. Unknown
// categories are ignored.
func (g *InventoryGrid) SetCategory(cat Category) {
	if cat < CategoryAll || int(cat) >= int(numCategories) || g.activeFilter == cat {
		return
	}
	g.activeFilter = cat
	g.refreshFilterCells()
	g.applyFilter()
}

// SetQuery updates the search text and re-renders the grid. The query
// matches item names case-insensitively.
func (g *InventoryGrid) SetQuery(query string) {
	if g.query == query {
		return
	}
	g.query = query
	g.applyFilter()
}

// Select shows the item at grid index idx in the detail panel and
// highlights its cell with the amber selection border (click-to-select,
// Story 005 Implementation Notes). Out-of-range indices are ignored.
func (g *InventoryGrid) Select(idx int) {
	if idx < 0 || idx >= len(g.cells) {
		return
	}
	g.selectItem(g.cells[idx].itemID)
}

// selectItem shows the given item in the detail panel and re-applies the
// selection highlight across the rendered cells. An item no longer in the
// pack clears the detail panel.
func (g *InventoryGrid) selectItem(id entity.GameEntityID) {
	g.selectedID = id
	for _, c := range g.cells {
		classes := []string{"inv-cell"}
		if c.itemID == id {
			classes = append(classes, "inv-cell-selected")
		}
		c.cell.SetClasses(classes...)
	}
	for i := range g.views {
		if g.views[i].ID != id {
			continue
		}
		v := g.views[i]
		if g.detailName != nil {
			g.detailName.SetText(v.Item.Name)
		}
		if g.detailDesc != nil {
			g.detailDesc.SetText(v.Item.Description)
		}
		if g.detailStats != nil {
			g.detailStats.SetText(itemStatsText(v.Item))
		}
		if g.detailWeight != nil {
			g.detailWeight.SetText(fmt.Sprintf("%d lbs", v.Item.Weight))
		}
		if g.detailValue != nil {
			g.detailValue.SetText(fmt.Sprintf("%s gold", FormatGold(v.Item.Value)))
		}
		return
	}
	g.clearDetail()
}

// clearDetail resets the detail panel and drops the selection.
func (g *InventoryGrid) clearDetail() {
	g.selectedID = 0
	if g.detailName != nil {
		g.detailName.SetText("—")
	}
	for _, l := range []*ui.Label{g.detailDesc, g.detailStats, g.detailWeight, g.detailValue} {
		if l != nil {
			l.SetText("")
		}
	}
}

// applyFilter re-renders the grid from the cached views under the active
// category and search query, then refreshes the weight bar, the empty
// state, and the detail selection. The weight bar always sums the full
// pack — filtering never changes the carried weight.
func (g *InventoryGrid) applyFilter() {
	filter := CategoryFilter{Category: g.activeFilter, Query: g.query}
	shown := make([]ItemView, 0, len(g.views))
	for _, v := range g.views {
		if filter.Matches(v.Item) {
			shown = append(shown, v)
		}
	}
	// A selection filtered out of the grid clears the detail panel before
	// the cells rebuild, so no stale highlight survives.
	if !itemViewsContain(shown, g.selectedID) {
		g.clearDetail()
	}
	g.renderCells(shown)
	g.refreshWeightBar()
	g.refreshEmpty()
}

// renderCells rebuilds the grid: every previously rendered cell is removed
// and one cell is duplicated from the hidden template per filtered item,
// carrying its stack-count label and data-index attribute (the markup
// invSelect handler resolves the index back to the binding). The selection
// highlight is re-applied from selectedID.
func (g *InventoryGrid) renderCells(views []ItemView) {
	if g.doc == nil || g.tpl == nil {
		return
	}
	for _, c := range g.cells {
		g.doc.RemoveElement(c.cell)
	}
	g.cells = g.cells[:0]
	for i, v := range views {
		cell := g.doc.DuplicateElementToParent(g.tpl, g.grid)
		cell.SetAttribute("data-index", strconv.Itoa(i))
		if lbl := cellCountLabel(cell); lbl != nil {
			lbl.SetText(StackCountLabel(v.Item.Quantity))
		}
		classes := []string{"inv-cell"}
		if v.ID == g.selectedID {
			classes = append(classes, "inv-cell-selected")
		}
		cell.SetClasses(classes...)
		g.cells = append(g.cells, cellBinding{cell: cell, itemID: v.ID})
	}
}

// cellCountLabel returns the stack-count label of a grid cell — the child
// element with class "inv-cell-count" (the template's bottom-right label).
func cellCountLabel(cell *document.Element) *ui.Label {
	for _, c := range cell.Children {
		if c.HasClass("inv-cell-count") {
			return c.InnerLabel()
		}
	}
	return nil
}

// refreshWeightBar writes the weight bar from the cached pack weight and
// max capacity: the row is hidden when the pack is empty (AC-14: no weight
// bar, gold display remains), the fill gets its width and palette color
// (amber → amber-oxblood → oxblood, WeightColor), and the ink label always
// carries the percentage text alongside the color (ADR-0012: never
// hue-only). The pulse opacity starts at full; Update oscillates it.
func (g *InventoryGrid) refreshWeightBar() {
	hasItems := len(g.views) > 0
	if g.doc != nil && g.weightRow != nil {
		display := "none"
		if hasItems {
			display = "flex"
		}
		g.doc.SetElementStyleProperty(g.weightRow, "display", display)
	}
	if g.doc != nil && g.weightFill != nil {
		bar := WeightBar{Current: g.currentWeight, Max: g.maxCapacity}
		g.doc.SetElementStyleProperty(g.weightFill, "width", fmt.Sprintf("%.1f%%", bar.Pct()))
		g.doc.SetElementStyleProperty(g.weightFill, "background-color", weightColorHex(bar.Color()))
		g.doc.SetElementStyleProperty(g.weightFill, "opacity", "1")
	}
	if g.weightLabel != nil {
		g.weightLabel.SetText(WeightBar{Current: g.currentWeight, Max: g.maxCapacity}.Label())
	}
}

// refreshEmpty toggles the "Your pack is empty." label (AC-14): shown
// centered over the grid area when the pack holds no items at all. A
// filter or search that hides every item of a non-empty pack leaves a
// blank grid — the message would be wrong there.
func (g *InventoryGrid) refreshEmpty() {
	if g.doc == nil || g.empty == nil {
		return
	}
	display := "none"
	if len(g.views) == 0 {
		display = "flex"
	}
	g.doc.SetElementStyleProperty(g.empty, "display", display)
}

// refreshFilterCells re-applies the amber underline accent to the active
// category filter cell (amber is accent, never text — ADR-0012).
func (g *InventoryGrid) refreshFilterCells() {
	for c := 0; c < int(numCategories); c++ {
		cell := g.filterCells[c]
		if cell == nil {
			continue
		}
		classes := []string{"inv-filter"}
		if Category(c) == g.activeFilter {
			classes = append(classes, "inv-filter-active")
		}
		cell.SetClasses(classes...)
	}
}

// pulseActive reports whether the weight bar pulse animation should run:
// WeightPulse at ≥90% of max (AC-10: 95% pulses; QA AC-12: no pulse at
// 70%).
func (g *InventoryGrid) pulseActive() bool {
	return WeightPulse(WeightPct(g.currentWeight, g.maxCapacity))
}

// Update advances the weight bar's oxblood pulse: the fill's opacity
// breathes 0.55 → 1.0 on a 1.2s cycle (Kaiju CSS has no animations — this
// is the Go-driven pattern of the level-up modal's hover pulse, Story
// 002). The pulse runs only while the inventory tab is visible, the pack
// is at ≥90% weight, and reduced motion is off. Called from the sheet's
// Update.
func (g *InventoryGrid) Update(dt float64) {
	if g.doc == nil || g.weightFill == nil || !g.visible || g.reducedMotion || !g.pulseActive() {
		g.pulseT = 0
		return
	}
	g.pulseT += dt
	const period = 1.2 // seconds per full breathe cycle
	a := 0.55 + 0.45*math.Abs(math.Sin(g.pulseT*math.Pi*2/period))
	g.doc.SetElementStyleProperty(g.weightFill, "opacity", fmt.Sprintf("%.3f", a))
}

// SetVisible shows or hides the inventory tab panel via the root's inline
// display property (hidden by default in the markup). The sheet's tab
// switch calls this; Story 007 owns the 200ms crossfade and the C/I key
// handling.
func (g *InventoryGrid) SetVisible(visible bool) {
	g.visible = visible
	g.pulseT = 0
	if g.doc == nil || g.panel == nil {
		return
	}
	display := "none"
	if visible {
		display = "flex"
	}
	g.doc.SetElementStyleProperty(g.panel, "display", display)
}

// SetReducedMotion disables the weight bar pulse when on (Story 009 wires
// the accessibility setting here) and settles the fill at full opacity.
func (g *InventoryGrid) SetReducedMotion(on bool) {
	g.reducedMotion = on
	g.pulseT = 0
	if on && g.doc != nil && g.weightFill != nil {
		g.doc.SetElementStyleProperty(g.weightFill, "opacity", "1")
	}
}

// Reset returns the tab to its default state: All category, empty search,
// no selection. The sheet calls it on Open so a session never carries a
// stale filter across viewings. The tab itself (character vs. inventory)
// is Story 007's session state.
func (g *InventoryGrid) Reset() {
	g.activeFilter = CategoryAll
	g.query = ""
	g.selectedID = 0
	if g.search != nil {
		g.search.SetTextWithoutEvent("")
	}
	g.refreshFilterCells()
}

// inventoryFuncMap returns the inventory tab's markup handler map (Story
// 005), dispatched through the markup loader's funcMap. The handlers read
// their state lazily from the sheet and no-op while the grid is nil (it is
// created after the document loads). Markup contract:
//
//	onclick="sheetTabSelect" — tab cell, data-tab="character"|"inventory"
//	onclick="invFilter"      — filter cell, data-cat="{Category.Key()}"
//	onclick="invSelect"      — grid cell, data-index="{cell index}"
//	onchange/onkeyup="invSearch" — search input (reads the input text)
func inventoryFuncMap(s *CharacterSheet) map[string]func(*document.Element) {
	return map[string]func(*document.Element){
		"sheetTabSelect": func(elm *document.Element) {
			s.showTab(elm.Attribute("data-tab"))
		},
		"invFilter": func(elm *document.Element) {
			if s.inventory == nil {
				return
			}
			if cat, ok := CategoryFromKey(elm.Attribute("data-cat")); ok {
				s.inventory.SetCategory(cat)
			}
		},
		"invSelect": func(elm *document.Element) {
			if s.inventory == nil {
				return
			}
			idx, err := strconv.Atoi(elm.Attribute("data-index"))
			if err != nil {
				slog.Warn("character sheet: inventory cell without data-index", "err", err)
				return
			}
			s.inventory.Select(idx)
		},
		"invSearch": func(elm *document.Element) {
			if s.inventory == nil || elm.UI == nil || !elm.UI.IsType(ui.ElementTypeInput) {
				return
			}
			s.inventory.SetQuery(elm.UI.ToInput().Text())
		},
	}
}
