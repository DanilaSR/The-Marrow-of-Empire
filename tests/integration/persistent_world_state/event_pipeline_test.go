// Integration tests for Story 002 of the Persistent World State epic: the
// Event Submission Pipeline.
//
// Evidence path: production/epics/persistent-world-state/
//
//	story-002-event-submission-pipeline.md
//
// Tests map to the five acceptance criteria:
//
//	AC-1  test_event_applies_flags_and_appends_log -> TestSubmitWorldEventAppliesFlagsAndAppendsLog
//	AC-2  test_non_world_flag_skipped              -> TestSubmitWorldEventFiltersNonWorldFlags
//	AC-3  test_same_frame_ordering                 -> TestSubmitWorldEventSameFrameOrderingLastWriterWins
//	AC-4  test_invalid_target_rejected             -> TestSubmitWorldEventInvalidTargetRejected
//	AC-5  test_broadcast_on_success                -> TestSubmitWorldEventBroadcastsOnSuccess
//
// QA edge cases: zero flag mutations (log entry still written), 10+
// mutations applied in deterministic order, all flags non-world (event still
// processed), schema-mismatched world.* flag skipped without failing the
// event, disjoint same-frame writes both applied, zero and invalid targets,
// invalid actor rejected, zero actor accepted (system-generated), empty
// event type / region rejected, no broadcast on rejection, and multiple
// subscribers all receiving the broadcast synchronously.
//
// AC-3 note: the QA case writes "A(priority=3), B(priority=5)", but the
// WorldEvent struct carries no per-event priority — PWS is the
// authoritative writer of the world.* namespace (ADR-0001), so same-flag
// conflicts resolve by submission order, last writer wins. Submission order
// IS the priority resolution inside the pipeline.
//
// The fake accessor, stub bus, and fixed game clock keep the pipeline
// engine-agnostic (persistent_world_state imports no Kaiju APIs).
package persistent_world_state_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"marrow-of-empire/entity"
	"marrow-of-empire/persistent_world_state"
)

// fakeAccessor is an in-memory PersistentAccessor mirroring the Entity
// Framework component store contract: an entity exists iff it has a
// Persistent component (components are removed on destruction).
type fakeAccessor struct {
	components map[entity.GameEntityID]entity.Persistent
}

func newFakeAccessor() *fakeAccessor {
	return &fakeAccessor{components: make(map[entity.GameEntityID]entity.Persistent)}
}

func (f *fakeAccessor) GetPersistent(id entity.GameEntityID) (entity.Persistent, bool) {
	p, ok := f.components[id]
	return p, ok
}

func (f *fakeAccessor) SetPersistent(id entity.GameEntityID, p entity.Persistent) error {
	f.components[id] = p
	return nil
}

// stubBus is a recording EventBus: it captures every emitted event in order
// so tests can assert broadcast behavior (AC-5) without a real subscriber
// registry.
type stubBus struct {
	emitted []entity.Event
}

func (b *stubBus) Subscribe(entity.EventType, int, entity.EventHandler) {}
func (b *stubBus) Emit(ev entity.Event)                                { b.emitted = append(b.emitted, ev) }
func (b *stubBus) Unsubscribe(entity.EventType, entity.EventHandler)   {}

// newTestPipeline wires a pipeline to a fresh fake accessor, a real flag
// store (story 001), a stub bus, and a fixed game clock (100.0 minutes) so
// entries carry a deterministic non-zero GameTimestamp (QA AC-1).
func newTestPipeline() (*persistent_world_state.EventPipeline, *persistent_world_state.WorldFlagStore, *fakeAccessor, *stubBus) {
	acc := newFakeAccessor()
	flags := persistent_world_state.NewWorldFlagStore(acc, persistent_world_state.WorldFlagStoreConfig{})
	bus := &stubBus{}
	pipe := persistent_world_state.NewEventPipeline(acc, flags, bus, persistent_world_state.EventPipelineConfig{
		Clock: func() float64 { return 100.0 },
	})
	return pipe, flags, acc, bus
}

// addEntity registers an entity with a Persistent component in the fake
// accessor.
func addEntity(acc *fakeAccessor, id entity.GameEntityID) {
	acc.components[id] = entity.Persistent{StateFlags: make(map[string]entity.StateFlagValue)}
}

// TestSubmitWorldEventAppliesFlagsAndAppendsLog covers AC-1: a
// location_cleared event sets world.ruin_cleared = true on the target AND
// appends a memory log entry with the event type, the current game
// timestamp, and the actor / target / region / details fields.
func TestSubmitWorldEventAppliesFlagsAndAppendsLog(t *testing.T) {
	pipe, flags, acc, bus := newTestPipeline()
	addEntity(acc, 1)

	ev := persistent_world_state.WorldEvent{
		EventType:      "location_cleared",
		ActorEntityID:  1,
		TargetEntityID: 1,
		RegionID:       "thornwell_ruins",
		Details:        `{"source":"player"}`,
		FlagMutations:  map[string]any{"world.ruin_cleared": true},
	}
	if err := pipe.SubmitWorldEvent(ev); err != nil {
		t.Fatalf("SubmitWorldEvent returned unexpected error: %v", err)
	}

	got, found := flags.GetWorldFlag(1, "world.ruin_cleared")
	if !found || got != true {
		t.Errorf("GetWorldFlag(world.ruin_cleared) = (%v, %v), want (true, true)", got, found)
	}

	log := pipe.MemoryLogFor(1)
	if log.Len() != 1 {
		t.Fatalf("MemoryLogFor(1) has %d entries, want 1", log.Len())
	}
	entry := log.Entries[0]
	if entry.EventType != "location_cleared" {
		t.Errorf("entry.EventType = %q, want %q", entry.EventType, "location_cleared")
	}
	if entry.GameTimestamp != 100.0 {
		t.Errorf("entry.GameTimestamp = %v, want 100.0 (the injected clock)", entry.GameTimestamp)
	}
	if entry.ActorID != 1 || entry.TargetID != 1 {
		t.Errorf("entry actor/target = %d/%d, want 1/1", entry.ActorID, entry.TargetID)
	}
	if entry.RegionID != "thornwell_ruins" {
		t.Errorf("entry.RegionID = %q, want thornwell_ruins", entry.RegionID)
	}
	if entry.Details != `{"source":"player"}` {
		t.Errorf("entry.Details = %q, want the submitted JSON blob", entry.Details)
	}
	if entry.EventID == 0 {
		t.Error("entry.EventID = 0, want a non-zero globally unique ID")
	}
	if entry.Protected {
		t.Error("entry.Protected = true, want false (WorldEvent has no Protected field; marking protected events is out of scope)")
	}
	if len(bus.emitted) != 1 {
		t.Errorf("broadcast count = %d, want 1", len(bus.emitted))
	}
}

// TestSubmitWorldEventZeroFlagMutationsStillLogs covers the QA AC-1 edge
// case: an event with no flag mutations is valid — no flags are written but
// the log entry is still appended and the event is still broadcast.
func TestSubmitWorldEventZeroFlagMutationsStillLogs(t *testing.T) {
	pipe, flags, acc, bus := newTestPipeline()
	addEntity(acc, 1)

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "npc_state_changed",
		TargetEntityID: 1,
		RegionID:       "thornwell",
	})
	if err != nil {
		t.Fatalf("SubmitWorldEvent returned unexpected error: %v", err)
	}
	if got := flags.GetWorldFlags(1); len(got) != 0 {
		t.Errorf("GetWorldFlags = %v, want empty", got)
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 1 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 1", log.Len())
	}
	if len(bus.emitted) != 1 {
		t.Errorf("broadcast count = %d, want 1", len(bus.emitted))
	}
}

// TestSubmitWorldEventManyFlagMutationsAllApplied covers the QA AC-1 edge
// case: an event with 10+ mutations applies every world.* flag, in
// deterministic order regardless of Go map iteration order (the pipeline
// sorts flag names). ints normalize to int64 on read-back (TR-pws-001).
func TestSubmitWorldEventManyFlagMutationsAllApplied(t *testing.T) {
	pipe, flags, acc, _ := newTestPipeline()
	addEntity(acc, 1)

	mutations := make(map[string]any)
	for i := 0; i < 12; i++ {
		mutations[fmt.Sprintf("world.flag_%02d", i)] = i
	}
	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "region_investigated",
		TargetEntityID: 1,
		RegionID:       "thornwell",
		FlagMutations:  mutations,
	})
	if err != nil {
		t.Fatalf("SubmitWorldEvent returned unexpected error: %v", err)
	}

	got := flags.GetWorldFlags(1)
	if len(got) != 12 {
		t.Fatalf("GetWorldFlags has %d flags, want 12", len(got))
	}
	for i := 0; i < 12; i++ {
		flag := fmt.Sprintf("world.flag_%02d", i)
		if v := got[flag]; v != int64(i) {
			t.Errorf("%s = %v (%T), want int64(%d)", flag, v, v, i)
		}
	}
}

// TestSubmitWorldEventFiltersNonWorldFlags covers AC-2: an event mixing
// world.a (true) and trust.x (5) applies world.a, skips trust.x with a
// warning, writes the log entry, and returns no error — the filter is
// per-flag, not per-event.
func TestSubmitWorldEventFiltersNonWorldFlags(t *testing.T) {
	pipe, flags, acc, _ := newTestPipeline()
	addEntity(acc, 1)

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "trust_observed",
		TargetEntityID: 1,
		RegionID:       "thornwell",
		FlagMutations:  map[string]any{"world.a": true, "trust.x": 5},
	})
	if err != nil {
		t.Fatalf("SubmitWorldEvent returned unexpected error: %v", err)
	}

	got := flags.GetWorldFlags(1)
	if len(got) != 1 || got["world.a"] != true {
		t.Errorf("GetWorldFlags = %v, want exactly {world.a: true}", got)
	}
	// trust.x must not leak into the Persistent component at all.
	if _, ok := acc.components[1].StateFlags["trust.x"]; ok {
		t.Error("trust.x leaked into the Persistent component; non-world.* flags must be skipped")
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 1 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 1 (log written despite skipped flag)", log.Len())
	}
}

// TestSubmitWorldEventAllFlagsNonWorldStillProcessed covers the QA AC-2 edge
// case: when every flag is non-world, the event is still processed — no
// flags are mutated, the log entry is written, and no error is returned.
func TestSubmitWorldEventAllFlagsNonWorldStillProcessed(t *testing.T) {
	pipe, flags, acc, bus := newTestPipeline()
	addEntity(acc, 1)

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "observation",
		TargetEntityID: 1,
		RegionID:       "thornwell",
		FlagMutations:  map[string]any{"trust.x": 5, "investigation.y": true},
	})
	if err != nil {
		t.Fatalf("SubmitWorldEvent returned unexpected error: %v", err)
	}
	if got := flags.GetWorldFlags(1); len(got) != 0 {
		t.Errorf("GetWorldFlags = %v, want empty", got)
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 1 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 1", log.Len())
	}
	if len(bus.emitted) != 1 {
		t.Errorf("broadcast count = %d, want 1", len(bus.emitted))
	}
}

// TestSubmitWorldEventSchemaMismatchFlagSkipped covers per-flag isolation
// (QA "mixed types: each flag independently validated"): a world.* flag
// whose value does not match the registered schema is skipped by
// SetWorldFlag (ErrTypeMismatch), while the other world.* flags in the same
// event are still applied, the log entry is still written, and no error is
// returned.
func TestSubmitWorldEventSchemaMismatchFlagSkipped(t *testing.T) {
	pipe, flags, acc, _ := newTestPipeline()
	addEntity(acc, 1)
	flags.RegisterFlagType("world.occupant", reflect.TypeOf("")) // string schema

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "occupancy_scan",
		TargetEntityID: 1,
		RegionID:       "thornwell",
		// bool against a string schema; sibling flag is valid.
		FlagMutations: map[string]any{"world.occupant": true, "world.a": true},
	})
	if err != nil {
		t.Fatalf("SubmitWorldEvent returned unexpected error: %v", err)
	}

	got := flags.GetWorldFlags(1)
	if _, ok := got["world.occupant"]; ok {
		t.Errorf("world.occupant = %v, want skipped (bool against string schema)", got["world.occupant"])
	}
	if v, ok := got["world.a"]; !ok || v != true {
		t.Errorf("world.a = (%v, %v), want (true, true) — sibling flag must still be applied", v, ok)
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 1 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 1", log.Len())
	}
}

// TestSubmitWorldEventSameFrameOrderingLastWriterWins covers AC-3: two
// events A then B submitted in the same frame both write world.count; both
// memory log entries are appended in A→B order and world.count holds B's
// value — submission order resolves the conflict, last writer wins. (PWS is
// the authoritative world.* writer, so no per-event priority exists; see
// the file header note.)
func TestSubmitWorldEventSameFrameOrderingLastWriterWins(t *testing.T) {
	pipe, flags, acc, _ := newTestPipeline()
	addEntity(acc, 1)

	eventA := persistent_world_state.WorldEvent{
		EventType:      "first_write",
		TargetEntityID: 1,
		RegionID:       "thornwell",
		FlagMutations:  map[string]any{"world.count": 1},
	}
	eventB := persistent_world_state.WorldEvent{
		EventType:      "second_write",
		TargetEntityID: 1,
		RegionID:       "thornwell",
		FlagMutations:  map[string]any{"world.count": 2},
	}
	if err := pipe.SubmitWorldEvent(eventA); err != nil {
		t.Fatalf("SubmitWorldEvent(A) returned unexpected error: %v", err)
	}
	if err := pipe.SubmitWorldEvent(eventB); err != nil {
		t.Fatalf("SubmitWorldEvent(B) returned unexpected error: %v", err)
	}

	if v, found := flags.GetWorldFlag(1, "world.count"); !found || v != int64(2) {
		t.Errorf("world.count = (%v, %v), want (int64(2), true) — B (last writer) wins", v, found)
	}

	log := pipe.MemoryLogFor(1)
	if log.Len() != 2 {
		t.Fatalf("MemoryLogFor(1) has %d entries, want 2", log.Len())
	}
	// Entries are stored oldest-first: A's entry must precede B's (EventID
	// is monotonic and assigned at submission time).
	if log.Entries[0].EventType != "first_write" || log.Entries[1].EventType != "second_write" {
		t.Errorf("log order = [%s, %s], want [first_write, second_write]",
			log.Entries[0].EventType, log.Entries[1].EventType)
	}
	if !(log.Entries[0].EventID < log.Entries[1].EventID) {
		t.Errorf("EventIDs = [%d, %d], want ascending (submission order)", log.Entries[0].EventID, log.Entries[1].EventID)
	}
}

// TestSubmitWorldEventSameFrameDisjointFlagsBothApplied covers the QA AC-3
// edge case: two same-frame events writing different flags do not conflict —
// both mutations are applied and both log entries are appended.
func TestSubmitWorldEventSameFrameDisjointFlagsBothApplied(t *testing.T) {
	pipe, flags, acc, _ := newTestPipeline()
	addEntity(acc, 1)

	if err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType: "first", TargetEntityID: 1, RegionID: "thornwell",
		FlagMutations: map[string]any{"world.f": true},
	}); err != nil {
		t.Fatalf("SubmitWorldEvent(A): %v", err)
	}
	if err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType: "second", TargetEntityID: 1, RegionID: "thornwell",
		FlagMutations: map[string]any{"world.g": true},
	}); err != nil {
		t.Fatalf("SubmitWorldEvent(B): %v", err)
	}

	got := flags.GetWorldFlags(1)
	if len(got) != 2 || got["world.f"] != true || got["world.g"] != true {
		t.Errorf("GetWorldFlags = %v, want both world.f and world.g", got)
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 2 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 2", log.Len())
	}
}

// TestSubmitWorldEventInvalidTargetRejected covers AC-4: an event targeting
// an entity that does not exist (never created, or destroyed — components
// are removed) is rejected with ErrInvalidTarget and produces no flag
// mutations, no log entry, and no broadcast.
func TestSubmitWorldEventInvalidTargetRejected(t *testing.T) {
	pipe, flags, acc, bus := newTestPipeline()
	addEntity(acc, 1)

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "location_cleared",
		TargetEntityID: 999, // never created
		RegionID:       "thornwell",
		FlagMutations:  map[string]any{"world.ruin_cleared": true},
	})
	if !errors.Is(err, persistent_world_state.ErrInvalidTarget) {
		t.Fatalf("SubmitWorldEvent error = %v, want ErrInvalidTarget", err)
	}

	if _, found := flags.GetWorldFlag(999, "world.ruin_cleared"); found {
		t.Error("flag mutated despite rejection")
	}
	if ids := flags.GetEntitiesWithFlag("world.ruin_cleared", ""); len(ids) != 0 {
		t.Errorf("GetEntitiesWithFlag = %v, want empty", ids)
	}
	if log := pipe.MemoryLogFor(999); log.Len() != 0 {
		t.Errorf("MemoryLogFor(999) has %d entries, want 0", log.Len())
	}
	if len(bus.emitted) != 0 {
		t.Errorf("broadcast count = %d, want 0 (no broadcast on rejection)", len(bus.emitted))
	}
}

// TestSubmitWorldEventZeroTargetRejected covers the QA AC-4 edge case "zero
// is not a valid target": a zero TargetEntityID is invalid — there is no
// entity to mutate and no log to append to.
func TestSubmitWorldEventZeroTargetRejected(t *testing.T) {
	pipe, _, acc, bus := newTestPipeline()
	addEntity(acc, 1)

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "location_cleared",
		TargetEntityID: 0,
		RegionID:       "thornwell",
	})
	if !errors.Is(err, persistent_world_state.ErrInvalidTarget) {
		t.Fatalf("SubmitWorldEvent error = %v, want ErrInvalidTarget", err)
	}
	if len(bus.emitted) != 0 {
		t.Errorf("broadcast count = %d, want 0", len(bus.emitted))
	}
}

// TestSubmitWorldEventInvalidActorRejected covers the QA AC-4 edge case "actor
// entity invalid": a non-zero actor that does not exist rejects the event
// with ErrInvalidActor and no side effects occur.
func TestSubmitWorldEventInvalidActorRejected(t *testing.T) {
	pipe, flags, acc, bus := newTestPipeline()
	addEntity(acc, 1)

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "location_cleared",
		ActorEntityID:  999, // never created
		TargetEntityID: 1,
		RegionID:       "thornwell",
		FlagMutations:  map[string]any{"world.ruin_cleared": true},
	})
	if !errors.Is(err, persistent_world_state.ErrInvalidActor) {
		t.Fatalf("SubmitWorldEvent error = %v, want ErrInvalidActor", err)
	}
	if _, found := flags.GetWorldFlag(1, "world.ruin_cleared"); found {
		t.Error("flag mutated despite actor rejection")
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 0 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 0", log.Len())
	}
	if len(bus.emitted) != 0 {
		t.Errorf("broadcast count = %d, want 0", len(bus.emitted))
	}
}

// TestSubmitWorldEventZeroActorSystemGenerated covers the QA AC-4 edge case
// "system-generated event": a zero actor skips the existence check and the
// entry records ActorID 0 (MemoryEntry.ActorID documents zero as valid for
// system-generated events).
func TestSubmitWorldEventZeroActorSystemGenerated(t *testing.T) {
	pipe, _, acc, _ := newTestPipeline()
	addEntity(acc, 1)

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "weather_change",
		TargetEntityID: 1,
		RegionID:       "thornwell",
	})
	if err != nil {
		t.Fatalf("SubmitWorldEvent returned unexpected error: %v", err)
	}
	if entry := pipe.MemoryLogFor(1).Entries[0]; entry.ActorID != 0 {
		t.Errorf("entry.ActorID = %d, want 0", entry.ActorID)
	}
}

// TestSubmitWorldEventInvalidEventTypeAndRegionRejected covers validation:
// an empty EventType returns ErrInvalidEventType and an empty RegionID
// returns ErrInvalidRegionID, with no side effects in either case.
func TestSubmitWorldEventInvalidEventTypeAndRegionRejected(t *testing.T) {
	pipe, flags, acc, bus := newTestPipeline()
	addEntity(acc, 1)

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		TargetEntityID: 1,
		RegionID:       "thornwell",
	})
	if !errors.Is(err, persistent_world_state.ErrInvalidEventType) {
		t.Fatalf("empty EventType error = %v, want ErrInvalidEventType", err)
	}

	err = pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "location_cleared",
		TargetEntityID: 1,
		FlagMutations:  map[string]any{"world.ruin_cleared": true},
	})
	if !errors.Is(err, persistent_world_state.ErrInvalidRegionID) {
		t.Fatalf("empty RegionID error = %v, want ErrInvalidRegionID", err)
	}

	if len(flags.GetWorldFlags(1)) != 0 {
		t.Error("flags mutated despite validation failure")
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 0 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 0", log.Len())
	}
	if len(bus.emitted) != 0 {
		t.Error("broadcast emitted despite validation failure")
	}
}

// TestSubmitWorldEventBroadcastsOnSuccess covers AC-5: a valid event
// broadcasts exactly one OnWorldStateChanged event with the submitted
// WorldEvent as payload, the pipeline timestamp, and the PWS source.
func TestSubmitWorldEventBroadcastsOnSuccess(t *testing.T) {
	pipe, _, acc, bus := newTestPipeline()
	addEntity(acc, 1)

	ev := persistent_world_state.WorldEvent{
		EventType:      "location_cleared",
		ActorEntityID:  1,
		TargetEntityID: 1,
		RegionID:       "thornwell",
		FlagMutations:  map[string]any{"world.ruin_cleared": true},
	}
	if err := pipe.SubmitWorldEvent(ev); err != nil {
		t.Fatalf("SubmitWorldEvent returned unexpected error: %v", err)
	}

	if len(bus.emitted) != 1 {
		t.Fatalf("broadcast count = %d, want exactly 1", len(bus.emitted))
	}
	got := bus.emitted[0]
	if got.Type != persistent_world_state.OnWorldStateChanged {
		t.Errorf("event.Type = %q, want %q", got.Type, persistent_world_state.OnWorldStateChanged)
	}
	if got.Timestamp != 100.0 {
		t.Errorf("event.Timestamp = %v, want 100.0 (the injected clock)", got.Timestamp)
	}
	if got.Source != "persistent_world_state" {
		t.Errorf("event.Source = %q, want %q", got.Source, "persistent_world_state")
	}
	payload, ok := got.Data.(persistent_world_state.WorldEvent)
	if !ok {
		t.Fatalf("event.Data = %T, want WorldEvent", got.Data)
	}
	if payload.EventType != ev.EventType || payload.TargetEntityID != ev.TargetEntityID {
		t.Errorf("payload = %+v, want EventType %q and TargetID %d", payload, ev.EventType, ev.TargetEntityID)
	}
}

// TestSubmitWorldEventNoBroadcastOnRejection covers the QA AC-5 edge case:
// a rejected event is not broadcast.
func TestSubmitWorldEventNoBroadcastOnRejection(t *testing.T) {
	pipe, _, acc, bus := newTestPipeline()
	addEntity(acc, 1)

	err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "location_cleared",
		TargetEntityID: 999,
		RegionID:       "thornwell",
	})
	if !errors.Is(err, persistent_world_state.ErrInvalidTarget) {
		t.Fatalf("SubmitWorldEvent error = %v, want ErrInvalidTarget", err)
	}
	if len(bus.emitted) != 0 {
		t.Errorf("broadcast count = %d, want 0", len(bus.emitted))
	}
}

// TestSubmitWorldEventBroadcastReachesAllSubscribers covers the QA AC-5 edge
// case "multiple subscribers (all receive)" and the same-frame guarantee:
// with the real EventBusImpl, two subscribers both receive the event during
// the SubmitWorldEvent call (synchronous dispatch, ADR-0011).
func TestSubmitWorldEventBroadcastReachesAllSubscribers(t *testing.T) {
	acc := newFakeAccessor()
	addEntity(acc, 1)
	flags := persistent_world_state.NewWorldFlagStore(acc, persistent_world_state.WorldFlagStoreConfig{})
	bus := entity.NewEventBus()

	received := make([]entity.Event, 0, 2)
	bus.Subscribe(persistent_world_state.OnWorldStateChanged, 5, func(ev entity.Event) {
		received = append(received, ev)
	})
	bus.Subscribe(persistent_world_state.OnWorldStateChanged, 5, func(ev entity.Event) {
		received = append(received, ev)
	})

	pipe := persistent_world_state.NewEventPipeline(acc, flags, bus, persistent_world_state.EventPipelineConfig{
		Clock: func() float64 { return 100.0 },
	})
	if err := pipe.SubmitWorldEvent(persistent_world_state.WorldEvent{
		EventType:      "location_cleared",
		TargetEntityID: 1,
		RegionID:       "thornwell",
	}); err != nil {
		t.Fatalf("SubmitWorldEvent returned unexpected error: %v", err)
	}

	if len(received) != 2 {
		t.Fatalf("subscribers received %d events, want 2", len(received))
	}
	for _, ev := range received {
		if ev.Type != persistent_world_state.OnWorldStateChanged {
			t.Errorf("subscriber event.Type = %q, want %q", ev.Type, persistent_world_state.OnWorldStateChanged)
		}
	}
}
