// Package character_sheet implements the character sheet stat display
// (Story 001, Epic: Character & Inventory UI): a Desk-register full-screen
// parchment overlay rendered through Kaiju's HTML/CSS-like UI system
// (ADR-0012).
//
// The sheet is read-only: it displays the six attributes with modifiers and
// the five derived stats of the viewed entity's Stats component (ADR-0001,
// ADR-0006) and never writes entity state. Story 002 adds the XP progress
// bar (bottom of the center panel) and the mandatory level-up modal
// overlay (level_up.go); the modal is the one exception to read-only —
// its Confirm writes the attribute point allocation back per the GDD.
// Story 003 adds the eight equipment slots in the bottom strip
// (equipment.go, drag_drop.go): equipping, unequipping, and moving items
// write the entity's Inventory component through the Entity Framework
// (ADR-0001), and the displayed attributes and derived stats recompute with
// the equipped items' attribute bonuses in the same frame (AC-6, AC-9). The
// display formatting functions (ModifierDisplay, AttributeDisplay,
// BarFillFraction, DerivedStats, DerivedValueDisplay, XPProgressPct,
// XPProgressDisplay, BonusDisplay, AttributeBaseDisplay,
// AttributeDisplayWithBonus) are pure Go with no engine dependency so they
// can be unit-tested without the engine running
// (tests/unit/character_sheet/).
//
// Story 005 adds the Inventory tab (inventory.go): a tab bar under the
// title switches the sheet between the character columns and the inventory
// panel (scrollable 6-column item grid with category filter and search,
// weight bar, gold display, and item detail panel). The tab chrome lives
// here (showTab); Story 007 owns the 200ms crossfade and the C/I key
// handling. The inventory grid reads the entity's Inventory and Item
// components only — it never writes entity state.
//
// Story 006 adds the toast notification overlay (toast.go): a reusable
// timed notification (queue, countdown, fade-out, reduced motion) rendered
// as a sibling of the sheet overlay — amber text on dark umber, never
// amber on parchment (control manifest) — that stays visible while the
// sheet is closed. The game layer fires it through ToastView() when
// PickupBlocked refuses a world-item pickup (AC-13: "Inventory full" for
// 2s, the item stays in the world).
//
// Motion follows the desk paper-slide language (TR-ui-004): 300ms ease-out
// enter, 200ms exit, translateX + opacity only. Enter and Exit are the
// animation hooks the screen state machine (Story 007) calls; this story
// does not own screen transitions.
//
// Story 008 adds the party sidebar and companion inspect (party_sidebar.go):
// the left 20% panel with party portraits (player at top, companions in
// party order, pool companions greyed and unselectable), the 300ms companion
// tooltip (name, level, role, HP bar, trust indicator, Inspect button),
// companion inspect — the Trust & Relationship section and the Knowledge
// Domains list in the center panel (AC-21) — and the AC-19 departure
// handling: when the viewed companion leaves the party (event
// party.member_departed or combat death), the sheet switches to the player
// character immediately and any level-up modal showing the departed
// companion closes. SwitchMember preserves the active tab across member
// switches (AC-22). The sidebar doubles as the level-up modal's PartyRoster
// (party order for simultaneous level-ups). Keyboard navigation and
// accessibility arrive with Story 009.
//
// Story 009 adds keyboard navigation and accessibility (accessibility.go):
// the AC-23 tab order (sidebar → attributes → equipment slots → tabs →
// close button, wrapping at both ends), in-group arrow keys, the oxblood
// focus ring (≥3:1 contrast, ADR-0012), the reduce-motion gate (AC-24)
// wired to every component, the 1280×720 resolution floor with its lock
// overlay (AC-25), and the close-button/Esc user-close path
// (SetCloseHandler).
//
// All player-facing strings are English placeholders pending localization,
// matching the title screen convention (the localization system does not
// exist yet).
package character_sheet

import (
	"fmt"
	"log/slog"
	"strconv"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup"
	"kaijuengine.com/engine/ui/markup/document"

	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// markupPath is the character sheet markup asset relative to the asset
// database root ("assets/ui/character_sheet.html").
const markupPath = "ui/character_sheet.html"

// maxAttributeValue is the top of the attribute range (ADR-0006,
// TR-stats-001: attributes range 3-18).
const maxAttributeValue = 18

// numDerivedStats is the number of derived stat rows on the right panel:
// Max Health, Action Points, Initiative, Evasion, Carry Capacity.
const numDerivedStats = 5

// slideDistance is the paper-slide travel distance in pixels: the parchment
// panel slides in from the right on enter and back out on exit.
const slideDistance = 48

// enterDuration is the paper-slide entrance duration (TR-ui-004:
// 200-300ms ease-out).
const enterDuration = 0.300

// exitDuration is the paper-slide exit duration (TR-ui-004: 200ms).
const exitDuration = 0.200

// minusSign is the proper Unicode minus sign (U+2212) used for negative
// modifiers and values — never the ASCII hyphen (AC-5).
const minusSign = "−"

// attributeRowIDs are the markup row ids of the six attribute rows in
// canonical order [STR, DEX, CON, INT, WIS, CHA], matching the
// stats.Attributes indices. Each row's value label and bar fill are
// "{id}-value" and "{id}-fill".
var attributeRowIDs = [stats.NumAttributes]string{
	"attr-str", "attr-dex", "attr-con", "attr-int", "attr-wis", "attr-cha",
}

// derivedRowIDs are the value label id suffixes of the five derived rows in
// DerivedStats order; full ids are "derived-{suffix}-value".
var derivedRowIDs = [numDerivedStats]string{
	"maxhp", "ap", "initiative", "evasion", "carry",
}

// ModifierDisplay formats an attribute modifier with an explicit sign:
// positives and zero show "+", negatives use the proper minus sign (U+2212,
// not a hyphen). Examples: 3 → "+3", -2 → "−2", 0 → "+0" (AC-5).
func ModifierDisplay(mod int) string {
	if mod < 0 {
		return fmt.Sprintf("%s%d", minusSign, -mod)
	}
	return fmt.Sprintf("+%d", mod)
}

// AttributeDisplay formats an attribute value with its modifier per the GDD
// display format "{value} ({sign}{modifier})" (AC-1, AC-5): STR 16 → "16
// (+3)", STR 7 → "7 (−2)", STR 10 → "10 (+0)". The modifier comes from
// stats.Modifier (floor((A-10)/2) clamped [-4,+4], ADR-0006).
func AttributeDisplay(value int) string {
	return fmt.Sprintf("%d (%s)", value, ModifierDisplay(stats.Modifier(value)))
}

// BarFillFraction returns the attribute bar fill fraction for a value in the
// attribute range 3-18: value/18, linear through zero, clamped to [0, 1].
// Bars are therefore proportional to the attribute value (AC-3): STR 16
// fills 16/18 ≈ 0.889 and CHA 8 fills 8/18 ≈ 0.444 — an exact 2:1 width
// ratio. (A normalized (value-3)/15 mapping would yield 2.6 and fail AC-3.)
// The clamp keeps buffs above 18 from overflowing the track.
func BarFillFraction(value int) float64 {
	f := float64(value) / float64(maxAttributeValue)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// DerivedStat is one derived stat row on the right-hand panel.
type DerivedStat struct {
	// Name is the display label, e.g. "Max Health".
	Name string
	// Value is the computed value from the Stats component.
	Value int
	// Unit is the display suffix ("HP", "AP", "lbs"); empty for unitless
	// stats (Initiative, Evasion).
	Unit string
}

// DerivedStats computes the five derived stat rows — Max Health, Action
// Points, Initiative, Evasion, Carry Capacity — from a Stats component using
// the StatsComponent methods (formulas per ADR-0006 / TR-stats-003:
// MaxHealth = 20 + CON×2 + Level×CON_mod; AP = 4 + DEX_mod (min 2);
// Initiative = DEX_mod + WIS_mod; Evasion = 10 + DEX_mod;
// CarryCapacity = STR×10 + 30).
func DerivedStats(c stats.StatsComponent) [numDerivedStats]DerivedStat {
	return [numDerivedStats]DerivedStat{
		{Name: "Max Health", Value: c.MaxHealth(), Unit: "HP"},
		{Name: "Action Points", Value: c.AP(), Unit: "AP"},
		{Name: "Initiative", Value: c.Initiative()},
		{Name: "Evasion", Value: c.Evasion()},
		{Name: "Carry Capacity", Value: c.CarryCapacity(), Unit: "lbs"},
	}
}

// formatInt renders an integer with the proper minus sign (U+2212) for
// negative values and no sign for zero and positive values. Derived totals
// (Initiative can be negative at low DEX/WIS) use this; the always-signed
// format applies to modifiers only (AC-5).
func formatInt(v int) string {
	if v < 0 {
		return fmt.Sprintf("%s%d", minusSign, -v)
	}
	return fmt.Sprintf("%d", v)
}

// DerivedValueDisplay renders a derived stat value with its unit suffix:
// "54 HP", "4 AP", "1", "9", "170 lbs" (AC-2).
func DerivedValueDisplay(d DerivedStat) string {
	if d.Unit == "" {
		return formatInt(d.Value)
	}
	return formatInt(d.Value) + " " + d.Unit
}

// CharacterSheet is the character sheet stat display. It owns the sheet's
// markup document, binds the stat rows by id, and repopulates them from the
// entity store on Open and Refresh. It reads Stats components only — it
// never writes entity state (ADR-0001: UI reads entity data).
type CharacterSheet struct {
	host  *engine.Host
	man   *ui.Manager
	store entity.EntityStore
	doc   *document.Document

	// viewingID is the entity whose stats are displayed; zero when closed.
	viewingID entity.GameEntityID
	visible   bool

	// reducedMotion snaps slides to their final state instead of animating.
	// Story 009 wires the accessibility setting here.
	reducedMotion bool

	// Elements bound from the markup.
	root  *document.Element // sheet-root
	paper *document.Element // sheet-paper

	// Tab bar (Story 005): cols is the character-tab attribute/derived
	// columns ("sheet-cols") and equipSection the character-tab equipment
	// strip ("equip-section") — both hidden while the Inventory tab is
	// active, replaced by the inventory panel. tabCharacter and
	// tabInventory are the two tab cells; the active one carries the
	// tab-active class.
	cols         *document.Element
	equipSection *document.Element
	tabCharacter *document.Element
	tabInventory *document.Element

	// inventory is the Inventory tab presenter (Story 005): item grid,
	// category filter, search bar, weight bar, gold display, and detail
	// panel (inventory.go).
	inventory *InventoryGrid

	// toast is the toast notification presenter (Story 006, toast.go):
	// the reusable timed-notification overlay (amber text on dark umber —
	// never amber on parchment) rendered as a sibling of the sheet overlay
	// so it stays visible while the sheet is closed. The game layer fires
	// it via ToastView() when PickupBlocked refuses a world-item pickup
	// (AC-13, "Inventory full", 2s).
	toast *ToastView

	// attrValues holds the six "{value} ({modifier})" labels and barFills the
	// six bar fill panels, indexed by stats attribute index.
	attrValues [stats.NumAttributes]*ui.Label
	barFills   [stats.NumAttributes]*document.Element

	// derivedVals holds the five derived value labels in DerivedStats order.
	derivedVals [numDerivedStats]*ui.Label

	// XP bar elements bound from the markup (Story 002): the level label,
	// the amber fill panel, and the "current / needed XP" label.
	xpLevel *ui.Label
	xpFill  *document.Element
	xpLabel *ui.Label

	// Equipment slot elements (Story 003): the eight 72×72 slot cells, their
	// icon layers, silhouette outlines, and grey-X reject labels, indexed by
	// SlotType; markup ids are "equip-{key}" (key from SlotType.Key()).
	equipCells       [NumSlots]*document.Element
	equipIcons       [NumSlots]*document.Element
	equipSilhouettes [NumSlots]*document.Element
	equipRejects     [NumSlots]*ui.Label

	// attrBonuses hold the green "(+N)" equipment bonus notes beside the
	// attribute values (AC-9), indexed by stats attribute; empty text when
	// the attribute has no equipment bonus.
	attrBonuses [stats.NumAttributes]*ui.Label

	// drag is the equipment drag-and-drop state machine (Story 003); its
	// equip callback repopulates the sheet in the same frame and its visual
	// callback refreshes the slot hover/reject classes.
	drag *DragDrop

	// modal is the level-up overlay (Story 002); nil when the markup never
	// loaded. The screen state machine (Story 007) drives it via Modal().
	modal *LevelUpModal

	// sidebar is the party sidebar (Story 008, party_sidebar.go): portraits
	// with the companion tooltip and inspect, selection state, and the AC-19
	// departure handling. It also serves as the level-up modal's PartyRoster
	// (party order for simultaneous level-ups). nil when the markup never
	// loaded.
	sidebar *PartySidebar

	// Companion inspect panel (Story 008, AC-21): the Trust & Relationship
	// section (indicator, level label, history) and the Knowledge Domains
	// list in the center panel, shown while a non-player companion is
	// viewed (populate's updateCompanionPanel). domainCells holds the
	// rendered domain rows (template duplicates, class-addressed children).
	companionPanel *document.Element // companion-panel
	trustIndicator *document.Element // companion-trust-indicator
	trustLevel     *ui.Label         // companion-trust-level
	trustHistory   *ui.Label         // companion-trust-history
	domains        *document.Element // companion-domains
	domainTpl      *document.Element // companion-domain-tpl
	domainsEmpty   *document.Element // companion-domains-empty
	domainCells    []*document.Element

	// slide drives the paper-slide enter/exit motion (TR-ui-004).
	slide *slideTween

	// Keyboard navigation and focus (Story 009, accessibility.go): nav is
	// the AC-23 tab-order cursor, focusElm the element currently carrying
	// the focus ring (nil when none), and closeHandler the user-close path
	// wired by the game layer (the state machine's user close, so the
	// level-up modal refusal, AC-18, and the animated exit hold).
	nav          KeyboardNav
	focusElm     *document.Element
	closeHandler func()

	// closeButton is the title-bar close button (Story 009, AC-23's final
	// tab stop); lockOverlay is the resolution-floor overlay (AC-25) and
	// lockShown tracks its display state.
	closeButton *document.Element
	lockOverlay *document.Element
	lockShown   bool

	// attrRows are the six attribute row elements ("attr-{key}"), the
	// focus-ring targets of the attributes group (Story 009).
	attrRows [stats.NumAttributes]*document.Element
}

// NewCharacterSheet creates the character sheet: loads the markup asset
// (with the level-up modal's onclick/onmouseenter handlers, Story 002, the
// equipment slots' mouse handlers, Story 003, the inventory tab handlers,
// Story 005, and the party sidebar handlers, Story 008), binds the stat row,
// XP bar, equipment slot, and companion inspect elements, creates the party
// sidebar, the level-up modal, the inventory grid, the toast overlay, and
// the equipment drag controller, and registers Update with the host updater.
//
// bus is the EventBus the party sidebar subscribes to (EventPartyMemberDeparted
// and combat death, AC-19); the game layer passes the world's event bus.
// A markup load failure degrades to a no-op sheet (slog warning) instead of
// crashing, matching the title screen behavior.
func NewCharacterSheet(host *engine.Host, man *ui.Manager, store entity.EntityStore, bus entity.EventBus) *CharacterSheet {
	s := &CharacterSheet{
		host:  host,
		man:   man,
		store: store,
	}
	// The drag controller is created before the document so the markup
	// handler closures can read s.drag; its equip callback repopulates the
	// sheet (same-frame derived stat update, AC-9) and its visual callback
	// refreshes the slot hover/reject classes (AC-7).
	s.drag = NewDragDrop(store)
	s.drag.OnEquip(func() {
		if err := s.Refresh(); err != nil {
			slog.Warn("character sheet: refresh after equipment change", "err", err)
		}
	})
	s.drag.OnVisual(s.applyEquipmentVisuals)
	funcMap := levelUpFuncMap(s)
	for name, handler := range equipmentFuncMap(s) {
		funcMap[name] = handler
	}
	for name, handler := range inventoryFuncMap(s) {
		funcMap[name] = handler
	}
	for name, handler := range partyFuncMap(s) {
		funcMap[name] = handler
	}
	// A nil host or UI manager (headless tests) degrades to a no-op sheet
	// instead of crashing: DocumentFromHTMLAsset dereferences the manager's
	// host immediately, and the host updater would be nil for the Update
	// registration below.
	if host == nil || man == nil {
		slog.Warn("character sheet: no UI manager; degraded sheet (no markup)")
		return s
	}
	doc, err := markup.DocumentFromHTMLAsset(man, markupPath, nil, funcMap)
	if err != nil {
		slog.Warn("character sheet: failed to load markup", "path", markupPath, "err", err)
		return s
	}
	s.doc = doc
	s.sidebar = NewPartySidebar(doc, store, bus)
	s.sidebar.SetViewer(s)
	s.inventory = NewInventoryGrid(doc, store)
	s.modal = NewLevelUpModal(doc, store, s.sidebar)
	s.toast = NewToastView(doc)
	s.bindElements()
	host.Updater.AddUpdate(s.Update)
	return s
}

// Open shows the character sheet for the given entity: it reads the entity's
// Stats component, populates every row (including the equipment slots,
// Story 003), points the drag controller at the entity, and plays the
// paper-slide entrance. It returns an error when the entity has no Stats
// component; the sheet stays closed in that case.
func (s *CharacterSheet) Open(id entity.GameEntityID) error {
	if s.doc == nil {
		// Degraded: markup never loaded. The error contract is the same as
		// the real path — an entity without Stats cannot be displayed.
		if _, ok := entity.GetComponent[stats.StatsComponent](s.store, id); !ok {
			return fmt.Errorf("character_sheet: entity %d has no Stats component", id)
		}
		s.viewingID = id
		return nil
	}
	comp, ok := entity.GetComponent[stats.StatsComponent](s.store, id)
	if !ok {
		return fmt.Errorf("character_sheet: entity %d has no Stats component", id)
	}
	s.viewingID = id
	// The game layer may open the sheet directly for an entity (e.g. the C
	// key opens the player sheet); keep the sidebar selection in sync.
	// Selection changes that originate in the sidebar go through SwitchMember
	// and set it themselves.
	if s.sidebar != nil {
		if m, ok := s.sidebar.Member(id); ok && !m.InPool {
			s.sidebar.setSelected(id)
		}
	}
	s.drag.SetOwner(id)
	s.drag.Reset() // a half-finished or rejected drag never carries over
	if s.inventory != nil {
		s.inventory.Reset() // a stale filter/search/selection never carries over
	}
	s.populate(comp)
	s.setVisible(true)
	s.Enter()
	return nil
}

// Close hides the sheet and forgets the viewed entity. It plays the 200ms
// paper-slide exit; the overlay is hidden when the slide completes. Calling
// Close on a closed sheet is a no-op.
func (s *CharacterSheet) Close() {
	if s.doc == nil {
		s.viewingID = 0
		s.drag.SetOwner(0)
		s.visible = false
		return
	}
	if !s.visible {
		return
	}
	s.viewingID = 0
	s.drag.Reset()
	s.drag.SetOwner(0)
	if s.sidebar != nil {
		s.sidebar.Reset() // a stale tooltip or selection never survives across opens
	}
	s.Exit()
}

// ForceClose hides the sheet immediately — no paper-slide exit — and
// forgets the viewed entity. The screen state machine (Story 007) calls it
// for forced closes (combat start, region transition): the overlay must be
// gone in the same frame, no animation, no prompt (GDD transition table;
// equipment changes are already committed). The level-up modal's pending
// state is untouched: a level-up interrupted by the forced close is
// deferred and reappears on the next open (Story 007 QA AC-16 edge).
// Calling ForceClose on a hidden sheet is a no-op.
func (s *CharacterSheet) ForceClose() {
	if s.doc == nil {
		s.viewingID = 0
		s.drag.SetOwner(0)
		s.visible = false
		return
	}
	if !s.visible {
		return
	}
	s.slide = nil // cancel any in-flight slide; the exit is not animated
	s.viewingID = 0
	s.drag.Reset()
	s.drag.SetOwner(0)
	if s.sidebar != nil {
		s.sidebar.Reset()
	}
	s.setVisible(false)
}

// Refresh re-reads the Stats component of the viewed entity and repopulates
// every displayed value. It returns an error when the sheet is closed or the
// entity no longer has a Stats component (e.g. the entity was destroyed
// while the sheet was open).
func (s *CharacterSheet) Refresh() error {
	if !s.visible {
		return fmt.Errorf("character_sheet: refresh on closed sheet")
	}
	comp, ok := entity.GetComponent[stats.StatsComponent](s.store, s.viewingID)
	if !ok {
		return fmt.Errorf("character_sheet: entity %d has no Stats component", s.viewingID)
	}
	s.populate(comp)
	return nil
}

// SwitchMember switches the sheet to another party member (AC-22): the
// viewed member's stats, equipment, XP bar, inventory, and companion inspect
// panel are repopulated for the new member, while the active tab is
// preserved — switching members never changes the Character/Inventory tab
// selection (the tab chrome is untouched). It is a no-op when id is already
// the viewed member (QA edge) and returns an error when the entity has no
// Stats component (the view stays on the current member). The party sidebar
// calls it via its Viewer contract.
func (s *CharacterSheet) SwitchMember(id entity.GameEntityID) error {
	if s.doc == nil {
		// Degraded: markup never loaded. Same error contract as the real
		// path so the party sidebar's departure switch behaves identically.
		if _, ok := entity.GetComponent[stats.StatsComponent](s.store, id); !ok {
			return fmt.Errorf("character_sheet: entity %d has no Stats component", id)
		}
		s.viewingID = id
		return nil
	}
	if id == s.viewingID {
		return nil
	}
	comp, ok := entity.GetComponent[stats.StatsComponent](s.store, id)
	if !ok {
		return fmt.Errorf("character_sheet: entity %d has no Stats component", id)
	}
	s.viewingID = id
	s.drag.Reset()
	s.drag.SetOwner(id)
	if s.inventory != nil {
		s.inventory.Reset()
	}
	s.populate(comp)
	if s.sidebar != nil {
		s.sidebar.setSelected(id)
	}
	return nil
}

// ViewedID returns the member currently viewed by the sheet; zero when the
// sheet is closed. The party sidebar (Story 008) uses it for the AC-19
// departure switch.
func (s *CharacterSheet) ViewedID() entity.GameEntityID { return s.viewingID }

// CompanionDeparted is the party sidebar's departure hook (Story 008,
// AC-19): when the level-up modal is showing the departed companion, the
// modal is aborted — the milestone attribute point is forfeited and the
// level-up queue discarded (LevelUpModal.Abort) — because Confirm's
// allocation write would target a character that is no longer in the party.
func (s *CharacterSheet) CompanionDeparted(id entity.GameEntityID) {
	if s.modal != nil && s.modal.Active() && s.modal.CurrentEntityID() == id {
		s.modal.Abort()
	}
}

// populate writes the stats values into the bound markup elements: six
// attribute labels and bar fills, five derived stat rows, the XP bar
// (Story 002), and the equipment slots (Story 003). Equipment attribute
// bonuses (AC-9) apply to the displayed values: the attribute label shows
// the base value with a green "(+N)" bonus note, the bar fill and the
// derived stats use the effective (base + bonus) attributes — all within
// the same populate call, so an equip lands on screen in the same frame.
// Bars always pair with numeric text so no information is encoded by color
// alone (ADR-0012).
func (s *CharacterSheet) populate(c stats.StatsComponent) {
	slots, hasInv := LoadEquipment(s.store, s.viewingID)
	if !hasInv {
		slog.Warn("character sheet: entity has no Inventory component; equipment shown empty",
			"entity", s.viewingID)
	}
	var bonuses [stats.NumAttributes]int
	if hasInv {
		bonuses = TotalAttributeBonuses(s.store, slots)
	}
	eff := EffectiveAttributes(c, bonuses)
	for i := 0; i < stats.NumAttributes; i++ {
		if l := s.attrValues[i]; l != nil {
			l.SetText(AttributeBaseDisplay(c.Attributes[i], bonuses[i]))
		}
		if b := s.attrBonuses[i]; b != nil {
			b.SetText(BonusDisplay(bonuses[i]))
		}
		if fill := s.barFills[i]; fill != nil && s.doc != nil {
			s.doc.SetElementStyleProperty(fill, "width",
				fmt.Sprintf("%.1f%%", BarFillFraction(eff.Attributes[i])*100))
		}
	}
	derived := DerivedStats(eff)
	for i := range derived {
		if l := s.derivedVals[i]; l != nil {
			l.SetText(DerivedValueDisplay(derived[i]))
		}
	}
	s.updateXPBar(c)
	s.populateEquipment(slots, hasInv)
	s.applyEquipmentVisuals()
	if s.inventory != nil {
		// Effective STR keeps the weight bar's max capacity in lockstep
		// with the Carry Capacity derived row (equipment bonuses included).
		s.inventory.Populate(s.viewingID, eff.CarryCapacity())
	}
	s.updateCompanionPanel()
}

// updateCompanionPanel renders the companion inspect section (Story 008,
// AC-21): while a non-player companion is viewed, the center panel shows the
// Trust & Relationship section — the amber trust indicator (glow at trust
// 1+, full glow at trust 3), the "Trusted (2/3)" level label, and the last
// five relationship events — plus the Knowledge Domains list with
// proficiency bars. The panel is hidden for the player character, for pool
// companions, and when the viewed entity is not in the party roster at all.
func (s *CharacterSheet) updateCompanionPanel() {
	show := false
	var m PartyMember
	if s.sidebar != nil {
		if member, ok := s.sidebar.Member(s.viewingID); ok && !member.InPool && member.ID != s.sidebar.PlayerID() {
			show = true
			m = member
		}
	}
	s.setShown(s.companionPanel, show)
	if !show {
		return
	}
	if s.trustIndicator != nil {
		s.trustIndicator.SetClasses("companion-trust-indicator", TrustIndicatorFor(int(m.Trust)).Class())
	}
	if s.trustLevel != nil {
		s.trustLevel.SetText(TrustLevelDisplay(m.Trust))
	}
	if s.trustHistory != nil {
		s.trustHistory.SetText(TrustHistoryDisplay(m.TrustHistory))
	}
	s.renderDomains(m.Domains)
}

// renderDomains renders the knowledge domain rows of the viewed companion
// (ADR-0005): one duplicated template row per domain with its name, an
// amber proficiency bar (KnowledgeDomainProficiency, 0-5), and the numeric
// "n/5" label. An empty list shows the "No knowledge domains." empty label
// instead. Bars always pair with numeric text — never color alone
// (ADR-0012). Out-of-range proficiency values are clamped for display.
func (s *CharacterSheet) renderDomains(domains []KnowledgeDomain) {
	if s.doc == nil {
		return
	}
	for _, row := range s.domainCells {
		s.doc.RemoveElement(row)
	}
	s.domainCells = s.domainCells[:0]
	if len(domains) == 0 {
		s.setShown(s.domainsEmpty, true)
		return
	}
	s.setShown(s.domainsEmpty, false)
	if s.domains == nil || s.domainTpl == nil {
		return
	}
	for i, d := range domains {
		proficiency := d.Proficiency
		if proficiency < 0 {
			proficiency = 0
		}
		if proficiency > 5 {
			proficiency = 5
		}
		row := s.doc.DuplicateElementToParent(s.domainTpl, s.domains)
		row.SetAttribute("data-index", strconv.Itoa(i))
		if name := findDescendant(row, "companion-domain-name"); name != nil {
			name.InnerLabel().SetText(d.Name)
		}
		if fill := findDescendant(row, "companion-domain-fill"); fill != nil {
			s.doc.SetElementStyleProperty(fill, "width",
				fmt.Sprintf("%.1f%%", float64(proficiency)/5*100))
		}
		if prof := findDescendant(row, "companion-domain-proficiency"); prof != nil {
			prof.InnerLabel().SetText(fmt.Sprintf("%d/5", proficiency))
		}
		s.domainCells = append(s.domainCells, row)
	}
}

// populateEquipment writes the equipped items into the eight slot cells
// (Story 003): a filled slot shows its icon layer and hides the silhouette
// outline; an empty slot shows the silhouette. The cells' hover/reject
// classes are refreshed separately by applyEquipmentVisuals (drag state).
func (s *CharacterSheet) populateEquipment(slots EquipmentSlots, hasInv bool) {
	for i, slot := range SlotTypes {
		s.setSlotFilled(i, hasInv && slots.Item(slot) != 0)
	}
}

// setSlotFilled shows the item icon layer (filled) or the silhouette
// outline (empty) for the slot at index i, and resets the cell to its base
// visual state (no hover, no grey X).
func (s *CharacterSheet) setSlotFilled(i int, filled bool) {
	if s.doc == nil {
		return
	}
	if icon := s.equipIcons[i]; icon != nil {
		display := "none"
		if filled {
			display = "flex"
		}
		s.doc.SetElementStyleProperty(icon, "display", display)
	}
	if sil := s.equipSilhouettes[i]; sil != nil {
		display := "flex"
		if filled {
			display = "none"
		}
		s.doc.SetElementStyleProperty(sil, "display", display)
	}
	if cell := s.equipCells[i]; cell != nil {
		cell.SetClasses("equip-slot")
	}
	if rej := s.equipRejects[i]; rej != nil {
		rej.SetText("")
	}
}

// applyEquipmentVisuals refreshes the drag-dependent slot visuals from the
// drag state (Story 003): during a drag the hovered compatible slot gets
// the amber border (equip-slot-hover), and the rejected slot shows the grey
// X (equip-slot-reject, AC-7). Idle, every slot is at its base state. The
// rejection X persists after a rejected drop until the pointer leaves the
// slot (HoverNone clears it).
func (s *CharacterSheet) applyEquipmentVisuals() {
	if s.doc == nil {
		return
	}
	hover := InvalidSlot
	rejected := InvalidSlot
	if s.drag != nil {
		if s.drag.Dragging() {
			hover = s.drag.HoverTarget()
		}
		rejected = s.drag.RejectedSlot()
	}
	for i, slot := range SlotTypes {
		if cell := s.equipCells[i]; cell != nil {
			classes := []string{"equip-slot"}
			if slot == hover {
				classes = append(classes, "equip-slot-hover")
			}
			if slot == rejected {
				classes = append(classes, "equip-slot-reject")
			}
			cell.SetClasses(classes...)
		}
		if rej := s.equipRejects[i]; rej != nil {
			text := ""
			if slot == rejected {
				text = "×" // grey X indicator (AC-7)
			}
			rej.SetText(text)
		}
	}
}

// updateXPBar writes the XP progress bar (Story 002, AC-4) from the Stats
// component: the "Level N" label, the amber fill width (XPProgressPct — the
// GDD formula composed from the stats package), and the "current / needed
// XP" label (XPProgressDisplay; "Max Level" at the level cap). The amber
// fill never carries text — the labels are ink on parchment (ADR-0012).
func (s *CharacterSheet) updateXPBar(c stats.StatsComponent) {
	if s.xpLevel != nil {
		s.xpLevel.SetText(fmt.Sprintf("Level %d", c.Level))
	}
	if s.xpFill != nil && s.doc != nil {
		s.doc.SetElementStyleProperty(s.xpFill, "width",
			fmt.Sprintf("%.1f%%", XPProgressPct(c)))
	}
	if s.xpLabel != nil {
		s.xpLabel.SetText(XPProgressDisplay(c))
	}
}

// setVisible shows or hides the sheet overlay via the root's inline display
// property (hidden by default in the markup).
func (s *CharacterSheet) setVisible(visible bool) {
	if s.doc == nil || s.root == nil {
		return
	}
	if visible {
		s.doc.SetElementStyleProperty(s.root, "display", "flex")
	} else {
		s.doc.SetElementStyleProperty(s.root, "display", "none")
	}
	s.visible = visible
}

// Visible reports whether the sheet overlay is currently shown. The screen
// state machine (Story 007) uses this to track the CharacterTab state.
func (s *CharacterSheet) Visible() bool { return s.visible }

// Modal returns the level-up modal overlay (Story 002); nil when the sheet
// markup never loaded. The game layer passes it to the screen state machine
// (Story 007), which shows it with Show (from the EventStatLevelUp
// subscriber), spends the milestone attribute point with AllocatePoint, and
// closes it with Confirm.
func (s *CharacterSheet) Modal() *LevelUpModal { return s.modal }

// ToastView returns the toast notification presenter (Story 006); nil when
// the sheet markup never loaded. The game layer fires it from the pickup
// interaction when PickupBlocked reports the pack cannot hold the item
// (AC-13): InventoryFullToast shows "Inventory full" for 2s and the item
// stays in the world. The toast is a World/Field notification, so it also
// fires while the sheet is closed.
func (s *CharacterSheet) ToastView() *ToastView { return s.toast }

// showTab switches the sheet between the Character and Inventory tabs
// (Story 005): the active tab cell carries the amber underline accent, and
// the character columns ("sheet-cols") are hidden while the inventory
// panel is shown. This story owns the tab chrome; Story 007 owns the
// 200ms crossfade, the C/I key handling, and the last-used-tab session
// state (the sheet opens on the Character tab by default).
func (s *CharacterSheet) showTab(tab string) {
	if s.doc == nil || s.cols == nil || s.inventory == nil {
		return
	}
	inventory := tab == "inventory"
	s.inventory.SetVisible(inventory)
	// The character-tab content (attribute/derived columns and the
	// equipment strip) is hidden while the inventory panel is shown.
	s.setShown(s.cols, !inventory)
	s.setShown(s.equipSection, !inventory)
	s.setTabActive(s.tabCharacter, !inventory)
	s.setTabActive(s.tabInventory, inventory)
}

// setShown shows or hides a tab content panel via its inline display
// property (panels default to flex in the sheet markup).
func (s *CharacterSheet) setShown(elm *document.Element, shown bool) {
	if s.doc == nil || elm == nil {
		return
	}
	display := "none"
	if shown {
		display = "flex"
	}
	s.doc.SetElementStyleProperty(elm, "display", display)
}

// setTabActive applies the tab-active class (amber underline accent —
// amber is accent, never text, ADR-0012) to the given tab cell.
func (s *CharacterSheet) setTabActive(tab *document.Element, active bool) {
	if tab == nil {
		return
	}
	classes := []string{"tab"}
	if active {
		classes = append(classes, "tab-active")
	}
	tab.SetClasses(classes...)
}

// bindElements resolves the markup ids into element references. Missing
// elements are logged and left nil; the update path skips nil elements.
func (s *CharacterSheet) bindElements() {
	s.root = s.element("sheet-root")
	s.paper = s.element("sheet-paper")

	for i, id := range attributeRowIDs {
		s.attrValues[i] = s.label(id + "-value")
		s.barFills[i] = s.element(id + "-fill")
		s.attrBonuses[i] = s.label(id + "-bonus")
	}
	for i, id := range derivedRowIDs {
		s.derivedVals[i] = s.label("derived-" + id + "-value")
	}
	s.xpLevel = s.label("xp-level")
	s.xpFill = s.element("xp-fill")
	s.xpLabel = s.label("xp-label")

	for i, slot := range SlotTypes {
		key := slot.Key()
		s.equipCells[i] = s.element("equip-" + key)
		s.equipIcons[i] = s.element("equip-" + key + "-icon")
		s.equipSilhouettes[i] = s.element("equip-" + key + "-silhouette")
		s.equipRejects[i] = s.label("equip-" + key + "-reject")
	}

	// Inventory tab chrome (Story 005): the character columns, the
	// equipment strip, and the two tab cells. The inventory panel elements
	// themselves bind in NewInventoryGrid.
	s.cols = s.element("sheet-cols")
	s.equipSection = s.element("equip-section")
	s.tabCharacter = s.element("tab-character")
	s.tabInventory = s.element("tab-inventory")

	// Companion inspect panel (Story 008, AC-21): the Trust & Relationship
	// section and the Knowledge Domains list in the center panel.
	s.companionPanel = s.element("companion-panel")
	s.trustIndicator = s.element("companion-trust-indicator")
	s.trustLevel = s.label("companion-trust-level")
	s.trustHistory = s.label("companion-trust-history")
	s.domains = s.element("companion-domains")
	s.domainTpl = s.element("companion-domain-tpl")
	s.domainsEmpty = s.element("companion-domains-empty")
}

func (s *CharacterSheet) element(id string) *document.Element {
	return sheetElement(s.doc, id)
}

func (s *CharacterSheet) label(id string) *ui.Label {
	return sheetLabel(s.doc, id)
}

// sheetElement resolves a markup id into an element reference, logging and
// returning nil when the id is missing. The level-up modal (Story 002)
// shares this helper; missing elements are skipped by the update paths.
func sheetElement(doc *document.Document, id string) *document.Element {
	elm, ok := doc.GetElementById(id)
	if !ok {
		slog.Warn("character sheet: markup element not found", "id", id)
		return nil
	}
	return elm
}

// sheetLabel resolves a markup id into its inner label, logging and
// returning nil when the id is missing.
func sheetLabel(doc *document.Document, id string) *ui.Label {
	elm, ok := doc.GetElementById(id)
	if !ok {
		slog.Warn("character sheet: markup element not found", "id", id)
		return nil
	}
	return elm.InnerLabel()
}

// Enter plays the paper-slide entrance: the parchment panel slides in from
// the right (transform translateX 48px → 0) while the soot dim overlay fades
// up (opacity 0 → 1), 300ms ease-out (TR-ui-004). The screen state machine
// that owns transitions is Story 007; this method is the hook it calls. Open
// calls it automatically.
func (s *CharacterSheet) Enter() {
	if s.doc == nil || s.root == nil || s.paper == nil {
		return
	}
	s.slide = newSlide(slideDistance, 0, enterDuration, true)
	if s.reducedMotion {
		s.slide.complete()
	}
	s.applySlide()
}

// Exit plays the paper-slide exit (opacity 1 → 0, translateX 0 → 48px,
// 200ms); the overlay is hidden when the slide completes. Story 007 calls
// this when leaving the CharacterTab state. An in-flight entrance is settled
// first so the exit starts from the fully shown state.
func (s *CharacterSheet) Exit() {
	if s.doc == nil || s.root == nil || s.paper == nil {
		s.setVisible(false)
		return
	}
	if s.slide != nil {
		s.slide.complete() // settle any in-flight slide before exiting
		s.applySlide()
	}
	s.slide = newSlide(0, slideDistance, exitDuration, false)
	if s.reducedMotion {
		s.slide.complete()
	}
	s.applySlide()
}

// Update advances the paper-slide tween, the level-up modal's hover pulse
// (Story 002), the inventory tab's weight-bar pulse (Story 005), and the
// toast notification countdown (Story 006) by dt seconds and applies the
// animated translateX/opacity values. Registered with the host updater in
// NewCharacterSheet; the toast keeps counting down even while the sheet is
// closed (world-interaction toasts, QA AC-13 edge).
func (s *CharacterSheet) Update(dt float64) {
	if s.modal != nil {
		s.modal.update(dt)
	}
	if s.inventory != nil {
		s.inventory.Update(dt)
	}
	if s.sidebar != nil {
		// Tooltip timer and dismiss grace (Story 008).
		s.sidebar.Update(dt)
	}
	if s.toast != nil {
		s.toast.Update(dt)
	}
	if s.slide == nil {
		return
	}
	s.slide.advance(dt)
	s.applySlide()
}

// applySlide applies the current slide values to the DOM. When a slide
// finishes, an exit hides the overlay and the slide is cleared.
func (s *CharacterSheet) applySlide() {
	if s.doc == nil || s.root == nil || s.paper == nil || s.slide == nil {
		return
	}
	s.doc.SetElementStyleProperty(s.paper, "transform",
		fmt.Sprintf("translateX(%.1fpx)", s.slide.value))
	s.doc.SetElementStyleProperty(s.root, "opacity", fmt.Sprintf("%.3f", s.slide.opacity()))
	if s.slide.done {
		if !s.slide.entering {
			s.setVisible(false)
		}
		s.slide = nil
	}
}

// SetReducedMotion gates the paper-slide motion, the level-up modal's
// hover pulse, the inventory tab's weight-bar pulse (Story 005), and the
// toast fade (Story 006): when on, slides complete instantly at their
// final state instead of animating, hovered attribute bars hold a steady
// fill, the weight bar pulse stops at full opacity, and toasts appear and
// dismiss instantly. Story 009 wires the accessibility setting here.
func (s *CharacterSheet) SetReducedMotion(on bool) {
	s.reducedMotion = on
	if s.modal != nil {
		s.modal.SetReducedMotion(on)
	}
	if s.inventory != nil {
		s.inventory.SetReducedMotion(on)
	}
	if s.toast != nil {
		s.toast.SetReducedMotion(on)
	}
	if on && s.slide != nil {
		s.slide.complete()
		s.applySlide()
	}
}
