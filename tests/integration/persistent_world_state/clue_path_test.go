// Integration tests for Story 004 of the Persistent World State epic:
// Clue-Path Protection.
//
// Evidence path: production/epics/persistent-world-state/
//
//	story-004-clue-path-protection.md
//
// Tests map to the four acceptance criteria:
//
//	AC-1  test_quest_relevant_destruction_records_protection
//	      -> TestCluePathProtectorQuestRelevantDestructionRecordsProtection
//	AC-2  test_non_quest_relevant_destruction_no_action
//	      -> TestCluePathProtectorNonQuestRelevantDestructionNoAction
//	AC-3  test_destroyed_with_clues_flag_queryable
//	      -> TestCluePathProtectorDestroyedWithCluesFlagQueryable
//	AC-4  test_memory_log_preserves_destruction_context
//	      -> TestCluePathProtectorMemoryLogPreservesDestructionContext
//
// QA edge cases: entity with no Persistent component (no error, no flag, no
// log), quest-relevance unmarked before destruction, never-destroyed entity
// and never-existing entity ID return not-found, second destruction event
// rejected (idempotency), game clock paused (frozen timestamp), region
// recorded from the registry, world.state_changed broadcast emitted for
// protected events, priority-2 dispatch before downstream consumers, Stop
// unsubscribes, and the RecordDestructionEvent direct path (destroyed target
// permitted, region resolution, validation).
//
// Two Story 004 implementation facts (deviations from the story's
// Implementation Notes, decided at implementation time) shape these tests:
//
//  1. Quest relevance is PWS-owned (MarkQuestRelevant) — the destroyed
//     entity's Persistent component is purged BEFORE entity.destroyed fires
//     (EntityStoreImpl.retire), so it cannot be read at handler time. The
//     fake accessor mirrors that: the component is removed before each
//     destroyed event is emitted.
//  2. Destroyed entities' flags live in PWS-owned tombstone storage
//     (WorldFlagStore.NoteEntityDestroyed) — GetWorldFlag must return
//     (true, true) for world.<id>.destroyed_with_clues AFTER the purge.
//
// The fake accessor, real EventBusImpl, and fixed game clock keep the tests
// engine-agnostic (persistent_world_state imports no Kaiju APIs).
package persistent_world_state_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"marrow-of-empire/entity"
	"marrow-of-empire/persistent_world_state"
)

// newTestProtector wires the Story 004 stack: fake accessor, real flag store,
// real event bus, pipeline with a fixed (frozen) game clock at 100.0 minutes,
// and a started CluePathProtector subscribed on the bus at priority 2. The
// pipeline broadcasts OnWorldStateChanged on the same bus, mirroring game
// wiring (ADR-0011: synchronous dispatch, one bus).
func newTestProtector() (*persistent_world_state.EventPipeline, *persistent_world_state.WorldFlagStore, *fakeAccessor, *entity.EventBusImpl, *persistent_world_state.CluePathProtector) {
	acc := newFakeAccessor()
	flags := persistent_world_state.NewWorldFlagStore(acc, persistent_world_state.WorldFlagStoreConfig{})
	bus := entity.NewEventBus()
	pipe := persistent_world_state.NewEventPipeline(acc, flags, bus, persistent_world_state.EventPipelineConfig{
		Clock: func() float64 { return 100.0 },
	})
	protector := persistent_world_state.NewCluePathProtector(pipe)
	protector.Start(bus)
	return pipe, flags, acc, bus, protector
}

// emitDestroyed fires an entity.destroyed event for id exactly as the Entity
// Framework's retire() does: component stores are purged BEFORE the event, so
// callers remove the Persistent component first.
func emitDestroyed(bus *entity.EventBusImpl, id entity.GameEntityID) {
	bus.Emit(entity.Event{
		Type:   entity.EventEntityDestroyed,
		Source: "entity_framework",
		Data: entity.EntityEventData{
			EntityID:    id,
			EntityType:  entity.EntityTypeNPC,
			DisplayName: "Acolyte Vex",
		},
	})
}

// destroyQuestRelevant marks id quest-relevant, registers its region, gives
// it a Persistent component, purges the component (mirroring
// EntityStoreImpl.retire: component stores are purged BEFORE entity.destroyed
// is emitted), and fires the destroyed event.
func destroyQuestRelevant(acc *fakeAccessor, flags *persistent_world_state.WorldFlagStore, bus *entity.EventBusImpl, protector *persistent_world_state.CluePathProtector, id entity.GameEntityID, region string) {
	protector.MarkQuestRelevant(id)
	flags.SetEntityRegion(id, region)
	addEntity(acc, id)
	delete(acc.components, id) // retire() purges components before entity.destroyed
	emitDestroyed(bus, id)
}

// TestCluePathProtectorQuestRelevantDestructionRecordsProtection covers AC-1:
// destroying a quest-relevant entity records clue-path protection — the flag
// world.<id>.destroyed_with_clues = true is set (through tombstone storage,
// since the component is already purged) and the memory log gains a
// clue_path_protection entry.
func TestCluePathProtectorQuestRelevantDestructionRecordsProtection(t *testing.T) {
	pipe, flags, acc, bus, protector := newTestProtector()
	destroyQuestRelevant(acc, flags, bus, protector, 1, "thornwell_ruins")

	flag := persistent_world_state.DestroyedWithCluesFlag(1)
	got, found := flags.GetWorldFlag(1, flag)
	if !found || got != true {
		t.Errorf("GetWorldFlag(1, %q) = (%v, %v), want (true, true)", flag, got, found)
	}
	log := pipe.MemoryLogFor(1)
	if log.Len() != 1 {
		t.Fatalf("MemoryLogFor(1) has %d entries, want 1", log.Len())
	}
	entry := log.Entries[0]
	if entry.EventType != persistent_world_state.EventTypeCluePathProtection {
		t.Errorf("entry.EventType = %q, want %q", entry.EventType, persistent_world_state.EventTypeCluePathProtection)
	}
	if !entry.Protected {
		t.Error("entry.Protected = false, want true (clue-path records are never evicted)")
	}
}

// TestCluePathProtectorNonQuestRelevantDestructionNoAction covers AC-2: a
// non-quest-relevant entity's destruction produces no flag, no log entry, and
// no broadcast — the event is observed but not acted upon. The component is
// purged exactly as at real destruction.
func TestCluePathProtectorNonQuestRelevantDestructionNoAction(t *testing.T) {
	pipe, flags, acc, bus, _ := newTestProtector()

	broadcasts := 0
	bus.Subscribe(persistent_world_state.OnWorldStateChanged, 5, func(entity.Event) { broadcasts++ })

	addEntity(acc, 1)
	delete(acc.components, 1) // retire() purges components before entity.destroyed
	emitDestroyed(bus, 1)

	if got, found := flags.GetWorldFlag(1, persistent_world_state.DestroyedWithCluesFlag(1)); found {
		t.Errorf("GetWorldFlag = (%v, %v), want not found — no flag for non-quest-relevant destruction", got, found)
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 0 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 0", log.Len())
	}
	if broadcasts != 0 {
		t.Errorf("broadcast count = %d, want 0", broadcasts)
	}
}

// TestCluePathProtectorDestroyedWithCluesFlagQueryable covers AC-3: after a
// clue-path-protected destruction, GetWorldFlag(entityID,
// "world.<entityID>.destroyed_with_clues") returns (true, true) — the exact
// shape the Investigation Journal queries. Tombstone storage keeps the flag
// alive after the component purge, and the flag surfaces through the
// full-flag query and the reverse index (unfiltered and region-scoped).
func TestCluePathProtectorDestroyedWithCluesFlagQueryable(t *testing.T) {
	_, flags, acc, bus, protector := newTestProtector()
	destroyQuestRelevant(acc, flags, bus, protector, 7, "hollow_moor")

	flag := fmt.Sprintf("world.%d.destroyed_with_clues", 7)
	got, found := flags.GetWorldFlag(7, flag)
	if got != true || !found {
		t.Errorf("GetWorldFlag(7, %q) = (%v, %v), want (true, true)", flag, got, found)
	}
	all := flags.GetWorldFlags(7)
	if len(all) != 1 || all[flag] != true {
		t.Errorf("GetWorldFlags(7) = %v, want exactly {%q: true}", all, flag)
	}
	if ids := flags.GetEntitiesWithFlag(flag, ""); len(ids) != 1 || ids[0] != 7 {
		t.Errorf("GetEntitiesWithFlag(%q, \"\") = %v, want [7]", flag, ids)
	}
	if ids := flags.GetEntitiesWithFlag(flag, "hollow_moor"); len(ids) != 1 || ids[0] != 7 {
		t.Errorf("GetEntitiesWithFlag(%q, hollow_moor) = %v, want [7]", flag, ids)
	}
}

// TestCluePathProtectorMemoryLogPreservesDestructionContext covers AC-4: the
// memory log entry for a protected destruction records which entity was
// destroyed (TargetID), the region (from the flag store registry), and the
// game timestamp (the injected frozen clock). ActorID is 0 — the
// destroyed-event payload carries no destroyer (QA: "or 0 for
// environmental"). The Details blob preserves the display name the Entity
// Framework deletes at destruction.
func TestCluePathProtectorMemoryLogPreservesDestructionContext(t *testing.T) {
	pipe, flags, acc, bus, protector := newTestProtector()
	destroyQuestRelevant(acc, flags, bus, protector, 3, "thornwell_ruins")

	entries := pipe.MemoryLogFor(3).GetEntries(0, 10)
	if len(entries) != 1 {
		t.Fatalf("GetEntries(0, 10) returned %d entries, want 1", len(entries))
	}
	entry := entries[0]
	if entry.EventType != persistent_world_state.EventTypeCluePathProtection {
		t.Errorf("entry.EventType = %q, want %q", entry.EventType, persistent_world_state.EventTypeCluePathProtection)
	}
	if entry.TargetID != 3 {
		t.Errorf("entry.TargetID = %d, want 3 (the destroyed entity)", entry.TargetID)
	}
	if entry.RegionID != "thornwell_ruins" {
		t.Errorf("entry.RegionID = %q, want thornwell_ruins (from the region registry)", entry.RegionID)
	}
	if entry.GameTimestamp != 100.0 {
		t.Errorf("entry.GameTimestamp = %v, want 100.0 (the frozen clock)", entry.GameTimestamp)
	}
	if entry.ActorID != 0 {
		t.Errorf("entry.ActorID = %d, want 0 (no destroyer in the payload)", entry.ActorID)
	}
	if !entry.Protected {
		t.Error("entry.Protected = false, want true")
	}
	if entry.EventID == 0 {
		t.Error("entry.EventID = 0, want a non-zero globally unique ID")
	}
	if !strings.Contains(entry.Details, `"display_name":"Acolyte Vex"`) ||
		!strings.Contains(entry.Details, `"entity_type":"NPC"`) {
		t.Errorf("entry.Details = %q, want the destroyed entity's display name and type", entry.Details)
	}
}

// TestCluePathProtectorEntityWithoutPersistentComponentNoAction covers the QA
// AC-2 edge case "entity without Persistent component at all": destroying an
// entity that never had a component produces no error, no flag, and no log
// entry. (Quest-relevant entities are always recorded — the component is
// never consulted, because at handler time EVERY destroyed entity has already
// had its component purged by retire().)
func TestCluePathProtectorEntityWithoutPersistentComponentNoAction(t *testing.T) {
	pipe, flags, _, bus, _ := newTestProtector()

	broadcasts := 0
	bus.Subscribe(persistent_world_state.OnWorldStateChanged, 5, func(entity.Event) { broadcasts++ })

	emitDestroyed(bus, 9) // never had a component, never quest-relevant

	if got, found := flags.GetWorldFlag(9, persistent_world_state.DestroyedWithCluesFlag(9)); found {
		t.Errorf("GetWorldFlag = (%v, %v), want not found", got, found)
	}
	if log := pipe.MemoryLogFor(9); log.Len() != 0 {
		t.Errorf("MemoryLogFor(9) has %d entries, want 0", log.Len())
	}
	if broadcasts != 0 {
		t.Errorf("broadcast count = %d, want 0", broadcasts)
	}
}

// TestCluePathProtectorQuestRelevantWithoutPersistentComponentStillProtected
// covers the QA AC-1 edge case "entity with no Persistent component (skip
// gracefully)": a quest-relevant entity whose component was never attached
// (or was purged before the event, as retire() always does) is still
// protected without error — quest relevance is PWS-owned and component-free.
func TestCluePathProtectorQuestRelevantWithoutPersistentComponentStillProtected(t *testing.T) {
	pipe, flags, _, bus, protector := newTestProtector()
	protector.MarkQuestRelevant(11)
	flags.SetEntityRegion(11, "thornwell_ruins")
	emitDestroyed(bus, 11) // no component ever attached

	got, found := flags.GetWorldFlag(11, persistent_world_state.DestroyedWithCluesFlag(11))
	if got != true || !found {
		t.Errorf("GetWorldFlag = (%v, %v), want (true, true)", got, found)
	}
	if log := pipe.MemoryLogFor(11); log.Len() != 1 {
		t.Errorf("MemoryLogFor(11) has %d entries, want 1", log.Len())
	}
}

// TestCluePathProtectorUnmarkQuestRelevantBeforeDestructionNoAction covers
// the QA AC-2 edge case "quest_relevant flag was removed before destruction":
// an entity whose mark was removed produces no flag, no log entry, and no
// broadcast.
func TestCluePathProtectorUnmarkQuestRelevantBeforeDestructionNoAction(t *testing.T) {
	pipe, flags, acc, bus, protector := newTestProtector()
	protector.MarkQuestRelevant(1)
	protector.UnmarkQuestRelevant(1)
	addEntity(acc, 1)
	delete(acc.components, 1)
	emitDestroyed(bus, 1)

	if _, found := flags.GetWorldFlag(1, persistent_world_state.DestroyedWithCluesFlag(1)); found {
		t.Error("flag set despite unmarked quest relevance")
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 0 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 0", log.Len())
	}
}

// TestCluePathProtectorDuplicateDestructionEventRejected covers the QA AC-4
// edge case "entity already destroyed; second event rejected": a second
// destruction event for the same entity is ignored — one log entry, one flag,
// one broadcast. The Entity Framework emits entity.destroyed exactly once;
// the guard keeps the record idempotent.
func TestCluePathProtectorDuplicateDestructionEventRejected(t *testing.T) {
	pipe, flags, acc, bus, protector := newTestProtector()

	broadcasts := 0
	bus.Subscribe(persistent_world_state.OnWorldStateChanged, 5, func(entity.Event) { broadcasts++ })

	destroyQuestRelevant(acc, flags, bus, protector, 1, "thornwell_ruins")
	emitDestroyed(bus, 1) // duplicate

	if log := pipe.MemoryLogFor(1); log.Len() != 1 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 1 (second event rejected)", log.Len())
	}
	if broadcasts != 1 {
		t.Errorf("broadcast count = %d, want 1", broadcasts)
	}
	got, found := flags.GetWorldFlag(1, persistent_world_state.DestroyedWithCluesFlag(1))
	if got != true || !found {
		t.Errorf("GetWorldFlag = (%v, %v), want (true, true)", got, found)
	}
}

// TestCluePathProtectorFrozenClockStampsFixedTimestamp covers the QA AC-1
// edge case "entity destroyed while game clock is paused (timestamp frozen)":
// the log entry carries the frozen clock value — no time-dependent behavior.
func TestCluePathProtectorFrozenClockStampsFixedTimestamp(t *testing.T) {
	pipe, flags, acc, bus, protector := newTestProtector()
	destroyQuestRelevant(acc, flags, bus, protector, 1, "thornwell_ruins")

	if entry := pipe.MemoryLogFor(1).Entries[0]; entry.GameTimestamp != 100.0 {
		t.Errorf("entry.GameTimestamp = %v, want 100.0 (frozen clock)", entry.GameTimestamp)
	}
}

// TestCluePathProtectorBroadcastEmittedForProtectedEvent covers the
// OnWorldStateChanged broadcast: a protected destruction emits exactly one
// world.state_changed event, synchronously during the destroyed dispatch,
// with the clue_path_protection WorldEvent as payload (ADR-0011).
func TestCluePathProtectorBroadcastEmittedForProtectedEvent(t *testing.T) {
	_, flags, acc, bus, protector := newTestProtector()

	var broadcasts []entity.Event
	bus.Subscribe(persistent_world_state.OnWorldStateChanged, 5, func(ev entity.Event) {
		broadcasts = append(broadcasts, ev)
	})

	destroyQuestRelevant(acc, flags, bus, protector, 1, "thornwell_ruins")

	if len(broadcasts) != 1 {
		t.Fatalf("broadcast count = %d, want 1", len(broadcasts))
	}
	got := broadcasts[0]
	if got.Type != persistent_world_state.OnWorldStateChanged {
		t.Errorf("event.Type = %q, want %q", got.Type, persistent_world_state.OnWorldStateChanged)
	}
	if got.Source != "persistent_world_state" {
		t.Errorf("event.Source = %q, want persistent_world_state", got.Source)
	}
	if got.Timestamp != 100.0 {
		t.Errorf("event.Timestamp = %v, want 100.0 (the injected clock)", got.Timestamp)
	}
	payload, ok := got.Data.(persistent_world_state.WorldEvent)
	if !ok {
		t.Fatalf("event.Data = %T, want WorldEvent", got.Data)
	}
	if payload.EventType != persistent_world_state.EventTypeCluePathProtection {
		t.Errorf("payload.EventType = %q, want %q", payload.EventType, persistent_world_state.EventTypeCluePathProtection)
	}
	if payload.TargetEntityID != 1 {
		t.Errorf("payload.TargetEntityID = %d, want 1", payload.TargetEntityID)
	}
	if payload.RegionID != "thornwell_ruins" {
		t.Errorf("payload.RegionID = %q, want thornwell_ruins (resolved from the registry)", payload.RegionID)
	}
}

// TestCluePathProtectorNeverDestroyedEntityFlagNotFound covers the AC-3 QA
// edge case: querying the destroyed_with_clues flag for a live entity that
// was never destroyed returns not-found — no tombstone is materialized.
func TestCluePathProtectorNeverDestroyedEntityFlagNotFound(t *testing.T) {
	_, flags, acc, _, _ := newTestProtector()
	addEntity(acc, 1) // live entity, never destroyed

	got, found := flags.GetWorldFlag(1, persistent_world_state.DestroyedWithCluesFlag(1))
	if got != nil || found {
		t.Errorf("GetWorldFlag = (%v, %v), want (nil, false)", got, found)
	}
}

// TestCluePathProtectorNonExistentEntityIDNotFound covers the AC-3 QA edge
// case: a GameEntityID that never existed returns not-found and holds no
// storage.
func TestCluePathProtectorNonExistentEntityIDNotFound(t *testing.T) {
	_, flags, _, _, _ := newTestProtector()

	got, found := flags.GetWorldFlag(999, persistent_world_state.DestroyedWithCluesFlag(999))
	if got != nil || found {
		t.Errorf("GetWorldFlag = (%v, %v), want (nil, false)", got, found)
	}
	if all := flags.GetWorldFlags(999); len(all) != 0 {
		t.Errorf("GetWorldFlags(999) = %v, want empty", all)
	}
}

// TestCluePathProtectorPriorityTwoDispatchesBeforeDownstreamConsumers covers
// ADR-0011's priority table: PWS subscribes at priority 2, so when a
// priority-3 consumer (Trust) and a priority-5 consumer (UI) run during the
// same entity.destroyed dispatch, the clue-path protection is already
// recorded — the flag is queryable.
func TestCluePathProtectorPriorityTwoDispatchesBeforeDownstreamConsumers(t *testing.T) {
	pipe, flags, acc, bus, protector := newTestProtector()

	sawFlagAtPriority3 := false
	sawFlagAtPriority5 := false
	bus.Subscribe(entity.EventEntityDestroyed, 3, func(entity.Event) {
		_, found := flags.GetWorldFlag(1, persistent_world_state.DestroyedWithCluesFlag(1))
		sawFlagAtPriority3 = found
	})
	bus.Subscribe(entity.EventEntityDestroyed, 5, func(entity.Event) {
		_, found := flags.GetWorldFlag(1, persistent_world_state.DestroyedWithCluesFlag(1))
		sawFlagAtPriority5 = found
	})

	destroyQuestRelevant(acc, flags, bus, protector, 1, "thornwell_ruins")

	if !sawFlagAtPriority3 {
		t.Error("priority-3 consumer did not see the destroyed_with_clues flag — PWS (priority 2) must run first")
	}
	if !sawFlagAtPriority5 {
		t.Error("priority-5 consumer did not see the destroyed_with_clues flag")
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 1 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 1", log.Len())
	}
}

// TestCluePathProtectorStopUnsubscribes covers the Stop() contract: after
// Stop, entity.destroyed events reach no handler — no flag, no log, no
// broadcast. Start after Stop re-subscribes: the registration never outlives
// its owner and the protector stays reusable.
func TestCluePathProtectorStopUnsubscribes(t *testing.T) {
	pipe, flags, acc, bus, protector := newTestProtector()

	protector.Stop()
	protector.MarkQuestRelevant(1)
	addEntity(acc, 1)
	delete(acc.components, 1)
	emitDestroyed(bus, 1)
	if _, found := flags.GetWorldFlag(1, persistent_world_state.DestroyedWithCluesFlag(1)); found {
		t.Error("flag set while stopped")
	}
	if log := pipe.MemoryLogFor(1); log.Len() != 0 {
		t.Errorf("MemoryLogFor(1) has %d entries, want 0 while stopped", log.Len())
	}

	protector.Start(bus)
	protector.MarkQuestRelevant(2)
	flags.SetEntityRegion(2, "thornwell_ruins")
	addEntity(acc, 2)
	delete(acc.components, 2)
	emitDestroyed(bus, 2)
	if got, found := flags.GetWorldFlag(2, persistent_world_state.DestroyedWithCluesFlag(2)); !found || got != true {
		t.Errorf("GetWorldFlag after restart = (%v, %v), want (true, true)", got, found)
	}
}

// TestCluePathProtectorQuestRelevanceRegistry covers the registry API
// lifecycle: unmarked by default, marked after MarkQuestRelevant, unmarked
// after UnmarkQuestRelevant.
func TestCluePathProtectorQuestRelevanceRegistry(t *testing.T) {
	_, _, _, _, protector := newTestProtector()

	if protector.IsQuestRelevant(1) {
		t.Error("IsQuestRelevant(1) = true, want false (unmarked by default)")
	}
	protector.MarkQuestRelevant(1)
	if !protector.IsQuestRelevant(1) {
		t.Error("IsQuestRelevant(1) = false, want true after MarkQuestRelevant")
	}
	protector.UnmarkQuestRelevant(1)
	if protector.IsQuestRelevant(1) {
		t.Error("IsQuestRelevant(1) = true, want false after UnmarkQuestRelevant")
	}
}

// TestRecordDestructionEventPermitsDestroyedTarget covers the dedicated
// pipeline path: it records a destruction for an entity whose component is
// gone (SubmitWorldEvent would reject it with ErrInvalidTarget), resolves the
// region from the flag store registry when the event leaves it empty, writes
// the flag through tombstone storage, appends a protected log entry, and
// broadcasts.
func TestRecordDestructionEventPermitsDestroyedTarget(t *testing.T) {
	acc := newFakeAccessor()
	flags := persistent_world_state.NewWorldFlagStore(acc, persistent_world_state.WorldFlagStoreConfig{})
	bus := entity.NewEventBus()
	pipe := persistent_world_state.NewEventPipeline(acc, flags, bus, persistent_world_state.EventPipelineConfig{
		Clock: func() float64 { return 42.0 },
	})

	// The entity existed once; retire() purged the component before the
	// destroyed event fired, exactly as in the real flow.
	addEntity(acc, 5)
	delete(acc.components, 5)
	flags.SetEntityRegion(5, "hollow_moor")

	err := pipe.RecordDestructionEvent(persistent_world_state.WorldEvent{
		EventType:      persistent_world_state.EventTypeCluePathProtection,
		TargetEntityID: 5,
		// RegionID intentionally empty: resolved from the registry.
		FlagMutations: map[string]any{
			persistent_world_state.DestroyedWithCluesFlag(5): true,
		},
	})
	if err != nil {
		t.Fatalf("RecordDestructionEvent returned unexpected error: %v", err)
	}

	got, found := flags.GetWorldFlag(5, persistent_world_state.DestroyedWithCluesFlag(5))
	if got != true || !found {
		t.Errorf("GetWorldFlag = (%v, %v), want (true, true) via tombstone", got, found)
	}
	entries := pipe.MemoryLogFor(5).GetEntries(0, 10)
	if len(entries) != 1 {
		t.Fatalf("GetEntries(0, 10) returned %d entries, want 1", len(entries))
	}
	if entries[0].RegionID != "hollow_moor" {
		t.Errorf("entry.RegionID = %q, want hollow_moor (resolved from registry)", entries[0].RegionID)
	}
	if entries[0].GameTimestamp != 42.0 {
		t.Errorf("entry.GameTimestamp = %v, want 42.0", entries[0].GameTimestamp)
	}
	if !entries[0].Protected {
		t.Error("entry.Protected = false, want true")
	}
}

// TestRecordDestructionEventValidation covers the direct-path validation:
// empty event type, unresolvable region, and zero target are rejected with
// the existing SubmitWorldEvent sentinels and no side effects.
func TestRecordDestructionEventValidation(t *testing.T) {
	acc := newFakeAccessor()
	flags := persistent_world_state.NewWorldFlagStore(acc, persistent_world_state.WorldFlagStoreConfig{})
	bus := entity.NewEventBus()
	pipe := persistent_world_state.NewEventPipeline(acc, flags, bus, persistent_world_state.EventPipelineConfig{})

	err := pipe.RecordDestructionEvent(persistent_world_state.WorldEvent{
		TargetEntityID: 1,
		RegionID:       "thornwell",
	})
	if !errors.Is(err, persistent_world_state.ErrInvalidEventType) {
		t.Fatalf("empty EventType error = %v, want ErrInvalidEventType", err)
	}

	// Region unresolvable: no registry entry for the target.
	err = pipe.RecordDestructionEvent(persistent_world_state.WorldEvent{
		EventType:      persistent_world_state.EventTypeCluePathProtection,
		TargetEntityID: 1,
	})
	if !errors.Is(err, persistent_world_state.ErrInvalidRegionID) {
		t.Fatalf("unresolvable RegionID error = %v, want ErrInvalidRegionID", err)
	}

	err = pipe.RecordDestructionEvent(persistent_world_state.WorldEvent{
		EventType: persistent_world_state.EventTypeCluePathProtection,
		RegionID:  "thornwell",
	})
	if !errors.Is(err, persistent_world_state.ErrInvalidTarget) {
		t.Fatalf("zero target error = %v, want ErrInvalidTarget", err)
	}

	if len(flags.GetWorldFlags(1)) != 0 {
		t.Error("flags mutated despite validation failure")
	}
}
