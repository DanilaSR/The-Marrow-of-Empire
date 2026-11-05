// Party roster event types (ADR-0011). The Party System is the sole
// producer; subscribers (TCE, UI) set flags and never write the roster.
//
// Registration follows the namespaced "system.action" convention of the
// EventType catalog in entity/event_bus.go — which predates the Party
// System and has no party.* types — so the party.* types are defined here
// like every other system's types (stats/events.go, combat/events.go,
// trust_relationship/events.go): a constant per event, documented with its
// payload, emitted by the owning system with Source "party".
//
// Note: party.member_departed (the departure event with cause) is defined
// by the character sheet's subscription contract
// (src/character_sheet/party_sidebar.go) and lands in this package with
// Story 002 (departure transitions, persisted cause). Story 001 emits no
// departure events.
package party

import "marrow-of-empire/entity"

const (
	// EventMemberAdded fires when a companion enters the active formation
	// (AddActive; AC-3 — the party screen's active slots update in the
	// same frame). Payload: MemberEventData.
	EventMemberAdded entity.EventType = "party.member_added"
	// EventMemberRemoved fires when a companion leaves the active formation
	// (Remove; the party may drop to a size-1 solo party). Payload:
	// MemberEventData.
	EventMemberRemoved entity.EventType = "party.member_removed"
	// EventPoolChanged fires when the pool changes: a member added
	// (AddToPool), a member removed (Remove from pool), or the pool's
	// recorded rest point changed (SetRestPoint, AC-5 relocation). Payload:
	// PoolChangedData.
	EventPoolChanged entity.EventType = "party.pool_changed"
)

// MemberEventData is the payload of EventMemberAdded and EventMemberRemoved.
// It is a value type so the event avoids heap allocation beyond the
// Event.Data interface boxing (ADR-0011).
type MemberEventData struct {
	// EntityID is the member the roster changed for.
	EntityID entity.GameEntityID
}

// PoolChangedData is the payload of EventPoolChanged. It is a value type
// (ADR-0011); the pool slice is a defensive copy — subscribers never see
// the store's internal slice.
type PoolChangedData struct {
	// Pool is the pool after the change, in pool order.
	Pool []entity.GameEntityID
	// RestPoint is the rest point the pool is recorded as present at
	// (AC-5).
	RestPoint entity.GameEntityID
}
