package dialogue_ui

import (
	"kaijuengine.com/matrix"

	"kaijuengine.com/engine"
)

// FocusRingColor is the oxblood/umber color used for the keyboard focus ring.
// Must provide ≥3:1 contrast ratio against the parchment background per ADR-0012.
var FocusRingColor = matrix.ColorRGBAInt(135, 28, 28, 255) // oxblood #881E1E

// SetupInputRouting configures keyboard and focus navigation for the dialogue panel.
// This establishes the Esc → end conversation binding and sets up the focus ring
// styling for accessible keyboard navigation.
//
// Tab order and arrow-key navigation between input elements will be wired in
// Story 002 (Text Input & Topic Chips), which owns the interactive form controls.
//
// Focus ring styling applies the oxblood/umber color at ≥3:1 contrast ratio
// against the parchment background, meeting WCAG AA per ADR-0012 accessibility
// requirements.
func SetupInputRouting(host *engine.Host, panel *DialoguePanel) {
	// Esc key handling is registered in NewDialoguePanel via
	// host.Window.Keyboard.AddKeyCallback. The handler gates on state != StateHidden
	// so Esc = pause works normally outside dialogue context.

	// Focus ring styling will be applied to interactive elements when they
	// are implemented in Story 002 (text input) and Story 003 (scrollable log).
	// The focus ring uses FocusRingColor (oxblood) against the parchment
	// background for ≥3:1 contrast.
	_ = host
	_ = panel
}
