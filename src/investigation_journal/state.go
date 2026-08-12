// Package investigation_journal implements the Investigation Journal UI —
// the player's desk-register interface for viewing investigation threads,
// chains, facts, and revelations (ADR-0012, Desk register).
//
// Story 001: journal shell — three-panel parchment overlay, open/close
// lifecycle, paper-slide animation, camera freeze coordination.
// Story 002: chain list sidebar, thread detail view, status indicators,
// ChainSelected state, sorting, and time formatting.
// Later stories populate the remaining panels with content.
//
// ADR: ADR-0012 (UI Framework — Desk register)
// Camera freeze wiring: Story 005.
package investigation_journal

// JournalState represents the current state of the investigation journal.
// States progress: Closed → Open → ChainSelected → EntitySelected (Story 003).
type JournalState int

const (
	// StateClosed is the default state. The journal overlay is not visible,
	// the camera is in exploration mode, and the world is at full opacity.
	StateClosed JournalState = iota
	// StateOpen is the base visible state. The journal overlay is visible,
	// the world is dimmed to 30%, and the camera is frozen. The sidebar
	// shows chain entries; the detail panel shows a "select a chain" prompt.
	StateOpen
	// StateChainSelected is entered when the player clicks a chain in the
	// sidebar. The detail panel shows the selected chain's threads.
	StateChainSelected
	// StateEntitySelected is entered when the player clicks an entity chip
	// (Story 003). The detail panel shows the entity's facts and connections.
	StateEntitySelected
)

// String returns a human-readable name for the journal state.
func (s JournalState) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateChainSelected:
		return "ChainSelected"
	case StateEntitySelected:
		return "EntitySelected"
	default:
		return "Unknown"
	}
}

// IsOpen reports whether the journal is currently visible (any non-Closed state).
func (s JournalState) IsOpen() bool {
	return s != StateClosed
}
