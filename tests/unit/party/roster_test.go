// Unit tests for Story 001 of the Party System epic: Party Roster & Active
// Slots.
//
// Evidence path: production/epics/party-system/
//
//	story-001-party-roster-active-slots.md
//
// Tests map to the five acceptance criteria (AC-1 PC slot locked, AC-2
// data-driven capacity, AC-3 roster visibility, AC-4 pc_controlled tagging,
// AC-5 pool relocation), the QA edge cases (solo party valid, double-remove
// idempotent, removed member's GameEntityID never reused), the event
// contract (party.member_added / party.member_removed / party.pool_changed,
// ADR-0011), and the serialization contract (EntityData via pod,
// ADR-0001/ADR-0002).
//
// Requirements TR-party-001; governing ADRs: ADR-0006 (component-store
// pattern), ADR-0011 (event bus). The store is exercised with a fake
// PCTagAccessor (the persistent_world_state fake-accessor pattern) so the
// roster logic is unit-tested without the engine; one test drives the
// production EFTagAdapter against a real EntityStoreImpl with a mock Kaiju
// host (the entity-framework test pattern) to prove the write-arbitration
// path.
package party_test

import (
	"errors"
	"reflect"
	"testing"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"

	"marrow-of-empire/entity"
	"marrow-of-empire/party"
)

// fakeTagAccessor is an in-memory PCTagAccessor for unit tests: a plain map
// from tagged entity to its controlling PC, mirroring the Entity Framework
// component store contract (presence in the map = the entity carries the
// pc_controlled tag).
type fakeTagAccessor struct {
	tags map[entity.GameEntityID]entity.GameEntityID
}

func newFakeTagAccessor() *fakeTagAccessor {
	return &fakeTagAccessor{tags: make(map[entity.GameEntityID]entity.GameEntityID)}
}

func (f *fakeTagAccessor) SetPCControlled(id, pcID entity.GameEntityID) error {
	f.tags[id] = pcID
	return nil
}

func (f *fakeTagAccessor) ClearPCControlled(id entity.GameEntityID) error {
	delete(f.tags, id)
	return nil
}

// tagged reports whether id currently carries the tag.
func (f *fakeTagAccessor) tagged(id entity.GameEntityID) bool {
	_, ok := f.tags[id]
	return ok
}

// failingTagAccessor fails tag writes (SetPCControlled) — used to prove an
// add-active aborts when the tag cannot be written (the roster is not
// mutated and no event fires).
type failingTagAccessor struct {
	fakeTagAccessor
}

func (f *failingTagAccessor) SetPCControlled(id, pcID entity.GameEntityID) error {
	return errors.New("tag write failed")
}

// clearFailingTagAccessor fails tag clears (ClearPCControlled) — used to
// prove a removal is never blocked by tag state (best-effort clear).
type clearFailingTagAccessor struct {
	fakeTagAccessor
}

func (f *clearFailingTagAccessor) ClearPCControlled(id entity.GameEntityID) error {
	return errors.New("tag clear failed")
}

// eventRecorder collects every party event it is subscribed to for later
// assertion.
type eventRecorder struct {
	received []entity.Event
}

func newEventRecorder() *eventRecorder { return &eventRecorder{} }

func (r *eventRecorder) subscribe(bus entity.EventBus, eventType entity.EventType) {
	bus.Subscribe(eventType, 5, r.handle)
}

func (r *eventRecorder) handle(ev entity.Event) { r.received = append(r.received, ev) }

// count returns how many events of the given type were received.
func (r *eventRecorder) count(eventType entity.EventType) int {
	n := 0
	for _, ev := range r.received {
		if ev.Type == eventType {
			n++
		}
	}
	return n
}

// find returns the first event of the given type.
func (r *eventRecorder) find(eventType entity.EventType) (entity.Event, bool) {
	for _, ev := range r.received {
		if ev.Type == eventType {
			return ev, true
		}
	}
	return entity.Event{}, false
}

// newTestStore returns a roster store wired to a fake tag accessor and a
// real event bus with a recorder on all three party event types. A vsSize
// of 0 falls back to the GDD default (4). Tests inject tiny sizes (2) to
// prove the capacity is data-driven, never hardcoded.
func newTestStore(vsSize int) (*party.PartyStore, *fakeTagAccessor, *entity.EventBusImpl, *eventRecorder) {
	acc := newFakeTagAccessor()
	bus := entity.NewEventBus()
	rec := newEventRecorder()
	rec.subscribe(bus, party.EventMemberAdded)
	rec.subscribe(bus, party.EventMemberRemoved)
	rec.subscribe(bus, party.EventPoolChanged)
	store := party.NewPartyStore(party.PartyStoreConfig{VSPartySize: vsSize}, acc, bus)
	return store, acc, bus, rec
}

// idsEqual compares two GameEntityID slices.
func idsEqual(a, b []entity.GameEntityID) bool { return reflect.DeepEqual(a, b) }

// ── AC-1: the PC slot is locked ───────────────────────────────────────────

// test_removePC_rejected covers AC-1: removing the player character is
// rejected in every party state — solo party and full party alike — and the
// PC stays in the active formation.
func Test_removePC_rejected(t *testing.T) {
	t.Helper()
	for _, withCompanions := range []bool{false, true} {
		store, _, _, rec := newTestStore(0)
		if err := store.SetPC(1); err != nil {
			t.Fatalf("SetPC: %v", err)
		}
		if withCompanions {
			if err := store.AddActive(2); err != nil {
				t.Fatalf("AddActive: %v", err)
			}
		}
		if err := store.Remove(1); !errors.Is(err, party.ErrPCSlotLocked) {
			t.Errorf("Remove(PC) withCompanions=%v: got %v, want ErrPCSlotLocked", withCompanions, err)
		}
		if !store.IsActive(1) {
			t.Errorf("Remove(PC) withCompanions=%v: PC removed from active formation", withCompanions)
		}
		if got := store.PC(); got != 1 {
			t.Errorf("PC() = %d, want 1", got)
		}
		if rec.count(party.EventMemberRemoved) != 0 {
			t.Errorf("member_removed fired %d times, want 0", rec.count(party.EventMemberRemoved))
		}
	}
}

// test_swapPC_rejected covers AC-1: swapping the player character with a
// companion — in either argument position — is rejected and the marching
// order is unchanged.
func Test_swapPC_rejected(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddActive(3); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.Swap(1, 2); !errors.Is(err, party.ErrPCSlotLocked) {
		t.Errorf("Swap(PC, c): got %v, want ErrPCSlotLocked", err)
	}
	if err := store.Swap(2, 1); !errors.Is(err, party.ErrPCSlotLocked) {
		t.Errorf("Swap(c, PC): got %v, want ErrPCSlotLocked", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1, 2, 3}) {
		t.Errorf("marching order after PC swap attempts = %v, want [1 2 3]", got)
	}
}

// test_addPCToPool_rejected covers AC-1: the player character can never be
// moved to the pool.
func Test_addPCToPool_rejected(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddToPool(1); !errors.Is(err, party.ErrPCSlotLocked) {
		t.Errorf("AddToPool(PC): got %v, want ErrPCSlotLocked", err)
	}
	if store.InPool(1) {
		t.Error("PC is in the pool")
	}
	if rec.count(party.EventPoolChanged) != 0 {
		t.Errorf("pool_changed fired %d times, want 0", rec.count(party.EventPoolChanged))
	}
}

// test_addPCAsActive_rejected covers AC-1 defense in depth: adding the PC as
// if it were a companion is rejected — the PC is already in the formation.
func Test_addPCAsActive_rejected(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(1); !errors.Is(err, party.ErrPCSlotLocked) {
		t.Errorf("AddActive(PC): got %v, want ErrPCSlotLocked", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1}) {
		t.Errorf("active formation = %v, want [1]", got)
	}
}

// ── AC-2: data-driven capacity ────────────────────────────────────────────

// test_addActiveAtCapacity_rejected covers AC-2 with a tiny injected
// capacity (vs_party_size 2): the third member is rejected with a
// full-roster result, the roster is unchanged, and no event fires.
func Test_addActiveAtCapacity_rejected(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(2)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if store.Capacity() != 2 {
		t.Fatalf("Capacity() = %d, want 2 (injected config)", store.Capacity())
	}
	if err := store.AddActive(3); !errors.Is(err, party.ErrPartyFull) {
		t.Errorf("AddActive at capacity: got %v, want ErrPartyFull", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1, 2}) {
		t.Errorf("active formation = %v, want [1 2] (unchanged)", got)
	}
	if rec.count(party.EventMemberAdded) != 1 {
		t.Errorf("member_added fired %d times, want 1", rec.count(party.EventMemberAdded))
	}
}

// test_fullRosterMemberRoutesToPool covers the AC-2 flow: when add-active is
// rejected at capacity, the member enters the pool instead — pool_changed
// fires, the member is never tagged pc_controlled (it was never active),
// and a later capacity change lets the member activate.
func Test_fullRosterMemberRoutesToPool(t *testing.T) {
	t.Helper()
	store, acc, _, rec := newTestStore(2)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddActive(3); !errors.Is(err, party.ErrPartyFull) {
		t.Fatalf("AddActive at capacity: got %v, want ErrPartyFull", err)
	}
	if err := store.AddToPool(3); err != nil {
		t.Fatalf("AddToPool after full rejection: %v", err)
	}
	if !store.InPool(3) {
		t.Error("member 3 not in the pool after routing")
	}
	if acc.tagged(3) {
		t.Error("pool-routed member carries pc_controlled, want untagged")
	}
	if rec.count(party.EventPoolChanged) != 1 {
		t.Errorf("pool_changed fired %d times, want 1", rec.count(party.EventPoolChanged))
	}
}

// test_defaultCapacityFour proves the GDD default: with zero config the
// capacity is vs_party_size 4 — the PC plus three companions fit and the
// fifth member is rejected.
func Test_defaultCapacityFour(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	for _, id := range []entity.GameEntityID{2, 3, 4} {
		if err := store.AddActive(id); err != nil {
			t.Fatalf("AddActive(%d): %v", id, err)
		}
	}
	if store.Capacity() != 4 {
		t.Fatalf("Capacity() = %d, want default 4", store.Capacity())
	}
	if err := store.AddActive(5); !errors.Is(err, party.ErrPartyFull) {
		t.Errorf("AddActive(5): got %v, want ErrPartyFull", err)
	}
}

// test_defaultFullVisionCapacitySix pins the second GDD knob: the Full
// Vision party size defaults to 6 (full_vision_party_size), and switching
// the store to it admits the PC plus five companions before rejecting the
// seventh.
func Test_defaultFullVisionCapacitySix(t *testing.T) {
	t.Helper()
	if party.DefaultFullVisionPartySize != 6 {
		t.Errorf("DefaultFullVisionPartySize = %d, want 6 (GDD full_vision_party_size)",
			party.DefaultFullVisionPartySize)
	}
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.SetMaxActiveSize(party.DefaultFullVisionPartySize); err != nil {
		t.Fatalf("SetMaxActiveSize(6): %v", err)
	}
	for _, id := range []entity.GameEntityID{2, 3, 4, 5, 6} {
		if err := store.AddActive(id); err != nil {
			t.Fatalf("AddActive(%d): %v", id, err)
		}
	}
	if err := store.AddActive(7); !errors.Is(err, party.ErrPartyFull) {
		t.Errorf("AddActive(7): got %v, want ErrPartyFull", err)
	}
}

// test_setMaxActiveSize_honored proves capacity is runtime-tunable: raising
// the limit (the Full Vision milestone path) admits more members.
func Test_setMaxActiveSize_honored(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(2)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddActive(3); !errors.Is(err, party.ErrPartyFull) {
		t.Fatalf("AddActive at capacity 2: got %v, want ErrPartyFull", err)
	}
	if err := store.SetMaxActiveSize(4); err != nil {
		t.Fatalf("SetMaxActiveSize(4): %v", err)
	}
	if err := store.AddActive(3); err != nil {
		t.Errorf("AddActive after capacity raise: %v", err)
	}
	if err := store.AddActive(4); err != nil {
		t.Errorf("AddActive(4) at capacity 4: %v", err)
	}
	if err := store.AddActive(5); !errors.Is(err, party.ErrPartyFull) {
		t.Errorf("AddActive(5) at capacity 4: got %v, want ErrPartyFull", err)
	}
}

// test_setMaxActiveSize_rejectsInvalidCapacity proves capacity never drops
// below 1 (a party always fits the player character).
func Test_setMaxActiveSize_rejectsInvalidCapacity(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetMaxActiveSize(0); !errors.Is(err, party.ErrInvalidCapacity) {
		t.Errorf("SetMaxActiveSize(0): got %v, want ErrInvalidCapacity", err)
	}
	if err := store.SetMaxActiveSize(-1); !errors.Is(err, party.ErrInvalidCapacity) {
		t.Errorf("SetMaxActiveSize(-1): got %v, want ErrInvalidCapacity", err)
	}
}

// test_setMaxActiveSize_shrinkBelowFormation_keepsMembers covers the shrink
// edge case: lowering the capacity below the current formation size never
// evicts existing members — the new limit takes effect on the next
// AddActive.
func Test_setMaxActiveSize_shrinkBelowFormation_keepsMembers(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(2)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.SetMaxActiveSize(1); err != nil {
		t.Fatalf("SetMaxActiveSize(1) below formation size: %v", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1, 2}) {
		t.Fatalf("shrink evicted a member: active = %v, want [1 2]", got)
	}
	if err := store.AddActive(3); !errors.Is(err, party.ErrPartyFull) {
		t.Errorf("AddActive after shrink: got %v, want ErrPartyFull", err)
	}
}

// ── AC-3: roster visibility ───────────────────────────────────────────────

// test_addActive_appearsInActiveSlots covers AC-3: a companion added to the
// active party appears in the active slots list, PC first, companions in
// marching order (append order).
func Test_addActive_appearsInActiveSlots(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(5); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddActive(3); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1, 5, 3}) {
		t.Errorf("active slots = %v, want [1 5 3] (PC first, marching order)", got)
	}
	if !store.IsActive(5) || !store.IsActive(3) {
		t.Error("added companions not reported active")
	}
	ev, ok := rec.find(party.EventMemberAdded)
	if !ok {
		t.Fatal("no member_added event")
	}
	data, ok := ev.Data.(party.MemberEventData)
	if !ok {
		t.Fatalf("member_added payload type %T, want party.MemberEventData", ev.Data)
	}
	if data.EntityID != 5 {
		t.Errorf("member_added payload entity = %d, want 5", data.EntityID)
	}
	if ev.Source != "party" {
		t.Errorf("member_added source = %q, want party", ev.Source)
	}
}

// test_swap_reordersMarchingOrder covers the AC-3 roster order primitive:
// Swap reorders two active companions; the PC stays first.
func Test_swap_reordersMarchingOrder(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddActive(3); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.Swap(2, 3); err != nil {
		t.Fatalf("Swap: %v", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1, 3, 2}) {
		t.Errorf("marching order after swap = %v, want [1 3 2]", got)
	}
	if rec.count(party.EventMemberAdded) != 2 || rec.count(party.EventMemberRemoved) != 0 {
		t.Errorf("swap emitted roster events: added=%d removed=%d, want 2 and 0",
			rec.count(party.EventMemberAdded), rec.count(party.EventMemberRemoved))
	}
}

// ── AC-4: pc_controlled tagging ───────────────────────────────────────────

// test_addActive_setsPCControlledTag covers AC-4: a companion in the active
// party carries the pc_controlled tag with the controlling PC.
func Test_addActive_setsPCControlledTag(t *testing.T) {
	t.Helper()
	store, acc, _, _ := newTestStore(0)
	if err := store.SetPC(7); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(8); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if !acc.tagged(8) {
		t.Error("active companion 8 not tagged pc_controlled")
	}
	if got := acc.tags[8]; got != 7 {
		t.Errorf("tag PCID = %d, want 7", got)
	}
	if acc.tagged(7) {
		t.Error("the PC itself must not carry the companion tag")
	}
}

// test_remove_clearsPCControlledTag covers AC-4: a companion removed from
// the active party loses the tag.
func Test_remove_clearsPCControlledTag(t *testing.T) {
	t.Helper()
	store, acc, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddActive(3); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.Remove(2); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if acc.tagged(2) {
		t.Error("removed companion 2 still tagged pc_controlled")
	}
	if !acc.tagged(3) {
		t.Error("remaining companion 3 lost its tag")
	}
}

// test_addToPool_clearsTag covers AC-4: a member entering the pool never
// carries the tag. A stray tag forced on a never-active member is cleared
// by AddToPool itself — the store is the tag's authoritative writer.
func Test_addToPool_clearsTag(t *testing.T) {
	t.Helper()
	store, acc, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	// Force a stray tag on a member that was never active, then pool it
	// directly: AddToPool must clear the tag (AC-4 — pool members are
	// never tagged).
	if err := acc.SetPCControlled(2, 1); err != nil {
		t.Fatalf("fake SetPCControlled: %v", err)
	}
	if !acc.tagged(2) {
		t.Fatal("precondition: stray tag must be present")
	}
	if err := store.AddToPool(2); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if acc.tagged(2) {
		t.Error("pool member 2 carries pc_controlled after AddToPool, want untagged")
	}
}

// test_poolMembersNeverTagged covers the AC-4 "companions in the pool do
// NOT carry the tag" state after a full active -> pool round trip.
func Test_poolMembersNeverTagged(t *testing.T) {
	t.Helper()
	store, acc, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if !acc.tagged(2) {
		t.Fatal("precondition: active member must be tagged")
	}
	if err := store.Remove(2); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := store.AddToPool(2); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if acc.tagged(2) {
		t.Error("pool member 2 carries pc_controlled after active -> pool round trip")
	}
}

// test_tagWriteFailureAbortsAddActive covers AC-4 defense: when the tag
// cannot be written, the add-active is rejected, the roster is not mutated,
// and no event fires.
func Test_tagWriteFailureAbortsAddActive(t *testing.T) {
	t.Helper()
	acc := &failingTagAccessor{}
	bus := entity.NewEventBus()
	rec := newEventRecorder()
	rec.subscribe(bus, party.EventMemberAdded)
	store := party.NewPartyStore(party.PartyStoreConfig{}, acc, bus)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err == nil {
		t.Fatal("AddActive with failing tag write: got nil, want error")
	}
	if store.IsActive(2) {
		t.Error("member 2 entered the formation despite the failed tag write")
	}
	if rec.count(party.EventMemberAdded) != 0 {
		t.Errorf("member_added fired %d times, want 0", rec.count(party.EventMemberAdded))
	}
}

// test_tagClearFailureDoesNotBlockRemoval covers the departure edge case: a
// companion whose tag cannot be cleared (destroyed entity) still leaves the
// roster — a removal never blocks on tag state.
func Test_tagClearFailureDoesNotBlockRemoval(t *testing.T) {
	t.Helper()
	acc := &clearFailingTagAccessor{}
	bus := entity.NewEventBus()
	store := party.NewPartyStore(party.PartyStoreConfig{}, acc, bus)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddToPool(2); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if err := store.Remove(2); err != nil {
		t.Errorf("Remove with failing tag clear: %v, want success (best-effort)", err)
	}
	if store.InPool(2) {
		t.Error("member 2 still in the pool after removal")
	}
}

// test_addToPool_tagClearFailure_bestEffort covers the best-effort tag
// contract on the pool path: AddToPool proceeds even when the tag cannot be
// cleared (destroyed entity) — pool entry never blocks on tag state.
func Test_addToPool_tagClearFailure_bestEffort(t *testing.T) {
	t.Helper()
	acc := &clearFailingTagAccessor{}
	bus := entity.NewEventBus()
	store := party.NewPartyStore(party.PartyStoreConfig{}, acc, bus)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddToPool(2); err != nil {
		t.Errorf("AddToPool with failing tag clear: %v, want success (best-effort)", err)
	}
	if !store.InPool(2) {
		t.Error("member 2 not in the pool despite best-effort tag clear")
	}
}

// ── AC-5: pool relocation ─────────────────────────────────────────────────

// test_setRestPoint_relocatesPool covers AC-5: when the party reaches a new
// rest point, the pool is recorded as present there and pool_changed fires
// with the new location and the pool contents.
func Test_setRestPoint_relocatesPool(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddToPool(2); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if err := store.AddToPool(3); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	// The earlier pool entries already fired pool_changed; clear the log so
	// the assertion below sees only the relocation event.
	rec.received = nil
	if err := store.SetRestPoint(100); err != nil {
		t.Fatalf("SetRestPoint: %v", err)
	}
	if got := store.RestPoint(); got != 100 {
		t.Errorf("RestPoint() = %d, want 100", got)
	}
	ev, ok := rec.find(party.EventPoolChanged)
	if !ok {
		t.Fatal("no pool_changed event")
	}
	data, ok := ev.Data.(party.PoolChangedData)
	if !ok {
		t.Fatalf("pool_changed payload type %T, want party.PoolChangedData", ev.Data)
	}
	if data.RestPoint != 100 {
		t.Errorf("pool_changed rest point = %d, want 100", data.RestPoint)
	}
	if !idsEqual(data.Pool, []entity.GameEntityID{2, 3}) {
		t.Errorf("pool_changed pool = %v, want [2 3]", data.Pool)
	}
}

// test_setRestPoint_sameValue_noop covers the AC-5 edge case: setting the
// same rest point again changes nothing and fires no event.
func Test_setRestPoint_sameValue_noop(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.SetRestPoint(100); err != nil {
		t.Fatalf("SetRestPoint: %v", err)
	}
	rec.received = nil
	if err := store.SetRestPoint(100); err != nil {
		t.Fatalf("SetRestPoint(same): %v", err)
	}
	if rec.count(party.EventPoolChanged) != 0 {
		t.Errorf("pool_changed fired %d times for a same-value rest point, want 0",
			rec.count(party.EventPoolChanged))
	}
}

// test_setRestPoint_clearsWithZero covers the zero-value contract: setting
// the zero rest point records "no rest point yet".
func Test_setRestPoint_clearsWithZero(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.SetRestPoint(100); err != nil {
		t.Fatalf("SetRestPoint: %v", err)
	}
	if err := store.SetRestPoint(0); err != nil {
		t.Fatalf("SetRestPoint(0): %v", err)
	}
	if got := store.RestPoint(); got != 0 {
		t.Errorf("RestPoint() after clear = %d, want 0", got)
	}
}

// ── Edge cases ────────────────────────────────────────────────────────────

// test_soloParty_valid covers the QA edge case: a size-1 party (PC only) is
// a valid state everywhere — after creation and after the last companion
// departs.
func Test_soloParty_valid(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1}) {
		t.Fatalf("fresh party = %v, want [1]", got)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	// The only non-PC active member departs mid-expedition.
	if err := store.Remove(2); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1}) {
		t.Errorf("party after companion departure = %v, want [1] (solo party valid)", got)
	}
	if err := store.AddActive(2); err != nil {
		t.Errorf("re-recruit after solo: %v", err)
	}
	if err := store.AddToPool(3); err != nil {
		t.Errorf("pool add while active: %v", err)
	}
}

// test_doubleRemove_idempotent covers the QA edge case: removing an
// already-removed member is a no-op success and fires no event.
func Test_doubleRemove_idempotent(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.Remove(2); err != nil {
		t.Fatalf("first Remove: %v", err)
	}
	if err := store.Remove(2); err != nil {
		t.Errorf("second Remove: %v, want nil (idempotent)", err)
	}
	if store.IsActive(2) {
		t.Error("double-removed member still active")
	}
	if rec.count(party.EventMemberRemoved) != 1 {
		t.Errorf("member_removed fired %d times, want 1", rec.count(party.EventMemberRemoved))
	}
	ev, ok := rec.find(party.EventMemberRemoved)
	if !ok {
		t.Fatal("no member_removed event")
	}
	data, ok := ev.Data.(party.MemberEventData)
	if !ok {
		t.Fatalf("member_removed payload type %T, want party.MemberEventData", ev.Data)
	}
	if data.EntityID != 2 {
		t.Errorf("member_removed payload entity = %d, want 2", data.EntityID)
	}
}

// test_removeUnknownMember_noop covers the QA edge case: removing a member
// that was never in the roster is a no-op success.
func Test_removeUnknownMember_noop(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.Remove(99); err != nil {
		t.Errorf("Remove(unknown): %v, want nil", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1}) {
		t.Errorf("roster = %v, want [1]", got)
	}
	if rec.count(party.EventMemberRemoved) != 0 {
		t.Errorf("member_removed fired %d times, want 0", rec.count(party.EventMemberRemoved))
	}
}

// test_storeCreatesNoEntityIDs covers the QA edge case "removed member's
// GameEntityID not reused": the party store never creates IDs — a fresh
// store holds no members and no counter — and never reserves or recycles
// IDs; only the Entity Framework's monotonic counter issues them (ADR-0001).
func Test_storeCreatesNoEntityIDs(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if store.PC() != 0 || store.RestPoint() != 0 {
		t.Errorf("fresh store: PC=%d RestPoint=%d, want 0 and 0", store.PC(), store.RestPoint())
	}
	if len(store.ActiveMembers()) != 0 || len(store.PoolMembers()) != 0 {
		t.Error("fresh store already holds members")
	}
}

// test_readdRemovedMemberAllowed covers the ID contract: after a removal the
// store drops the reference but neither recycles nor blocks the ID — the
// caller (and the Entity Framework) decide validity.
func Test_readdRemovedMemberAllowed(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.Remove(2); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := store.AddToPool(2); err != nil {
		t.Errorf("re-adding removed ID to pool: %v, want success", err)
	}
}

// test_addActive_duplicateRejected covers the duplicate guard: a member
// already active cannot be added again, and no event fires.
func Test_addActive_duplicateRejected(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddActive(2); !errors.Is(err, party.ErrMemberAlreadyActive) {
		t.Errorf("duplicate AddActive: got %v, want ErrMemberAlreadyActive", err)
	}
	if rec.count(party.EventMemberAdded) != 1 {
		t.Errorf("member_added fired %d times, want 1", rec.count(party.EventMemberAdded))
	}
}

// test_addToPool_duplicateRejected covers the duplicate guard: a member
// already in the pool cannot be added again.
func Test_addToPool_duplicateRejected(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddToPool(2); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if err := store.AddToPool(2); !errors.Is(err, party.ErrAlreadyInPool) {
		t.Errorf("duplicate AddToPool: got %v, want ErrAlreadyInPool", err)
	}
	if rec.count(party.EventPoolChanged) != 1 {
		t.Errorf("pool_changed fired %d times, want 1", rec.count(party.EventPoolChanged))
	}
}

// test_addActive_poolMemberRejected covers the disjointness invariant: a
// pool member must be removed before it can enter the active formation
// (Story 004 swap-in composes the primitives).
func Test_addActive_poolMemberRejected(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddToPool(2); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if err := store.AddActive(2); !errors.Is(err, party.ErrMemberInPool) {
		t.Errorf("AddActive on pool member: got %v, want ErrMemberInPool", err)
	}
}

// test_addToPool_activeMemberRejected covers the disjointness invariant: an
// active member must be removed before it can enter the pool (Story 002
// transition composition).
func Test_addToPool_activeMemberRejected(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddToPool(2); !errors.Is(err, party.ErrMemberActive) {
		t.Errorf("AddToPool on active member: got %v, want ErrMemberActive", err)
	}
}

// test_setPC_twice_rejected covers the AC-1 "fixed at creation" contract:
// the player character is set exactly once.
func Test_setPC_twice_rejected(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.SetPC(2); !errors.Is(err, party.ErrPCAlreadySet) {
		t.Errorf("second SetPC: got %v, want ErrPCAlreadySet", err)
	}
	if store.PC() != 1 {
		t.Errorf("PC() = %d, want 1 (unchanged)", store.PC())
	}
}

// test_mutationsBeforeSetPC_rejected covers the defensive guard: no roster
// mutation runs against a leaderless roster.
func Test_mutationsBeforeSetPC_rejected(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.AddActive(2); !errors.Is(err, party.ErrNoPC) {
		t.Errorf("AddActive before SetPC: got %v, want ErrNoPC", err)
	}
	if err := store.AddToPool(2); !errors.Is(err, party.ErrNoPC) {
		t.Errorf("AddToPool before SetPC: got %v, want ErrNoPC", err)
	}
	if err := store.Remove(2); !errors.Is(err, party.ErrNoPC) {
		t.Errorf("Remove before SetPC: got %v, want ErrNoPC", err)
	}
	if err := store.Swap(2, 3); !errors.Is(err, party.ErrNoPC) {
		t.Errorf("Swap before SetPC: got %v, want ErrNoPC", err)
	}
	if err := store.SetRestPoint(100); !errors.Is(err, party.ErrNoPC) {
		t.Errorf("SetRestPoint before SetPC: got %v, want ErrNoPC", err)
	}
}

// test_zeroIDs_rejected covers the invalid-member guard: a zero
// GameEntityID is never a valid roster member (the Entity Framework never
// issues zero).
func Test_zeroIDs_rejected(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(0); !errors.Is(err, party.ErrInvalidMember) {
		t.Errorf("AddActive(0): got %v, want ErrInvalidMember", err)
	}
	if err := store.AddToPool(0); !errors.Is(err, party.ErrInvalidMember) {
		t.Errorf("AddToPool(0): got %v, want ErrInvalidMember", err)
	}
	if err := store.Remove(0); !errors.Is(err, party.ErrInvalidMember) {
		t.Errorf("Remove(0): got %v, want ErrInvalidMember", err)
	}
	if err := store.Swap(0, 2); !errors.Is(err, party.ErrInvalidMember) {
		t.Errorf("Swap(0, 2): got %v, want ErrInvalidMember", err)
	}
	if err := store.SetPC(0); !errors.Is(err, party.ErrInvalidMember) {
		t.Errorf("SetPC(0): got %v, want ErrInvalidMember", err)
	}
}

// test_swap_nonActiveMember_rejected covers Swap's membership guard: both
// members must be in the active formation.
func Test_swap_nonActiveMember_rejected(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.Swap(2, 3); !errors.Is(err, party.ErrMemberNotActive) {
		t.Errorf("Swap(active, unknown): got %v, want ErrMemberNotActive", err)
	}
}

// test_swapSelf_regression covers the Swap membership-guard regression: a
// self-swap of a member that is not in the formation is rejected, while a
// self-swap of a real active member is a no-op that leaves the order
// unchanged.
func Test_swapSelf_regression(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.Swap(99, 99); !errors.Is(err, party.ErrMemberNotActive) {
		t.Errorf("Swap(unknown, unknown): got %v, want ErrMemberNotActive", err)
	}
	if err := store.Swap(2, 2); err != nil {
		t.Errorf("Swap(active, active): %v, want nil (no-op)", err)
	}
	if got := store.ActiveMembers(); !idsEqual(got, []entity.GameEntityID{1, 2}) {
		t.Errorf("marching order after self-swap = %v, want [1 2]", got)
	}
}

// test_events_firedOnRosterChanges covers the ADR-0011 event contract end to
// end: member_added on activation, member_removed on removal, pool_changed
// on pool entry — all synchronous, with value payloads and Source "party".
func Test_events_firedOnRosterChanges(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.Remove(2); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := store.AddToPool(2); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if rec.count(party.EventMemberAdded) != 1 {
		t.Errorf("member_added fired %d times, want 1", rec.count(party.EventMemberAdded))
	}
	if rec.count(party.EventMemberRemoved) != 1 {
		t.Errorf("member_removed fired %d times, want 1", rec.count(party.EventMemberRemoved))
	}
	if rec.count(party.EventPoolChanged) != 1 {
		t.Errorf("pool_changed fired %d times, want 1", rec.count(party.EventPoolChanged))
	}
	for _, ev := range rec.received {
		if ev.Source != "party" {
			t.Errorf("event %s source = %q, want party", ev.Type, ev.Source)
		}
	}
}

// test_removeFromPool_firesPoolChanged covers the pool-removal event path: a
// dismissal at a rest point fires pool_changed (not member_removed — the
// member never left the active formation) with the member absent from the
// new pool snapshot.
func Test_removeFromPool_firesPoolChanged(t *testing.T) {
	t.Helper()
	store, _, _, rec := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddToPool(2); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if err := store.AddToPool(3); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	// Earlier pool entries already fired pool_changed; clear the log so the
	// assertions below see only the removal event.
	rec.received = nil
	if err := store.Remove(2); err != nil {
		t.Fatalf("Remove from pool: %v", err)
	}
	if rec.count(party.EventMemberRemoved) != 0 {
		t.Errorf("member_removed fired %d times for a pool removal, want 0",
			rec.count(party.EventMemberRemoved))
	}
	if rec.count(party.EventPoolChanged) != 1 {
		t.Errorf("pool_changed fired %d times, want 1", rec.count(party.EventPoolChanged))
	}
	ev, ok := rec.find(party.EventPoolChanged)
	if !ok {
		t.Fatal("no pool_changed event")
	}
	data, ok := ev.Data.(party.PoolChangedData)
	if !ok {
		t.Fatalf("pool_changed payload type %T, want party.PoolChangedData", ev.Data)
	}
	if !idsEqual(data.Pool, []entity.GameEntityID{3}) {
		t.Errorf("pool_changed pool = %v, want [3] (removed member absent)", data.Pool)
	}
}

// ── Serialization (ADR-0001/ADR-0002) ─────────────────────────────────────

// test_serialization_roundTrip proves the EntityData contract: a populated
// store Encode()s to a pod blob that Decode()s into an identical roster on
// a fresh store — the Save/Load path for the party state.
func Test_serialization_roundTrip(t *testing.T) {
	t.Helper()
	store, _, _, _ := newTestStore(0)
	if err := store.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := store.AddActive(2); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddActive(3); err != nil {
		t.Fatalf("AddActive: %v", err)
	}
	if err := store.AddToPool(4); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if err := store.SetRestPoint(100); err != nil {
		t.Fatalf("SetRestPoint: %v", err)
	}

	blob, err := store.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(blob) == 0 {
		t.Fatal("Encode returned an empty blob for a populated roster")
	}

	loaded, _, _, _ := newTestStore(0)
	if err := loaded.Decode(blob); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if loaded.PC() != 1 {
		t.Errorf("decoded PC = %d, want 1", loaded.PC())
	}
	if !idsEqual(loaded.ActiveMembers(), []entity.GameEntityID{1, 2, 3}) {
		t.Errorf("decoded active = %v, want [1 2 3]", loaded.ActiveMembers())
	}
	if !idsEqual(loaded.PoolMembers(), []entity.GameEntityID{4}) {
		t.Errorf("decoded pool = %v, want [4]", loaded.PoolMembers())
	}
	if loaded.RestPoint() != 100 {
		t.Errorf("decoded rest point = %d, want 100", loaded.RestPoint())
	}

	// The decoded store is fully operational (mutations and events work).
	if err := loaded.AddActive(5); err != nil {
		t.Errorf("AddActive after decode: %v", err)
	}
}

// test_serialization_roundTrip_edgeStates covers the remaining serialization
// shapes: an empty store, a PC-only solo party, a pool-only roster, and the
// load path replacing a populated store wholesale (Decode over a different
// roster).
func Test_serialization_roundTrip_edgeStates(t *testing.T) {
	t.Helper()

	// Empty store: pod may normalize nil slices to empty ones — both read
	// back as an empty roster, and the decoded store reports no members.
	empty, _, _, _ := newTestStore(0)
	emptyBlob, err := empty.Encode()
	if err != nil {
		t.Fatalf("empty Encode: %v", err)
	}
	decodedEmpty, _, _, _ := newTestStore(0)
	if err := decodedEmpty.Decode(emptyBlob); err != nil {
		t.Fatalf("empty Decode: %v", err)
	}
	if decodedEmpty.PC() != 0 || decodedEmpty.RestPoint() != 0 {
		t.Errorf("decoded empty store: PC=%d RestPoint=%d, want 0 and 0",
			decodedEmpty.PC(), decodedEmpty.RestPoint())
	}
	if len(decodedEmpty.ActiveMembers()) != 0 || len(decodedEmpty.PoolMembers()) != 0 {
		t.Error("decoded empty store holds members")
	}

	// PC-only solo party.
	solo, _, _, _ := newTestStore(0)
	if err := solo.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	soloBlob, err := solo.Encode()
	if err != nil {
		t.Fatalf("solo Encode: %v", err)
	}
	decodedSolo, _, _, _ := newTestStore(0)
	if err := decodedSolo.Decode(soloBlob); err != nil {
		t.Fatalf("solo Decode: %v", err)
	}
	if !idsEqual(decodedSolo.ActiveMembers(), []entity.GameEntityID{1}) {
		t.Errorf("decoded solo active = %v, want [1]", decodedSolo.ActiveMembers())
	}
	if len(decodedSolo.PoolMembers()) != 0 {
		t.Errorf("decoded solo pool = %v, want empty", decodedSolo.PoolMembers())
	}

	// Pool-only roster (no active companions).
	poolOnly, _, _, _ := newTestStore(0)
	if err := poolOnly.SetPC(1); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := poolOnly.AddToPool(2); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	poolBlob, err := poolOnly.Encode()
	if err != nil {
		t.Fatalf("pool-only Encode: %v", err)
	}
	decodedPool, _, _, _ := newTestStore(0)
	if err := decodedPool.Decode(poolBlob); err != nil {
		t.Fatalf("pool-only Decode: %v", err)
	}
	if !idsEqual(decodedPool.PoolMembers(), []entity.GameEntityID{2}) {
		t.Errorf("decoded pool = %v, want [2]", decodedPool.PoolMembers())
	}
	if !idsEqual(decodedPool.ActiveMembers(), []entity.GameEntityID{1}) {
		t.Errorf("decoded active = %v, want [1]", decodedPool.ActiveMembers())
	}

	// Decode over a populated store replaces the roster wholesale.
	source, _, _, _ := newTestStore(0)
	if err := source.SetPC(1); err != nil {
		t.Fatalf("source SetPC: %v", err)
	}
	if err := source.AddActive(2); err != nil {
		t.Fatalf("source AddActive: %v", err)
	}
	if err := source.AddActive(3); err != nil {
		t.Fatalf("source AddActive: %v", err)
	}
	if err := source.AddToPool(4); err != nil {
		t.Fatalf("source AddToPool: %v", err)
	}
	if err := source.SetRestPoint(100); err != nil {
		t.Fatalf("source SetRestPoint: %v", err)
	}
	sourceBlob, err := source.Encode()
	if err != nil {
		t.Fatalf("source Encode: %v", err)
	}
	target, _, _, _ := newTestStore(0)
	if err := target.SetPC(11); err != nil {
		t.Fatalf("target SetPC: %v", err)
	}
	if err := target.AddActive(12); err != nil {
		t.Fatalf("target AddActive: %v", err)
	}
	if err := target.AddToPool(13); err != nil {
		t.Fatalf("target AddToPool: %v", err)
	}
	if err := target.SetRestPoint(50); err != nil {
		t.Fatalf("target SetRestPoint: %v", err)
	}
	if err := target.Decode(sourceBlob); err != nil {
		t.Fatalf("target Decode: %v", err)
	}
	if target.PC() != 1 || target.RestPoint() != 100 {
		t.Errorf("decoded target: PC=%d RestPoint=%d, want 1 and 100",
			target.PC(), target.RestPoint())
	}
	if !idsEqual(target.ActiveMembers(), []entity.GameEntityID{1, 2, 3}) {
		t.Errorf("decoded target active = %v, want [1 2 3]", target.ActiveMembers())
	}
	if !idsEqual(target.PoolMembers(), []entity.GameEntityID{4}) {
		t.Errorf("decoded target pool = %v, want [4]", target.PoolMembers())
	}
}

// ── Production tag adapter (write arbitration, ADR-0001) ──────────────────

// mockKaijuHost is a test double for entity.KaijuHost (the entity-framework
// test pattern): it registers engine entities by EntityId so a real
// EntityStoreImpl can run without a window or Vulkan context.
type mockKaijuHost struct {
	entities  map[engine.EntityId]*engine.Entity
	workGroup concurrent.WorkGroup
}

func (m *mockKaijuHost) EntityById(id engine.EntityId) *engine.Entity {
	return m.entities[id]
}

func (m *mockKaijuHost) SetEntityId(e *engine.Entity, id engine.EntityId) bool {
	if id == "" {
		return false
	}
	if _, exists := m.entities[id]; exists {
		return false
	}
	m.entities[id] = e
	return true
}

func (m *mockKaijuHost) DestroyEntity(e *engine.Entity) {
	for id, ent := range m.entities {
		if ent == e {
			delete(m.entities, id)
			return
		}
	}
}

func (m *mockKaijuHost) WorkGroup() *concurrent.WorkGroup { return &m.workGroup }

func newMockHost() *mockKaijuHost {
	m := &mockKaijuHost{entities: make(map[engine.EntityId]*engine.Entity)}
	m.workGroup.Init()
	return m
}

// test_efTagAdapter_writesThroughArbitration proves the production tag
// path: EFTagAdapter writes PCControlled through the Entity Framework's
// write arbitration (SetComponentArbitrated) on add-active and clears it on
// removal, with the tag observable in the real component store (the
// debug-overlay read path, AC-4).
func Test_efTagAdapter_writesThroughArbitration(t *testing.T) {
	t.Helper()
	host := newMockHost()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(host, bus, entity.EntityStoreConfig{})
	arbiter := entity.NewWriteArbiter(store)
	adapter := party.NewEFTagAdapter(store, arbiter)
	roster := party.NewPartyStore(party.PartyStoreConfig{}, adapter, bus)

	pcID, err := store.CreateEntity(entity.EntityTypeCharacter, "Player", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity(PC): %v", err)
	}
	compID, err := store.CreateEntity(entity.EntityTypeCharacter, "Companion", matrix.NewVec3(11, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity(companion): %v", err)
	}
	if err := roster.SetPC(pcID); err != nil {
		t.Fatalf("SetPC: %v", err)
	}
	if err := roster.AddActive(compID); err != nil {
		t.Fatalf("AddActive: %v", err)
	}

	tag, ok := entity.GetComponent[party.PCControlled](store, compID)
	if !ok {
		t.Fatal("companion has no PCControlled component after AddActive")
	}
	if tag.PCID != pcID {
		t.Errorf("tag PCID = %d, want %d", tag.PCID, pcID)
	}
	if _, ok := entity.GetComponent[party.PCControlled](store, pcID); ok {
		t.Error("the PC entity itself carries the companion tag")
	}

	if err := roster.Remove(compID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, ok := entity.GetComponent[party.PCControlled](store, compID); ok {
		t.Error("companion still carries PCControlled after Remove")
	}

	// Pool entry also clears the tag through the adapter.
	if err := roster.AddActive(compID); err != nil {
		t.Fatalf("re-AddActive: %v", err)
	}
	if err := roster.Remove(compID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := roster.AddToPool(compID); err != nil {
		t.Fatalf("AddToPool: %v", err)
	}
	if _, ok := entity.GetComponent[party.PCControlled](store, compID); ok {
		t.Error("pool member still carries PCControlled after AddToPool")
	}
}
