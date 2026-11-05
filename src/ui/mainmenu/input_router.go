package mainmenu

// Action is a device-independent navigation intent. Every input device —
// keyboard, gamepad, mouse — translates into these same actions, so no
// screen requires a specific input device (AC-1/AC-2/AC-3, CR-8).
type Action int

const (
	// ActionNone is the null action (unmapped input).
	ActionNone Action = iota
	// ActionFocusNext is Tab: move to the next focusable.
	ActionFocusNext
	// ActionFocusPrev is Shift+Tab: move to the previous focusable.
	ActionFocusPrev
	// ActionMoveUp/Down/Left/Right are arrows/D-pad movement within a group.
	ActionMoveUp
	ActionMoveDown
	ActionMoveLeft
	ActionMoveRight
	// ActionActivate is Enter/Space/gamepad A/click.
	ActionActivate
	// ActionBack is Escape/gamepad B: back/cancel at every level.
	ActionBack
	// ActionTogglePause is gamepad Start: pause toggle (CR-4).
	ActionTogglePause
	// ActionScrollUp/Down are mouse wheel movements for sliders and lists.
	ActionScrollUp
	ActionScrollDown
)

// InputSource tags which device family produced an input, so the flow can
// manage focus-ring visibility (AC-4: ring shown on first keyboard/gamepad
// input, hidden during mouse navigation).
type InputSource int

const (
	SourceNone InputSource = iota
	SourceKeyboard
	SourceGamepad
	SourceMouse
)

// KeyEvent is a keyboard input in device-neutral form. Key names follow
// the hid constant names ("Tab", "Escape", "Enter", "Space", "ArrowUp",
// "ArrowDown", "ArrowLeft", "ArrowRight"); the engine adapter maps
// hid.KeyboardKeyX values onto them.
type KeyEvent struct {
	Key         string
	Shift, Ctrl bool
	Alt         bool
}

// GamepadEvent is a gamepad button input in device-neutral form
// ("up", "down", "left", "right", "a", "b", "start").
type GamepadEvent struct {
	Button string
}

// MouseEvent is a mouse input in device-neutral form.
type MouseEvent struct {
	// ClickedID is the element under the cursor when clicked ("" = none).
	// Screens that own hit-testing fill this; the flow focuses+activates.
	ClickedID string
	// Scroll is the wheel delta: +1 up, -1 down, 0 none.
	Scroll int
}

// Input is a translated navigation intent ready for ScreenFlow.Dispatch.
type Input struct {
	Action  Action
	Element string // click target for mouse activation
	Source  InputSource
}

// TranslateKey maps a keyboard event to a navigation action
// (AC-1: Tab/Shift+Tab, arrows, Enter/Space, Escape).
func TranslateKey(e KeyEvent) Input {
	switch e.Key {
	case "Tab":
		if e.Shift {
			return Input{Action: ActionFocusPrev, Source: SourceKeyboard}
		}
		return Input{Action: ActionFocusNext, Source: SourceKeyboard}
	case "ArrowUp":
		return Input{Action: ActionMoveUp, Source: SourceKeyboard}
	case "ArrowDown":
		return Input{Action: ActionMoveDown, Source: SourceKeyboard}
	case "ArrowLeft":
		return Input{Action: ActionMoveLeft, Source: SourceKeyboard}
	case "ArrowRight":
		return Input{Action: ActionMoveRight, Source: SourceKeyboard}
	case "Enter", "Space":
		return Input{Action: ActionActivate, Source: SourceKeyboard}
	case "Escape":
		return Input{Action: ActionBack, Source: SourceKeyboard}
	}
	return Input{}
}

// TranslateGamepad maps a gamepad event to a navigation action
// (AC-2: D-pad moves focus, A activates, B backs, Start toggles pause).
func TranslateGamepad(e GamepadEvent) Input {
	switch e.Button {
	case "up":
		return Input{Action: ActionMoveUp, Source: SourceGamepad}
	case "down":
		return Input{Action: ActionMoveDown, Source: SourceGamepad}
	case "left":
		return Input{Action: ActionMoveLeft, Source: SourceGamepad}
	case "right":
		return Input{Action: ActionMoveRight, Source: SourceGamepad}
	case "a":
		return Input{Action: ActionActivate, Source: SourceGamepad}
	case "b":
		return Input{Action: ActionBack, Source: SourceGamepad}
	case "start":
		return Input{Action: ActionTogglePause, Source: SourceGamepad}
	}
	return Input{}
}

// RouteRawKey implements the host input-routing contract (Story 008 AC-3):
// while the keybindings screen captures input, every raw key is routed to
// the screen FIRST; a HandleKey return of false means the key is not part of
// the capture and must be dispatched to the flow (dialog buttons stay
// keyboard-reachable). With no capture active the key is always dispatched.
//
// Returns true when the key was consumed by the screen (no key leakage to
// the flow or the game world — AC-4); false when it was dispatched.
//
// The game layer calls this for every raw key event before any game-world
// input handling. screen may be nil when no keybindings screen is wired.
func RouteRawKey(screen *KeybindingsScreen, flow *ScreenFlow, e KeyEvent) bool {
	if screen != nil && screen.CapturesInput() {
		if screen.HandleKey(e) {
			return true
		}
	}
	flow.Dispatch(TranslateKey(e))
	return false
}

// TranslateMouse maps a mouse event to a navigation action
// (AC-3: click to focus and activate, scroll wheel for sliders and lists).
func TranslateMouse(e MouseEvent) Input {
	switch {
	case e.ClickedID != "":
		return Input{Action: ActionActivate, Element: e.ClickedID, Source: SourceMouse}
	case e.Scroll > 0:
		return Input{Action: ActionScrollUp, Element: e.ClickedID, Source: SourceMouse}
	case e.Scroll < 0:
		return Input{Action: ActionScrollDown, Element: e.ClickedID, Source: SourceMouse}
	}
	return Input{}
}
