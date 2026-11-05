// pause_menu.go implements the PauseMenu controller (Story 006): the
// in-game pause overlay with the full world freeze (AC-1), the 200ms dim
// overlay (AC-1b), the five options (AC-1d), resume (AC-2), the combat
// pause variant with the Field register above the dim (AC-3), the quit
// confirmation dialog (AC-4), the 300ms toggle cooldown (AC-5), and the
// restricted-state gates (AC-6).
//
// Like the other screens the controller is pure Go: it owns state, focus,
// and timing only. All timing derives from the injectable clock. The world
// freeze is observable through the injected callbacks — the game layer
// implements them by suspending the engine update loop (ADR-0007: the game
// clock freezes; ADR-0011: GamePaused/GameResumed events) and tests inject
// counters. Kaiju rendering (dim quad, parchment panel) is intentionally
// absent — this file exposes the data model (RenderLayer order, DimAlpha,
// DrawCallDelta, Options) a Kaiju-aware rendering pass consumes.

package mainmenu

import "fmt"

// Timing and cooldown tuning knobs (GDD main-menu-and-settings Formulas
// section; data-driven, safe ranges from the Tuning Knobs table).
const (
	// PauseDimMs is pause_dim_ms: the dim overlay ramp duration (GDD:
	// 200ms). The world dims to 30% brightness over this interval on
	// pause entry and lifts back on exit.
	PauseDimMs = 200.0
	// PauseToggleCooldownMs is pause_toggle_cooldown_ms: two Escape
	// presses closer than this are collapsed to one (GDD edge cases:
	// rapid pause/unpause/pause within 500ms must not flicker). Boundary
	// inclusive — delta == cooldown toggles.
	PauseToggleCooldownMs = 300.0
	// PauseDimMaxAlpha is the target dim alpha: 0.7 (world dims to 30%
	// brightness, TR-ui-001 "world dims to 30%").
	PauseDimMaxAlpha = 0.7
)

// Pause menu option labels and the quit confirmation copy.
//
// LOCALIZATION: user-facing strings. Exposed as data here because the
// project has no string table yet; route through the localization system
// when one exists (UI rule: no hardcoded strings in logic). See
// NewGameConfirmMessage in title_screen.go for the convention.
const (
	PauseLabelResume         = "Resume"
	PauseLabelSave           = "Save"
	PauseLabelLoad           = "Load"
	PauseLabelSettings       = "Settings"
	PauseLabelQuitToMainMenu = "Quit to Main Menu"

	// PauseQuitConfirmMessage is the quit confirmation dialog body (AC-4).
	PauseQuitConfirmMessage = "Quit to Main Menu? Unsaved progress will be lost."
	PauseLabelSaveAndQuit   = "Save & Quit"
	// PauseLabelQuitWithoutSaving is the destructive option label; it uses
	// oxblood fill with ivory text (see QuitWithoutSavingStyle).
	PauseLabelQuitWithoutSaving = "Quit Without Saving"
	PauseLabelCancel            = "Cancel"
)

// PauseState is the inline UI state of the pause menu. The quit
// confirmation is an inline state — not a ScreenFlow state (GDD CR-4) —
// following the CloudConflictModal pattern: while it is open the game
// layer routes input to PauseMenu.Dispatch instead of the flow.
type PauseState int

const (
	// PauseStateMenu is the five-option pause menu (AC-1d).
	PauseStateMenu PauseState = iota
	// PauseStateConfirmQuit is the quit confirmation dialog (AC-4).
	PauseStateConfirmQuit
)

// String returns the state name for diagnostics and error messages.
func (s PauseState) String() string {
	switch s {
	case PauseStateMenu:
		return "Menu"
	case PauseStateConfirmQuit:
		return "ConfirmQuit"
	}
	return fmt.Sprintf("PauseState(%d)", int(s))
}

// Focusable element ids. Registration order IS the Tab order; the Kaiju
// rendering pass binds these ids to markup elements.
const (
	PauseFocusResume            = "resume"
	PauseFocusSave              = "save"
	PauseFocusLoad              = "load"
	PauseFocusSettings          = "settings"
	PauseFocusQuit              = "quit"
	PauseFocusSaveAndQuit       = "save-and-quit"
	PauseFocusQuitWithoutSaving = "quit-without-saving"
	PauseFocusCancel            = "cancel"
)

// RenderLayer is the z-order contract for the pause overlay (AC-1b, AC-3):
// the dim quad renders above the game world and below the Field register
// (combat HUD chrome), so no tactical information is obscured while paused
// (ADR-0012 register assignment). The desk panel renders above everything.
type RenderLayer int

const (
	// RenderLayerWorld is the game world pass (entities, world-space
	// anchored health bars and damage numbers).
	RenderLayerWorld RenderLayer = iota
	// RenderLayerDim is the pause dim quad (Desk register): a full-screen
	// ColorDimOverlay quad whose alpha is animated by DimAlpha.
	RenderLayerDim
	// RenderLayerField is the combat HUD chrome (turn order, AP bar,
	// ability grid) — renders ABOVE the dim and stays fully visible while
	// paused (AC-3).
	RenderLayerField
	// RenderLayerDesk is the pause menu panel itself.
	RenderLayerDesk
)

// PauseOption describes one of the five pause menu options (AC-1d) for the
// rendering pass: the focusable id and its localization-marked label.
type PauseOption struct {
	ID    string
	Label string
}

// PauseMenuOption configures a PauseMenu at construction time.
type PauseMenuOption func(*PauseMenu)

// WithPauseClock installs the game-time clock (milliseconds) that drives
// the dim animation and the toggle cooldown. Pass the same function given
// to NewTransitionSystem so overlay timing and screen transitions share
// one time source.
func WithPauseClock(now func() float64) PauseMenuOption {
	return func(p *PauseMenu) { p.now = now }
}

// WithGamePausedEvent installs the GamePaused event emission (ADR-0011):
// invoked on pause entry, before the world freeze. The game layer routes
// it to the EventBus.
func WithGamePausedEvent(fn func()) PauseMenuOption {
	return func(p *PauseMenu) { p.gamePausedFn = fn }
}

// WithGameResumedEvent installs the GameResumed event emission (ADR-0011):
// invoked on pause exit, after the world unfreezes.
func WithGameResumedEvent(fn func()) PauseMenuOption {
	return func(p *PauseMenu) { p.gameResumedFn = fn }
}

// WithPauseWorld installs the world-freeze callback (AC-1): invoked on
// pause entry. The game layer implements it by suspending the engine
// update loop — all simulation systems (NPC schedules, combat, wildlife,
// weather) stop advancing and the game clock freezes (ADR-0007). The tick
// in progress completes before the freeze (GDD edge case).
func WithPauseWorld(fn func()) PauseMenuOption {
	return func(p *PauseMenu) { p.pauseWorldFn = fn }
}

// WithResumeWorld installs the world-unfreeze callback (AC-2): invoked on
// pause exit. Systems resume from their frozen state; the game clock
// continues from the exact tick it was frozen at — no game time passes
// during the pause.
func WithResumeWorld(fn func()) PauseMenuOption {
	return func(p *PauseMenu) { p.resumeWorldFn = fn }
}

// WithCanPause installs the restricted-state probe (AC-6): false while a
// cutscene is playing, a region transition is in progress, or a save write
// is running. A blocked pause-open consumes the Escape without opening the
// menu. Without it the pause is always allowed.
func WithCanPause(fn func() bool) PauseMenuOption {
	return func(p *PauseMenu) { p.canPauseFn = fn }
}

// WithSkipCutscene installs the cutscene-skip routing callback (AC-6).
// The cutscene system owns the skip behavior (out of scope); the menu only
// provides the routing rule: when a pause-open is blocked by the CanPause
// probe, this callback is invoked. The game layer wires it to a guarded
// skip (checks the cutscene is active and skippable) so unskippable
// cutscenes and non-cutscene blocks (region transitions, mid-save) ignore
// the press entirely.
func WithSkipCutscene(fn func()) PauseMenuOption {
	return func(p *PauseMenu) { p.skipCutsceneFn = fn }
}

// WithHasUnsavedProgress installs the unsaved-progress probe (AC-4): true
// when the dirty flag is set since the last save. Quit to Main Menu shows
// the confirmation dialog only when this returns true. Without it the
// confirmation is skipped.
func WithHasUnsavedProgress(fn func() bool) PauseMenuOption {
	return func(p *PauseMenu) { p.hasUnsavedFn = fn }
}

// WithSaveAndQuit installs the Save & Quit callback (AC-4): runs the save
// flow (Story 004) and returns an error on failure. On success the menu
// transitions to the title screen; on failure the error is surfaced
// (QuitError) and the menu stays open — the player does NOT quit.
func WithSaveAndQuit(fn func() error) PauseMenuOption {
	return func(p *PauseMenu) { p.saveAndQuitFn = fn }
}

// WithQuitWithoutSaving installs the Quit Without Saving callback (AC-4):
// discards the session; the menu then transitions to the title screen. The
// button uses destructive styling (oxblood fill, see QuitWithoutSavingStyle).
func WithQuitWithoutSaving(fn func()) PauseMenuOption {
	return func(p *PauseMenu) { p.quitWithoutFn = fn }
}

// PauseMenu is the pause overlay controller (Story 006). It implements
// ScreenController and registers for ScreenPauseMenu.
//
// World freeze contract: OnEnter emits GamePaused and freezes the world via
// the injected callbacks; OnExit emits GameResumed and unfreezes. The
// freeze is observable through the callbacks — tests inject counters, and
// each simulation system exposes a TicksAdvanced() counter that must stay
// identical across all paused frames (AC-1c).
//
// All probes and callbacks are injected (functional options), so the
// controller is fully headless-testable: no engine, no goroutines. Timing
// is a pure function of the injectable clock.
type PauseMenu struct {
	flow  *ScreenFlow
	focus *FocusManager
	now   func() float64

	gamePausedFn   func()
	gameResumedFn  func()
	pauseWorldFn   func()
	resumeWorldFn  func()
	canPauseFn     func() bool
	skipCutsceneFn func()
	hasUnsavedFn   func() bool
	saveAndQuitFn  func() error
	quitWithoutFn  func()

	state PauseState

	// enteredAt is the clock value of the last OnEnter, or -1 while the
	// pause overlay is not active (world running). PauseActive is true
	// exactly while enteredAt >= 0 — the world freeze interval.
	enteredAt float64
	// exiting and exitStartedAt drive the dim lift after OnExit (AC-2):
	// the dim fades 0.7 -> 0 over PauseDimMs once the screen is left.
	exiting       bool
	exitStartedAt float64

	// lastToggleMs is the clock value of the last pause enter/exit toggle
	// (AC-5 cooldown). Initialized to a far-past sentinel so the first
	// toggle is always permitted.
	lastToggleMs float64

	// quitError surfaces a failed Save & Quit (AC-4 edge case); the
	// rendering pass shows it on the menu.
	quitError string
}

// NewPauseMenu creates the pause menu controller for the given flow and
// installs the pause gates (AC-5 cooldown, AC-6 restricted states) into
// it. The caller registers the screen with flow.RegisterController(
// ScreenPauseMenu, pause).
//
// The gates are installed here — not at registration — because the first
// Escape press that opens the menu arrives while InGame is current and is
// processed by the flow (Back/togglePause). The flow consults the gates
// through the closures installed by this constructor.
func NewPauseMenu(flow *ScreenFlow, opts ...PauseMenuOption) *PauseMenu {
	p := &PauseMenu{
		flow:         flow,
		focus:        NewFocusManager(NewFocusStyle(1.0)),
		now:          func() float64 { return 0 },
		state:        PauseStateMenu,
		enteredAt:    -1,
		lastToggleMs: -1e9,
	}
	for _, o := range opts {
		if o != nil {
			o(p)
		}
	}
	if p.now == nil {
		p.now = func() float64 { return 0 }
	}
	if flow != nil {
		flow.SetPauseEntryGate(p.entryGate)
		flow.SetPauseToggleGate(p.toggleGate)
	}
	p.registerFocusables()
	p.refreshEnabled()
	return p
}

// registerFocusables registers every focusable element in Tab order: the
// five options in the "menu" group, then the three confirmation buttons in
// the "confirm" group. The destructive buttons declare the oxblood
// background so the focus ring is selected with >= 3:1 contrast against it.
func (p *PauseMenu) registerFocusables() {
	for _, opt := range p.Options() {
		id := opt.ID
		_ = p.focus.Register(id, WithGroup("menu"), WithBackground(ColorParchment), WithActivate(p.activationFor(id)))
	}
	_ = p.focus.Register(PauseFocusSaveAndQuit, WithGroup("confirm"), WithBackground(ColorOxblood), WithActivate(p.onSaveAndQuit))
	_ = p.focus.Register(PauseFocusQuitWithoutSaving, WithGroup("confirm"), WithBackground(ColorOxblood), WithActivate(p.onQuitWithoutSaving))
	_ = p.focus.Register(PauseFocusCancel, WithGroup("confirm"), WithBackground(ColorParchment), WithActivate(p.onCancel))
}

// activationFor returns the activation handler for a menu option id.
func (p *PauseMenu) activationFor(id string) func() {
	switch id {
	case PauseFocusResume:
		return p.onResume
	case PauseFocusSave:
		return p.onSave
	case PauseFocusLoad:
		return p.onLoad
	case PauseFocusSettings:
		return p.onSettings
	case PauseFocusQuit:
		return p.onQuit
	}
	return func() {}
}

// OnEnter starts the pause overlay (AC-1): the dim begins ramping to
// PauseDimMaxAlpha over PauseDimMs, the menu focuses its first option, the
// GamePaused event is emitted (ADR-0011), and the world freezes via the
// injected callback. The tick in progress completes before the freeze —
// the game layer's freeze implementation finishes the current tick first
// (GDD edge case).
func (p *PauseMenu) OnEnter(prev Screen) {
	p.enteredAt = p.now()
	p.exiting = false
	p.state = PauseStateMenu
	p.quitError = ""
	p.refreshEnabled()
	p.focus.FocusNext()
	if p.gamePausedFn != nil {
		p.gamePausedFn()
	}
	if p.pauseWorldFn != nil {
		p.pauseWorldFn()
	}
}

// OnExit ends the pause overlay (AC-2): the world unfreezes via the
// injected callback, the GameResumed event is emitted, and the dim starts
// lifting (0.7 -> 0 over PauseDimMs). The game clock resumes from the exact
// tick it was frozen at — no game time passes while paused.
func (p *PauseMenu) OnExit() {
	p.exiting = true
	p.exitStartedAt = p.now()
	p.enteredAt = -1
	p.focus.ClearFocus()
	if p.resumeWorldFn != nil {
		p.resumeWorldFn()
	}
	if p.gameResumedFn != nil {
		p.gameResumedFn()
	}
}

// Focus returns the screen's focus manager for input routing.
func (p *PauseMenu) Focus() *FocusManager { return p.focus }

// Dispatch routes a translated input to the pause menu's inline states.
// While the quit confirmation dialog is open, the game layer must call
// this instead of the flow (CloudConflictModal pattern): flow-level Escape
// would resume the game (AC-2), but Escape in the confirmation cancels the
// quit (AC-4). In the menu state it is a no-op — the flow owns input
// routing there.
func (p *PauseMenu) Dispatch(in Input) bool {
	if p.state != PauseStateConfirmQuit {
		return false
	}
	if in.Action == ActionBack {
		p.setState(PauseStateMenu)
		return true
	}
	return dispatchFocusAction(p.focus, in)
}

// State returns the current inline UI state.
func (p *PauseMenu) State() PauseState { return p.state }

// PauseActive reports whether the world is frozen behind the overlay —
// exactly the interval between OnEnter and OnExit (AC-1c). The game layer
// uses it to gate world ticking and to skip combat HUD re-rendering
// (AC-3: the HUD is a static snapshot while paused; draw-call delta = 0).
func (p *PauseMenu) PauseActive() bool { return p.enteredAt >= 0 }

// Options returns the five pause menu options in registration order
// (AC-1d): Resume, Save, Load, Settings, Quit to Main Menu. The returned
// slice is a fresh copy; callers may not mutate the menu through it.
func (p *PauseMenu) Options() []PauseOption {
	return []PauseOption{
		{PauseFocusResume, PauseLabelResume},
		{PauseFocusSave, PauseLabelSave},
		{PauseFocusLoad, PauseLabelLoad},
		{PauseFocusSettings, PauseLabelSettings},
		{PauseFocusQuit, PauseLabelQuitToMainMenu},
	}
}

// DimAlpha returns the dim overlay alpha in [0, 0.7] (AC-1b):
// ease-out(clamp(elapsed / PauseDimMs, 0, 1)) × 0.7 while entering, and the
// mirrored fall while exiting (AC-2: the dim lifts 0.7 -> 0 over
// PauseDimMs after OnExit). The rendering pass applies this to the
// RenderLayerDim quad each frame. Exact and tolerance-free at the unit
// level (GDD measurement protocol) because it is a pure function of the
// injectable clock.
func (p *PauseMenu) DimAlpha() float64 {
	switch {
	case p.exiting:
		return (1 - p.dimProgress(p.now()-p.exitStartedAt)) * PauseDimMaxAlpha
	case p.enteredAt >= 0:
		return p.dimProgress(p.now()-p.enteredAt) * PauseDimMaxAlpha
	}
	return 0
}

// dimProgress returns the eased dim progress in [0,1]: quadratic ease-out
// of clamp(elapsed / PauseDimMs, 0, 1) (TR-ui-004 paper-slide motion:
// enter 200-300ms ease-out, no bounce/scale/spin/flip).
func (p *PauseMenu) dimProgress(elapsed float64) float64 {
	raw := elapsed / PauseDimMs
	if raw <= 0 {
		return 0
	}
	if raw >= 1 {
		return 1
	}
	e := 1 - raw
	return 1 - e*e
}

// DrawCallDelta returns the number of draw calls the pause overlay adds
// (AC-7): exactly 1 — the dim quad (RenderLayerDim) — while the dim is
// visible, 0 otherwise. The pause panel belongs to the desk register UI
// pass (already counted), and the combat HUD is NOT re-rendered while
// paused (AC-3), so the pause state totals in-combat draw calls + 1.
func (p *PauseMenu) DrawCallDelta() int {
	if p.DimAlpha() > 0 {
		return 1
	}
	return 0
}

// QuitError returns the last Save & Quit failure message (AC-4 edge case:
// a failed save does NOT quit; the message is shown on the menu), or ""
// when the last quit attempt succeeded or none was made.
func (p *PauseMenu) QuitError() string { return p.quitError }

// QuitConfirmMessage returns the confirmation dialog body (AC-4).
//
// LOCALIZATION: user-facing string (see PauseQuitConfirmMessage).
func (p *PauseMenu) QuitConfirmMessage() string { return PauseQuitConfirmMessage }

// QuitWithoutSavingStyle returns the destructive button styling contract
// (AC-4): oxblood fill with ivory text at >= 4.5:1 contrast (WCAG AA),
// amber hover edge.
func (p *PauseMenu) QuitWithoutSavingStyle() ButtonStyle {
	return ButtonStyle{
		Background:      ColorOxblood,
		Text:            ColorIvory,
		HoverEdge:       ColorAmber,
		ActiveUnderline: ColorAmber,
		DisabledOpacity: DisabledButtonOpacity,
	}
}

// entryGate is the AC-6 gate the flow consults before opening the pause
// menu: the pause opens only when CanPause() permits. When blocked, the
// Escape press routes to the injected cutscene-skip callback — the menu
// provides the routing rule only; the cutscene system owns the skip
// behavior (out of scope), so the injected callback must self-guard
// (active and skippable) to ignore unskippable cutscenes and non-cutscene
// blocks (region transitions, mid-save).
func (p *PauseMenu) entryGate() bool {
	if p.canPauseFn != nil && !p.canPauseFn() {
		if p.skipCutsceneFn != nil {
			p.skipCutsceneFn()
		}
		return false
	}
	return true
}

// toggleGate is the AC-5 cooldown gate the flow consults before every
// pause enter and exit: two toggles closer than PauseToggleCooldownMs are
// collapsed to one (no state transition, no dim animation reversal). The
// timestamp updates on each permitted toggle, so the cooldown window is
// measured from the toggle request (GDD measurement protocol: delta ==
// cooldown is permitted — boundary inclusive).
func (p *PauseMenu) toggleGate() bool {
	now := p.now()
	if now-p.lastToggleMs < PauseToggleCooldownMs {
		return false
	}
	p.lastToggleMs = now
	return true
}

// onResume is the Resume option activation (AC-2): fade out the dim,
// unfreeze the world (OnExit), and return to InGame.
func (p *PauseMenu) onResume() {
	if p.state != PauseStateMenu {
		return
	}
	p.resume()
}

// resume is the shared resume path (Resume button): stamps the cooldown
// timestamp (AC-5 — a rapid Escape after resuming must not reopen the
// menu) and requests the InGame transition. The world unfreezes in OnExit
// when the crossfade completes.
func (p *PauseMenu) resume() {
	p.lastToggleMs = p.now()
	_ = p.requestTransition(ScreenInGame)
}

// onSave opens the Save screen (Story 004) — the world stays frozen behind
// it.
func (p *PauseMenu) onSave() {
	if p.state != PauseStateMenu {
		return
	}
	_ = p.requestTransition(ScreenSaveGame)
}

// onLoad opens the Load screen (Story 004) — the world stays frozen behind
// it.
func (p *PauseMenu) onLoad() {
	if p.state != PauseStateMenu {
		return
	}
	_ = p.requestTransition(ScreenLoadGame)
}

// onSettings opens the Settings screen (Story 005); its in-game variant
// limits the Graphics category (no resolution/fullscreen changes while
// in-game, GDD CR-4).
func (p *PauseMenu) onSettings() {
	if p.state != PauseStateMenu {
		return
	}
	_ = p.requestTransition(ScreenSettings)
}

// onQuit is the Quit to Main Menu option (AC-4): with unsaved progress the
// confirmation dialog appears; without it the quit proceeds directly.
func (p *PauseMenu) onQuit() {
	if p.state != PauseStateMenu {
		return
	}
	if p.hasUnsaved() {
		p.setState(PauseStateConfirmQuit)
		return
	}
	_ = p.requestTransition(ScreenTitleScreen)
}

// onSaveAndQuit confirms the quit with a save (AC-4): the injected
// callback runs the save flow (Story 004), then the menu transitions to
// the title screen. A failed save does NOT quit — the error is surfaced on
// the menu and the player returns to the pause menu.
func (p *PauseMenu) onSaveAndQuit() {
	if p.state != PauseStateConfirmQuit {
		return
	}
	if p.saveAndQuitFn != nil {
		if err := p.saveAndQuitFn(); err != nil {
			p.quitError = err.Error()
			p.setState(PauseStateMenu)
			return
		}
	}
	p.quitError = ""
	_ = p.requestTransition(ScreenTitleScreen)
}

// onQuitWithoutSaving confirms the destructive quit (AC-4): the injected
// callback discards the session, then the menu transitions to the title
// screen.
func (p *PauseMenu) onQuitWithoutSaving() {
	if p.state != PauseStateConfirmQuit {
		return
	}
	if p.quitWithoutFn != nil {
		p.quitWithoutFn()
	}
	_ = p.requestTransition(ScreenTitleScreen)
}

// onCancel dismisses the confirmation dialog and returns to the pause menu
// (AC-4).
func (p *PauseMenu) onCancel() {
	if p.state != PauseStateConfirmQuit {
		return
	}
	p.setState(PauseStateMenu)
}

func (p *PauseMenu) hasUnsaved() bool { return p.hasUnsavedFn != nil && p.hasUnsavedFn() }

// requestTransition asks the flow to transition to target. Errors are
// propagated to the caller; the flow enforces the transition table. A nil
// flow (constructor misuse) makes navigation a silent no-op.
func (p *PauseMenu) requestTransition(target Screen) error {
	if p.flow == nil {
		return nil
	}
	return p.flow.RequestTransition(target)
}

// setState switches the inline UI state and refocuses the first enabled
// element of the new state.
func (p *PauseMenu) setState(s PauseState) {
	p.state = s
	p.refreshEnabled()
	p.focus.FocusNext()
}

// refreshEnabled re-applies the enabled state of every focusable element
// from the current inline state.
func (p *PauseMenu) refreshEnabled() {
	menu := p.state == PauseStateMenu
	for _, opt := range p.Options() {
		p.setEnabled(opt.ID, menu)
	}
	confirm := p.state == PauseStateConfirmQuit
	p.setEnabled(PauseFocusSaveAndQuit, confirm)
	p.setEnabled(PauseFocusQuitWithoutSaving, confirm)
	p.setEnabled(PauseFocusCancel, confirm)
}

func (p *PauseMenu) setEnabled(id string, on bool) {
	_ = p.focus.SetEnabled(id, on)
}
