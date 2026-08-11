// Screen state machine (Story 007, Epic: Character & Inventory UI): the
// GDD screen states (design/gdd/character-inventory-ui.md) and their
// transitions, plus the combat integration via the synchronous EventBus
// (ADR-0011): combat start force-closes the UI and blocks the C/I open
// keys; combat end restores them. Tab switches crossfade in 200ms
// (ADR-0012 / TR-ui-004); open/close use the sheet's paper-slide hooks
// (300ms enter / 200ms exit). Last-used tab and last-viewed party member
// are session-only state (never persisted to save files, per the GDD).
//
// The state machine owns transition logic only. The sheet's Enter and Exit
// are the animation hooks it calls (Story 009's reduce-motion setting gates
// them on the sheet side), and the level-up modal (Story 002) is driven
// through the injected *LevelUpModal — the EventStatLevelUp subscriber that
// shows it lives here per level_up.go's Show contract: prevLevel and hpGain
// derive from the pre-award component the store still holds during the
// synchronous dispatch (stats.AwardXPAndNotify writes the post-award
// component back only after the dispatch returns).
//
// Explicit transition table:
//
//	Closed ────── OpenCharacter()/C or menu command ──▶ CharacterTab  (paper-slide enter 300ms, opens to last-viewed member)
//	Closed ────── OpenInventory()/I or menu command ──▶ InventoryTab  (paper-slide enter 300ms)
//	CharacterTab ─ SwitchTab()/C ──────────────────────▶ InventoryTab  (content crossfade 200ms, member kept)
//	InventoryTab ─ SwitchTab()/I ──────────────────────▶ CharacterTab  (content crossfade 200ms, member kept)
//	CharacterTab ─ ToggleCharacter()/C ────────────────▶ Closed        (paper-slide exit 200ms)
//	InventoryTab ─ ToggleInventory()/I ────────────────▶ Closed        (paper-slide exit 200ms)
//	CharacterTab/InventoryTab ─ Esc or close button ───▶ Closed        (paper-slide exit 200ms)
//	CharacterTab ─ EventStatLevelUp ───────────────────▶ LevelUp       (modal overlay; only Confirm dismisses it, AC-18)
//	Any open state ─ combat start / region transition ─▶ Closed        (forced: immediate, no animation, no prompt)
//
// While combat is active (EventCombatStarted .. EventCombatEnded) every
// open command — keyboard, gamepad menu button, or menu command — routes
// through these methods and is silently ignored (AC-15). Conversation
// start does not close the UI and does not block C/I (Story 007
// Implementation Notes); CloseReasonConversationStart exists for the
// future producer per the GDD transition table.
package character_sheet

import (
	"fmt"
	"log/slog"

	"kaijuengine.com/engine"
	"kaijuengine.com/platform/hid"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// ScreenState is the state of the Character & Inventory UI (GDD screen
// states, design/gdd/character-inventory-ui.md).
type ScreenState int

const (
	// Closed: the UI is hidden; C/I keys or menu commands open it.
	Closed ScreenState = iota
	// CharacterTab: the character sheet is displayed for the selected party
	// member.
	CharacterTab
	// InventoryTab: the inventory grid is displayed for the selected party
	// member (content arrives with Story 005).
	InventoryTab
	// LevelUp: the mandatory level-up modal overlays the sheet (Story 002);
	// only Confirm dismisses it (AC-18).
	LevelUp
)

// CloseReason distinguishes user-initiated closes from forced closes (GDD
// transition table: "Any open state → Closed (forced): combat starts,
// conversation starts, or region transition begins. UI closes immediately
// with no save prompt").
type CloseReason int

const (
	// CloseReasonUserRequest: Esc or the close button. Animated — the
	// 200ms paper-slide exit runs.
	CloseReasonUserRequest CloseReason = iota
	// CloseReasonCombatStart: EventCombatStarted. Forced — immediate, no
	// animation, no prompt (AC-16). Wired in this story.
	CloseReasonCombatStart
	// CloseReasonConversationStart: defined per the GDD transition table;
	// not wired to any event in this story (per Story 007 Implementation
	// Notes the UI stays open during conversation).
	CloseReasonConversationStart
	// CloseReasonRegionTransition: defined per the GDD transition table;
	// the region-transition event producer arrives with the exploration
	// epic.
	CloseReasonRegionTransition
)

// crossfadeDuration is the tab-switch content crossfade time (TR-ui-004 /
// ADR-0012: 200-300ms; the GDD pins 200ms). The timer is the animation
// hook the state machine owns; the content panels' DOM fade is wired by
// Story 005 (inventory grid). A pure timer — no frame drops during the
// transition.
const crossfadeDuration = 0.200

// subscriptionPriority places the C&I UI after the systems that own game
// state (ADR-0011: presentation is last — priority 1 dispatches first),
// matching the combat HUD's presentation-last placement.
const subscriptionPriority = 10

// crossfadeEpsilon is the residual below which the crossfade timer snaps to
// zero. Float accumulation from exact-sum frame times (e.g. 0.19 + 0.01
// against a 0.2 budget) must not leave the transition technically in flight
// forever; anything below a nanosecond is settled.
const crossfadeEpsilon = 1e-9

// Sheet is the subset of CharacterSheet the screen state machine drives:
// open/close (animated and forced) and visibility. It is an interface so
// the machine is testable without the markup layer — a real CharacterSheet
// can only be constructed with a live UI manager (headless tests use a
// stub). CharacterSheet satisfies it.
type Sheet interface {
	// Open shows the sheet for the given entity with the paper-slide
	// entrance; it returns an error when the entity has no Stats component
	// (the UI must stay closed then).
	Open(id entity.GameEntityID) error
	// Close hides the sheet with the 200ms paper-slide exit; the overlay
	// hides when the slide completes.
	Close()
	// ForceClose hides the sheet immediately — no animation, no prompt.
	ForceClose()
	// Visible reports whether the overlay is currently shown.
	Visible() bool
}

// sheetImpl assertion: the real sheet satisfies the interface.
var _ Sheet = (*CharacterSheet)(nil)

// LevelUpHpGain returns the hit points gained by a level-up from the
// pre-award Stats component per ADR-0006 and TR-stats-005: max(1, CON_mod)
// per level gained (stats.HealthGainPerLevel × levels). Derived from the
// pre-award component the store still holds during the synchronous
// EventStatLevelUp dispatch; the payload's NewMaxHealth is not used
// because the closed-form MaxHealth delta can be zero or negative at low
// CON while the GDD guarantees a minimum of +1 HP per level.
func LevelUpHpGain(pre stats.StatsComponent, newLevel int) int {
	return pre.HealthGainPerLevel() * (newLevel - pre.Level)
}

// LevelUpGrantsAttributePoint reports whether the level range (prevLevel,
// newLevel] crosses a milestone level — 4, 8, 12, 16, every 4th level
// below the cap — matching the UnspentPoints grant in stats.AwardXP (a
// 4→5 level-up is not a milestone; a 3→4 level-up is, and a 3→5 award
// crosses 4). Level 20 grants none.
func LevelUpGrantsAttributePoint(prevLevel, newLevel int) bool {
	for l := prevLevel + 1; l <= newLevel; l++ {
		if l%4 == 0 && l < stats.LevelCap {
			return true
		}
	}
	return false
}

// ScreenStateMachine owns the C&I UI's screen states and transitions. It
// routes open/close/toggle/switch requests (keyboard polling, menu
// commands, future gamepad), subscribes to the combat lifecycle events and
// EventStatLevelUp on the synchronous EventBus, and preserves the
// session-only last-used tab and last-viewed party member across close/open
// cycles.
type ScreenStateMachine struct {
	sheet Sheet
	modal *LevelUpModal
	store entity.EntityStore
	bus   entity.EventBus
	host  *engine.Host

	// playerID is the player character: the last-viewed member a new
	// session starts on (session-only state resets per session).
	playerID entity.GameEntityID

	// tab is the machine's current state: Closed when the UI is hidden,
	// CharacterTab or InventoryTab while open. LevelUp is derived (State).
	tab ScreenState

	// lastTab and lastMember are session-only (GDD: "persisted per-session,
	// not per-save — resets to player character on new session").
	lastTab    ScreenState
	lastMember entity.GameEntityID

	// combatActive blocks C/I open commands while an encounter runs
	// (AC-15); set by EventCombatStarted, cleared by EventCombatEnded.
	combatActive bool

	// closing is true while a user-initiated close's paper-slide exit is in
	// flight (the overlay stays visible until the slide completes). Forced
	// closes never set it.
	closing bool

	// fadeRemaining is the tab crossfade timer in seconds (200ms).
	fadeRemaining float64
}

// NewScreenStateMachine creates the screen state machine over the sheet,
// subscribes to the combat lifecycle events (EventCombatStarted,
// EventCombatEnded) and EventStatLevelUp on the EventBus, and registers
// Update with the host updater (nil host: no keyboard polling — headless
// tests drive the methods directly).
//
// modal is the level-up overlay the state machine drives (Story 002); pass
// sheet.Modal() in the game. store is the entity store the level-up
// subscriber reads the pre-award Stats component from (the sheet's store).
// playerID seeds the session-only last-viewed member (the GDD resets it to
// the player character per session). A nil bus skips subscription (headless
// contexts without events).
func NewScreenStateMachine(sheet Sheet, modal *LevelUpModal, store entity.EntityStore, bus entity.EventBus, playerID entity.GameEntityID, host *engine.Host) *ScreenStateMachine {
	sm := &ScreenStateMachine{
		sheet:      sheet,
		modal:      modal,
		store:      store,
		bus:        bus,
		host:       host,
		playerID:   playerID,
		tab:        Closed,
		lastTab:    CharacterTab,
		lastMember: playerID,
	}
	if bus != nil {
		bus.Subscribe(combat.EventCombatStarted, subscriptionPriority, sm.onCombatStarted)
		bus.Subscribe(combat.EventCombatEnded, subscriptionPriority, sm.onCombatEnded)
		bus.Subscribe(stats.EventStatLevelUp, subscriptionPriority, sm.onLevelUp)
	}
	if host != nil {
		host.Updater.AddUpdate(sm.Update)
	}
	return sm
}

// OpenCharacter opens the Character tab (C key or menu command): from
// Closed it shows the sheet with the paper-slide enter and opens to the
// last-viewed party member; from InventoryTab it crossfades to the
// Character tab. On the Character tab it is a no-op (the toggle key closes
// instead). Silently ignored while combat is active (AC-15).
func (sm *ScreenStateMachine) OpenCharacter() {
	if sm.combatActive {
		return
	}
	switch sm.tab {
	case Closed:
		sm.open(CharacterTab)
	case InventoryTab:
		sm.SwitchTab(CharacterTab)
	}
}

// OpenInventory mirrors OpenCharacter for the Inventory tab (I key or menu
// command).
func (sm *ScreenStateMachine) OpenInventory() {
	if sm.combatActive {
		return
	}
	switch sm.tab {
	case Closed:
		sm.open(InventoryTab)
	case CharacterTab:
		sm.SwitchTab(InventoryTab)
	}
}

// ToggleCharacter is the C key handler (GDD: "C toggles character sheet"):
// on the Character tab it closes (animated exit), on the Inventory tab it
// switches with a crossfade, from Closed it opens the Character tab.
// Silently ignored while combat is active (AC-15).
func (sm *ScreenStateMachine) ToggleCharacter() {
	if sm.combatActive {
		return
	}
	switch sm.tab {
	case Closed:
		sm.open(CharacterTab)
	case CharacterTab:
		sm.Close(CloseReasonUserRequest)
	case InventoryTab:
		sm.SwitchTab(CharacterTab)
	}
}

// ToggleInventory mirrors ToggleCharacter for the Inventory tab (I key
// handler).
func (sm *ScreenStateMachine) ToggleInventory() {
	if sm.combatActive {
		return
	}
	switch sm.tab {
	case Closed:
		sm.open(InventoryTab)
	case InventoryTab:
		sm.Close(CloseReasonUserRequest)
	case CharacterTab:
		sm.SwitchTab(InventoryTab)
	}
}

// SwitchTab switches between CharacterTab and InventoryTab with the 200ms
// content crossfade (AC-17). The selected party member is preserved — the
// switch never touches lastMember. Rapid switches do not stack or queue:
// the latest target wins and the crossfade timer restarts. No-op while
// combat is active, while the UI is closed, on the same tab, or for
// non-tab targets.
func (sm *ScreenStateMachine) SwitchTab(target ScreenState) {
	if sm.combatActive || sm.tab == Closed {
		return
	}
	if target != CharacterTab && target != InventoryTab {
		return
	}
	if target == sm.tab {
		return
	}
	sm.tab = target
	sm.lastTab = target
	sm.startCrossfade()
}

// Close transitions any open state to Closed. CloseReasonUserRequest (Esc,
// close button) runs the 200ms paper-slide exit; the other reasons are
// forced closes — the overlay disappears in the same frame, no animation,
// no prompt (GDD transition table; equipment changes are already
// committed). A user close is refused while the level-up modal is active:
// only Confirm dismisses it (AC-18). Closing an already-closed UI is a
// no-op.
func (sm *ScreenStateMachine) Close(reason CloseReason) {
	if sm.tab == Closed {
		return
	}
	if reason == CloseReasonUserRequest && sm.modalActive() {
		return
	}
	sm.tab = Closed
	sm.fadeRemaining = 0
	if reason == CloseReasonUserRequest {
		sm.closing = true
		sm.sheet.Close()
		return
	}
	sm.closing = false
	sm.sheet.ForceClose()
}

// State returns the current screen state. LevelUp is reported while the
// level-up modal is active and the UI is open; a modal left pending by a
// forced close does not report LevelUp while the UI is hidden (the
// level-up is deferred and reappears on the next open).
func (sm *ScreenStateMachine) State() ScreenState {
	if sm.tab == Closed {
		return Closed
	}
	if sm.modalActive() {
		return LevelUp
	}
	return sm.tab
}

// Visible reports whether the overlay is on screen. True while open and
// while a user-initiated close's exit slide is still running; false
// immediately after a forced close (AC-16: the parchment overlay is gone).
func (sm *ScreenStateMachine) Visible() bool {
	if sm.tab != Closed {
		return true
	}
	return sm.closing && sm.sheet.Visible()
}

// Transitioning reports whether a tab-switch crossfade (200ms) is in
// flight.
func (sm *ScreenStateMachine) Transitioning() bool { return sm.fadeRemaining > 0 }

// Closing reports whether a user-initiated close's paper-slide exit is in
// flight. Forced closes never set it — the UI is gone in the same frame.
func (sm *ScreenStateMachine) Closing() bool { return sm.closing }

// CombatActive reports whether an encounter is running (C/I open commands
// are blocked while true, AC-15).
func (sm *ScreenStateMachine) CombatActive() bool { return sm.combatActive }

// LastTab returns the session-only last-used tab (CharacterTab initially).
func (sm *ScreenStateMachine) LastTab() ScreenState { return sm.lastTab }

// LastMember returns the session-only last-viewed party member — the
// member the UI opens to (the player character on a new session).
func (sm *ScreenStateMachine) LastMember() entity.GameEntityID { return sm.lastMember }

// SetMember records the party member the player is now viewing (the
// session-only last-viewed member). Story 008's sidebar calls this on
// selection; the state machine preserves the member across tab switches
// and close/open cycles.
func (sm *ScreenStateMachine) SetMember(id entity.GameEntityID) { sm.lastMember = id }

// Update advances the tab crossfade by dt seconds and polls the keyboard
// (C/I/Esc) when a host is attached — mirroring the combat HUD's polling.
// Esc closes the UI only while it is open (never steals Esc for pause when
// closed) and never fires while the level-up modal is active (AC-18). Key
// polling is edge-triggered (KeyDown), so held keys cannot repeat.
func (sm *ScreenStateMachine) Update(dt float64) {
	sm.advanceCrossfade(dt)
	if sm.closing && !sm.sheet.Visible() {
		sm.closing = false // the exit slide completed; the overlay hid
	}
	if sm.host == nil || sm.host.Window == nil {
		return
	}
	if sm.modalActive() {
		return // the level-up modal is mandatory; Esc/C/I never dismiss it
	}
	kb := &sm.host.Window.Keyboard
	if kb.KeyDown(hid.KeyboardKeyC) {
		sm.ToggleCharacter()
	}
	if kb.KeyDown(hid.KeyboardKeyI) {
		sm.ToggleInventory()
	}
	if kb.KeyDown(hid.KeyboardKeyEscape) {
		sm.Close(CloseReasonUserRequest)
	}
}

// Event handlers ------------------------------------------------------------

// onCombatStarted handles EventCombatStarted (AC-16): the combat-active
// flag blocks C/I open commands (AC-15) and an open UI is force-closed
// immediately — no animation, no prompt. The payload (turn order for the
// combat HUD) is not consumed; the flag semantics must hold regardless.
func (sm *ScreenStateMachine) onCombatStarted(entity.Event) {
	sm.combatActive = true
	sm.Close(CloseReasonCombatStart)
}

// onCombatEnded handles EventCombatEnded: the combat-active flag clears and
// C/I open commands work again (QA AC-15 edge: the flag is cleared
// synchronously, before the next input is processed).
func (sm *ScreenStateMachine) onCombatEnded(entity.Event) {
	sm.combatActive = false
}

// onLevelUp handles EventStatLevelUp (Story 002's forward contract: the
// subscriber lives here): it shows the level-up modal, deriving prevLevel
// and hpGain from the pre-award Stats component the store still holds
// during the synchronous dispatch, and hasAttributePoint from the
// milestone levels the award crossed (LevelUpGrantsAttributePoint). A
// level-up that arrives while the UI is closed is queued by the modal and
// reappears on the next open (deferred, matching the AC-16 forced-close
// edge).
func (sm *ScreenStateMachine) onLevelUp(ev entity.Event) {
	data, ok := ev.Data.(stats.LevelUpEventData)
	if !ok {
		slog.Warn("screen state machine: level-up payload type mismatch", "type", fmt.Sprintf("%T", ev.Data))
		return
	}
	if sm.modal == nil {
		return // markup never loaded; nothing to display
	}
	pre, ok := entity.GetComponent[stats.StatsComponent](sm.store, data.EntityID)
	if !ok {
		slog.Warn("screen state machine: level-up entity has no Stats component", "entity", data.EntityID)
		return
	}
	sm.modal.Show(data.EntityID, pre.Level, data.NewLevel,
		LevelUpHpGain(pre, data.NewLevel),
		LevelUpGrantsAttributePoint(pre.Level, data.NewLevel))
}

// Teardown unsubscribes the state machine's event handlers from the
// EventBus. Call when the UI system shuts down (event subscriptions must
// not outlive their owner). Safe to call once; subsequent calls are
// no-ops. Handlers are matched by code-pointer identity (see
// EventBusImpl.Unsubscribe); with a single state machine per bus the
// exact registrations are removed.
func (sm *ScreenStateMachine) Teardown() {
	if sm.bus == nil {
		return
	}
	sm.bus.Unsubscribe(combat.EventCombatStarted, sm.onCombatStarted)
	sm.bus.Unsubscribe(combat.EventCombatEnded, sm.onCombatEnded)
	sm.bus.Unsubscribe(stats.EventStatLevelUp, sm.onLevelUp)
	sm.bus = nil
}

// open shows the sheet on the given tab with the paper-slide enter,
// opening to the last-viewed party member (the player character on a new
// session). A failed open (the member no longer has a Stats component)
// leaves the UI closed.
func (sm *ScreenStateMachine) open(tab ScreenState) {
	member := sm.lastMember
	if member == 0 {
		member = sm.playerID
	}
	if err := sm.sheet.Open(member); err != nil {
		slog.Warn("screen state machine: open failed", "member", member, "err", err)
		return
	}
	sm.closing = false
	sm.tab = tab
	sm.lastTab = tab
}

// startCrossfade starts (or restarts) the 200ms tab-switch crossfade.
// Restarting replaces the in-flight crossfade — switches never queue; the
// latest target wins.
func (sm *ScreenStateMachine) startCrossfade() {
	sm.fadeRemaining = crossfadeDuration
}

// advanceCrossfade counts the crossfade timer down by dt seconds. Residual
// drift below crossfadeEpsilon snaps to zero so the timer settles
// deterministically.
func (sm *ScreenStateMachine) advanceCrossfade(dt float64) {
	if sm.fadeRemaining <= 0 {
		return
	}
	sm.fadeRemaining -= dt
	if sm.fadeRemaining < crossfadeEpsilon {
		sm.fadeRemaining = 0
	}
}

// modalActive reports whether the level-up modal is currently displayed.
func (sm *ScreenStateMachine) modalActive() bool {
	return sm.modal != nil && sm.modal.Active()
}
