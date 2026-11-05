// Renderer half of the combat HUD: it translates the TurnOrder model into
// Kaiju document operations — portrait chip creation, ordering, removal,
// highlight state, the hover tooltip, and the Go-driven border-color
// cross-fade that stands in for the CSS transition Kaiju does not support.
//
// All methods are no-ops when the document is nil (headless mode: tests
// exercise the model and event wiring without a rendering context).
package combat_hud

import (
	"fmt"
	"log/slog"
	"strconv"

	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup/document"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
)

// Highlight cross-fade duration (AC-3: the amber highlight moves from the
// current portrait to the next over ~150ms). Kaiju CSS has no transitions,
// so the fade is interpolated per frame in Go.
const highlightFadeDuration = 0.15 // seconds

// Highlight colors (art bible Section 6.6 / palette).
const (
	amberHex      = "#D4A040" // current-turn amber border
	umberBorderHex = "#6E6350" // resting umber border (fade target)
)

// portraitChip is one rendered turn-order portrait.
type portraitChip struct {
	elm  *document.Element
	init *ui.Label // initiative number label (combat face domain)
}

// borderFade drives the 150ms border-color cross-fade between the outgoing
// and incoming portrait on a turn change. Both chips animate together from
// the same tween; inline border-color overrides the class value while the
// fade runs and converges with it on completion, so no stale inline value
// can fight a later highlight (the next fade always writes both endpoints).
type borderFade struct {
	tween *Tween

	outElm       *document.Element
	outFrom, outTo string

	inElm        *document.Element
	inFrom, inTo string
}

// start begins the cross-fade: outElm (amber -> umber) and inElm
// (umber -> amber). Either chip may be nil.
func (f *borderFade) start(doc *document.Document, outElm, inElm *document.Element) {
	if doc == nil {
		return
	}
	f.outElm, f.outFrom, f.outTo = outElm, amberHex, umberBorderHex
	f.inElm, f.inFrom, f.inTo = inElm, umberBorderHex, amberHex
	f.tween = NewPaperSlideTween(0, 1, highlightFadeDuration)
}

// drop clears the fade references for a chip being removed, so a removal
// mid-fade cannot write styles to a detached element.
func (f *borderFade) drop(elm *document.Element) {
	if f.outElm == elm {
		f.outElm = nil
	}
	if f.inElm == elm {
		f.inElm = nil
	}
}

// update advances the fade by dt seconds and applies the interpolated
// border colors.
func (f *borderFade) update(doc *document.Document, dt float64) {
	if f.tween == nil || doc == nil {
		return
	}
	f.tween.Update(dt)
	p := f.tween.Value()
	if f.outElm != nil {
		doc.SetElementStyleProperty(f.outElm, "border-color", lerpHexColor(f.outFrom, f.outTo, p))
	}
	if f.inElm != nil {
		doc.SetElementStyleProperty(f.inElm, "border-color", lerpHexColor(f.inFrom, f.inTo, p))
	}
	if f.tween.Done() {
		f.tween = nil
		f.outElm, f.inElm = nil, nil
	}
}

// hudRenderer owns the bound document elements and the per-combatant
// portrait chips. Story 002 adds the action bar elements (AP meter,
// ability slots, End Turn) and the models they render (meter, bar).
type hudRenderer struct {
	doc   *document.Document
	store entity.EntityStore
	names NameReader
	model *TurnOrder
	meter *APMeter
	bar   *ActionBar

	topBar      *document.Element
	bottomBar   *document.Element
	tooltip     *document.Element
	tooltipName *ui.Label
	tooltipHP   *ui.Label
	template    *document.Element // hidden clone source for portrait chips

	// Story 002 action bar elements.
	apFill      *document.Element // ap-fill: amber fill, width follows the AP fill fraction
	apGlow      *document.Element // ap-glow: amber pulse overlay, opacity follows PulseStrength
	apText      *ui.Label         // ap-text: "current / max" readout
	abilityRow  *document.Element // ability-row: menu container, hidden off player turns
	abilityElms []*document.Element // ability-1..6 slots
	endTurn     *document.Element // end-turn button, hidden off player turns

	// onSelect/onEndTurn are the input reactions supplied by the HUD via
	// bindActions (see action_render.go).
	onSelect  func(hotkey int)
	onEndTurn func()

	chips       map[entity.GameEntityID]*portraitChip
	prevCurrent entity.GameEntityID
	fade        borderFade
}

// newHudRenderer creates the renderer and binds the markup elements. A nil
// document puts the renderer in headless mode (all operations no-op).
func newHudRenderer(doc *document.Document, store entity.EntityStore, names NameReader, model *TurnOrder, meter *APMeter, bar *ActionBar) *hudRenderer {
	r := &hudRenderer{
		doc:   doc,
		store: store,
		names: names,
		model: model,
		meter: meter,
		bar:   bar,
		chips: make(map[entity.GameEntityID]*portraitChip),
	}
	r.bind()
	return r
}

// bind resolves the markup ids and parks the HUD off-screen: the bars sit
// outside the viewport (translateY positive = up in Kaiju's layout, so the
// top bar rests above the screen and the bottom bar below it) and the
// tooltip and template are hidden. CombatStarted slides them in. The
// action bar elements (Story 002) start hidden — the ability menu and End
// Turn button appear on the first player turn (AC-2).
func (r *hudRenderer) bind() {
	if !r.online() {
		return
	}
	r.topBar = r.element("top-bar")
	r.bottomBar = r.element("bottom-bar")
	r.template = r.element("portrait-template")
	r.tooltip = r.element("tooltip")
	if r.tooltip != nil {
		r.tooltipName = r.tooltip.FindElementLabelById("tooltip-name")
		r.tooltipHP = r.tooltip.FindElementLabelById("tooltip-hp")
	}
	if r.template != nil {
		r.doc.SetElementStyleProperty(r.template, "display", "none")
	}
	if r.tooltip != nil {
		r.doc.SetElementStyleProperty(r.tooltip, "display", "none")
	}
	r.bindActionBar()
	r.applySlides(topBarOffset, -bottomBarOffset)
}

// bindActionBar resolves the Story 002 action bar elements: the AP meter
// parts (fill, glow, readout), the six ability slots, and the End Turn
// button. The static per-slot labels (hotkey number, AP cost) come from
// the action bar model's table — the markup keeps placeholder text. The
// menu and End Turn button start hidden (display: none); applyBar drives
// visibility per frame from the model.
func (r *hudRenderer) bindActionBar() {
	apMeter := r.element("ap-meter")
	r.apFill = r.element("ap-fill")
	r.apGlow = r.element("ap-glow")
	if apMeter != nil {
		r.apText = apMeter.FindElementLabelById("ap-text")
	}
	r.abilityRow = r.element("ability-row")
	r.endTurn = r.element("end-turn")
	for i, slot := range r.bar.Slots() {
		id := fmt.Sprintf("ability-%d", slot.Hotkey)
		elm := r.element(id)
		r.abilityElms = append(r.abilityElms, elm)
		if elm == nil {
			continue
		}
		hotkeyID := fmt.Sprintf("ability-hotkey-%d", slot.Hotkey)
		if lbl := elm.FindElementLabelById(hotkeyID); lbl != nil {
			lbl.SetText(strconv.Itoa(i + 1))
		}
		costID := fmt.Sprintf("ability-cost-%d", slot.Hotkey)
		if lbl := elm.FindElementLabelById(costID); lbl != nil {
			lbl.SetText(slot.CostLabel())
		}
	}
	if r.abilityRow != nil {
		r.doc.SetElementStyleProperty(r.abilityRow, "display", "none")
	}
	if r.endTurn != nil {
		r.doc.SetElementStyleProperty(r.endTurn, "display", "none")
	}
}

// online reports whether the renderer has a document to operate on.
func (r *hudRenderer) online() bool { return r.doc != nil }

// element resolves a markup id, logging a warning when missing.
func (r *hudRenderer) element(id string) *document.Element {
	elm, ok := r.doc.GetElementById(id)
	if !ok {
		// Bound ids are part of the HUD markup contract; a missing one is
		// a content bug, so warn loudly rather than fail silently.
		slog.Warn("combat hud: markup element not found", "id", id)
		return nil
	}
	return elm
}

// rebuild (re)creates the portrait chips for a CombatStarted payload.
// Chips are duplicated from the hidden template in entry order — the
// document appends each clone after the previous one, so the row is born
// in descending initiative order with no reordering pass needed (the
// order is fixed per encounter; only removals change it).
func (r *hudRenderer) rebuild(entries []combat.TurnEntry) {
	r.removeAllChips()
	if !r.online() || r.template == nil {
		return
	}
	clones := r.doc.DuplicateElementRepeat(r.template, len(entries))
	for i, e := range entries {
		elm := clones[i]
		r.doc.SetElementIdWithoutApplyStyles(elm, fmt.Sprintf("portrait-%d", e.EntityID))
		r.doc.SetElementClassesWithoutApply(elm, "portrait-chip", r.chipState(e.EntityID))
		r.doc.SetElementStylePropertyWithoutApply(elm, "display", "flex")
		initLabel := firstLabel(elm)
		if initLabel != nil {
			initLabel.SetText(strconv.Itoa(e.Initiative))
		}
		id := e.EntityID
		panel := elm.UI.ToPanel()
		panel.Base().AddEvent(ui.EventTypeEnter, func() { r.showTooltip(id) })
		panel.Base().AddEvent(ui.EventTypeExit, func() { r.hideTooltip() })
		r.chips[id] = &portraitChip{elm: elm, init: initLabel}
	}
	r.doc.ApplyStyles()
}

// setCurrent moves the highlight to the combatant whose turn is active:
// the outgoing chip drops the current class and fades amber -> umber while
// the incoming chip takes the current class and fades umber -> amber
// (AC-3: the highlight slides, no bounce/scale/spin).
func (r *hudRenderer) setCurrent(id entity.GameEntityID) {
	if !r.online() {
		r.prevCurrent = id
		return
	}
	var outElm *document.Element
	if r.prevCurrent != 0 {
		if old, ok := r.chips[r.prevCurrent]; ok {
			r.setChipClasses(old.elm, r.chipState(r.prevCurrent))
			outElm = old.elm
		}
	}
	var inElm *document.Element
	if chip, ok := r.chips[id]; ok {
		r.setChipClasses(chip.elm, "portrait-current")
		inElm = chip.elm
	}
	r.prevCurrent = id
	r.fade.start(r.doc, outElm, inElm)
}

// refreshChip re-derives a chip's state class — used when a turn ends and
// the combatant becomes acted.
func (r *hudRenderer) refreshChip(id entity.GameEntityID) {
	chip, ok := r.chips[id]
	if !ok || !r.online() {
		return
	}
	r.setChipClasses(chip.elm, r.chipState(id))
}

// removeChip removes the combatant's portrait from the row
// (EventCombatantRemoved). The remaining chips keep their order.
func (r *hudRenderer) removeChip(id entity.GameEntityID) {
	chip, ok := r.chips[id]
	if !ok {
		return
	}
	r.fade.drop(chip.elm)
	if r.online() {
		r.doc.RemoveElement(chip.elm)
	}
	delete(r.chips, id)
	if r.prevCurrent == id {
		r.prevCurrent = 0
	}
}

// removeAllChips clears the portrait row (used on rebuild and after the
// combat-end slide-out completes).
func (r *hudRenderer) removeAllChips() {
	if r.online() {
		for _, chip := range r.chips {
			r.doc.RemoveElement(chip.elm)
		}
	}
	r.chips = make(map[entity.GameEntityID]*portraitChip)
	r.prevCurrent = 0
	r.fade = borderFade{}
}

// chipState derives the visual state class for a combatant from the model:
// current wins over acted (the combat system never marks a combatant acted
// while its turn runs — TurnEnded precedes the next TurnStarted).
func (r *hudRenderer) chipState(id entity.GameEntityID) string {
	if cur, ok := r.model.Current(); ok && cur == id {
		return "portrait-current"
	}
	if r.model.IsActed(id) {
		return "portrait-acted"
	}
	return "portrait-upcoming"
}

// setChipClasses applies the base chip class plus exactly one state class.
func (r *hudRenderer) setChipClasses(elm *document.Element, state string) {
	if elm == nil || !r.online() {
		return
	}
	r.doc.SetElementClasses(elm, "portrait-chip", state)
}

// showTooltip displays the hover tooltip for the combatant (AC-5): name
// from the injected NameReader, "HP: current/max" from the Health
// component, both in combat face. The tooltip is positioned under the bar
// near the hovered chip (index-based percentage, clamped on-screen).
func (r *hudRenderer) showTooltip(id entity.GameEntityID) {
	if !r.online() || r.tooltip == nil {
		return
	}
	if r.tooltipName != nil {
		r.tooltipName.SetText(ResolveCombatantName(r.names, id))
	}
	if r.tooltipHP != nil {
		r.tooltipHP.SetText(r.hpText(id))
	}
	if idx, n := r.indexAndCount(id); n > 0 {
		pct := (float64(idx) + 0.5) / float64(n) * 100
		if pct < 4 {
			pct = 4
		}
		if pct > 84 {
			pct = 84
		}
		r.doc.SetElementStyleProperty(r.tooltip, "left", fmt.Sprintf("%.1f%%", pct))
	}
	r.doc.SetElementStyleProperty(r.tooltip, "display", "flex")
}

// hideTooltip dismisses the hover tooltip.
func (r *hudRenderer) hideTooltip() {
	if !r.online() || r.tooltip == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.tooltip, "display", "none")
}

// hpText renders the combatant's hit points in combat face with tabular
// figures. A combatant without a Health component (should not occur for a
// visible portrait) renders an unknown marker.
func (r *hudRenderer) hpText(id entity.GameEntityID) string {
	if r.store != nil {
		if hp, ok := entity.GetComponent[entity.Health](r.store, id); ok {
			// TODO(localization): user-facing string — route through the
			// localization system when it lands.
			return fmt.Sprintf("HP: %d/%d", hp.Current, hp.Max)
		}
	}
	// TODO(localization): user-facing string.
	return "HP: ?/?"
}

// indexAndCount returns the combatant's position in the current order and
// the order length, for tooltip placement.
func (r *hudRenderer) indexAndCount(id entity.GameEntityID) (int, int) {
	entries := r.model.Entries()
	for i, e := range entries {
		if e.EntityID == id {
			return i, len(entries)
		}
	}
	return 0, 0
}

// applySlides positions the bars via translateY (paper-slide motion; the
// values come from the HUD's Go-driven tweens).
func (r *hudRenderer) applySlides(top, bottom float64) {
	if !r.online() {
		return
	}
	if r.topBar != nil {
		r.doc.SetElementStyleProperty(r.topBar, "translateY", fmt.Sprintf("%.0fpx", top))
	}
	if r.bottomBar != nil {
		r.doc.SetElementStyleProperty(r.bottomBar, "translateY", fmt.Sprintf("%.0fpx", bottom))
	}
}

// update advances the highlight cross-fade and re-renders the action bar
// (AP meter fill/readout/pulse, ability slot states, menu visibility —
// Story 002).
func (r *hudRenderer) update(dt float64) {
	if !r.online() {
		return
	}
	r.fade.update(r.doc, dt)
	r.applyAP()
	r.applyBar()
}

// firstLabel returns the first Label child of the element subtree — the
// portrait chip's initiative label. Traversal avoids id collisions between
// the clone and its siblings (DuplicateElementRepeat clones keep the
// template's child ids).
func firstLabel(elm *document.Element) *ui.Label {
	if elm == nil {
		return nil
	}
	for _, child := range elm.Children {
		if child.UI != nil && child.UI.IsType(ui.ElementTypeLabel) {
			return child.UI.ToLabel()
		}
	}
	return nil
}

// lerpHexColor interpolates between two #RRGGBB colors at t in [0, 1].
func lerpHexColor(from, to string, t float64) string {
	fr, fg, fb := parseHex(from)
	tr, tg, tb := parseHex(to)
	r := fr + (tr-fr)*t
	g := fg + (tg-fg)*t
	b := fb + (tb-fb)*t
	return fmt.Sprintf("#%02X%02X%02X", int(r), int(g), int(b))
}

// parseHex decodes a #RRGGBB color into [0, 255] float components.
func parseHex(s string) (float64, float64, float64) {
	var r, g, b uint8
	fmt.Sscanf(s, "#%02X%02X%02X", &r, &g, &b)
	return float64(r), float64(g), float64(b)
}
