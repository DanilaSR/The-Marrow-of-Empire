// Unit tests for Story EF-002: Component Model.
//
// Evidence path: production/epics/entity-framework/story-002-component-model.md
// Each test function maps to one acceptance criterion (AC-1 .. AC-5); the
// remaining tests cover the edge semantics defined in the story and the GDD
// (idempotent removal, destroyed-entity rejection, store independence,
// query filtering, erase-path safety, destruction purge, suspension
// preservation).
//
// The typed stores are exercised with the built-in component types (Health,
// Knowledge, Faction, Combatant, Interactable) and, in the EF-001 suite,
// with arbitrary named types (HealthComponent) — proving lazy store
// creation works for any named type.
package entity_test

import (
	"reflect"
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
)

// AC-1: SetComponent stores a component on an Active entity and subsequent
// GetComponent returns the same value.
func TestSetGet_RoundTrip(t *testing.T) {
	store, _, _ := newTestStore()
	id, err := store.CreateEntity(entity.EntityTypeCharacter, "Player", matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	want := entity.Health{Current: 100, Max: 100}
	if err := entity.SetComponent(store, id, want); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	got, ok := entity.GetComponent[entity.Health](store, id)
	if !ok {
		t.Fatal("expected found=true for a stored component (AC-2)")
	}
	if got != want {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}
}

// AC-3: GetComponent for an entity without the component returns found=false
// with the zero value.
func TestGet_MissingComponentReturnsZeroFalse(t *testing.T) {
	store, _, _ := newTestStore()
	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	got, ok := entity.GetComponent[entity.Knowledge](store, id)
	if ok {
		t.Fatal("expected found=false for a missing component")
	}
	if !reflect.DeepEqual(got, entity.Knowledge{}) {
		t.Fatalf("expected zero value, got %+v", got)
	}
}

// AC-4: RemoveComponent fires component.removed with the removed value and
// subsequent reads return not-found.
func TestRemove_FiresEventAndClearsComponent(t *testing.T) {
	store, _, bus := newTestStore()
	recorder := &eventRecorder{}
	recorder.subscribe(bus, entity.EventComponentRemoved)

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponent(store, id, entity.Health{Current: 7, Max: 10}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	if err := entity.RemoveComponent[entity.Health](store, id); err != nil {
		t.Fatalf("RemoveComponent returned error: %v", err)
	}

	// All subscribed systems received component.removed, synchronously.
	if got := recorder.count(entity.EventComponentRemoved); got != 1 {
		t.Fatalf("received %d component.removed events, want 1", got)
	}
	ev, ok := recorder.find(entity.EventComponentRemoved)
	if !ok {
		t.Fatal("no component.removed event recorded")
	}
	data, ok := ev.Data.(entity.ComponentEventData)
	if !ok {
		t.Fatalf("payload is not ComponentEventData: %T", ev.Data)
	}
	if data.EntityID != id {
		t.Fatalf("event EntityID %d != removed id %d", data.EntityID, id)
	}
	if data.ComponentType != "Health" {
		t.Fatalf("event ComponentType %q != %q", data.ComponentType, "Health")
	}
	removed, ok := data.OldValue.(entity.Health)
	if !ok || removed.Current != 7 || removed.Max != 10 {
		t.Fatalf("event OldValue %+v does not carry the removed component", data.OldValue)
	}
	if data.NewValue != nil {
		t.Fatal("component.removed must not carry NewValue")
	}

	// Subsequent reads return not-found.
	if _, ok := entity.GetComponent[entity.Health](store, id); ok {
		t.Fatal("component still readable after removal")
	}
}

// AC-5: SetComponent replacing an existing component fires component.changed
// with old_value and new_value.
func TestSet_ExistingComponentFiresChangedWithOldNew(t *testing.T) {
	store, _, bus := newTestStore()
	changed := &eventRecorder{}
	changed.subscribe(bus, entity.EventComponentChanged)
	added := &eventRecorder{}
	added.subscribe(bus, entity.EventComponentAdded)

	id, err := store.CreateEntity(entity.EntityTypeCharacter, "Player", matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	first := entity.Health{Current: 10, Max: 100}
	if err := entity.SetComponent(store, id, first); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}
	if err := entity.SetComponent(store, id, entity.Health{Current: 5, Max: 100}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	if got := changed.count(entity.EventComponentChanged); got != 1 {
		t.Fatalf("received %d component.changed events, want 1", got)
	}
	ev, ok := changed.find(entity.EventComponentChanged)
	if !ok {
		t.Fatal("no component.changed event recorded")
	}
	data, ok := ev.Data.(entity.ComponentEventData)
	if !ok {
		t.Fatalf("payload is not ComponentEventData: %T", ev.Data)
	}
	if data.EntityID != id || data.ComponentType != "Health" {
		t.Fatalf("changed event targets (%d, %q), want (%d, %q)",
			data.EntityID, data.ComponentType, id, "Health")
	}
	oldHealth, ok := data.OldValue.(entity.Health)
	if !ok || oldHealth != first {
		t.Fatalf("changed OldValue %+v, want %+v", data.OldValue, first)
	}
	newHealth, ok := data.NewValue.(entity.Health)
	if !ok || newHealth.Current != 5 {
		t.Fatalf("changed NewValue %+v, want Current=5", data.NewValue)
	}

	// The replacement fired changed, not a second added.
	if got := added.count(entity.EventComponentAdded); got != 1 {
		t.Fatalf("received %d component.added events, want 1", got)
	}
}

// First attachment fires component.added with the new value and no changed
// event.
func TestSet_FirstAttachmentFiresAdded(t *testing.T) {
	store, _, bus := newTestStore()
	added := &eventRecorder{}
	added.subscribe(bus, entity.EventComponentAdded)
	changed := &eventRecorder{}
	changed.subscribe(bus, entity.EventComponentChanged)

	id, err := store.CreateEntity(entity.EntityTypeProp, "Chest", matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	want := entity.Interactable{InteractionType: entity.InteractionOpen, PromptText: "Open", Range: 2}
	if err := entity.SetComponent(store, id, want); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	if got := added.count(entity.EventComponentAdded); got != 1 {
		t.Fatalf("received %d component.added events, want 1", got)
	}
	ev, ok := added.find(entity.EventComponentAdded)
	if !ok {
		t.Fatal("no component.added event recorded")
	}
	data, ok := ev.Data.(entity.ComponentEventData)
	if !ok {
		t.Fatalf("payload is not ComponentEventData: %T", ev.Data)
	}
	if data.OldValue != nil {
		t.Fatal("component.added must not carry OldValue")
	}
	got, ok := data.NewValue.(entity.Interactable)
	if !ok || got != want {
		t.Fatalf("added NewValue %+v, want %+v", data.NewValue, want)
	}
	if got := changed.count(entity.EventComponentChanged); got != 0 {
		t.Fatalf("received %d component.changed events on first attachment, want 0", got)
	}
}

// Removing a component the entity does not have is a no-op: nil error, no
// event.
func TestRemove_AbsentComponentIsNoOp(t *testing.T) {
	store, _, bus := newTestStore()
	recorder := &eventRecorder{}
	recorder.subscribe(bus, entity.EventComponentRemoved)

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	if err := entity.RemoveComponent[entity.Faction](store, id); err != nil {
		t.Fatalf("removing an absent component must not error, got %v", err)
	}
	if got := recorder.count(entity.EventComponentRemoved); got != 0 {
		t.Fatalf("received %d component.removed events for an absent component, want 0", got)
	}
}

// Set and Remove reject destroyed and unknown entities; Get returns
// not-found (GDD edge case: "entity destroyed while a system holds its
// GameEntityID").
func TestComponentOps_OnDestroyedEntity(t *testing.T) {
	store, _, _ := newTestStore()
	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponent(store, id, entity.Health{Current: 5, Max: 5}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}
	if err := store.DestroyEntity(id); err != nil {
		t.Fatalf("DestroyEntity returned error: %v", err)
	}

	if err := entity.SetComponent(store, id, entity.Health{Current: 1, Max: 1}); err == nil {
		t.Fatal("SetComponent on a destroyed entity must error")
	}
	if err := entity.RemoveComponent[entity.Health](store, id); err == nil {
		t.Fatal("RemoveComponent on a destroyed entity must error")
	}
	if _, ok := entity.GetComponent[entity.Health](store, id); ok {
		t.Fatal("GetComponent on a destroyed entity must be not-found")
	}

	// Unknown IDs behave identically.
	if err := entity.SetComponent(store, 9999, entity.Health{}); err == nil {
		t.Fatal("SetComponent on an unknown entity must error")
	}
	if err := entity.RemoveComponent[entity.Health](store, 9999); err == nil {
		t.Fatal("RemoveComponent on an unknown entity must error")
	}
	if _, ok := entity.GetComponent[entity.Health](store, 9999); ok {
		t.Fatal("GetComponent on an unknown entity must be not-found")
	}
}

// Each entity's component data is independent of other entities.
func TestStores_IndependentPerEntity(t *testing.T) {
	store, _, _ := newTestStore()
	idA, err := store.CreateEntity(entity.EntityTypeNPC, "A", matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	idB, err := store.CreateEntity(entity.EntityTypeNPC, "B", matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponent(store, idA, entity.Health{Current: 42, Max: 100}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	// Entity B has no Health of its own.
	if _, ok := entity.GetComponent[entity.Health](store, idB); ok {
		t.Fatal("entity B must not see entity A's component")
	}
	// Writing entity B does not affect entity A.
	if err := entity.SetComponent(store, idB, entity.Health{Current: 7, Max: 100}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}
	gotA, ok := entity.GetComponent[entity.Health](store, idA)
	if !ok || gotA.Current != 42 {
		t.Fatalf("entity A component changed to %+v after writing entity B", gotA)
	}
	gotB, _ := entity.GetComponent[entity.Health](store, idB)
	if gotB.Current != 7 {
		t.Fatalf("entity B component is %+v, want Current=7", gotB)
	}
}

// QueryEntities filters by required component names backed by the typed
// stores (CR-3: "All entities with Health AND Combatant within ...").
func TestQueryEntities_ComponentFilter(t *testing.T) {
	store, _, _ := newTestStore()
	withHealth, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	without, err := store.CreateEntity(entity.EntityTypeProp, "Barrel", matrix.NewVec3(2, 2, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponent(store, withHealth, entity.Health{Current: 10, Max: 10}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	results := store.QueryEntities(entity.Query{Components: []string{"Health"}})
	if !contains(results, withHealth) {
		t.Fatal("entity with Health missing from query result")
	}
	if contains(results, without) {
		t.Fatal("entity without Health returned by query")
	}

	// Multiple required components: all must be present.
	if err := entity.SetComponent(store, withHealth, entity.Combatant{ActionPoints: 4}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}
	both := store.QueryEntities(entity.Query{Components: []string{"Health", "Combatant"}})
	if !contains(both, withHealth) {
		t.Fatal("entity with Health+Combatant missing from combined query")
	}
	if contains(both, without) {
		t.Fatal("entity without components returned by combined query")
	}
	if got := len(store.QueryEntities(entity.Query{Components: []string{"Knowledge"}})); got != 0 {
		t.Fatalf("query for Knowledge returned %d entities, want 0", got)
	}
}

// Erased path: GetComponentData/SetComponentData/RemoveComponentData remain
// functional for string-keyed access, reject type mismatches without
// corrupting stored data, and reject unknown type names.
func TestErasedPath_TypeSafetyAndRemoval(t *testing.T) {
	store, _, _ := newTestStore()
	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponent(store, id, entity.Health{Current: 1, Max: 10}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	v, ok := store.GetComponentData(id, "Health")
	if !ok {
		t.Fatal("erased get found=false for a stored component")
	}
	if h, ok := v.(entity.Health); !ok || h.Current != 1 {
		t.Fatalf("erased get value %+v, want Current=1", v)
	}

	// A type mismatch on set is rejected and leaves the stored value intact.
	if err := store.SetComponentData(id, "Health", entity.Faction{FactionID: "thornwell"}); err == nil {
		t.Fatal("SetComponentData with a wrong component type must error")
	}
	if h, _ := entity.GetComponent[entity.Health](store, id); h.Current != 1 {
		t.Fatalf("failed erased set corrupted the store: %+v", h)
	}

	// Unknown type name: get returns not-found, set errors (the erased path
	// cannot create a typed store without the Go type).
	if _, ok := store.GetComponentData(id, "NoSuchComponent"); ok {
		t.Fatal("erased get for an unknown type must be not-found")
	}
	if err := store.SetComponentData(id, "NoSuchComponent", 42); err == nil {
		t.Fatal("SetComponentData for an unknown type must error")
	}

	// Erased removal mirrors the generic path, including idempotency.
	if err := store.RemoveComponentData(id, "Health"); err != nil {
		t.Fatalf("RemoveComponentData returned error: %v", err)
	}
	if _, ok := store.GetComponentData(id, "Health"); ok {
		t.Fatal("component still present after erased removal")
	}
	if err := store.RemoveComponentData(id, "Health"); err != nil {
		t.Fatalf("removing an absent component via erased path must not error, got %v", err)
	}
}

// Destroying an entity purges its data from every typed store; reads return
// not-found afterwards.
func TestDestroy_PurgesComponentStores(t *testing.T) {
	store, _, _ := newTestStore()
	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponent(store, id, entity.Health{Current: 5, Max: 5}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}
	if err := entity.SetComponent(store, id, entity.Faction{FactionID: "thornwell"}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	if err := store.DestroyEntity(id); err != nil {
		t.Fatalf("DestroyEntity returned error: %v", err)
	}
	if _, ok := entity.GetComponent[entity.Health](store, id); ok {
		t.Fatal("destroyed entity still returns Health")
	}
	if _, ok := entity.GetComponent[entity.Faction](store, id); ok {
		t.Fatal("destroyed entity still returns Faction")
	}
}

// Component data is preserved across suspension and remains writable while
// suspended (GDD CR-4: "Position and component state are preserved").
func TestSuspension_PreservesAndAcceptsWrites(t *testing.T) {
	store, _, _ := newTestStore()
	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponent(store, id, entity.Health{Current: 30, Max: 100}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	store.UpdateSuspension(matrix.NewVec3(200, 200, 0)) // suspend
	if err := entity.SetComponent(store, id, entity.Health{Current: 29, Max: 100}); err != nil {
		t.Fatalf("SetComponent while suspended returned error: %v", err)
	}
	store.UpdateSuspension(matrix.NewVec3(15, 20, 0)) // reactivate

	hp, ok := entity.GetComponent[entity.Health](store, id)
	if !ok {
		t.Fatal("Health lost across suspension")
	}
	if hp.Current != 29 || hp.Max != 100 {
		t.Fatalf("Health changed across suspension: %+v", hp)
	}
}
