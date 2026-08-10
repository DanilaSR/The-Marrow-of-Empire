// Unit tests for Story EF-001: Entity Lifecycle.
//
// Evidence path: production/epics/entity-framework/story-001-entity-lifecycle.md
// Each test function maps to one acceptance criterion (AC-1 .. AC-5).
//
// The Kaiju host is a mock (mockKaijuHost): a real engine.Host requires a
// window and Vulkan context, which unit tests must not depend on. The mock
// records EntityById lookups and native destroy calls so the tests can verify
// the GameEntityID -> EntityId mapping lifecycle end to end.
package entity_test

import (
	"fmt"
	"testing"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"

	"marrow-of-empire/entity"
)

// mockKaijuHost is a test double for entity.KaijuHost. It registers engine
// entities by EntityId, records every EntityById lookup, and records which
// entities were passed to the native destroy path.
type mockKaijuHost struct {
	entities       map[engine.EntityId]*engine.Entity
	entityByIdCalls []engine.EntityId
	destroyed       []engine.EntityId
	workGroup       concurrent.WorkGroup
}

func (m *mockKaijuHost) EntityById(id engine.EntityId) *engine.Entity {
	m.entityByIdCalls = append(m.entityByIdCalls, id)
	return m.entities[id]
}

func (m *mockKaijuHost) SetEntityId(entity *engine.Entity, id engine.EntityId) bool {
	if id == "" {
		return false
	}
	if _, exists := m.entities[id]; exists {
		return false
	}
	m.entities[id] = entity
	return true
}

func (m *mockKaijuHost) DestroyEntity(entity *engine.Entity) {
	for id, e := range m.entities {
		if e == entity {
			delete(m.entities, id)
			m.destroyed = append(m.destroyed, id)
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

// eventRecorder collects every event it is subscribed to for later assertion.
type eventRecorder struct {
	received []entity.Event
}

func (r *eventRecorder) subscribe(bus *entity.EventBusImpl, eventType entity.EventType) {
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

// HealthComponent is a named component type used to verify that component
// data survives suspension and reactivation. Named types are required by the
// generic component key resolution.
type HealthComponent struct {
	Current int
	Max     int
}

// newTestStore wires a store to a mock host and a fresh event bus.
// The config zero values fall back to the GDD defaults:
// suspension_distance = 100, max_entities_per_region = 500.
func newTestStore() (*entity.EntityStoreImpl, *mockKaijuHost, *entity.EventBusImpl) {
	host := newMockHost()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(host, bus, entity.EntityStoreConfig{})
	return store, host, bus
}

func contains(ids []entity.GameEntityID, want entity.GameEntityID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func samePosition(a, b matrix.Vec3) bool {
	return a.Distance(b) == 0
}

// AC-1: CreateEntity returns a valid GameEntityID > 0, the Kaiju entity
// exists with EntityId "game_{ID}", and the entity state is Active.
func TestCreateEntity_MapsToKaijuEntityId(t *testing.T) {
	store, host, bus := newTestStore()
	recorder := &eventRecorder{}
	recorder.subscribe(bus, entity.EventEntityCreated)

	id, err := store.CreateEntity(entity.EntityTypeCharacter, "Player", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if id == 0 {
		t.Fatal("expected GameEntityID > 0, got 0")
	}

	// Kaiju entity exists with the mapped EntityId "game_{ID}".
	kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", id)))
	if kaiju == nil {
		t.Fatalf("kaiju entity %q not registered in host", fmt.Sprintf("game_%d", id))
	}
	if !kaiju.IsActive() {
		t.Fatal("kaiju entity should be active after creation")
	}
	if !samePosition(kaiju.Transform.Position(), matrix.NewVec3(10, 20, 0)) {
		t.Fatalf("entity position not applied, got %v", kaiju.Transform.Position())
	}

	// Game-layer state is Active (Pending -> Active transition completed).
	st, ok := store.EntityState(id)
	if !ok {
		t.Fatalf("no state recorded for entity %d", id)
	}
	if st != entity.EntityStateActive {
		t.Fatalf("expected state Active, got %s", st)
	}

	// entity.created dispatched with the full payload.
	ev, ok := recorder.find(entity.EventEntityCreated)
	if !ok {
		t.Fatal("subscribed system did not receive entity.created")
	}
	data, ok := ev.Data.(entity.EntityEventData)
	if !ok {
		t.Fatalf("event payload is not EntityEventData: %T", ev.Data)
	}
	if data.EntityID != id {
		t.Fatalf("event EntityID %d != created id %d", data.EntityID, id)
	}
	if data.EntityType != entity.EntityTypeCharacter {
		t.Fatalf("event EntityType %s != Character", data.EntityType)
	}
	if data.DisplayName != "Player" {
		t.Fatalf("event DisplayName %q != %q", data.DisplayName, "Player")
	}
}

// AC-2: DestroyEntity transitions to Destroyed, on_entity_destroyed is
// dispatched to all subscribers, the ID is removed from QueryEntities, and
// the counter never reuses the ID.
func TestDestroyEntity_IdNeverReused(t *testing.T) {
	store, host, bus := newTestStore()
	recorderA := &eventRecorder{}
	recorderB := &eventRecorder{}
	recorderA.subscribe(bus, entity.EventEntityDestroyed)
	recorderB.subscribe(bus, entity.EventEntityDestroyed)

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(5, 5, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if !contains(store.QueryEntities(entity.Query{}), id) {
		t.Fatal("entity should be queryable before destruction")
	}

	if err := store.DestroyEntity(id); err != nil {
		t.Fatalf("DestroyEntity returned error: %v", err)
	}

	// State is Destroyed.
	st, ok := store.EntityState(id)
	if !ok {
		t.Fatalf("destroyed entity %d lost its state record", id)
	}
	if st != entity.EntityStateDestroyed {
		t.Fatalf("expected state Destroyed, got %s", st)
	}

	// All subscribed systems received entity.destroyed, synchronously.
	if got := recorderA.count(entity.EventEntityDestroyed); got != 1 {
		t.Fatalf("subscriber A received %d entity.destroyed events, want 1", got)
	}
	if got := recorderB.count(entity.EventEntityDestroyed); got != 1 {
		t.Fatalf("subscriber B received %d entity.destroyed events, want 1", got)
	}
	if ev, found := recorderA.find(entity.EventEntityDestroyed); found {
		data, ok := ev.Data.(entity.EntityEventData)
		if !ok || data.EntityID != id {
			t.Fatalf("entity.destroyed payload has wrong EntityID: %+v", ev.Data)
		}
		if !samePosition(data.Position, matrix.NewVec3(5, 5, 0)) {
			t.Fatalf("entity.destroyed payload has wrong position: %v (expected (5,5,0))", data.Position)
		}
	}

	// Mapping cleaned up: the GameEntityID -> EntityId mapping is removed and
	// the mock host no longer resolves the engine entity.
	if kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", id))); kaiju != nil {
		t.Fatal("mapping not cleaned up: engine entity still registered after destroy")
	}
	expectedDestroy := engine.EntityId(fmt.Sprintf("game_%d", id))
	found := false
	for _, d := range host.destroyed {
		if d == expectedDestroy {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("native destroy was not called for the engine entity")
	}

	// ID removed from QueryEntities.
	if contains(store.QueryEntities(entity.Query{}), id) {
		t.Fatal("destroyed entity still returned by QueryEntities")
	}

	// Double destroy is rejected (Destroyed -> any is an invalid transition).
	if err := store.DestroyEntity(id); err == nil {
		t.Fatal("expected error destroying an already-destroyed entity")
	}

	// Counter never reuses the destroyed ID.
	next, err := store.CreateEntity(entity.EntityTypeItem, "Key", matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if next <= id {
		t.Fatalf("counter reused destroyed ID: next %d <= destroyed %d", next, id)
	}
	if kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", next))); kaiju == nil {
		t.Fatalf("new entity %d not registered as %q", next, fmt.Sprintf("game_%d", next))
	}
}

// AC-3: distance > suspension_distance -> Suspended, component data
// preserved, hidden from update loops, still visible to spatial queries.
func TestSuspend_OnDistanceExceeded(t *testing.T) {
	store, host, bus := newTestStore()
	recorder := &eventRecorder{}
	recorder.subscribe(bus, entity.EventEntitySuspended)

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponent(store, id, HealthComponent{Current: 7, Max: 10}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	// Player far away: distance (200,200,0) - (10,20,0) ~= 280 > 100.
	store.UpdateSuspension(matrix.NewVec3(200, 200, 0))

	st, ok := store.EntityState(id)
	if !ok {
		t.Fatalf("no state recorded for entity %d", id)
	}
	if st != entity.EntityStateSuspended {
		t.Fatalf("expected state Suspended, got %s", st)
	}

	// Component data preserved across suspension.
	hp, ok := entity.GetComponent[HealthComponent](store, id)
	if !ok {
		t.Fatal("component lost during suspension")
	}
	if hp.Current != 7 || hp.Max != 10 {
		t.Fatalf("component data changed during suspension: %+v", hp)
	}

	// Hidden from update loops: engine entity deactivated, game state not Active.
	if kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", id))); kaiju == nil {
		t.Fatal("engine entity vanished during suspension")
	} else if kaiju.IsActive() {
		t.Fatal("engine entity still active while suspended")
	}

	// Still visible to spatial queries.
	results := store.QueryEntities(entity.Query{
		Position: matrix.NewVec3(10, 20, 0),
		Range:    5,
	})
	if !contains(results, id) {
		t.Fatal("suspended entity not returned by spatial query")
	}

	if got := recorder.count(entity.EventEntitySuspended); got != 1 {
		t.Fatalf("received %d entity.suspended events, want 1", got)
	}
}

// AC-4: player returns within suspension_distance -> Active,
// on_entity_reactivated fired, component data unchanged from suspension.
func TestReactivate_OnPlayerReturn(t *testing.T) {
	store, host, bus := newTestStore()
	recorder := &eventRecorder{}
	recorder.subscribe(bus, entity.EventEntityReactivated)

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponent(store, id, HealthComponent{Current: 3, Max: 10}); err != nil {
		t.Fatalf("SetComponent returned error: %v", err)
	}

	// Suspend first.
	store.UpdateSuspension(matrix.NewVec3(200, 200, 0))
	if st, _ := store.EntityState(id); st != entity.EntityStateSuspended {
		t.Fatalf("setup failed: expected Suspended, got %s", st)
	}

	// Player returns: distance (15,20,0) - (10,20,0) = 5 <= 100.
	store.UpdateSuspension(matrix.NewVec3(15, 20, 0))

	st, ok := store.EntityState(id)
	if !ok {
		t.Fatalf("no state recorded for entity %d", id)
	}
	if st != entity.EntityStateActive {
		t.Fatalf("expected state Active after reactivation, got %s", st)
	}

	if got := recorder.count(entity.EventEntityReactivated); got != 1 {
		t.Fatalf("received %d entity.reactivated events, want 1", got)
	}

	// Component data unchanged from suspension.
	hp, ok := entity.GetComponent[HealthComponent](store, id)
	if !ok {
		t.Fatal("component lost during reactivation")
	}
	if hp.Current != 3 || hp.Max != 10 {
		t.Fatalf("component data changed during reactivation: %+v", hp)
	}

	// Engine entity active again.
	if kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", id))); kaiju == nil {
		t.Fatal("engine entity vanished during reactivation")
	} else if !kaiju.IsActive() {
		t.Fatal("engine entity not active after reactivation")
	}

	// Repeated check while Active does not re-emit reactivated.
	store.UpdateSuspension(matrix.NewVec3(16, 20, 0))
	if got := recorder.count(entity.EventEntityReactivated); got != 1 {
		t.Fatalf("duplicate reactivation: received %d entity.reactivated events, want 1", got)
	}
}

// AC-5: every lifecycle transition dispatches events to subscribers in
// priority order (1 = highest), ties broken by registration order, all
// synchronously within the same call (same frame).
func TestLifecycleEvents_InPriorityOrder(t *testing.T) {
	store, _, bus := newTestStore()

	// Track the order in which test subscribers receive each event type.
	// The store's own priority-1 subscription (registered at construction)
	// runs first; these assertions verify the relative order of everything
	// registered afterwards: p1 (priority 1), p3a/p3b (priority 3, tie),
	// p5 (priority 5), p10 (priority 10).
	type tracker struct {
		eventType entity.EventType
		received  []string
	}
	trackers := make(map[entity.EventType]*tracker)
	for _, eventType := range []entity.EventType{
		entity.EventEntityCreated,
		entity.EventEntityDestroyed,
		entity.EventEntitySuspended,
		entity.EventEntityReactivated,
	} {
		tr := &tracker{eventType: eventType}
		trackers[eventType] = tr
		bus.Subscribe(eventType, 10, func(ev entity.Event) { tr.received = append(tr.received, "p10") })
		bus.Subscribe(eventType, 5, func(ev entity.Event) { tr.received = append(tr.received, "p5") })
		bus.Subscribe(eventType, 3, func(ev entity.Event) { tr.received = append(tr.received, "p3b") })
		bus.Subscribe(eventType, 3, func(ev entity.Event) { tr.received = append(tr.received, "p3a") })
		bus.Subscribe(eventType, 1, func(ev entity.Event) { tr.received = append(tr.received, "p1") })
	}
	assertOrder := func(eventType entity.EventType) {
		t.Helper()
		tr := trackers[eventType]
		want := []string{"p1", "p3b", "p3a", "p5", "p10"}
		if len(tr.received) != len(want) {
			t.Fatalf("%s: received %d events (%v), want %d (%v)",
				eventType, len(tr.received), tr.received, len(want), want)
		}
		for i := range want {
			if tr.received[i] != want[i] {
				t.Fatalf("%s: dispatch order %v, want %v", eventType, tr.received, want)
			}
		}
	}

	id, err := store.CreateEntity(entity.EntityTypeWildlife, "Fox", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	assertOrder(entity.EventEntityCreated)

	// Suspend (far), reactivate (near), destroy — each in the same frame.
	store.UpdateSuspension(matrix.NewVec3(500, 500, 0))
	assertOrder(entity.EventEntitySuspended)
	store.UpdateSuspension(matrix.NewVec3(11, 20, 0))
	assertOrder(entity.EventEntityReactivated)
	if err := store.DestroyEntity(id); err != nil {
		t.Fatalf("DestroyEntity returned error: %v", err)
	}
	assertOrder(entity.EventEntityDestroyed)
}
