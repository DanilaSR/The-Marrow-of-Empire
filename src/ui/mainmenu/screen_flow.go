package mainmenu

import (
	"errors"
	"fmt"
)

// Screen is a state of the 7-state screen-flow machine (GDD states and
// transitions table).
type Screen int

const (
	// ScreenTitleScreen is the initial state after engine boot: title
	// background with menu options (New Game, Load Game, Settings, Quit).
	ScreenTitleScreen Screen = iota
	// ScreenLoadGame is the save-slot browser for loading.
	ScreenLoadGame
	// ScreenInGame is normal gameplay; the pause menu is NOT visible.
	ScreenInGame
	// ScreenPauseMenu is the in-game overlay; the world is frozen.
	ScreenPauseMenu
	// ScreenSaveGame is the save-slot browser, entered from PauseMenu.
	ScreenSaveGame
	// ScreenSettings is the settings screen, entered from TitleScreen or
	// PauseMenu. It remembers its entry point for the back destination
	// (AC-8).
	ScreenSettings
	// ScreenLoading is the transitional state: black screen, data being
	// loaded, input locked. Uses a hard cut, not a crossfade.
	ScreenLoading
	// ScreenKeybindings is the Controls → Keybindings sub-screen (Story
	// 007): the per-action keybinding table with capture mode. Entered from
	// the Settings screen's Controls category only (the GDD transition
	// table lists no direct PauseMenu → Keybindings edge).
	ScreenKeybindings
)

// String returns the state name for diagnostics and error messages.
func (s Screen) String() string {
	switch s {
	case ScreenTitleScreen:
		return "TitleScreen"
	case ScreenLoadGame:
		return "LoadGame"
	case ScreenInGame:
		return "InGame"
	case ScreenPauseMenu:
		return "PauseMenu"
	case ScreenSaveGame:
		return "SaveGame"
	case ScreenSettings:
		return "Settings"
	case ScreenLoading:
		return "Loading"
	case ScreenKeybindings:
		return "Keybindings"
	}
	return fmt.Sprintf("Screen(%d)", int(s))
}

// validTransitions is the GDD transition table. Loading is hard-cut into
// from every screen that can start a load or new-game flow; the Loading
// state itself locks input until it resolves to InGame or TitleScreen.
var validTransitions = map[Screen][]Screen{
	ScreenTitleScreen: {ScreenLoadGame, ScreenSettings, ScreenInGame, ScreenLoading},
	ScreenLoadGame:    {ScreenTitleScreen, ScreenInGame, ScreenPauseMenu, ScreenLoading},
	ScreenInGame:      {ScreenPauseMenu, ScreenTitleScreen, ScreenLoading},
	ScreenPauseMenu:   {ScreenInGame, ScreenSaveGame, ScreenLoadGame, ScreenSettings, ScreenTitleScreen, ScreenLoading},
	ScreenSaveGame:    {ScreenPauseMenu},
	ScreenSettings:    {ScreenTitleScreen, ScreenPauseMenu, ScreenKeybindings},
	ScreenKeybindings: {ScreenSettings},
	ScreenLoading:     {ScreenInGame, ScreenTitleScreen},
}

// screenCanTransition reports whether target is a valid transition from
// current per the GDD table.
func screenCanTransition(current, target Screen) bool {
	for _, s := range validTransitions[current] {
		if s == target {
			return true
		}
	}
	return false
}

// Flow errors. Callers branch on these with errors.Is to drive dialogs
// (the quit confirmation UI is Story 006's concern; the flow enforces the
// rule).
var (
	// ErrInvalidTransition is returned when the target is not reachable
	// from the current screen per the GDD transition table.
	ErrInvalidTransition = errors.New("invalid screen transition")
	// ErrTransitionInProgress is returned when a transition is requested
	// while another is still fading (AC-7: no double-transitions).
	ErrTransitionInProgress = errors.New("a screen transition is already in progress")
	// ErrUnsavedProgress is returned when InGame → TitleScreen is
	// requested with unsaved progress and the quit is not confirmed
	// (AC-10). Call ConfirmQuit to proceed.
	ErrUnsavedProgress = errors.New("unsaved progress requires confirmation before quitting to the title screen")
	// ErrNoPendingQuit is returned by ConfirmQuit when there is no
	// pending unconfirmed quit to confirm.
	ErrNoPendingQuit = errors.New("no pending quit to confirm")
	// ErrInputLocked is returned for navigation while Loading.
	ErrInputLocked = errors.New("input is locked while loading")
)

// noPendingScreen marks "no transition pending" (the zero Screen value is
// a real state, so a sentinel is required).
const noPendingScreen Screen = -1

// SaveLoader is the seam between the flow and the save/load system
// (Story 004 implements the real loader; tests inject stubs). The flow
// never touches the filesystem.
type SaveLoader interface {
	// Load loads the given slot. A non-nil error (checksum mismatch,
	// version mismatch, missing file) aborts into TitleScreen (AC-9).
	Load(slot string) error
}

// ScreenController is implemented by screens plugged into the flow
// (Stories 002-006). The flow drives lifecycle callbacks and reads the
// screen's focus manager for input routing.
type ScreenController interface {
	// OnEnter is called when the screen becomes current, with the screen
	// it was entered from.
	OnEnter(prev Screen)
	// OnExit is called when the screen stops being current.
	OnExit()
	// Focus returns the screen's focus manager, or nil if the screen has
	// no interactive elements.
	Focus() *FocusManager
}

// FlowOption configures a ScreenFlow at construction time.
type FlowOption func(*ScreenFlow)

// WithUnsavedProgressCheck installs the unsaved-progress probe used by the
// AC-10 confirmation gate. When it returns true, InGame → TitleScreen is
// blocked until ConfirmQuit.
func WithUnsavedProgressCheck(fn func() bool) FlowOption {
	return func(f *ScreenFlow) { f.unsavedProgress = fn }
}

// WithSaveLoader installs the save loader used by BeginLoad (AC-9).
func WithSaveLoader(loader SaveLoader) FlowOption {
	return func(f *ScreenFlow) { f.saveLoader = loader }
}

// ScreenFlow is the 7-state screen-flow state machine: it validates
// transitions against the GDD table, runs the crossfade/hard-cut via
// TransitionSystem, maintains the back-navigation stack (AC-8), enforces
// the quit confirmation gate (AC-10), and discards input during
// transitions and while Loading (AC-7).
type ScreenFlow struct {
	transition *TransitionSystem
	current    Screen
	pending    Screen
	backStack  []Screen

	controllers     map[Screen]ScreenController
	unsavedProgress func() bool
	saveLoader      SaveLoader

	// Story 006 pause gates, installed by the PauseMenu controller at
	// construction: consulted before every pause menu enter/exit.
	pauseEntryGate  func() bool // AC-6 restricted states (open only)
	pauseToggleGate func() bool // AC-5 toggle cooldown (open and close)

	quitConfirmed bool
	lastError     string
}

// NewScreenFlow creates the flow in the ScreenTitleScreen state, the GDD
// initial state after engine boot.
func NewScreenFlow(ts *TransitionSystem, opts ...FlowOption) *ScreenFlow {
	f := &ScreenFlow{
		transition:  ts,
		current:     ScreenTitleScreen,
		pending:     noPendingScreen,
		controllers: map[Screen]ScreenController{},
	}
	for _, o := range opts {
		if o != nil {
			o(f)
		}
	}
	return f
}

// RegisterController plugs a screen controller into the flow. Each screen
// may register at most once.
func (f *ScreenFlow) RegisterController(s Screen, ctrl ScreenController) error {
	if ctrl == nil {
		return errors.New("nil screen controller")
	}
	if _, ok := f.controllers[s]; ok {
		return fmt.Errorf("screen %s already has a controller", s)
	}
	f.controllers[s] = ctrl
	return nil
}

// SetPauseEntryGate installs the pause-open gate (Story 006 AC-6): called
// when a pause toggle would open the pause menu (Escape at InGame, gamepad
// Start). Returning false consumes the toggle without a transition. The
// PauseMenu controller installs this from its CanPause probe (cutscenes,
// region transitions, mid-save). A nil gate allows the pause to open.
func (f *ScreenFlow) SetPauseEntryGate(fn func() bool) { f.pauseEntryGate = fn }

// SetPauseToggleGate installs the pause toggle gate (Story 006 AC-5):
// called before every pause enter AND exit; returning false consumes the
// toggle without a transition (no dim animation reversal). The PauseMenu
// controller installs this from its 300ms cooldown state. A nil gate
// allows every toggle.
func (f *ScreenFlow) SetPauseToggleGate(fn func() bool) { f.pauseToggleGate = fn }

// Current returns the active screen.
func (f *ScreenFlow) Current() Screen { return f.current }

// ActiveFocus returns the focus manager of the active screen, or nil.
func (f *ScreenFlow) ActiveFocus() *FocusManager {
	if ctrl, ok := f.controllers[f.current]; ok && ctrl != nil {
		return ctrl.Focus()
	}
	return nil
}

// InputBlocked reports whether input must be discarded: during a
// transition (AC-7) or while Loading.
func (f *ScreenFlow) InputBlocked() bool {
	return f.transition.Active() || f.current == ScreenLoading
}

// LastError returns the most recent load failure message (AC-9), or ""
// when the last load succeeded. The UI layer renders this; the flow owns
// the state only.
func (f *ScreenFlow) LastError() string { return f.lastError }

// RequestTransition validates and starts a transition to target.
//
//   - While another transition is active the request is rejected
//     (AC-7: no double-transitions).
//   - Transitions not in the GDD table are rejected.
//   - InGame → TitleScreen with unsaved progress requires confirmation
//     (AC-10): the request is held pending and ErrUnsavedProgress is
//     returned; ConfirmQuit proceeds, CancelQuit abandons.
//   - All screen-to-screen transitions crossfade at ScreenCrossfadeMs;
//     Loading uses a hard cut.
//
// The new screen becomes current when the transition completes
// (call Update each frame).
func (f *ScreenFlow) RequestTransition(target Screen) error {
	if target == f.current {
		return nil
	}
	if f.transition.Active() {
		return ErrTransitionInProgress
	}
	if !screenCanTransition(f.current, target) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, f.current, target)
	}
	// AC-10: quitting to the title screen from gameplay with unsaved
	// progress always requires confirmation.
	if f.current == ScreenInGame && target == ScreenTitleScreen &&
		f.hasUnsavedProgress() && !f.quitConfirmed {
		f.pending = target
		return ErrUnsavedProgress
	}
	f.begin(target)
	return nil
}

// ConfirmQuit confirms the pending unconfirmed InGame → TitleScreen quit
// (AC-10) and starts the transition. Returns ErrNoPendingQuit when no
// unconfirmed quit is pending.
func (f *ScreenFlow) ConfirmQuit() error {
	if f.current != ScreenInGame || f.pending != ScreenTitleScreen {
		return ErrNoPendingQuit
	}
	f.quitConfirmed = true
	f.begin(ScreenTitleScreen)
	return nil
}

// CancelQuit abandons the pending unconfirmed quit; the player stays
// in-game.
func (f *ScreenFlow) CancelQuit() {
	if f.pending == ScreenTitleScreen {
		f.pending = noPendingScreen
	}
	f.quitConfirmed = false
}

// Back implements the universal back/cancel: with a back stack it returns
// to the previous screen (AC-8: Settings and Load/Save return to their
// entry point); at InGame it opens the pause menu (Escape = pause, CR-4 —
// InGame is gameplay, not a sub-screen, so the back stack does not apply);
// at TitleScreen it is a no-op.
//
// Pause toggles are gated (Story 006): the entry gate (AC-6 restricted
// states) applies when opening the menu, and the toggle gate (AC-5
// cooldown) applies to every pause enter and exit. A blocked gate consumes
// the press without a transition.
//
// The back-stack entry is validated before removal: if the transition is
// rejected (e.g. AC-10 quit-confirmation gate), the entry is preserved and
// the error propagates to the caller.
func (f *ScreenFlow) Back() error {
	if f.InputBlocked() {
		return ErrInputLocked
	}
	if f.current == ScreenInGame {
		if !f.pauseToggleAllowed() {
			return nil
		}
		return f.RequestTransition(ScreenPauseMenu)
	}
	if n := len(f.backStack); n > 0 {
		prev := f.backStack[n-1]
		// Escape inside the pause menu is a resume toggle (AC-2): the
		// cooldown gate applies, so rapid pause/unpause cannot flicker
		// (AC-5).
		if f.current == ScreenPauseMenu && !f.pauseToggleAllowed() {
			return nil
		}
		if err := f.RequestTransition(prev); err != nil {
			return err
		}
		f.backStack = f.backStack[:n-1]
		return nil
	}
	return nil
}

// BeginLoad starts a load operation (AC-9): hard cut to Loading, run the
// save loader, then transition to InGame on success or TitleScreen with
// the failure message on error. With an asynchronous loader, Story 004
// calls CompleteLoad instead once the result arrives.
func (f *ScreenFlow) BeginLoad(slot string) error {
	if f.current == ScreenLoading {
		return ErrInputLocked
	}
	if err := f.RequestTransition(ScreenLoading); err != nil {
		return err
	}
	if f.saveLoader == nil {
		f.lastError = "no save loader configured"
		return f.RequestTransition(ScreenTitleScreen)
	}
	f.CompleteLoad(f.saveLoader.Load(slot))
	return nil
}

// CompleteLoad resolves a Loading state with the load result: success
// proceeds to InGame, failure transitions to TitleScreen with the error
// message stored for the UI (AC-9). Safe to call only while Loading.
func (f *ScreenFlow) CompleteLoad(err error) {
	if f.current != ScreenLoading {
		return
	}
	if err != nil {
		f.lastError = err.Error()
		f.RequestTransition(ScreenTitleScreen)
		return
	}
	f.lastError = ""
	f.RequestTransition(ScreenInGame)
}

// Dispatch routes a translated input to the active screen. Input arriving
// during a crossfade or while Loading is discarded (AC-7), returning
// false. Flow-level actions (Back, TogglePause) work even when the current
// screen has no focus manager; focus actions are delegated to the active
// screen's focus manager.
func (f *ScreenFlow) Dispatch(in Input) bool {
	if f.InputBlocked() {
		return false
	}
	// Flow-level actions: Back and TogglePause are handled by the screen
	// flow itself and do not require a focus manager.
	if ok, result := f.tryFlowAction(in); ok {
		return result
	}
	fm := f.ActiveFocus()
	if fm == nil {
		return false
	}
	return dispatchFocusAction(fm, in)
}

// tryFlowAction handles actions that are owned by the screen flow, not by
// the active screen's focus manager. It reports (true, result) when the
// action was consumed; (false, _) means it was not a flow-level action.
func (f *ScreenFlow) tryFlowAction(in Input) (bool, bool) {
	switch in.Action {
	case ActionBack:
		return true, f.Back() == nil
	case ActionTogglePause:
		return true, f.togglePause()
	}
	return false, false
}

// togglePause implements the Escape/Start pause toggle (CR-4): InGame →
// PauseMenu, PauseMenu → InGame. Other screens ignore the toggle. Both
// directions are gated by the Story 006 toggle gate (AC-5 cooldown); the
// open direction additionally by the entry gate (AC-6 restricted states).
func (f *ScreenFlow) togglePause() bool {
	switch f.current {
	case ScreenInGame:
		if !f.pauseToggleAllowed() {
			return false
		}
		return f.RequestTransition(ScreenPauseMenu) == nil
	case ScreenPauseMenu:
		if !f.pauseToggleAllowed() {
			return false
		}
		return f.RequestTransition(ScreenInGame) == nil
	}
	return false
}

// pauseToggleAllowed consults the Story 006 pause gates. The entry gate
// (AC-6) applies only when opening the pause menu from InGame; the toggle
// gate (AC-5 cooldown) applies to every pause enter and exit. A false
// result consumes the press (no transition). With no gates installed every
// toggle is allowed.
func (f *ScreenFlow) pauseToggleAllowed() bool {
	if f.current == ScreenInGame && f.pauseEntryGate != nil && !f.pauseEntryGate() {
		return false
	}
	return f.pauseToggleGate == nil || f.pauseToggleGate()
}

// dispatchFocusAction delegates a navigation action to a focus manager,
// setting the input method from the action's source device so ring
// visibility tracks correctly (AC-4). Shared by the screen flow and the
// AccessibilityPrompt overlay (Story 003), which routes its own input while
// visible.
func dispatchFocusAction(fm *FocusManager, in Input) bool {
	fm.SetInputMethod(inputMethodFor(in.Source))
	switch in.Action {
	case ActionFocusNext:
		return fm.FocusNext()
	case ActionFocusPrev:
		return fm.FocusPrev()
	case ActionMoveUp:
		return fm.MoveGroup(DirectionUp)
	case ActionMoveDown:
		return fm.MoveGroup(DirectionDown)
	case ActionMoveLeft:
		return fm.MoveGroup(DirectionLeft)
	case ActionMoveRight:
		return fm.MoveGroup(DirectionRight)
	case ActionActivate:
		if in.Element != "" {
			return fm.ActivateAt(in.Element)
		}
		return fm.Activate()
	case ActionScrollUp:
		return fm.Scroll(1)
	case ActionScrollDown:
		return fm.Scroll(-1)
	}
	return false
}

// Update advances the transition; when a crossfade completes (or a hard
// cut resolves), the pending screen becomes current and the controllers
// receive OnExit/OnEnter. Call once per frame.
func (f *ScreenFlow) Update() {
	if f.pending == noPendingScreen || !f.transition.Active() {
		return
	}
	if !f.transition.Update() {
		return
	}
	f.finishSwap()
}

// begin pushes the current screen onto the back stack and starts the
// transition (crossfade, or hard cut for Loading which resolves
// immediately).
func (f *ScreenFlow) begin(target Screen) {
	f.backStack = append(f.backStack, f.current)
	f.pending = target
	mode := TransitionCrossfade
	if target == ScreenLoading {
		mode = TransitionHardCut
	}
	f.transition.Begin(mode)
	if mode == TransitionHardCut {
		f.Update()
	}
}

// finishSwap makes the pending screen current and fires lifecycle
// callbacks. The quit confirmation resets so the next InGame → TitleScreen
// requires confirmation again (AC-10).
func (f *ScreenFlow) finishSwap() {
	prev, target := f.current, f.pending
	f.pending = noPendingScreen
	if ctrl, ok := f.controllers[prev]; ok {
		ctrl.OnExit()
	}
	f.current = target
	if ctrl, ok := f.controllers[target]; ok {
		ctrl.OnEnter(prev)
	}
	f.quitConfirmed = false
}

// hasUnsavedProgress reports whether the AC-10 gate applies; with no probe
// installed the flow assumes no unsaved progress.
func (f *ScreenFlow) hasUnsavedProgress() bool {
	return f.unsavedProgress != nil && f.unsavedProgress()
}

// inputMethodFor maps an input source onto the ring-visibility method:
// keyboard and gamepad show the ring, mouse hides it (AC-4).
func inputMethodFor(src InputSource) InputMethod {
	switch src {
	case SourceKeyboard:
		return InputMethodKeyboard
	case SourceGamepad:
		return InputMethodGamepad
	case SourceMouse:
		return InputMethodMouse
	}
	return InputMethodNone
}
