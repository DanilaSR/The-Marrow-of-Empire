// Integration tests for Story EF-005: Serialization.
//
// Evidence path: production/epics/entity-framework/story-005-serialization.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_encode_captures_active_and_suspended_entities -> TestEncodeCapturesActiveAndSuspendedEntities
//	AC-2  test_round_trip_reconstructs_identical_state        -> TestRoundTripReconstructsIdenticalState
//	AC-3  test_empty_store_encodes_as_zero_bytes              -> TestEmptyStoreEncodesZeroBytes
//
// Edge cases: Pending/Destroyed exclusion (Pending is not externally
// observable — CreateEntity transitions to Active atomically — so the
// filter is exercised through Destroyed), max component count per entity
// (all 15 component types on one entity), round-trip suspension state,
// GameEntityID counter resume, reload into the same store, unknown store
// rejection, and lifecycle event suppression during load (GDD CR-8).
//
// The Kaiju host mock, eventRecorder, and newTestStore come from
// spatial_index_test.go in this package (Go cannot import test helpers
// across packages, so each test file shares them within the package).
package entity_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/encoding/pod"
	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
)

func createTestEntity(t *testing.T, store *entity.EntityStoreImpl, etype entity.EntityType, name string, pos matrix.Vec3) entity.GameEntityID {
	t.Helper()
	id, err := store.CreateEntity(etype, name, pos)
	if err != nil {
		t.Fatalf("CreateEntity(%q) error: %v", name, err)
	}
	return id
}

func mustSetComponent[T any](t *testing.T, store *entity.EntityStoreImpl, id entity.GameEntityID, component T) {
	t.Helper()
	if err := entity.SetComponent(store, id, component); err != nil {
		t.Fatalf("SetComponent on entity %d error: %v", id, err)
	}
}

func samePosition(a, b matrix.Vec3) bool {
	return a.Distance(b) == 0
}

// kaijuPosition reads the entity position from the mock host's engine entity.
func kaijuPosition(host *mockKaijuHost, id entity.GameEntityID) (matrix.Vec3, bool) {
	kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", id)))
	if kaiju == nil {
		return matrix.Vec3{}, false
	}
	return kaiju.Transform.Position(), true
}

// TestEncodeCapturesActiveAndSuspendedEntities is AC-1: 100 entities with
// various components; after suspension, Serialize must encode every Active
// and Suspended entity with full component state. Verified by deserializing
// into a fresh store and comparing state, components, and positions.
func TestEncodeCapturesActiveAndSuspendedEntities(t *testing.T) {
	source, _, _ := newTestStore()
	const count = 100
	ids := make([]entity.GameEntityID, 0, count)
	for i := 0; i < count; i++ {
		id := createTestEntity(t, source, entity.EntityTypeNPC, fmt.Sprintf("Guard%d", i), matrix.NewVec3(float32(i)*2, 0, 0))
		ids = append(ids, id)
		mustSetComponent(t, source, id, entity.Health{Current: 100 - i, Max: 100, Temporary: i})
		if i%2 == 0 {
			mustSetComponent(t, source, id, entity.Stats{Strength: 10 + i%9, Dexterity: 11, Charisma: 12})
		}
		if i%4 == 0 {
			mustSetComponent(t, source, id, entity.Visual{ModelRef: "meshes/guard.fbx", RenderLayer: 2})
		}
	}
	// Entities at distance > 100 units (i > 50) become Suspended.
	source.UpdateSuspension(matrix.NewVec3(0, 0, 0))

	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}
	if len(blob) == 0 {
		t.Fatal("expected a non-empty blob for 100 entities")
	}

	restored, restoredHost, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}

	restoredIDs := restored.QueryEntities(entity.Query{})
	if len(restoredIDs) != count {
		t.Fatalf("expected %d restored entities, got %d", count, len(restoredIDs))
	}
	for i, id := range ids {
		state, ok := restored.EntityState(id)
		if !ok {
			t.Fatalf("entity %d missing after round-trip", id)
		}
		wantSuspended := i > 50
		if wantSuspended && state != entity.EntityStateSuspended {
			t.Errorf("entity %d: expected Suspended, got %v", id, state)
		}
		if !wantSuspended && state != entity.EntityStateActive {
			t.Errorf("entity %d: expected Active, got %v", id, state)
		}
		// Component state is fully encoded.
		hp, ok := entity.GetComponent[entity.Health](restored, id)
		if !ok || hp != (entity.Health{Current: 100 - i, Max: 100, Temporary: i}) {
			t.Errorf("entity %d: Health not round-tripped, got %+v ok=%v", id, hp, ok)
		}
		if i%2 == 0 {
			if _, ok := entity.GetComponent[entity.Stats](restored, id); !ok {
				t.Errorf("entity %d: Stats missing after round-trip", id)
			}
		} else if _, ok := entity.GetComponent[entity.Stats](restored, id); ok {
			t.Errorf("entity %d: Stats present on entity that never had them", id)
		}
		if i%4 == 0 {
			if _, ok := entity.GetComponent[entity.Visual](restored, id); !ok {
				t.Errorf("entity %d: Visual missing after round-trip", id)
			}
		}
		// The Kaiju transform position is restored.
		pos, ok := kaijuPosition(restoredHost, id)
		if !ok || !samePosition(pos, matrix.NewVec3(float32(i)*2, 0, 0)) {
			t.Errorf("entity %d: position not restored, got %v ok=%v", id, pos, ok)
		}
	}
}

// TestRoundTripReconstructsIdenticalState is AC-2 (QA case): 50 Active and
// Suspended entities with Health, Stats, and Visual components at known
// positions round-trip with identical GameEntityIDs, positions, and
// component values.
func TestRoundTripReconstructsIdenticalState(t *testing.T) {
	source, _, _ := newTestStore()
	const count = 50
	wantHealth := make(map[entity.GameEntityID]entity.Health, count)
	wantStats := make(map[entity.GameEntityID]entity.Stats, count)
	wantVisual := make(map[entity.GameEntityID]entity.Visual, count)
	wantPos := make(map[entity.GameEntityID]matrix.Vec3, count)
	for i := 0; i < count; i++ {
		pos := matrix.NewVec3(float32(i)*5, float32(i%7), 0)
		id := createTestEntity(t, source, entity.EntityTypeCharacter, fmt.Sprintf("Hero%d", i), pos)
		wantPos[id] = pos
		wantHealth[id] = entity.Health{Current: 50 + i, Max: 120, Temporary: i % 3}
		mustSetComponent(t, source, id, wantHealth[id])
		if i%2 == 0 {
			wantStats[id] = entity.Stats{Strength: 8 + i%11, Dexterity: 12, Constitution: 10, Intelligence: 9, Wisdom: 11, Charisma: 14, Evasion: 12}
			mustSetComponent(t, source, id, wantStats[id])
		}
		if i%3 == 0 {
			wantVisual[id] = entity.Visual{ModelRef: "meshes/hero.fbx", AnimationState: "idle", VisibilityFlag: 3}
			mustSetComponent(t, source, id, wantVisual[id])
		}
	}
	source.UpdateSuspension(matrix.NewVec3(0, 0, 0))

	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}

	restored, restoredHost, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}

	// Identical GameEntityIDs, in registration order.
	gotIDs := restored.QueryEntities(entity.Query{})
	if !reflect.DeepEqual(gotIDs, source.QueryEntities(entity.Query{})) {
		t.Errorf("GameEntityID sets differ:\n got %v\nwant %v", gotIDs, source.QueryEntities(entity.Query{}))
	}

	for _, id := range gotIDs {
		// Identical positions, read back from the restored Kaiju transform.
		pos, ok := kaijuPosition(restoredHost, id)
		if !ok || !samePosition(pos, wantPos[id]) {
			t.Errorf("entity %d: position not restored, got %v ok=%v want %v", id, pos, ok, wantPos[id])
		}
		// Identical component values.
		hp, ok := entity.GetComponent[entity.Health](restored, id)
		if !ok || hp != wantHealth[id] {
			t.Errorf("entity %d: Health mismatch, got %+v want %+v ok=%v", id, hp, wantHealth[id], ok)
		}
		stats, hasStats := wantStats[id]
		gotStats, okStats := entity.GetComponent[entity.Stats](restored, id)
		if hasStats != okStats || (okStats && gotStats != stats) {
			t.Errorf("entity %d: Stats mismatch, got %+v want %+v ok=%v", id, gotStats, stats, okStats)
		}
		visual, hasVisual := wantVisual[id]
		gotVisual, okVisual := entity.GetComponent[entity.Visual](restored, id)
		if hasVisual != okVisual || (okVisual && gotVisual != visual) {
			t.Errorf("entity %d: Visual mismatch, got %+v want %+v ok=%v", id, gotVisual, visual, okVisual)
		}
	}
}

// TestEmptyStoreEncodesZeroBytes is AC-3. Two cases: (a) a fresh store with
// no entities at all encodes as literally zero bytes and round-trips as a
// no-op; (b) a store with entities where a component type has no entities
// serializes without error and the empty store contributes zero bytes (its
// data is absent after the round-trip).
func TestEmptyStoreEncodesZeroBytes(t *testing.T) {
	// Case (a): fully empty store -> zero bytes, no error.
	empty, _, _ := newTestStore()
	blob, err := empty.Serialize()
	if err != nil {
		t.Fatalf("Serialize on empty store error: %v", err)
	}
	if len(blob) != 0 {
		t.Fatalf("empty store encoded %d bytes, want 0", len(blob))
	}
	reloaded, _, _ := newTestStore()
	if err := reloaded.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize of empty blob error: %v", err)
	}
	if got := reloaded.QueryEntities(entity.Query{}); len(got) != 0 {
		t.Fatalf("deserialized empty store has %d entities, want 0", len(got))
	}

	// Case (b): entities exist, but a component type has no entities.
	source, _, _ := newTestStore()
	for i := 0; i < 5; i++ {
		id := createTestEntity(t, source, entity.EntityTypeProp, fmt.Sprintf("Crate%d", i), matrix.NewVec3(float32(i), 0, 0))
		mustSetComponent(t, source, id, entity.Health{Current: 10, Max: 10})
	}
	// Schedule, Stats, Audio, ... were never written: those stores are
	// empty (or absent) and must contribute zero bytes without error.
	blob, err = source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}
	if len(blob) == 0 {
		t.Fatal("expected a non-empty blob when entities exist")
	}
	restored, _, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}
	if got := len(restored.QueryEntities(entity.Query{})); got != 5 {
		t.Fatalf("expected 5 restored entities, got %d", got)
	}
	for _, name := range []string{"Schedule", "Stats", "Audio", "Inventory", "Knowledge"} {
		if got, _ := restored.GetComponentData(1, name); got != nil {
			t.Errorf("component type %q should be absent after round-trip (empty store)", name)
		}
	}
}

// TestRoundTripWithAllComponentTypes covers the max-component-count edge
// case: one entity carrying all 15 component types, including the
// slice/map-bearing ones and the StateFlags tagged union, round-trips
// bit-identically. A second entity with a single component verifies that
// per-entity distribution does not leak components.
func TestRoundTripWithAllComponentTypes(t *testing.T) {
	source, _, _ := newTestStore()
	id := createTestEntity(t, source, entity.EntityTypeNPC, "Everything", matrix.NewVec3(12, 34, 0))

	want := map[string]any{
		"Health":    entity.Health{Current: 42, Max: 100, Temporary: 5},
		"Stats":     entity.Stats{Strength: 12, Dexterity: 10, Constitution: 14, Intelligence: 9, Wisdom: 11, Charisma: 8, MaxHealth: 48, ActionPoints: 4, Initiative: 1, Evasion: 10},
		"Inventory": entity.Inventory{Items: []entity.GameEntityID{2, 3}, EquipmentSlots: map[string]entity.GameEntityID{"weapon": 4}, Gold: 150},
		"Knowledge": entity.Knowledge{Domains: []string{"alchemy", "history"}, KnownFacts: []string{"f1", "f2"}, TrustThreshold: 2},
		"Faction":   entity.Faction{FactionID: "thornwell", Standing: 25},
		"Dialogue":  entity.Dialogue{Greeting: "Hail", Topics: []string{"weather", "work"}, PersonalityProfile: "grumpy"},
		"StatusEffect": entity.StatusEffect{EffectID: "bleed", SourceID: 7, Duration: 3, StackCount: 2,
			Parameters: map[string]float64{"damage_per_tick": 1.5}},
		"Combatant": entity.Combatant{Initiative: 9, ActionPoints: 4, MovementPoints: 5, MovementSpeed: 3.5,
			Actions: []string{"attack", "guard"}, ReactionAvailable: true,
			StatusEffects: []entity.StatusEffect{{EffectID: "bleed", SourceID: 7, Duration: 3, StackCount: 2}}},
		"Interactable": entity.Interactable{InteractionType: entity.InteractionTalk, PromptText: "Talk", Range: 2.5, IsOnceOnly: true},
		"MemoryLogEntry": entity.MemoryLogEntry{Timestamp: 90.5, EventType: "kill",
			Details: map[string]string{"target": "rat", "place": "cellar"}},
		"Persistent": entity.Persistent{
			StateFlags: map[string]entity.StateFlagValue{
				"world.door_open":       {Kind: entity.StateFlagBool, Bool: true},
				"trust.level":           {Kind: entity.StateFlagInt, Int: 2},
				"world.rain_intensity":  {Kind: entity.StateFlagFloat, Float: 0.75},
				"investigation.clue":    {Kind: entity.StateFlagString, Str: "dagger"},
			},
			MemoryLog:  []entity.MemoryLogEntry{{Timestamp: 90.5, EventType: "kill", Details: map[string]string{"target": "rat"}}},
			ModifiedAt: 120.5,
		},
		"Visual": entity.Visual{ModelRef: "meshes/guard.fbx", AnimationState: "idle",
			Tint: matrix.NewVec3(1, 0.5, 0.25), VisibilityFlag: 3, RenderLayer: 2},
		"Audio": entity.Audio{AmbientSound: "forest_loop", FootstepType: "gravel", OcclusionFlag: true},
		"ScheduleEntry": entity.ScheduleEntry{StartTime: 480, EndTime: 720, Activity: "working",
			TargetPosition: matrix.NewVec3(5, 6, 7)},
		"Schedule": entity.Schedule{
			Entries:         []entity.ScheduleEntry{{StartTime: 480, EndTime: 720, Activity: "working", TargetPosition: matrix.NewVec3(5, 6, 7)}},
			CurrentActivity: "working",
		},
	}
	mustSetComponent(t, source, id, want["Health"].(entity.Health))
	mustSetComponent(t, source, id, want["Stats"].(entity.Stats))
	mustSetComponent(t, source, id, want["Inventory"].(entity.Inventory))
	mustSetComponent(t, source, id, want["Knowledge"].(entity.Knowledge))
	mustSetComponent(t, source, id, want["Faction"].(entity.Faction))
	mustSetComponent(t, source, id, want["Dialogue"].(entity.Dialogue))
	mustSetComponent(t, source, id, want["StatusEffect"].(entity.StatusEffect))
	mustSetComponent(t, source, id, want["Combatant"].(entity.Combatant))
	mustSetComponent(t, source, id, want["Interactable"].(entity.Interactable))
	mustSetComponent(t, source, id, want["MemoryLogEntry"].(entity.MemoryLogEntry))
	mustSetComponent(t, source, id, want["Persistent"].(entity.Persistent))
	mustSetComponent(t, source, id, want["Visual"].(entity.Visual))
	mustSetComponent(t, source, id, want["Audio"].(entity.Audio))
	mustSetComponent(t, source, id, want["ScheduleEntry"].(entity.ScheduleEntry))
	mustSetComponent(t, source, id, want["Schedule"].(entity.Schedule))

	// A second entity with only Health must not pick up any of the others.
	other := createTestEntity(t, source, entity.EntityTypeItem, "Bare", matrix.NewVec3(1, 1, 1))
	mustSetComponent(t, source, other, entity.Health{Current: 1, Max: 1})

	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}
	restored, _, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}

	for name, w := range want {
		got, _ := restored.GetComponentData(id, name)
		if !reflect.DeepEqual(got, w) {
			t.Errorf("component %s mismatch after round-trip:\n got %#v\nwant %#v", name, got, w)
		}
	}
	for _, name := range []string{"Stats", "Inventory", "Knowledge", "Faction", "Dialogue", "StatusEffect",
		"Combatant", "Interactable", "MemoryLogEntry", "Persistent", "Visual", "Audio", "ScheduleEntry", "Schedule"} {
		if got, _ := restored.GetComponentData(other, name); got != nil {
			t.Errorf("component %s leaked onto entity %d, got %#v", name, other, got)
		}
	}
}

// TestDestroyedEntitiesExcludedFromEncode covers the GDD CR-8 exclusion of
// Destroyed entities (Pending is not externally observable: CreateEntity
// transitions to Active atomically, so the state filter is exercised through
// Destroyed).
func TestDestroyedEntitiesExcludedFromEncode(t *testing.T) {
	source, _, _ := newTestStore()
	var keep []entity.GameEntityID
	for i := 1; i <= 5; i++ {
		id := createTestEntity(t, source, entity.EntityTypeNPC, fmt.Sprintf("N%d", i), matrix.NewVec3(float32(i), 0, 0))
		mustSetComponent(t, source, id, entity.Health{Current: i * 10, Max: 100})
		if i != 2 && i != 4 {
			keep = append(keep, id)
		}
	}
	if err := source.DestroyEntity(2); err != nil {
		t.Fatalf("DestroyEntity(2) error: %v", err)
	}
	if err := source.DestroyEntity(4); err != nil {
		t.Fatalf("DestroyEntity(4) error: %v", err)
	}

	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}
	restored, _, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}
	got := restored.QueryEntities(entity.Query{})
	if !reflect.DeepEqual(got, keep) {
		t.Errorf("restored IDs %v, want %v (destroyed entities must be excluded)", got, keep)
	}
	for _, id := range keep {
		if _, ok := entity.GetComponent[entity.Health](restored, id); !ok {
			t.Errorf("entity %d: Health missing after round-trip", id)
		}
	}
}

// TestDeserializePreservesSuspensionState covers the edge case that a
// round-trip preserves each entity's Suspended/Active state and the native
// Kaiju entity activation flag.
func TestDeserializePreservesSuspensionState(t *testing.T) {
	source, _, _ := newTestStore()
	positions := []matrix.Vec3{
		matrix.NewVec3(10, 0, 0),  // Active (within 100 units)
		matrix.NewVec3(250, 0, 0), // Suspended
		matrix.NewVec3(5, 5, 0),   // Active
		matrix.NewVec3(999, 0, 0), // Suspended
	}
	var ids []entity.GameEntityID
	for i, pos := range positions {
		ids = append(ids, createTestEntity(t, source, entity.EntityTypeNPC, fmt.Sprintf("S%d", i), pos))
	}
	source.UpdateSuspension(matrix.NewVec3(0, 0, 0))

	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}
	restored, host, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}
	for i, id := range ids {
		wantSuspended := i == 1 || i == 3
		state, ok := restored.EntityState(id)
		if !ok {
			t.Fatalf("entity %d missing after round-trip", id)
		}
		if wantSuspended && state != entity.EntityStateSuspended {
			t.Errorf("entity %d: expected Suspended, got %v", id, state)
		}
		if !wantSuspended && state != entity.EntityStateActive {
			t.Errorf("entity %d: expected Active, got %v", id, state)
		}
		kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", id)))
		if kaiju == nil {
			t.Fatalf("entity %d: kaiju entity missing after round-trip", id)
		}
		if kaiju.IsActive() == wantSuspended {
			t.Errorf("entity %d: native activation flag %v does not match state %v", id, kaiju.IsActive(), state)
		}
	}
}

// TestDeserializeResumesEntityCounter covers GDD CR-1: after a round-trip,
// the monotonic counter resumes past the highest ID ever assigned, so
// post-load entity creation never collides with restored or retired IDs.
func TestDeserializeResumesEntityCounter(t *testing.T) {
	source, _, _ := newTestStore()
	_ = createTestEntity(t, source, entity.EntityTypeNPC, "A", matrix.NewVec3(1, 0, 0))
	id2 := createTestEntity(t, source, entity.EntityTypeNPC, "B", matrix.NewVec3(2, 0, 0))
	id3 := createTestEntity(t, source, entity.EntityTypeNPC, "C", matrix.NewVec3(3, 0, 0))
	if err := source.DestroyEntity(id2); err != nil {
		t.Fatalf("DestroyEntity error: %v", err)
	}

	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}
	restored, host, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}

	// The counter was 3 at save time; the next ID must be 4 — the destroyed
	// ID 2 must never be reused (GDD CR-1).
	next, err := restored.CreateEntity(entity.EntityTypeNPC, "D", matrix.NewVec3(4, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity after load error: %v", err)
	}
	if next != id3+1 {
		t.Errorf("next GameEntityID = %d, want %d (counter must resume past the saved counter)", next, id3+1)
	}
	if host.EntityById(engine.EntityId("game_2")) != nil {
		t.Error("destroyed entity 2 was recreated by the load")
	}
	if host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", next))) == nil {
		t.Errorf("new entity %d not registered with the host", next)
	}
}

// TestDeserializeIntoExistingStore covers reloading the same store: native
// entities with IDs present in the save are reused (no SetEntityId
// collision), prior state is replaced, and the counter resumes from the
// snapshot.
func TestDeserializeIntoExistingStore(t *testing.T) {
	store, _, _ := newTestStore()
	id1 := createTestEntity(t, store, entity.EntityTypeNPC, "One", matrix.NewVec3(1, 0, 0))
	id2 := createTestEntity(t, store, entity.EntityTypeNPC, "Two", matrix.NewVec3(2, 0, 0))
	mustSetComponent(t, store, id1, entity.Health{Current: 50, Max: 100})
	mustSetComponent(t, store, id2, entity.Health{Current: 75, Max: 100})

	blob, err := store.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}

	// Mutate the live state: destroy one entity, create another, and change
	// a component value.
	if err := store.DestroyEntity(id1); err != nil {
		t.Fatalf("DestroyEntity error: %v", err)
	}
	createTestEntity(t, store, entity.EntityTypeItem, "Three", matrix.NewVec3(3, 0, 0))
	mustSetComponent(t, store, id2, entity.Health{Current: 99, Max: 100})

	if err := store.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}

	got := store.QueryEntities(entity.Query{})
	if len(got) != 2 || got[0] != id1 || got[1] != id2 {
		t.Fatalf("state not replaced after reload: %v", got)
	}
	hp, ok := entity.GetComponent[entity.Health](store, id1)
	if !ok || hp != (entity.Health{Current: 50, Max: 100}) {
		t.Errorf("entity %d: Health not restored, got %+v ok=%v", id1, hp, ok)
	}
	hp, ok = entity.GetComponent[entity.Health](store, id2)
	if !ok || hp != (entity.Health{Current: 75, Max: 100}) {
		t.Errorf("entity %d: Health not restored, got %+v ok=%v", id2, hp, ok)
	}
	if state, _ := store.EntityState(id1); state != entity.EntityStateActive {
		t.Errorf("entity %d: state %v after reload, want Active", id1, state)
	}
	// The counter resumes from the snapshot (2), so the next ID is 3, not a
	// reuse of any pre-load ID.
	next, err := store.CreateEntity(entity.EntityTypeNPC, "Four", matrix.NewVec3(4, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity after reload error: %v", err)
	}
	if next != id2+1 {
		t.Errorf("next GameEntityID = %d, want %d", next, id2+1)
	}
}

// testSnapshot mirrors the entity package's unexported snapshot layout for
// corruption testing: pod decodes by field name, so a blob encoded from this
// layout decodes into the real snapshot. If the real layout's field names
// change, this test must be updated.
type testSnapshot struct {
	NextEntityID entity.GameEntityID
	Table        []testEntityRecord
	Stores       map[string][]byte
}

// testEntityRecord mirrors the entity package's unexported entityRecord.
type testEntityRecord struct {
	ID          entity.GameEntityID
	Type        entity.EntityType
	DisplayName string
	Position    matrix.Vec3
	State       entity.EntityState
}

// TestDeserializeRejectsUnknownComponentType covers the corruption guard:
// a blob referencing a component type with no registered store is rejected
// before any live state is touched.
func TestDeserializeRejectsUnknownComponentType(t *testing.T) {
	_ = pod.Register(testSnapshot{})     // registration errors are expected if run twice
	_ = pod.Register(testEntityRecord{}) // (test binaries run once per process)

	fake := testSnapshot{
		Stores: map[string][]byte{"Bogus": {1, 2, 3}},
	}
	var buf bytes.Buffer
	if err := pod.NewEncoder(&buf).Encode(fake); err != nil {
		t.Fatalf("encoding fake snapshot error: %v", err)
	}

	store, _, _ := newTestStore()
	id := createTestEntity(t, store, entity.EntityTypeNPC, "Survivor", matrix.NewVec3(1, 0, 0))
	err := store.Deserialize(buf.Bytes())
	if err == nil || !strings.Contains(err.Error(), "unknown component type") {
		t.Fatalf("Deserialize error = %v, want rejection of unknown component type", err)
	}
	// The live state must be untouched by the rejected load.
	if state, ok := store.EntityState(id); !ok || state != entity.EntityStateActive {
		t.Errorf("entity %d state changed by rejected load: %v ok=%v", id, state, ok)
	}
}

// TestDeserializeSuppressesLifecycleEvents covers GDD CR-8: loading a save
// is not entity creation — no lifecycle or component events fire.
func TestDeserializeSuppressesLifecycleEvents(t *testing.T) {
	source, _, _ := newTestStore()
	id1 := createTestEntity(t, source, entity.EntityTypeNPC, "A", matrix.NewVec3(1, 0, 0))
	id2 := createTestEntity(t, source, entity.EntityTypeNPC, "B", matrix.NewVec3(2, 0, 0))
	mustSetComponent(t, source, id1, entity.Health{Current: 50, Max: 100})
	mustSetComponent(t, source, id2, entity.Stats{Strength: 12})
	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}

	restored, _, bus := newTestStore()
	recorder := &eventRecorder{}
	for _, ev := range []entity.EventType{
		entity.EventEntityCreated, entity.EventEntityDestroyed,
		entity.EventEntitySuspended, entity.EventEntityReactivated,
		entity.EventComponentAdded, entity.EventComponentRemoved, entity.EventComponentChanged,
	} {
		recorder.subscribe(bus, ev)
	}
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}
	if n := len(recorder.received); n != 0 {
		t.Errorf("expected no lifecycle events during load, got %d: %v", n, recorder.received)
	}
}
