// Action bar rendering (Story 002): the AP meter (bottom-left), the six
// ability slots (bottom-center), and the End Turn button (bottom-right).
// All values are applied per frame from the engine-free models
// (ap_meter.go, action_bar.go) — the Kaiju CSS engine has no transitions.
//
// Engine note: the engine's CSS parser implements box-shadow's parse but
// its Process returns "BoxShadow not implemented" (verified in the pinned
// engine source), so the AC-5 amber pulse renders as an amber glow
// overlay (ap-glow) whose opacity follows the pulse tween — a brightness
// spike by opacity, not shadow. All methods are no-ops when the document
// is nil (headless mode: tests exercise the models and event wiring
// without a rendering context).
package combat_hud

import (
	"fmt"

	"kaijuengine.com/engine/ui"
)

// bindActions registers the click handlers for the ability slots and the
// End Turn button. The callbacks are supplied by the HUD: onSelect feeds
// CombatHUD.SelectAbility (selection toggle + EventAbilitySelected
// emission), onEndTurn feeds CombatHUD.PressEndTurn (guard + manager
// call). The model guards (affordability, lockout) apply before any
// callback effect, so a disabled slot's click is a no-op (AC-3).
func (r *hudRenderer) bindActions(onSelect func(hotkey int), onEndTurn func()) {
	if !r.online() {
		return
	}
	r.onSelect = onSelect
	r.onEndTurn = onEndTurn
	for i, elm := range r.abilityElms {
		if elm == nil {
			continue
		}
		hotkey := i + 1
		panel := elm.UI.ToPanel()
		panel.Base().AddEvent(ui.EventTypeClick, func() {
			if r.onSelect != nil {
				r.onSelect(hotkey)
			}
		})
	}
	if r.endTurn != nil {
		panel := r.endTurn.UI.ToPanel()
		panel.Base().AddEvent(ui.EventTypeClick, func() {
			if r.onEndTurn != nil {
				r.onEndTurn()
			}
		})
	}
}

// applyAP renders the AP meter (AC-1/AC-5): fill width follows the
// animated fill fraction, the numeric readout follows the count, and the
// glow overlay opacity follows the refill pulse strength.
func (r *hudRenderer) applyAP() {
	if !r.online() {
		return
	}
	if r.apFill != nil {
		r.doc.SetElementStyleProperty(r.apFill, "width", fmt.Sprintf("%.1f%%", r.meter.FillFraction()*100))
	}
	if r.apText != nil {
		r.apText.SetText(r.meter.DisplayText())
	}
	if r.apGlow != nil {
		r.doc.SetElementStyleProperty(r.apGlow, "opacity", fmt.Sprintf("%.2f", r.meter.PulseStrength()))
	}
}

// applyBar renders the action bar (AC-2/AC-3/AC-4): the menu and End Turn
// button are visible only during player turns; each slot carries its
// disabled class (insufficient AP) and the selected slot its highlight.
func (r *hudRenderer) applyBar() {
	if !r.online() {
		return
	}
	visible := "none"
	if r.bar.Active() {
		visible = "flex"
	}
	if r.abilityRow != nil {
		r.doc.SetElementStyleProperty(r.abilityRow, "display", visible)
	}
	if r.endTurn != nil {
		r.doc.SetElementStyleProperty(r.endTurn, "display", visible)
	}
	selected := r.bar.Selected()
	for i, elm := range r.abilityElms {
		if elm == nil {
			continue
		}
		slot := r.bar.Slots()[i]
		classes := []string{"ability-slot"}
		if !r.bar.CanAfford(slot.Hotkey) {
			classes = append(classes, "ability-disabled")
		}
		if selected == slot.Hotkey {
			classes = append(classes, "ability-selected")
		}
		r.doc.SetElementClasses(elm, classes...)
	}
}
