// Roster mutations for Story 001 (storage, capacity, locking, tagging,
// rest-point relocation). Operations are deliberately granular — AddActive,
// AddToPool, Remove, Swap, SetRestPoint — so Story 002 can compose the
// four companion states (Unrecruited/InParty/InPool/Departed) and their
// transitions on top without reworking this API. The store enforces the
// roster invariants (PC slot locked, Active and Pool disjoint and
// duplicate-free, capacity) but carries no state-machine semantics.
package party

import (
	"fmt"
	"log/slog"

	"marrow-of-empire/entity"
)

// SetPC fixes the player character at character creation (AC-1: the PC slot
// is locked for the lifetime of the party). One-shot: the PC is fixed at
// creation, so a second call is rejected with ErrPCAlreadySet (restores
// from save/load go through Decode, which replaces the state wholesale).
//
// The call establishes the always-valid solo party: Active becomes [pcID]
// with an empty pool, so a size-1 party (PC only) is a valid state
// everywhere (edge case — "the party size drops to 1 and remains a valid
// state").
func (s *PartyStore) SetPC(id entity.GameEntityID) error {
	if id == 0 {
		return ErrInvalidMember
	}
	if s.state.PCID != 0 {
		return ErrPCAlreadySet
	}
	s.state.PCID = id
	s.state.Active = []entity.GameEntityID{id}
	s.state.Pool = nil
	return nil
}

// AddActive adds a companion to the active formation in marching order —
// appended after the PC (AC-3: the party screen's active slots then show
// the companion in the roster). Rejections, in order:
//
//  1. No player character yet (ErrNoPC) — the party begins at creation.
//  2. Zero id (ErrInvalidMember) — the Entity Framework never issues zero.
//  3. The PC is never "added" — it is already active (ErrPCSlotLocked,
//     defense in depth).
//  4. A pool member cannot enter the formation directly (ErrMemberInPool):
//     Story 004's swap-in composes Remove (from pool) + AddActive.
//  5. A member already active is rejected (ErrMemberAlreadyActive).
//  6. A formation at capacity is rejected with ErrPartyFull (AC-2, data-
//     driven capacity); the caller routes the member to the pool instead
//     via AddToPool.
//  7. The pc_controlled tag must be written on the companion (AC-4): a
//     failed tag write aborts the add — the roster is not mutated and no
//     event fires (an entity the framework cannot tag is not a valid party
//     member).
//
// On success the member is appended, the tag is set, and party.member_added
// fires with MemberEventData.
func (s *PartyStore) AddActive(id entity.GameEntityID) error {
	if s.state.PCID == 0 {
		return ErrNoPC
	}
	if id == 0 {
		return ErrInvalidMember
	}
	if id == s.state.PCID {
		return ErrPCSlotLocked
	}
	if s.InPool(id) {
		return ErrMemberInPool
	}
	if s.IsActive(id) {
		return ErrMemberAlreadyActive
	}
	if len(s.state.Active) >= s.capacity {
		return ErrPartyFull
	}
	if s.tags != nil {
		if err := s.tags.SetPCControlled(id, s.state.PCID); err != nil {
			return fmt.Errorf("party: failed to tag member %d as pc_controlled: %w", id, err)
		}
	}
	s.state.Active = append(s.state.Active, id)
	s.emit(entity.Event{
		Type:   EventMemberAdded,
		Source: "party",
		Data:   MemberEventData{EntityID: id},
	})
	return nil
}

// AddToPool adds a member to the recruitment pool — the AC-2 pool-routing
// path ("the member must enter the pool instead") and the Story 002/004
// pool-entry primitive. Pool members are present at the current rest point
// (AC-5). Rejections: no PC yet, zero id, the PC never enters the pool
// (ErrPCSlotLocked), an active member must be removed first
// (ErrMemberActive — Active and Pool are disjoint), and a member already in
// the pool is rejected (ErrAlreadyInPool).
//
// The pc_controlled tag is cleared defensively — a pool member never
// carries the tag (AC-4) — best-effort: entering the pool never blocks on
// tag state (a destroyed entity's tag was already purged by the Entity
// Framework). On success party.pool_changed fires with PoolChangedData.
func (s *PartyStore) AddToPool(id entity.GameEntityID) error {
	if s.state.PCID == 0 {
		return ErrNoPC
	}
	if id == 0 {
		return ErrInvalidMember
	}
	if id == s.state.PCID {
		return ErrPCSlotLocked
	}
	if s.IsActive(id) {
		return ErrMemberActive
	}
	if s.InPool(id) {
		return ErrAlreadyInPool
	}
	s.clearTag(id)
	s.state.Pool = append(s.state.Pool, id)
	s.emitPoolChanged()
	return nil
}

// Remove removes a member from the roster: from the active formation
// (departure mid-expedition — the party may drop to the size-1 solo party,
// which remains valid, edge case) or from the pool (dismissal at a rest
// point). The PC is never removable (AC-1, ErrPCSlotLocked — checked before
// any state lookup, so the rejection holds in every party state).
//
// The pc_controlled tag is cleared best-effort (AC-4): a companion whose
// entity is destroyed still leaves the roster — a departure must never
// block on tag state, and the Entity Framework already purged the tag with
// the entity.
//
// Idempotent: removing a member that is not in the roster is a no-op
// success (double-remove safe) and no event fires. Emissions: an active
// removal fires party.member_removed; a pool removal fires
// party.pool_changed. GameEntityIDs are never reused by this store — the
// removed ID simply leaves the roster.
func (s *PartyStore) Remove(id entity.GameEntityID) error {
	if s.state.PCID == 0 {
		return ErrNoPC
	}
	if id == 0 {
		return ErrInvalidMember
	}
	if id == s.state.PCID {
		return ErrPCSlotLocked
	}
	for i, m := range s.state.Active {
		if m == id {
			s.state.Active = append(s.state.Active[:i], s.state.Active[i+1:]...)
			s.clearTag(id)
			s.emit(entity.Event{
				Type:   EventMemberRemoved,
				Source: "party",
				Data:   MemberEventData{EntityID: id},
			})
			return nil
		}
	}
	for i, m := range s.state.Pool {
		if m == id {
			s.state.Pool = append(s.state.Pool[:i], s.state.Pool[i+1:]...)
			s.clearTag(id)
			s.emitPoolChanged()
			return nil
		}
	}
	return nil // idempotent no-op: the member is not in the roster
}

// Swap reorders two members of the active formation (marching order, AC-3
// roster order). Any involvement of the player character is rejected with
// ErrPCSlotLocked in every state (AC-1, defense in depth); both members
// must be active (ErrMemberNotActive) — checked before the self-swap
// shortcut, so a self-swap of a member that is not in the formation is
// rejected, while a self-swap of a real active member is a documented
// no-op. No event fires: the roster contents are unchanged — Story 004's
// formation editing consumes this primitive.
func (s *PartyStore) Swap(a, b entity.GameEntityID) error {
	if s.state.PCID == 0 {
		return ErrNoPC
	}
	if a == 0 || b == 0 {
		return ErrInvalidMember
	}
	if a == s.state.PCID || b == s.state.PCID {
		return ErrPCSlotLocked
	}
	ia, ib := -1, -1
	for i, m := range s.state.Active {
		if m == a {
			ia = i
		}
		if m == b {
			ib = i
		}
	}
	if ia < 0 || ib < 0 {
		return ErrMemberNotActive
	}
	if a == b {
		return nil // reordering a member with itself is a no-op
	}
	s.state.Active[ia], s.state.Active[ib] = s.state.Active[ib], s.state.Active[ia]
	return nil
}

// SetRestPoint records the party's current rest point (AC-5, CR-1): pool
// companions are recorded as present there — they automatically relocate
// with the party, so they are always accessible for management when the
// party rests. Setting the same rest point again is a no-op (no event); a
// different value fires party.pool_changed carrying the new location. The
// zero value clears the recorded rest point (the party has not rested yet).
func (s *PartyStore) SetRestPoint(restPoint entity.GameEntityID) error {
	if s.state.PCID == 0 {
		return ErrNoPC
	}
	if restPoint == s.state.RestPoint {
		return nil
	}
	s.state.RestPoint = restPoint
	s.emitPoolChanged()
	return nil
}

// SetMaxActiveSize sets the active formation capacity at runtime. The Full
// Vision milestone switches the store from vs_party_size to
// full_vision_party_size through this method — the GDD knobs stay
// data-driven. Capacity is at least 1: a party always fits the player
// character (ErrInvalidCapacity). Shrinking below the current formation
// size takes effect on the next AddActive (existing members are never
// evicted). The player character counts against the capacity.
func (s *PartyStore) SetMaxActiveSize(n int) error {
	if n < 1 {
		return ErrInvalidCapacity
	}
	s.capacity = n
	return nil
}

// emitPoolChanged fires party.pool_changed with a snapshot of the pool and
// the recorded rest point (defensive copies — subscribers never see the
// store's internal slices).
func (s *PartyStore) emitPoolChanged() {
	s.emit(entity.Event{
		Type:   EventPoolChanged,
		Source: "party",
		Data: PoolChangedData{
			Pool:      append([]entity.GameEntityID(nil), s.state.Pool...),
			RestPoint: s.state.RestPoint,
		},
	})
}

// emit dispatches an event on the injected bus; a nil bus (headless
// operation) skips the emission.
func (s *PartyStore) emit(ev entity.Event) {
	if s.bus != nil {
		s.bus.Emit(ev)
	}
}

// clearTag clears the pc_controlled tag best-effort (AC-4: pool members and
// removed members never carry the tag). A failure is logged, never fatal:
// removal and pool-entry must not block on a destroyed entity whose tag the
// Entity Framework already purged.
func (s *PartyStore) clearTag(id entity.GameEntityID) {
	if s.tags == nil {
		return
	}
	if err := s.tags.ClearPCControlled(id); err != nil {
		slog.Warn("party: failed to clear pc_controlled tag", "entity", id, "err", err)
	}
}
