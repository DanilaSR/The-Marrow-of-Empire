// Package dialogue_ui implements the dialogue panel framework for Marrow of Empire.
// It provides the panel lifecycle, state machine, and input routing for NPC conversations.
//
// Story: EPIC-DialogueUI-001 — Dialogue Panel Framework & Lifecycle
// ADR: ADR-0012 (UI Framework — Desk edge register)
package dialogue_ui

// PanelState represents the current state of the dialogue panel.
// States transition linearly: Hidden → Active → Processing → Response → Active (loop) or Hidden (exit).
type PanelState int

const (
	// StateHidden is the initial state. The panel is not visible.
	StateHidden PanelState = iota
	// StateActive is the default conversation state. Input is enabled, player can type.
	StateActive
	// StateProcessing is entered after the player submits a question. Input is disabled,
	// a thinking indicator is shown. The panel waits for the LLM response.
	StateProcessing
	// StateResponse is entered when the LLM response arrives. The typewriter effect runs,
	// and input is re-enabled on completion (controlled by Story 003).
	StateResponse
)

// String returns a human-readable name for the panel state.
func (s PanelState) String() string {
	switch s {
	case StateHidden:
		return "Hidden"
	case StateActive:
		return "Active"
	case StateProcessing:
		return "Processing"
	case StateResponse:
		return "Response"
	default:
		return "Unknown"
	}
}

// IsInputEnabled reports whether player text input should be accepted in this state.
// Input is enabled only in Active and Response states.
func (s PanelState) IsInputEnabled() bool {
	return s == StateActive || s == StateResponse
}

// IsVisible reports whether the panel should be rendered in this state.
// The panel is visible in all states except Hidden.
func (s PanelState) IsVisible() bool {
	return s != StateHidden
}
