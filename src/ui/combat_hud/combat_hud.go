// Package combat_hud implements the Field-register combat HUD (ADR-0012,
// Story 001): the turn order bar, its paper-slide enter/exit transitions,
// and the combatant hover tooltip. Story 002 adds the action bar: the AP
// meter (bottom-left), the six-slot ability menu (bottom-center), and the
// End Turn button (bottom-right).
//
// The HUD is a pure consumer (ui-code rules): it subscribes to the combat
// lifecycle events on the synchronous EventBus (ADR-0011) and never
// mutates game state — the one sanctioned exception is End Turn, which
// the combat layer explicitly delegates to the input/UI layer
// (combat.Manager.EndTurn). Ability selection is published as
// EventAbilitySelected for the targeting (Story 005) and audio
// (AUDIO-004) consumers. It renders in screen space on the UI orthographic
// camera — immune to world camera shake, the desk never trembles
// (TR-ui-004). Field register styling (panel umber, no world dimming) is
// defined in assets/ui/combat_hud.html.
//
// Design: the engine-free models own all state — TurnOrder (turn_order.go)
// plus APMeter and ActionBar (ap_meter.go, action_bar.go); the hudRenderer
// (render.go, action_render.go) translates them into Kaiju document
// operations; tween.go provides the paper-slide motion (Kaiju CSS has no
// transitions, so motion is Go-driven per frame).
package combat_hud

import (
	"fmt"
	"log/slog"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup"
	"kaijuengine.com/platform/hid"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
)

// Paper-slide timings per ADR-0012 / TR-ui-004: enter 200ms ease-out,
// exit 200ms, no bounce/scale/spin/flip.
const (
	slideEnterDuration = 0.2 // seconds
	slideExitDuration  = 0.2 // seconds
)

// Off-screen resting offsets, matching the bar heights in the markup.
const (
	topBarOffset    = 64.0 // px
	bottomBarOffset = 96.0 // px
)

// subscriptionPriority places the HUD after combat's own bookkeeping
// (priority 1 = first). Subscribers at equal priority dispatch in
// registration order; the HUD only reads, so the exact value is not
// semantically load-bearing.
const subscriptionPriority = 10

// NameReader supplies combatant display names for portraits and tooltips.
// The game layer (party system, NPC registry) implements it; the HUD
// depends on the interface, not the source, and falls back to a generic
// label when a name is unavailable.
type NameReader interface {
	// Name returns the combatant's display name. An empty string is
	// treated as unavailable.
	Name(entity.GameEntityID) string
}

// fallbackNameFormat is the generic combatant label used when no name is
// available.
//
// TODO(localization): user-facing string — route through the localization
// system when it lands (ui-code rules).
const fallbackNameFormat = "Combatant %d"

// ResolveCombatantName returns the display name for the combatant from the
// NameReader, falling back to "Combatant N" when the reader is nil or
// returns an empty name.
func ResolveCombatantName(names NameReader, id entity.GameEntityID) string {
	if names != nil {
		if n := names.Name(id); n != "" {
			return n
		}
	}
	return fmt.Sprintf(fallbackNameFormat, id)
}

// CombatHUD is the combat HUD entry point: it owns the turn order and
// action bar models, the document renderer, and the paper-slide tweens,
// and subscribes to the combat lifecycle events (EventCombatStarted,
// EventTurnStarted, EventTurnEnded, EventCombatantRemoved,
// EventCombatEnded).
type CombatHUD struct {
	model    *TurnOrder
	renderer *hudRenderer

	// Story 002: the action bar models and the combat manager they read
	// from and act through. manager may be nil in headless contexts (the
	// meter stays empty and End Turn is a no-op).
	manager combat.Manager
	meter   *APMeter
	bar     *ActionBar

	// bus is the EventBus the HUD emits EventAbilitySelected on (AC-4);
	// nil in headless contexts without events.
	bus entity.EventBus

	// host is retained for hotkey polling (pollHotkeys); nil in headless
	// mode.
	host *engine.Host

	topSlide    *Tween // top bar paper-slide (translateY)
	bottomSlide *Tween // bottom bar paper-slide (translateY)

	// Settled bar offsets once the active slide completes (or under
	// reduced motion, immediately).
	settledTop    float64
	settledBottom float64
	// exiting is true while the combat-end slide-out runs; the portrait
	// chips are cleared when it completes.
	exiting bool

	reducedMotion bool
}

// NewCombatHUD loads the combat HUD markup, parks the bars off-screen,
// registers the per-frame update, and subscribes to the combat events.
//
// manager supplies the AP readouts and the End Turn action (Story 002);
// it may be nil in headless contexts — the meter stays empty and End Turn
// becomes a no-op. The HUD depends on the combat.Manager interface, not
// the concrete manager (dependency-injection contract).
//
// Headless mode: pass nil host and nil ui.Manager (as the integration
// tests do) to skip markup loading and the per-frame update while keeping
// the event wiring and the models fully active. store supplies Health
// components for the hover tooltip; names supplies display names — both
// may be nil (tooltip renders an unknown marker / fallback name).
func NewCombatHUD(host *engine.Host, man *ui.Manager, bus entity.EventBus, store entity.EntityStore, names NameReader, manager combat.Manager) *CombatHUD {
	h := &CombatHUD{
		model:         NewTurnOrder(),
		meter:         NewAPMeter(),
		bar:           NewActionBar(),
		manager:       manager,
		bus:           bus,
		host:          host,
		settledTop:    topBarOffset,
		settledBottom: -bottomBarOffset,
	}
	h.renderer = newHudRenderer(nil, store, names, h.model, h.meter, h.bar)

	if host != nil && man != nil {
		doc, err := markup.DocumentFromHTMLAsset(man, "ui/combat_hud.html", nil, nil)
		if err != nil {
			slog.Warn("combat hud: failed to load markup", "err", err)
			return h
		}
		h.renderer = newHudRenderer(doc, store, names, h.model, h.meter, h.bar)
		h.renderer.bindActions(func(hotkey int) { h.selectAbility(hotkey) }, func() { h.pressEndTurn() })
		host.Updater.AddUpdate(h.Update)
	}
	h.subscribe(bus)
	return h
}

// subscribe registers the five combat lifecycle event handlers. A nil bus
// (headless context without events) is tolerated.
func (h *CombatHUD) subscribe(bus entity.EventBus) {
	if bus == nil {
		return
	}
	bus.Subscribe(combat.EventCombatStarted, subscriptionPriority, h.onCombatStarted)
	bus.Subscribe(combat.EventTurnStarted, subscriptionPriority, h.onTurnStarted)
	bus.Subscribe(combat.EventTurnEnded, subscriptionPriority, h.onTurnEnded)
	bus.Subscribe(combat.EventCombatantRemoved, subscriptionPriority, h.onCombatantRemoved)
	bus.Subscribe(combat.EventCombatEnded, subscriptionPriority, h.onCombatEnded)
}

// Event handlers -----------------------------------------------------------

// onCombatStarted handles EventCombatStarted: populate the turn order,
// rebuild the portrait row, reset the action bar models (Story 002), and
// slide both bars in from the screen edges (AC-2: top bar down from the
// top, bottom bar up from the bottom, 200ms ease-out).
func (h *CombatHUD) onCombatStarted(ev entity.Event) {
	data, ok := ev.Data.(combat.CombatStartedData)
	if !ok {
		slog.Warn("combat hud: CombatStarted payload type mismatch", "type", fmt.Sprintf("%T", ev.Data))
		return
	}
	h.model.BeginCombat(data.TurnOrder)
	h.meter.Reset()
	h.bar.Reset()
	h.renderer.rebuild(data.TurnOrder)
	h.startSlideIn()
}

// onTurnStarted handles EventTurnStarted: advance the highlight to the
// active combatant (AC-3: the amber border cross-fades between portraits,
// 150ms, no bounce/scale/spin), refill the AP meter from the manager's
// current readouts (AC-5 — the pool is already reset when the event
// fires), and arm the action bar for player turns (AC-2).
func (h *CombatHUD) onTurnStarted(ev entity.Event) {
	if !h.model.Active() {
		return
	}
	data, ok := ev.Data.(combat.TurnStartedData)
	if !ok {
		slog.Warn("combat hud: TurnStarted payload type mismatch", "type", fmt.Sprintf("%T", ev.Data))
		return
	}
	h.model.OnTurnStarted(data.Combatant, data.Round)
	h.renderer.setCurrent(data.Combatant)
	ap, max := 0, 0
	if h.manager != nil {
		ap = h.manager.CurrentAP()
		max = h.manager.MaxAP()
	}
	h.meter.OnTurnStarted(ap, max)
	h.bar.OnTurnStarted(data.Side, ap)
}

// onTurnEnded handles EventTurnEnded: mark the combatant as acted this
// round (further dimmed portrait) and hide the action bar (Story 002 —
// the menu belongs to the player's turn, AC-2).
func (h *CombatHUD) onTurnEnded(ev entity.Event) {
	if !h.model.Active() {
		return
	}
	data, ok := ev.Data.(combat.TurnEndedData)
	if !ok {
		slog.Warn("combat hud: TurnEnded payload type mismatch", "type", fmt.Sprintf("%T", ev.Data))
		return
	}
	h.model.OnTurnEnded(data.Combatant)
	h.bar.OnTurnEnded()
	h.renderer.refreshChip(data.Combatant)
}

// onCombatantRemoved handles EventCombatantRemoved: drop the combatant's
// portrait from the row; the remaining portraits keep their order.
func (h *CombatHUD) onCombatantRemoved(ev entity.Event) {
	if !h.model.Active() {
		return
	}
	data, ok := ev.Data.(combat.CombatantRemovedData)
	if !ok {
		slog.Warn("combat hud: CombatantRemoved payload type mismatch", "type", fmt.Sprintf("%T", ev.Data))
		return
	}
	h.model.Remove(data.EntityID)
	h.renderer.removeChip(data.EntityID)
}

// onCombatEnded handles EventCombatEnded: clear the turn order state and
// slide both bars out — top bar exits upward, bottom bar exits downward,
// 200ms (AC-4). The portrait chips are cleared when the exit completes.
// The outcome payload is not consumed here (Story 006: resolution
// screens).
func (h *CombatHUD) onCombatEnded(ev entity.Event) {
	if !h.model.Active() {
		return
	}
	h.model.EndCombat()
	h.meter.Reset()
	h.bar.Reset()
	h.startSlideOut()
}

// Slides ----------------------------------------------------------------

// startSlideIn animates both bars into view from their off-screen resting
// offsets (top bar down, bottom bar up; 200ms ease-out). Under reduced
// motion the bars jump to their final positions immediately.
func (h *CombatHUD) startSlideIn() {
	if h.reducedMotion {
		h.settledTop, h.settledBottom = 0, 0
		h.topSlide, h.bottomSlide = nil, nil
		h.renderer.applySlides(0, 0)
		return
	}
	h.topSlide = NewPaperSlideTween(topBarOffset, 0, slideEnterDuration)
	h.bottomSlide = NewPaperSlideTween(-bottomBarOffset, 0, slideEnterDuration)
}

// startSlideOut animates both bars out of view (top bar up, bottom bar
// down; 200ms). The portrait row is cleared when the exit completes.
func (h *CombatHUD) startSlideOut() {
	if h.reducedMotion {
		h.settledTop, h.settledBottom = topBarOffset, -bottomBarOffset
		h.topSlide, h.bottomSlide = nil, nil
		h.renderer.applySlides(topBarOffset, -bottomBarOffset)
		h.exiting = false
		h.renderer.removeAllChips()
		h.renderer.hideTooltip()
		return
	}
	h.topSlide = NewPaperSlideTween(0, topBarOffset, slideExitDuration)
	h.bottomSlide = NewPaperSlideTween(0, -bottomBarOffset, slideExitDuration)
	h.exiting = true
}

// Update advances the paper-slide tweens, the highlight cross-fade, the
// AP meter and action bar models (Story 002), and routes hotkeys (1-6,
// E, Enter during a player turn). Registered on the host updater by
// NewCombatHUD (no-op in headless mode).
func (h *CombatHUD) Update(dt float64) {
	if h.topSlide != nil {
		h.topSlide.Update(dt)
		if h.topSlide.Done() {
			h.settledTop = h.topSlide.To
			h.topSlide = nil
		}
	}
	if h.bottomSlide != nil {
		h.bottomSlide.Update(dt)
		if h.bottomSlide.Done() {
			h.settledBottom = h.bottomSlide.To
			h.bottomSlide = nil
		}
	}
	top, bottom := h.settledTop, h.settledBottom
	if h.topSlide != nil {
		top = h.topSlide.Value()
	}
	if h.bottomSlide != nil {
		bottom = h.bottomSlide.Value()
	}
	h.renderer.applySlides(top, bottom)
	h.meter.Update(dt)
	h.bar.Update(dt)
	h.pollHotkeys()
	h.renderer.update(dt)

	if h.exiting && h.topSlide == nil && h.bottomSlide == nil {
		h.exiting = false
		h.renderer.removeAllChips()
		h.renderer.hideTooltip()
	}
}

// pollHotkeys routes keyboard input during a player turn (AC-2): 1-6
// select abilities, E and Enter end the turn. KeyDown is edge-triggered
// in the engine (KeyStateDown, distinct from KeyHeld), so a held key
// cannot repeat or toggle the selection. No-op without a host (headless
// mode — tests drive SelectAbility/PressEndTurn directly).
func (h *CombatHUD) pollHotkeys() {
	if h.host == nil || !h.bar.Active() {
		return
	}
	kb := &h.host.Window.Keyboard
	for _, slot := range h.bar.Slots() {
		if kb.KeyDown(hotkeyKey(slot.Hotkey)) {
			h.selectAbility(slot.Hotkey)
		}
	}
	if kb.KeyDown(hid.KeyboardKeyE) || kb.KeyDown(hid.KeyboardKeyEnter) {
		h.pressEndTurn()
	}
}

// hotkeyKey maps a 1-6 action-bar hotkey to the engine keyboard key
// (KeyboardKey1..6). Callers only pass valid hotkeys from the action bar
// table.
func hotkeyKey(hotkey int) hid.KeyboardKey {
	return hid.KeyboardKey1 + hid.KeyboardKey(hotkey-1)
}

// SetReducedMotion enables or disables the reduced-motion gate: while on,
// paper-slide transitions and the AP fill/pulse animations complete
// instantly (accessibility requirement: "Combat HUD slide-in -> 0ms").
// The setting applies to transitions that start after the call; the game
// layer wires it from the accessibility settings.
func (h *CombatHUD) SetReducedMotion(on bool) {
	h.reducedMotion = on
	h.meter.SetReducedMotion(on)
}

// Action bar input -------------------------------------------------------

// SelectAbility handles an ability selection by click or hotkey (AC-4):
// toggles the selection (re-selecting deselects, one ability at a time)
// and emits EventAbilitySelected when an ability becomes selected —
// consumers are the targeting system (Story 005) and the audio system
// (AUDIO-004: soft select sound). Returns the newly selected hotkey
// (0 = deselected) and whether the request was honored. Requests for
// unknown, unaffordable (AC-3), or out-of-turn abilities are ignored:
// (0, false).
func (h *CombatHUD) SelectAbility(hotkey int) (int, bool) {
	next, ok := h.bar.Select(hotkey)
	if !ok || next == 0 {
		return next, ok
	}
	if h.bus != nil {
		if cur, curOK := h.model.Current(); curOK {
			h.bus.Emit(entity.Event{
				Type:   combat.EventAbilitySelected,
				Source: "combat_hud",
				Data:   combat.AbilitySelectedData{Combatant: cur, Slot: next},
			})
		}
	}
	return next, true
}

// PressEndTurn handles an End Turn press by click or key (AC-4): the
// 300ms double-trigger guard is applied, then the turn ends via
// combat.Manager.EndTurn — the one state mutation the combat layer
// explicitly delegates to the input/UI layer. Returns the manager's
// error, or nil when the press was ignored (guard active, no player
// turn) or no manager is injected (headless mode).
func (h *CombatHUD) PressEndTurn() error {
	if !h.bar.TryEndTurn() {
		return nil
	}
	if h.manager == nil {
		return nil
	}
	if err := h.manager.EndTurn(); err != nil {
		slog.Warn("combat hud: end turn failed", "err", err)
		return err
	}
	return nil
}

// SetAP applies an AP change to the meter and the action bar's
// affordability (AC-1, AC-3): the bar fill depletes to current/max over
// 150ms with the readout counting down. The combat system does not emit
// an AP change event yet (AP spending lands with the targeting and
// attack stories), so nothing calls this in live play — the future
// EventAPChanged wiring routes here.
func (h *CombatHUD) SetAP(current int) {
	h.meter.SetAP(current)
	h.bar.SetAP(current)
}

// selectAbility and pressEndTurn are the unexported wrappers the renderer
// click handlers and the hotkey polling call.
func (h *CombatHUD) selectAbility(hotkey int) { _, _ = h.SelectAbility(hotkey) }

func (h *CombatHUD) pressEndTurn() { _ = h.PressEndTurn() }

// Model accessors ---------------------------------------------------------

// Active reports whether an encounter is running.
func (h *CombatHUD) Active() bool { return h.model.Active() }

// TurnOrder returns the displayed turn order — the combatants still in the
// encounter, in descending initiative order.
func (h *CombatHUD) TurnOrder() []combat.TurnEntry { return h.model.Entries() }

// CurrentCombatant returns the highlighted combatant, or (0, false) when
// no turn is active.
func (h *CombatHUD) CurrentCombatant() (entity.GameEntityID, bool) {
	return h.model.Current()
}

// Round returns the current round number, 1-based.
func (h *CombatHUD) Round() int { return h.model.Round() }

// IsActed reports whether the combatant has taken its turn this round.
func (h *CombatHUD) IsActed(id entity.GameEntityID) bool {
	return h.model.IsActed(id)
}

// Action bar model accessors ----------------------------------------------

// APCurrent returns the active combatant's settled AP (the readout shows
// the animated value during fill/depletion transitions).
func (h *CombatHUD) APCurrent() int { return h.meter.Current() }

// APMax returns the active combatant's per-turn AP budget.
func (h *CombatHUD) APMax() int { return h.meter.Max() }

// APText returns the meter's numeric readout, "current / max".
func (h *CombatHUD) APText() string { return h.meter.DisplayText() }

// APFill returns the animated fill fraction in [0,1] (bar width %).
func (h *CombatHUD) APFill() float64 { return h.meter.FillFraction() }

// APPulseStrength returns the refill pulse intensity in [0,1], 0 outside
// the post-refill spike window (AC-5).
func (h *CombatHUD) APPulseStrength() float64 { return h.meter.PulseStrength() }

// AbilityActive reports whether the ability menu and End Turn button are
// shown (player-controlled turn, AC-2).
func (h *CombatHUD) AbilityActive() bool { return h.bar.Active() }

// AbilitySlots returns the six ability slots in hotkey order (AC-2).
func (h *CombatHUD) AbilitySlots() []AbilitySlot { return h.bar.Slots() }

// SelectedAbility returns the selected slot's hotkey (1-6), 0 when none
// (AC-4).
func (h *CombatHUD) SelectedAbility() int { return h.bar.Selected() }

// CanAffordAbility reports whether the ability with the given hotkey is
// selectable with the active turn's AP (AC-3).
func (h *CombatHUD) CanAffordAbility(hotkey int) bool { return h.bar.CanAfford(hotkey) }

// EndTurnLocked reports whether the End Turn double-trigger guard is
// active (300ms after a press).
func (h *CombatHUD) EndTurnLocked() bool { return h.bar.EndTurnLocked() }
