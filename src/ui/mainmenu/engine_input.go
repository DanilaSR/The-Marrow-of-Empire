package mainmenu

import (
	"kaijuengine.com/engine"
	"kaijuengine.com/platform/hid"
)

// This file is the only engine-coupled code in the package: it polls the
// Kaiju host window's keyboard, gamepad, and mouse state and feeds the
// translated actions into the ScreenFlow. The polling patterns mirror the
// verified title screen implementation (src/ui/title_screen/
// title_screen_focus.go) and the hid API surface in kaiju/src/platform/hid/.

// Poll reads keyboard, gamepad, and mouse state from the host window and
// dispatches translated navigation actions to the flow. Call once per
// frame from the host UI update.
func Poll(host *engine.Host, flow *ScreenFlow) {
	pollKeyboard(host, flow)
	pollGamepad(host, flow)
	pollMouse(host, flow)
}

// pollKeyboard translates Tab/Shift+Tab, arrows, Enter/Space, and Escape
// (AC-1). First-match-wins per frame — the same polling semantics as the
// title screen focus controller.
func pollKeyboard(host *engine.Host, flow *ScreenFlow) {
	kb := &host.Window.Keyboard
	switch {
	case kb.KeyDown(hid.KeyboardKeyTab):
		flow.Dispatch(TranslateKey(KeyEvent{Key: "Tab", Shift: kb.HasShift()}))
	case kb.KeyDown(hid.KeyboardKeyEscape):
		flow.Dispatch(TranslateKey(KeyEvent{Key: "Escape"}))
	case kb.KeyDown(hid.KeyboardKeyEnter), kb.KeyDown(hid.KeyboardKeySpace):
		flow.Dispatch(TranslateKey(KeyEvent{Key: "Enter"}))
	case kb.KeyDown(hid.KeyboardKeyUp):
		flow.Dispatch(TranslateKey(KeyEvent{Key: "ArrowUp"}))
	case kb.KeyDown(hid.KeyboardKeyDown):
		flow.Dispatch(TranslateKey(KeyEvent{Key: "ArrowDown"}))
	case kb.KeyDown(hid.KeyboardKeyLeft):
		flow.Dispatch(TranslateKey(KeyEvent{Key: "ArrowLeft"}))
	case kb.KeyDown(hid.KeyboardKeyRight):
		flow.Dispatch(TranslateKey(KeyEvent{Key: "ArrowRight"}))
	}
}

// pollGamepad translates D-pad, A/B, and Start (AC-2). All connected
// devices are polled, mirroring the title screen's device loop.
func pollGamepad(host *engine.Host, flow *ScreenFlow) {
	ctl := &host.Window.Controller
	for i := 0; i < hid.ControllerMaxDevices; i++ {
		if !ctl.Available(i) {
			continue
		}
		dev := hid.ControllerButton(i) // device index is typed ControllerButton
		switch {
		case ctl.IsButtonDown(dev, hid.ControllerButtonUp):
			flow.Dispatch(TranslateGamepad(GamepadEvent{Button: "up"}))
		case ctl.IsButtonDown(dev, hid.ControllerButtonDown):
			flow.Dispatch(TranslateGamepad(GamepadEvent{Button: "down"}))
		case ctl.IsButtonDown(dev, hid.ControllerButtonLeft):
			flow.Dispatch(TranslateGamepad(GamepadEvent{Button: "left"}))
		case ctl.IsButtonDown(dev, hid.ControllerButtonRight):
			flow.Dispatch(TranslateGamepad(GamepadEvent{Button: "right"}))
		case ctl.IsButtonDown(dev, hid.ControllerButtonA):
			flow.Dispatch(TranslateGamepad(GamepadEvent{Button: "a"}))
		case ctl.IsButtonDown(dev, hid.ControllerButtonB):
			flow.Dispatch(TranslateGamepad(GamepadEvent{Button: "b"}))
		case ctl.IsButtonDown(dev, hid.ControllerButtonStart):
			flow.Dispatch(TranslateGamepad(GamepadEvent{Button: "start"}))
		}
	}
}

// pollMouse translates the scroll wheel (AC-3). Mouse clicks are wired by
// the screens themselves: hit-testing needs element geometry, which the
// screens own (Story 002+), and they dispatch ActivateAt via their
// document onclick handlers — the same action the router produces.
func pollMouse(host *engine.Host, flow *ScreenFlow) {
	m := &host.Window.Mouse
	if m.Scrolled() {
		scroll := 0
		switch {
		case m.ScrollY > 0:
			scroll = 1
		case m.ScrollY < 0:
			scroll = -1
		}
		flow.Dispatch(TranslateMouse(MouseEvent{Scroll: scroll}))
	}
}
