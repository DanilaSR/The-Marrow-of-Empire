package mainmenu

// title_screen.go implements the TitleScreen controller (Story 002): the
// title screen with background fade-in, the four menu buttons, the New Game
// flow (confirmation dialog, difficulty selection, character name entry),
// and the disabled-state probes (AC-5/AC-6).
//
// The controller is pure Go: it owns state, focus, and timing only. All
// timing derives from the injectable clock (the same function passed to
// NewTransitionSystem), so fade-in assertions are exact and tolerance-free
// at the unit level (GDD measurement protocol). Kaiju HTML/CSS rendering
// (document creation, element styling) is intentionally absent — this file
// exposes the data model (BackgroundSpec, ButtonStyle, Particle) and
// per-frame getters (BackgroundOpacity, MenuVisible, State) that a
// Kaiju-aware rendering pass consumes.

import (
	"fmt"
	"strings"
)

// Timing and layout tuning knobs (GDD main-menu-and-settings Formulas
// section; data-driven, safe ranges from the Tuning Knobs table).
const (
	// TitleFadeInMs is title_fade_in_ms: the background fade-in duration
	// after the splash screens (GDD: 500ms, safe range 200-1000ms).
	TitleFadeInMs = 500.0
	// MenuElementDelayMs is menu_element_delay_ms: the delay before the
	// menu elements appear after the background fade-in starts (GDD: 300ms,
	// safe range 100-600ms).
	MenuElementDelayMs = 300.0
	// MenuElementFadeMs is the menu-element reveal fade duration. The
	// control manifest's paper-slide motion language is 200-300ms ease-out
	// (TR-ui-004); the reveal uses the lower bound.
	MenuElementFadeMs = 200.0
	// NewGameFadeMs is new_game_fade_ms: the crossfade to black when a new
	// game begins (GDD: 500ms, safe range 200-1000ms). The Loading screen
	// owns rendering this fade; the constant is exposed so the loading UI
	// can honor it.
	NewGameFadeMs = 500.0
	// DisabledButtonOpacity is the greyed-out opacity for disabled buttons
	// (GDD: 40% — "saturation is life; a dead control is a bleached
	// control", art bible 6.5).
	DisabledButtonOpacity = 0.4
	// GameVersion is the build version string shown bottom-right (AC-2).
	GameVersion = "0.1.0"
	// DefaultCharacterName is the pre-made character template name used
	// when the player confirms an empty name (MVP character creation
	// handoff, GDD CR-5).
	DefaultCharacterName = "Wayfarer"
	// NewGameConfirmMessage is the confirmation dialog body shown when
	// save files exist (AC-4).
	//
	// LOCALIZATION: user-facing string. Exposed as data here because the
	// project has no string table yet; route through the localization
	// system when one exists (UI rule: no hardcoded strings in logic).
	NewGameConfirmMessage = "Starting a new game will not overwrite your existing saves."
	// NoSavesTooltip is the Load Game tooltip when no saves exist (AC-6).
	//
	// LOCALIZATION: user-facing string (see NewGameConfirmMessage).
	NoSavesTooltip = "No saved games found."
	// Menu button labels. LOCALIZATION: user-facing strings (see
	// NewGameConfirmMessage); exposed as data for the Kaiju markup pass.
	LabelNewGame       = "New Game"
	LabelLoadGame      = "Load Game"
	LabelSettings      = "Settings"
	LabelQuitToDesktop = "Quit to Desktop"
)

// Difficulty is the new-game difficulty selection (AC-3). The menu owns the
// selection UI and stores the choice in the new game session; the Tactical
// Combat Engine owns the actual difficulty parameters (GDD CR-5).
type Difficulty int

const (
	// DifficultyNarrative is the story-focused difficulty.
	DifficultyNarrative Difficulty = iota
	// DifficultyNormal is the default difficulty.
	DifficultyNormal
	// DifficultyTactical is the challenge-focused difficulty.
	DifficultyTactical
)

// String returns the player-facing difficulty label.
func (d Difficulty) String() string {
	switch d {
	case DifficultyNarrative:
		return "Narrative"
	case DifficultyNormal:
		return "Normal"
	case DifficultyTactical:
		return "Tactical"
	}
	return fmt.Sprintf("Difficulty(%d)", int(d))
}

// TitleState is the inline UI state of the title screen. Difficulty
// selection, character name entry, and the new-game confirmation dialog are
// inline states of the TitleScreen — not ScreenFlow states (GDD CR-5).
type TitleState int

const (
	// TitleStateMenu is the main menu: New Game, Load Game, Settings,
	// Quit to Desktop.
	TitleStateMenu TitleState = iota
	// TitleStateConfirmNew is the new-game confirmation dialog shown when
	// save files exist (AC-4).
	TitleStateConfirmNew
	// TitleStateDifficulty is the difficulty selection (AC-3).
	TitleStateDifficulty
	// TitleStateNameEntry is the character name entry (AC-3).
	TitleStateNameEntry
	// TitleStateLoading is entered when the new game starts; the Loading
	// screen owns rendering. The state guards against double-activation
	// through the direct ActivateAt path (which bypasses the flow's
	// transition input block).
	TitleStateLoading
)

// Focusable element ids. Registration order IS the Tab order; the Kaiju
// rendering pass binds these ids to markup elements.
const (
	FocusNewGame             = "new-game"
	FocusLoadGame            = "load-game"
	FocusSettings            = "settings"
	FocusQuit                = "quit"
	FocusConfirmBegin        = "confirm-begin"
	FocusConfirmCancel       = "confirm-cancel"
	FocusDifficultyNarrative = "difficulty-narrative"
	FocusDifficultyNormal    = "difficulty-normal"
	FocusDifficultyTactical  = "difficulty-tactical"
	FocusNameBegin           = "name-begin"
	FocusNameCancel          = "name-cancel"
)

// Particle is a drifting ember/pollen particle of the title background
// (MVP stub: data model only — the Kaiju rendering pass animates and draws
// it). Coordinates are normalized [0,1] viewport units; velocity is in
// normalized units per second.
type Particle struct {
	X, Y    float64 // base position
	VX, VY  float64 // drift velocity
	Opacity float64 // base opacity [0,1]
	Radius  float64 // particle radius (normalized)
}

// BackgroundSpec is the title background composition (MVP stub). The art
// director owns the final composition (art bible State 6 storybook dusk
// vista); the menu owns the fade-in timing and the UI overlay.
type BackgroundSpec struct {
	// TopColor is the upper gradient color: deep warm umber dusk sky.
	TopColor Color
	// BottomColor is the lower gradient color: warm golden-hour glow.
	BottomColor Color
	// Particles are the drifting ember/pollen particles.
	Particles []Particle
}

// backgroundStub is the MVP background composition: a dusk gradient from
// dark stone (deep umber) to amber (warm gold) with a handful of drifting
// particles. Treat the returned spec as read-only.
var backgroundStub = BackgroundSpec{
	TopColor:    ColorDarkStone,
	BottomColor: ColorAmber,
	Particles: []Particle{
		{X: 0.35, Y: 0.60, VX: 0.000, VY: -0.040, Opacity: 0.80, Radius: 0.004},  // ember rising
		{X: 0.40, Y: 0.55, VX: 0.010, VY: -0.050, Opacity: 0.60, Radius: 0.003},  // ember rising
		{X: 0.30, Y: 0.65, VX: -0.010, VY: -0.030, Opacity: 0.90, Radius: 0.005}, // ember rising
		{X: 0.62, Y: 0.35, VX: 0.020, VY: -0.020, Opacity: 0.40, Radius: 0.002},  // pollen drift
		{X: 0.70, Y: 0.30, VX: 0.010, VY: 0.010, Opacity: 0.50, Radius: 0.003},   // pollen drift
		{X: 0.55, Y: 0.80, VX: 0.015, VY: -0.010, Opacity: 0.30, Radius: 0.002},  // dust mote
	},
}

// ButtonStyle is the menu button styling contract for the Kaiju CSS pass
// (AC-2): parchment-toned background, ink-dark text, warm gold hover edge,
// thread-red active underline. No skeuomorphic bevels or gradients
// (control manifest: pen-and-ink flat only).
type ButtonStyle struct {
	Background      Color
	Text            Color
	HoverEdge       Color
	ActiveUnderline Color
	DisabledOpacity float64
}

// TitleScreenOption configures a TitleScreen at construction time.
type TitleScreenOption func(*TitleScreen)

// WithClock installs the game-time clock (milliseconds) that drives all
// title screen timing. Pass the same function given to NewTransitionSystem
// so fade-in and screen transitions share one time source.
func WithClock(now func() float64) TitleScreenOption {
	return func(t *TitleScreen) { t.now = now }
}

// WithHasSaves installs the save-existence probe (AC-6): true when at
// least one save file exists. Without it the screen assumes no saves.
func WithHasSaves(fn func() bool) TitleScreenOption {
	return func(t *TitleScreen) { t.hasSavesFn = fn }
}

// WithIsSaving installs the save-write-in-progress probe (AC-5): true
// while a save write is running. Without it the screen assumes idle.
func WithIsSaving(fn func() bool) TitleScreenOption {
	return func(t *TitleScreen) { t.isSavingFn = fn }
}

// WithBeginNewGame installs the new-game start callback (AC-3): invoked
// with the chosen difficulty and character name after the player confirms.
func WithBeginNewGame(fn func(d Difficulty, name string)) TitleScreenOption {
	return func(t *TitleScreen) { t.beginNewGameFn = fn }
}

// WithQuitToDesktop installs the application-quit callback (AC-2).
func WithQuitToDesktop(fn func()) TitleScreenOption {
	return func(t *TitleScreen) { t.quitFn = fn }
}

// TitleScreen is the title screen controller (Story 002): the inline menu
// state machine, focus registry, and fade-in timing for the first screen
// the player sees. It implements ScreenController and registers for
// ScreenTitleScreen.
//
// All probes and callbacks are injected (functional options), so the
// controller is fully headless-testable: no filesystem, no engine, no
// goroutines. Timing is a pure function of the injectable clock.
type TitleScreen struct {
	flow  *ScreenFlow
	focus *FocusManager
	now   func() float64

	hasSavesFn     func() bool
	isSavingFn     func() bool
	beginNewGameFn func(Difficulty, string)
	quitFn         func()

	state       TitleState
	interactive bool
	enteredAt   float64

	selectedDifficulty Difficulty
	characterName      string
}

// NewTitleScreen creates the title screen controller for the given flow.
// The flow is required for navigation (RequestTransition); the caller
// registers the screen with flow.RegisterController(ScreenTitleScreen,
// title). Options inject the clock, save probes, and game callbacks.
//
// The fade-in sequence starts at construction: the flow begins in
// ScreenTitleScreen without a transition, so OnEnter is not guaranteed to
// fire at boot. OnEnter restarts the sequence on re-entry.
func NewTitleScreen(flow *ScreenFlow, opts ...TitleScreenOption) *TitleScreen {
	t := &TitleScreen{
		flow:               flow,
		focus:              NewFocusManager(NewFocusStyle(1.0)),
		now:                func() float64 { return 0 },
		state:              TitleStateMenu,
		interactive:        true,
		selectedDifficulty: DifficultyNormal,
	}
	for _, o := range opts {
		if o != nil {
			o(t)
		}
	}
	if t.now == nil {
		t.now = func() float64 { return 0 }
	}
	t.enteredAt = t.now() // boot-path sequence start (see doc comment)
	t.registerFocusables()
	t.refreshEnabled()
	return t
}

// registerFocusables registers every focusable element of the screen in Tab
// order. Elements are registered once and enabled/disabled per state, so
// the Tab order is stable across inline state changes.
func (t *TitleScreen) registerFocusables() {
	menu := []struct {
		id       string
		activate func()
	}{
		{FocusNewGame, t.onNewGame},
		{FocusLoadGame, t.onLoadGame},
		{FocusSettings, t.onSettings},
		{FocusQuit, t.onQuit},
	}
	for _, b := range menu {
		_ = t.focus.Register(b.id, WithGroup("menu"), WithBackground(ColorParchment), WithActivate(b.activate))
	}
	_ = t.focus.Register(FocusConfirmBegin, WithGroup("confirm"), WithBackground(ColorParchment), WithActivate(t.onConfirmBegin))
	_ = t.focus.Register(FocusConfirmCancel, WithGroup("confirm"), WithBackground(ColorParchment), WithActivate(t.onConfirmCancel))
	difficulties := []struct {
		id string
		d  Difficulty
	}{
		{FocusDifficultyNarrative, DifficultyNarrative},
		{FocusDifficultyNormal, DifficultyNormal},
		{FocusDifficultyTactical, DifficultyTactical},
	}
	for _, e := range difficulties {
		_ = t.focus.Register(e.id, WithGroup("difficulty"), WithBackground(ColorParchment), WithActivate(func() { t.onDifficulty(e.d) }))
	}
	_ = t.focus.Register(FocusNameBegin, WithGroup("name"), WithBackground(ColorParchment), WithActivate(t.onNameBegin))
	_ = t.focus.Register(FocusNameCancel, WithGroup("name"), WithBackground(ColorParchment), WithActivate(t.onNameCancel))
}

// OnEnter starts the entrance sequence (AC-1): the background fade-in over
// TitleFadeInMs and the menu reveal MenuElementDelayMs later. The screen
// resets to the menu state, re-applies the button enabled states, and
// focuses the first menu button.
func (t *TitleScreen) OnEnter(prev Screen) {
	t.enteredAt = t.now()
	t.state = TitleStateMenu
	t.selectedDifficulty = DifficultyNormal
	t.characterName = ""
	t.refreshEnabled()
	t.focus.FocusNext()
}

// OnExit stops the entrance sequence and clears focus. With the
// clock-driven timing model there are no timers or goroutines to cancel:
// the fade state resets so a stale render shows nothing mid-sequence.
func (t *TitleScreen) OnExit() {
	t.enteredAt = -1
	t.focus.ClearFocus()
}

// Focus returns the screen's focus manager for input routing.
func (t *TitleScreen) Focus() *FocusManager { return t.focus }

// SetInteractive toggles menu interactivity. Story 003 calls this when the
// first-launch accessibility prompt modal appears (false) and when it is
// dismissed (true): buttons stay visible (registered) but are skipped by
// focus traversal and reject activation while non-interactive (AC-1).
// Defaults to true until Story 003 exists.
func (t *TitleScreen) SetInteractive(on bool) {
	t.interactive = on
	t.refreshEnabled()
}

// Refresh re-polls the save probes and re-applies the button enabled
// states. Call once per frame from the game layer so probe changes (a save
// completing, saves being created) update the buttons reactively (AC-5 and
// AC-6 edge cases).
func (t *TitleScreen) Refresh() {
	t.refreshEnabled()
}

// elapsed returns the time since the entrance sequence started, or 0
// before the first entry.
func (t *TitleScreen) elapsed() float64 {
	if t.enteredAt < 0 {
		return 0
	}
	return t.now() - t.enteredAt
}

// BackgroundOpacity returns the background fade-in progress in [0,1]:
// clamp(elapsed / TitleFadeInMs). The rendering pass sets the background
// element's opacity from this every frame (AC-1).
func (t *TitleScreen) BackgroundOpacity() float64 {
	p := t.elapsed() / TitleFadeInMs
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		return 1
	}
	return p
}

// MenuVisible reports whether the menu elements have appeared (AC-1):
// true once MenuElementDelayMs has elapsed after the background fade-in
// started.
func (t *TitleScreen) MenuVisible() bool {
	return t.elapsed() >= MenuElementDelayMs
}

// MenuRevealProgress returns the menu reveal fade in [0,1]: 0 while hidden,
// ramping to 1 over MenuElementFadeMs after the delay.
func (t *TitleScreen) MenuRevealProgress() float64 {
	if !t.MenuVisible() {
		return 0
	}
	p := (t.elapsed() - MenuElementDelayMs) / MenuElementFadeMs
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		return 1
	}
	return p
}

// State returns the current inline UI state.
func (t *TitleScreen) State() TitleState { return t.state }

// Background returns the MVP background composition data model (gradient +
// particles). The art director owns the final composition; the returned
// spec is read-only data for the rendering pass.
func (t *TitleScreen) Background() BackgroundSpec { return backgroundStub }

// MenuButtonStyle returns the menu button styling contract for the Kaiju
// CSS pass (AC-2).
func (t *TitleScreen) MenuButtonStyle() ButtonStyle {
	return ButtonStyle{
		Background:      ColorParchment,
		Text:            ColorInk,
		HoverEdge:       ColorAmber,
		ActiveUnderline: ColorOxblood,
		DisabledOpacity: DisabledButtonOpacity,
	}
}

// Version returns the version label shown bottom-right (AC-2). The label
// is not focusable.
func (t *TitleScreen) Version() string { return "v" + GameVersion }

// LoadGameDisabled reports whether Load Game must be greyed out (AC-6):
// no save files exist. The rendering pass derives the 40% opacity and the
// NoSavesTooltip from this.
func (t *TitleScreen) LoadGameDisabled() bool { return !t.hasSaves() }

// NewGameDisabled reports whether New Game must be greyed out (AC-5): a
// save write is in progress.
func (t *TitleScreen) NewGameDisabled() bool { return t.isSaving() }

// SelectedDifficulty returns the currently selected difficulty
// (defaults to DifficultyNormal).
func (t *TitleScreen) SelectedDifficulty() Difficulty { return t.selectedDifficulty }

// SetSelectedDifficulty sets the difficulty selection. Invalid values are
// ignored.
func (t *TitleScreen) SetSelectedDifficulty(d Difficulty) {
	switch d {
	case DifficultyNarrative, DifficultyNormal, DifficultyTactical:
		t.selectedDifficulty = d
	}
}

// CharacterName returns the entered character name (MVP: simple string
// field on the pre-made character template).
func (t *TitleScreen) CharacterName() string { return t.characterName }

// SetCharacterName sets the character name entry, trimmed of surrounding
// whitespace.
func (t *TitleScreen) SetCharacterName(name string) {
	t.characterName = strings.TrimSpace(name)
}

func (t *TitleScreen) hasSaves() bool { return t.hasSavesFn != nil && t.hasSavesFn() }
func (t *TitleScreen) isSaving() bool { return t.isSavingFn != nil && t.isSavingFn() }

// requestTransition asks the flow to transition to target. Errors are
// propagated to the caller; the flow enforces the transition table and the
// no-double-transition rule. A nil flow (constructor misuse) makes
// navigation a silent no-op.
func (t *TitleScreen) requestTransition(target Screen) error {
	if t.flow == nil {
		return nil
	}
	return t.flow.RequestTransition(target)
}

// setState switches the inline UI state and refocuses the first enabled
// element of the new state.
func (t *TitleScreen) setState(s TitleState) {
	t.state = s
	t.refreshEnabled()
	t.focus.FocusNext()
}

// refreshEnabled re-applies the enabled state of every focusable element
// from the current state, the interactivity flag (Story 003 hook), and the
// save probes (AC-5/AC-6). Call after any of those change; the rendering
// pass also samples the LoadGameDisabled/NewGameDisabled getters per frame.
func (t *TitleScreen) refreshEnabled() {
	menu := t.interactive && t.state == TitleStateMenu
	t.setEnabled(FocusNewGame, menu && !t.isSaving())
	t.setEnabled(FocusLoadGame, menu && t.hasSaves())
	t.setEnabled(FocusSettings, menu)
	t.setEnabled(FocusQuit, menu)

	confirm := t.interactive && t.state == TitleStateConfirmNew
	t.setEnabled(FocusConfirmBegin, confirm)
	t.setEnabled(FocusConfirmCancel, confirm)

	difficulty := t.interactive && t.state == TitleStateDifficulty
	t.setEnabled(FocusDifficultyNarrative, difficulty)
	t.setEnabled(FocusDifficultyNormal, difficulty)
	t.setEnabled(FocusDifficultyTactical, difficulty)

	name := t.interactive && t.state == TitleStateNameEntry
	t.setEnabled(FocusNameBegin, name)
	t.setEnabled(FocusNameCancel, name)
}

func (t *TitleScreen) setEnabled(id string, on bool) {
	_ = t.focus.SetEnabled(id, on)
}

// onNewGame is the New Game activation (AC-3/AC-4): with existing saves a
// confirmation dialog appears first; without saves the difficulty selection
// appears. Ignored while a save write is in progress (AC-5) or when the
// accessibility prompt is active.
func (t *TitleScreen) onNewGame() {
	if t.state != TitleStateMenu || !t.interactive || t.isSaving() {
		return
	}
	if t.hasSaves() {
		t.setState(TitleStateConfirmNew)
		return
	}
	t.setState(TitleStateDifficulty)
}

// onLoadGame is the Load Game activation (AC-6): greyed out when no saves
// exist.
func (t *TitleScreen) onLoadGame() {
	if t.state != TitleStateMenu || !t.interactive || !t.hasSaves() {
		return
	}
	_ = t.requestTransition(ScreenLoadGame)
}

// onSettings is the Settings activation.
func (t *TitleScreen) onSettings() {
	if t.state != TitleStateMenu || !t.interactive {
		return
	}
	_ = t.requestTransition(ScreenSettings)
}

// onQuit is the Quit to Desktop activation (AC-2).
func (t *TitleScreen) onQuit() {
	if t.state != TitleStateMenu || !t.interactive {
		return
	}
	if t.quitFn != nil {
		t.quitFn()
	}
}

// onDifficulty selects a difficulty and proceeds to character name entry
// (AC-3).
func (t *TitleScreen) onDifficulty(d Difficulty) {
	if t.state != TitleStateDifficulty || !t.interactive {
		return
	}
	t.selectedDifficulty = d
	t.setState(TitleStateNameEntry)
}

// onConfirmBegin confirms the new-game dialog and proceeds to difficulty
// selection (AC-4).
func (t *TitleScreen) onConfirmBegin() {
	if t.state != TitleStateConfirmNew || !t.interactive {
		return
	}
	t.setState(TitleStateDifficulty)
}

// onConfirmCancel cancels the new-game dialog, returning to the menu
// (AC-4).
func (t *TitleScreen) onConfirmCancel() {
	if t.state != TitleStateConfirmNew || !t.interactive {
		return
	}
	t.setState(TitleStateMenu)
}

// onNameBegin starts the new game (AC-3): transition to the Loading state
// (hard cut per Story 001), then invoke the injected BeginNewGame callback
// with the chosen difficulty and name. An empty name falls back to the
// pre-made template name (DefaultCharacterName).
func (t *TitleScreen) onNameBegin() {
	if t.state != TitleStateNameEntry || !t.interactive {
		return
	}
	name := t.characterName
	if name == "" {
		name = DefaultCharacterName
	}
	if err := t.requestTransition(ScreenLoading); err != nil {
		return
	}
	t.state = TitleStateLoading
	if t.beginNewGameFn != nil {
		t.beginNewGameFn(t.selectedDifficulty, name)
	}
}

// onNameCancel cancels character name entry and returns to the menu.
func (t *TitleScreen) onNameCancel() {
	if t.state != TitleStateNameEntry || !t.interactive {
		return
	}
	t.setState(TitleStateMenu)
}
