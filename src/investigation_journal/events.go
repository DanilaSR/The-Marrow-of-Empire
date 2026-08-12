package investigation_journal

import "marrow-of-empire/entity"

// Journal-specific event types. The Investigation Journal is an event producer
// (it emits state changes that other systems consume) and an event consumer
// (it subscribes to Thread Investigation events to update its display in
// later stories).
//
// Event types follow the namespaced string pattern from ADR-0011.
const (
	// EventJournalStateChanged fires when the journal opens or closes.
	// Payload: JournalStateEvent{Open: bool}.
	// Consumers: Camera System (freeze/unfreeze), input routing (suppress/restore).
	EventJournalStateChanged entity.EventType = "journal.state_changed"
)

// JournalStateEvent is the payload for EventJournalStateChanged.
type JournalStateEvent struct {
	// Open is true when the journal opened, false when it closed.
	Open bool
}
