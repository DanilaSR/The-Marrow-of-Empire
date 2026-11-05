package party

import "errors"

// Sentinel errors returned by the PartyStore API. Callers test with
// errors.Is; every error wraps its underlying cause where one exists.
var (
	// ErrPCSlotLocked rejects any operation that would remove, swap out, or
	// move the player character out of the active formation (AC-1). The PC
	// slot is locked in every party state: the check runs before any state
	// lookup, so even a corrupt roster cannot touch the PC (defense in
	// depth — no UI should ever reach these operations).
	ErrPCSlotLocked = errors.New("party: the player character slot is locked")

	// ErrPartyFull rejects an add-active request when the formation is at
	// capacity (AC-2). The caller routes the member to the pool instead via
	// AddToPool.
	ErrPartyFull = errors.New("party: active formation is at capacity")

	// ErrMemberAlreadyActive rejects adding a member that is already in the
	// active formation (duplicate guard — Active holds each member once).
	ErrMemberAlreadyActive = errors.New("party: member is already in the active formation")

	// ErrMemberInPool rejects adding a pool member to the active formation:
	// Active and Pool are disjoint, so the caller removes the member from
	// the pool first (Story 004 swap-in composition).
	ErrMemberInPool = errors.New("party: member is in the pool, remove it before activating")

	// ErrMemberActive rejects adding an active member to the pool: Active
	// and Pool are disjoint, so the caller removes the member from the
	// formation first (Story 002 transition composition).
	ErrMemberActive = errors.New("party: member is in the active formation, remove it before pooling")

	// ErrAlreadyInPool rejects adding a member that is already in the pool
	// (duplicate guard — Pool holds each member once).
	ErrAlreadyInPool = errors.New("party: member is already in the pool")

	// ErrMemberNotActive rejects an active-formation operation (Swap) on a
	// member that is not in the active formation.
	ErrMemberNotActive = errors.New("party: member is not in the active formation")

	// ErrNoPC rejects roster mutations before the player character is set:
	// the party begins at character creation (SetPC), and the PC anchors
	// the active formation (Active[0]). Defensive — no operation may run
	// against a leaderless roster.
	ErrNoPC = errors.New("party: no player character set; call SetPC first")

	// ErrPCAlreadySet rejects a second SetPC: the player character is fixed
	// at character creation and never changes (AC-1).
	ErrPCAlreadySet = errors.New("party: the player character is fixed at creation and already set")

	// ErrInvalidMember rejects a zero GameEntityID: entity IDs come from
	// the Entity Framework's monotonic counter and are never zero.
	ErrInvalidMember = errors.New("party: invalid member id (zero)")

	// ErrInvalidCapacity rejects a capacity below 1: the active formation
	// must always fit the player character.
	ErrInvalidCapacity = errors.New("party: capacity must be at least 1")
)
