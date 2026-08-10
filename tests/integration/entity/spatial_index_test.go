// Integration tests for Story EF-003: Spatial Index.
//
// Evidence path: production/epics/entity-framework/story-003-spatial-index.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria map to test functions 1:1. Go only executes
// Test-prefixed functions, so each story name ("test_...") is exported with
// the Go prefix:
//
//	AC-1  test_range_query_entities_within_radius      -> TestRangeQueryEntitiesWithinRadius
//	AC-2  test_position_update_migrates_bucket         -> TestPositionUpdateMigratesBucket
//	AC-3  test_multi_bucket_query_aggregates_correctly -> TestMultiBucketQueryAggregatesCorrectly
//	AC-4  test_component_filter_within_range           -> TestComponentFilterWithinRange
//	AC-5  test_bucket_overflow_degrades_to_linear_scan -> TestBucketOverflowDegradesToLinearScan
//
// The Kaiju host is the same mock used by the unit tests (tests/unit/entity).
// Go cannot import test helpers across packages, so newTestStore, contains,
// and eventRecorder are mirrored here from that package.
package entity_test

import (
	"fmt"
	"testing"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"

	"marrow-of-empire/entity"
)

// mockKaijuHost is a test double for entity.KaijuHost (mirrored from
// tests/unit/entity). It registers engine entities by EntityId and records
// native destroy calls.
type mockKaijuHost struct {
	entities        map[engine.EntityId]*engine.Entity
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

// eventRecorder collects every event it is subscribed to (mirrored from
// tests/unit/entity; kept for parity with the shared test helpers).
type eventRecorder struct {
	received []entity.Event
}

func (r *eventRecorder) subscribe(bus *entity.EventBusImpl, eventType entity.EventType) {
	bus.Subscribe(eventType, 5, r.handle)
}

func (r *eventRecorder) handle(ev entity.Event) { r.received = append(r.received, ev) }

func (r *eventRecorder) count(eventType entity.EventType) int {
	n := 0
	for _, ev := range r.received {
		if ev.Type == eventType {
			n++
		}
	}
	return n
}

// newTestStore wires a store to a mock host and a fresh event bus (mirrored
// from tests/unit/entity). Config zero values fall back to the GDD defaults.
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

// AC-1: QueryEntities with a position and range returns only entities within
// range. 100 entities are scattered across several 32-unit buckets; the
// entity at (10,20) is returned, the entity at (50,80) is not, and every
// returned entity is verified to be within range.
func TestRangeQueryEntitiesWithinRadius(t *testing.T) {
	store, host, _ := newTestStore()

	// 100 entities at deterministic scattered positions: a 10x10 grid
	// spanning x in [0,63], y in [0,117] — several buckets.
	for i := 0; i < 100; i++ {
		pos := matrix.NewVec3(float64(i%10)*7, float64((i/10)%10)*13, 0)
		if _, err := store.CreateEntity(entity.EntityTypeProp, fmt.Sprintf("marker_%d", i), pos); err != nil {
			t.Fatalf("CreateEntity %d returned error: %v", i, err)
		}
	}
	near, err := store.CreateEntity(entity.EntityTypeNPC, "near", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity near returned error: %v", err)
	}
	far, err := store.CreateEntity(entity.EntityTypeNPC, "far", matrix.NewVec3(50, 80, 0))
	if err != nil {
		t.Fatalf("CreateEntity far returned error: %v", err)
	}

	results := store.QueryEntities(entity.Query{
		Position: matrix.NewVec3(10, 20, 0),
		Range:    15,
	})

	if !contains(results, near) {
		t.Fatal("entity at (10,20) not returned by range query")
	}
	if contains(results, far) {
		t.Fatal("entity at (50,80) returned despite being outside range")
	}
	center := matrix.NewVec3(10, 20, 0)
	for _, id := range results {
		kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", id)))
		if kaiju == nil {
			t.Fatalf("entity %d not registered in host", id)
		}
		if d := kaiju.Transform.Position().Distance(center); d > 15 {
			t.Fatalf("entity %d at distance %v returned despite range 15", id, d)
		}
	}
}

// AC-2: an entity moved from one bucket to another is indexed in the new
// bucket. After the move, queries near the new position return the entity and
// queries near the old position do not; same-cell moves keep working; unknown
// and destroyed IDs are rejected.
func TestPositionUpdateMigratesBucket(t *testing.T) {
	store, host, _ := newTestStore()

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Wanderer", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	queryAt := func(pos matrix.Vec3) []entity.GameEntityID {
		return store.QueryEntities(entity.Query{Position: pos, Range: 5})
	}

	if !contains(queryAt(matrix.NewVec3(10, 20, 0)), id) {
		t.Fatal("entity not found near its creation position")
	}

	// A system moves the Kaiju Transform (authoritative source), then syncs
	// the spatial index with UpdatePosition. (10,20) is in bucket (0,0);
	// (100,100) is in bucket (3,3).
	kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", id)))
	if kaiju == nil {
		t.Fatalf("entity %d not registered in host", id)
	}
	kaiju.Transform.SetPosition(matrix.NewVec3(100, 100, 0))
	if err := store.UpdatePosition(id, matrix.NewVec3(100, 100, 0)); err != nil {
		t.Fatalf("UpdatePosition returned error: %v", err)
	}

	if contains(queryAt(matrix.NewVec3(10, 20, 0)), id) {
		t.Fatal("entity still returned near its old position after migration")
	}
	if !contains(queryAt(matrix.NewVec3(100, 100, 0)), id) {
		t.Fatal("entity not returned near its new position after migration")
	}

	// Move within the same bucket (11,21 is still bucket (0,0)): the index
	// must keep serving the entity at its new location.
	kaiju.Transform.SetPosition(matrix.NewVec3(11, 21, 0))
	if err := store.UpdatePosition(id, matrix.NewVec3(11, 21, 0)); err != nil {
		t.Fatalf("UpdatePosition (same bucket) returned error: %v", err)
	}
	if !contains(queryAt(matrix.NewVec3(11, 21, 0)), id) {
		t.Fatal("entity not found after same-bucket move")
	}

	// Unknown and destroyed IDs are rejected.
	if err := store.UpdatePosition(9999, matrix.NewVec3(0, 0, 0)); err == nil {
		t.Fatal("expected error updating unknown entity")
	}
	if err := store.DestroyEntity(id); err != nil {
		t.Fatalf("DestroyEntity returned error: %v", err)
	}
	if err := store.UpdatePosition(id, matrix.NewVec3(0, 0, 0)); err == nil {
		t.Fatal("expected error updating destroyed entity")
	}
}

// AC-3: a query spanning bucket boundaries aggregates candidates from all
// overlapping buckets, and the exact-distance filter still excludes entities
// inside the query bounding box but outside the search circle.
func TestMultiBucketQueryAggregatesCorrectly(t *testing.T) {
	store, _, _ := newTestStore()

	// One entity in each quadrant around the (32,32) bucket boundary:
	// buckets (0,0), (1,0), (0,1), (1,1).
	positions := []matrix.Vec3{
		matrix.NewVec3(5, 5, 0), matrix.NewVec3(35, 5, 0),
		matrix.NewVec3(5, 35, 0), matrix.NewVec3(35, 35, 0),
	}
	ids := make([]entity.GameEntityID, 0, len(positions))
	for _, pos := range positions {
		id, err := store.CreateEntity(entity.EntityTypeProp, "corner", pos)
		if err != nil {
			t.Fatalf("CreateEntity returned error: %v", err)
		}
		ids = append(ids, id)
	}
	// (44,44) lies in bucket (1,1) — inside the query's bounding box — but
	// is ~33.9 units from the center, outside the range-25 circle.
	outside, err := store.CreateEntity(entity.EntityTypeProp, "outside", matrix.NewVec3(44, 44, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	results := store.QueryEntities(entity.Query{
		Position: matrix.NewVec3(20, 20, 0),
		Range:    25,
	})

	for _, id := range ids {
		if !contains(results, id) {
			t.Fatalf("entity %d from overlapping bucket not returned", id)
		}
	}
	if contains(results, outside) {
		t.Fatal("entity outside the search circle returned despite exact-distance filter")
	}
	if len(results) != len(ids) {
		t.Fatalf("expected %d results, got %d: %v", len(ids), len(results), results)
	}
}

// AC-4: component filtering applies within the spatial range. Entity A has
// Health + Combatant, entity B has Health only; querying "Health AND
// Combatant" within range returns only A, and out-of-range entities with
// matching components are excluded.
func TestComponentFilterWithinRange(t *testing.T) {
	store, _, _ := newTestStore()

	a, err := store.CreateEntity(entity.EntityTypeNPC, "A", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity A returned error: %v", err)
	}
	b, err := store.CreateEntity(entity.EntityTypeNPC, "B", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity B returned error: %v", err)
	}
	c, err := store.CreateEntity(entity.EntityTypeNPC, "C", matrix.NewVec3(10, 20, 0))
	if err != nil {
		t.Fatalf("CreateEntity C returned error: %v", err)
	}
	d, err := store.CreateEntity(entity.EntityTypeNPC, "D", matrix.NewVec3(200, 200, 0))
	if err != nil {
		t.Fatalf("CreateEntity D returned error: %v", err)
	}

	for id, set := range map[entity.GameEntityID]func() error{
		a: func() error {
			if err := entity.SetComponent(store, a, entity.Health{Current: 10, Max: 10}); err != nil {
				return err
			}
			return entity.SetComponent(store, a, entity.Combatant{ActionPoints: 4})
		},
		b: func() error { return entity.SetComponent(store, b, entity.Health{Current: 6, Max: 10}) },
		c: func() error { return entity.SetComponent(store, c, entity.Combatant{ActionPoints: 6}) },
		d: func() error {
			if err := entity.SetComponent(store, d, entity.Health{Current: 10, Max: 10}); err != nil {
				return err
			}
			return entity.SetComponent(store, d, entity.Combatant{ActionPoints: 4})
		},
	} {
		if err := set(); err != nil {
			t.Fatalf("SetComponent for entity %d returned error: %v", id, err)
		}
	}

	center := matrix.NewVec3(10, 20, 0)
	both := store.QueryEntities(entity.Query{
		Components: []string{"Health", "Combatant"},
		Position:   center,
		Range:      10,
	})
	if len(both) != 1 || !contains(both, a) {
		t.Fatalf("Health AND Combatant query returned %v, want only entity %d", both, a)
	}

	healthOnly := store.QueryEntities(entity.Query{
		Components: []string{"Health"},
		Position:   center,
		Range:      10,
	})
	if !contains(healthOnly, a) || !contains(healthOnly, b) || contains(healthOnly, c) {
		t.Fatalf("Health query returned %v, want entities %d and %d only", healthOnly, a, b)
	}

	// Out-of-range entity D has Health + Combatant but sits 268+ units away.
	both = store.QueryEntities(entity.Query{
		Components: []string{"Health", "Combatant"},
		Position:   center,
		Range:      10,
	})
	if contains(both, d) {
		t.Fatal("out-of-range entity with matching components returned")
	}
}

// AC-5: a bucket holding more than the overflow threshold (>500 entities)
// degrades the range query to a linear scan with a warning. Results stay
// correct (all 501 returned), the degradation is observable via
// SpatialIndexDegradationCount, and queries that never touch the overloaded
// bucket do not degrade.
func TestBucketOverflowDegradesToLinearScan(t *testing.T) {
	host := newMockHost()
	bus := entity.NewEventBus()
	// Raised region cap keeps the AC-5 entity flood out of the
	// max_entities_per_region warning path.
	store := entity.NewEntityStore(host, bus, entity.EntityStoreConfig{
		MaxEntitiesPerRegion: 1000,
	})

	const bucketPopulation = 501 // > DefaultSpatialOverflowThreshold (500)
	for i := 0; i < bucketPopulation; i++ {
		if _, err := store.CreateEntity(entity.EntityTypeProp, "crowd", matrix.NewVec3(10, 10, 0)); err != nil {
			t.Fatalf("CreateEntity %d returned error: %v", i, err)
		}
	}
	if got := store.SpatialIndexDegradationCount(); got != 0 {
		t.Fatalf("expected no degradation before overflow query, got %d", got)
	}

	results := store.QueryEntities(entity.Query{
		Position: matrix.NewVec3(10, 10, 0),
		Range:    5,
	})
	if len(results) != bucketPopulation {
		t.Fatalf("degraded query returned %d entities, want %d", len(results), bucketPopulation)
	}
	if got := store.SpatialIndexDegradationCount(); got != 1 {
		t.Fatalf("expected exactly 1 degraded query, got %d", got)
	}

	// A query far from the overloaded bucket must not degrade.
	far := store.QueryEntities(entity.Query{
		Position: matrix.NewVec3(1000, 1000, 0),
		Range:    5,
	})
	if len(far) != 0 {
		t.Fatalf("query far from all entities returned %d entities, want 0", len(far))
	}
	if got := store.SpatialIndexDegradationCount(); got != 1 {
		t.Fatalf("non-overlapping query degraded the index: count %d, want 1", got)
	}
}
