// Integration tests for Story STAT-005: Stats Component Integration.
//
// Evidence path: production/epics/character-stats-progression/story-005-stats-component-integration.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_attach_component_via_ef          -> TestAttachStatsComponentToEntity
//	AC-2  test_serialize_preserves_stats        -> TestSerializePreservesStatsInBlob
//	AC-3  test_deserialize_restores_stats       -> TestDeserializeRestoresStatsAndRecomputesDerived
//	AC-4  test_level_up_emits_event             -> TestLevelUpEmitsEventWithNewDerivedStats
//
// Bridge edge cases beyond the story ACs: sub-threshold awards and LevelCap
// awards emit no event (TestAwardXPBelowThresholdEmitsNoEvent,
// TestAwardXPAtLevelCapEmitsNoEvent), and a multi-level award emits a single
// aggregate event carrying the final values
// (TestLevelUpMultiLevelJumpEmitsSingleAggregateEvent).
//
// The mock KaijuHost, eventRecorder, and newTestStore helpers are mirrored
// from tests/integration/knowledge (Go cannot import test helpers across
// packages). StatsComponent is registered as an EF component by the stats
// package init (component.go), so the typed accessors and the serialization
// pass treat it like any other component.
package stats_test

import (
	"bytes"
	"reflect"
	"testing"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"

	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// fixtureBaseline is a level-1 character used by AC-1, AC-2, and AC-4:
// STR 14, DEX 12, CON 14, INT 10, WIS 10, CHA 8. Derived stats at level 1
// (hand-computed per ADR-0006): MaxHealth 20+14*2+1*2 = 50, AP 4+1 = 5,
// Initiative 1+0 = 1, Evasion 11, CarryCapacity 14*10+30 = 170.
func fixtureBaseline() stats.StatsComponent {
	return stats.StatsComponent{
		Attributes: [stats.NumAttributes]int{14, 12, 14, 10, 10, 8}, // STR, DEX, CON, INT, WIS, CHA
		Level:      1,
		XP:         0,
	}
}

// fixtureLevel4 is a level-4 character with accumulated XP and an unspent
// attribute point, used by AC-3 to prove level, XP, and UnspentPoints survive
// the round-trip. XP 1800 is exactly TotalXPForLevel(4) = 150*4*3. Derived
// stats (hand-computed): MaxHealth 20+12*2+4*1 = 48, AP 4+3 = 7,
// Initiative 3+(-1) = 2, Evasion 13, CarryCapacity 10*10+30 = 130.
func fixtureLevel4() stats.StatsComponent {
	return stats.StatsComponent{
		Attributes:    [stats.NumAttributes]int{10, 16, 12, 10, 8, 10}, // STR, DEX, CON, INT, WIS, CHA
		Level:         4,
		XP:            1800,
		UnspentPoints: 1, // level 4 grants one attribute point (STAT-003 AC-4)
	}
}

// mockKaijuHost is a test double for entity.KaijuHost (mirrored from
// tests/integration/knowledge). It registers engine entities by EntityId and
// records native destroy calls.
type mockKaijuHost struct {
	entities  map[engine.EntityId]*engine.Entity
	destroyed []engine.EntityId
	workGroup concurrent.WorkGroup
}

func (m *mockKaijuHost) EntityById(id engine.EntityId) *engine.Entity {
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
// tests/integration/knowledge).
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

// firstEventOf returns the first recorded event of the given type.
func firstEventOf(r *eventRecorder, eventType entity.EventType) (entity.Event, bool) {
	for _, ev := range r.received {
		if ev.Type == eventType {
			return ev, true
		}
	}
	return entity.Event{}, false
}

// newTestStore wires a store to a mock host and a fresh event bus (mirrored
// from tests/integration/knowledge). Config zero values fall back to the GDD
// defaults.
func newTestStore() (*entity.EntityStoreImpl, *mockKaijuHost, *entity.EventBusImpl) {
	host := newMockHost()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(host, bus, entity.EntityStoreConfig{})
	return store, host, bus
}

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

// TestAttachStatsComponentToEntity is AC-1: GIVEN a character entity, WHEN
// the Stats component is attached via the Entity Framework, THEN all six
// attributes and derived stats are accessible via GetComponent[StatsComponent].
// It also covers component.added payload, retrieval isolation across entities,
// and the type-erased GetComponentData path.
func TestAttachStatsComponentToEntity(t *testing.T) {
	store, _, bus := newTestStore()
	want := fixtureBaseline()
	id := createTestEntity(t, store, entity.EntityTypeCharacter, "Elena", matrix.NewVec3(5, 5, 0))

	recorder := &eventRecorder{}
	recorder.subscribe(bus, entity.EventComponentAdded)

	if err := entity.SetComponent(store, id, want); err != nil {
		t.Fatalf("SetComponent error: %v", err)
	}

	got, ok := entity.GetComponent[stats.StatsComponent](store, id)
	if !ok {
		t.Fatal("GetComponent did not find the attached StatsComponent")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("retrieved component mismatch:\n got %+v\nwant %+v", got, want)
	}

	// Derived stats are computed on demand from the retrieved component
	// (ADR-0006): hand-computed expected values for the fixture.
	if v := got.MaxHealth(); v != 50 {
		t.Errorf("MaxHealth = %d, want 50", v)
	}
	if v := got.AP(); v != 5 {
		t.Errorf("AP = %d, want 5", v)
	}
	if v := got.Initiative(); v != 1 {
		t.Errorf("Initiative = %d, want 1", v)
	}
	if v := got.Evasion(); v != 11 {
		t.Errorf("Evasion = %d, want 11", v)
	}
	if v := got.CarryCapacity(); v != 170 {
		t.Errorf("CarryCapacity = %d, want 170", v)
	}

	// component.added fires with the component as NewValue and the store's
	// type name for StatsComponent.
	ev, ok := firstEventOf(recorder, entity.EventComponentAdded)
	if !ok {
		t.Fatal("component.added did not fire on attachment")
	}
	data, ok := ev.Data.(entity.ComponentEventData)
	if !ok || data.EntityID != id || data.ComponentType != "StatsComponent" {
		t.Errorf("component.added payload = %+v, want entity %d, type StatsComponent", ev.Data, id)
	}
	if !reflect.DeepEqual(data.NewValue, want) {
		t.Errorf("component.added NewValue mismatch:\n got %+v\nwant %+v", data.NewValue, want)
	}
	if data.OldValue != nil {
		t.Errorf("component.added OldValue = %+v, want nil", data.OldValue)
	}

	// An entity without a stats component reports (zero, false) on both
	// access paths.
	bareID := createTestEntity(t, store, entity.EntityTypeProp, "Bare", matrix.NewVec3(6, 6, 0))
	if _, ok := entity.GetComponent[stats.StatsComponent](store, bareID); ok {
		t.Error("bare entity reported a StatsComponent")
	}
	if got, ok := store.GetComponentData(bareID, "StatsComponent"); ok || got != nil {
		t.Errorf("bare entity erased lookup = %#v ok=%v, want nil/false", got, ok)
	}

	// The erased path returns the same value as the typed path.
	raw, ok := store.GetComponentData(id, "StatsComponent")
	if !ok {
		t.Fatal("erased GetComponentData did not find Elena's stats")
	}
	typed, ok := raw.(stats.StatsComponent)
	if !ok || !reflect.DeepEqual(typed, want) {
		t.Errorf("erased lookup mismatch:\n got %#v\nwant %+v", raw, want)
	}
}

// TestSerializePreservesStatsInBlob is AC-2: GIVEN a Save operation, WHEN
// the Stats component is serialized via pod, THEN all attributes, level, and
// XP are preserved in the save blob. The blob is non-empty, differs from a
// control store without stats (proving the stats store contributes bytes,
// not just the entity table), and a fresh store deserialized from it
// recovers the exact component.
func TestSerializePreservesStatsInBlob(t *testing.T) {
	source, _, _ := newTestStore()
	want := fixtureBaseline()
	id := createTestEntity(t, source, entity.EntityTypeCharacter, "Elena", matrix.NewVec3(5, 5, 0))
	mustSetComponent(t, source, id, want)

	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}
	if len(blob) == 0 {
		t.Fatal("Serialize returned an empty blob for a store with a stats component")
	}

	// Control: an identical entity table without a stats component must
	// produce a different blob — the difference is the encoded stats store.
	// (Each store has its own GameEntityID counter, so both stores create
	// entity id 1; the blobs must still differ.)
	control, _, _ := newTestStore()
	createTestEntity(t, control, entity.EntityTypeCharacter, "Elena", matrix.NewVec3(5, 5, 0))
	controlBlob, err := control.Serialize()
	if err != nil {
		t.Fatalf("control Serialize error: %v", err)
	}
	if bytes.Equal(blob, controlBlob) {
		t.Error("blob with a StatsComponent equals the control blob without one; stats data is not in the blob")
	}

	// The blob decodes: the stats data was in the blob all along.
	restored, _, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}
	got, ok := entity.GetComponent[stats.StatsComponent](restored, id)
	if !ok {
		t.Fatal("StatsComponent missing after deserializing the saved blob")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("component after round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

// TestDeserializeRestoresStatsAndRecomputesDerived is AC-3: GIVEN a Load
// operation, WHEN the Stats component is deserialized, THEN attributes,
// level, and XP match the saved values — and derived stats are recomputed
// from the restored attributes on demand (ADR-0006: derived stats are NOT
// serialized; recomputation is idempotent).
func TestDeserializeRestoresStatsAndRecomputesDerived(t *testing.T) {
	source, _, _ := newTestStore()
	want := fixtureLevel4()
	id := createTestEntity(t, source, entity.EntityTypeNPC, "Voss", matrix.NewVec3(2, 2, 0))
	mustSetComponent(t, source, id, want)

	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}

	restored, _, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}
	if got := restored.QueryEntities(entity.Query{}); len(got) != 1 {
		t.Fatalf("restored %d entities, want 1", len(got))
	}

	got, ok := entity.GetComponent[stats.StatsComponent](restored, id)
	if !ok {
		t.Fatal("StatsComponent missing after load")
	}

	// Persisted state matches the saved values exactly: attributes, level,
	// XP, and the unspent attribute point.
	if !reflect.DeepEqual(got.Attributes, want.Attributes) {
		t.Errorf("attributes mismatch:\n got %+v\nwant %+v", got.Attributes, want.Attributes)
	}
	if got.Level != want.Level {
		t.Errorf("Level = %d, want %d", got.Level, want.Level)
	}
	if got.XP != want.XP {
		t.Errorf("XP = %d, want %d", got.XP, want.XP)
	}
	if got.UnspentPoints != want.UnspentPoints {
		t.Errorf("UnspentPoints = %d, want %d", got.UnspentPoints, want.UnspentPoints)
	}

	// Derived stats are recomputed from the restored attributes + level, not
	// read from the blob (hand-computed expected values for the fixture).
	if v := got.MaxHealth(); v != 48 {
		t.Errorf("MaxHealth = %d, want 48", v)
	}
	if v := got.AP(); v != 7 {
		t.Errorf("AP = %d, want 7", v)
	}
	if v := got.Initiative(); v != 2 {
		t.Errorf("Initiative = %d, want 2", v)
	}
	if v := got.Evasion(); v != 13 {
		t.Errorf("Evasion = %d, want 13", v)
	}
	if v := got.CarryCapacity(); v != 130 {
		t.Errorf("CarryCapacity = %d, want 130", v)
	}

	// Value isolation after load: the restored component is a fresh value —
	// mutating it must not leak into the source store's component.
	got.Attributes[stats.STR]++
	if src, _ := entity.GetComponent[stats.StatsComponent](source, id); src.Attributes[stats.STR] != want.Attributes[stats.STR] {
		t.Errorf("source store attribute changed by mutating the restored copy: got %d, want %d", src.Attributes[stats.STR], want.Attributes[stats.STR])
	}
	if v := got.MaxHealth(); v != 48 {
		t.Errorf("MaxHealth after mutating the copy = %d, want unchanged 48", v)
	}
}

// TestLevelUpEmitsEventWithNewDerivedStats is AC-4: GIVEN a level-up during
// combat, WHEN XP is awarded, THEN a stats.level_up event is emitted with the
// entity ID and the new MaxHealth and AP values, so the Combat UI updates
// without re-querying the component (ADR-0011 synchronous dispatch).
func TestLevelUpEmitsEventWithNewDerivedStats(t *testing.T) {
	store, _, bus := newTestStore()
	want := fixtureBaseline()
	id := createTestEntity(t, store, entity.EntityTypeCharacter, "Elena", matrix.NewVec3(5, 5, 0))
	mustSetComponent(t, store, id, want)

	recorder := &eventRecorder{}
	recorder.subscribe(bus, stats.EventStatLevelUp)

	// Read the component copy, award XP through the bridge, write back
	// (components are stored by value in the EF).
	c, ok := entity.GetComponent[stats.StatsComponent](store, id)
	if !ok {
		t.Fatal("GetComponent did not find the attached StatsComponent")
	}
	gained := stats.AwardXPAndNotify(bus, id, &c, 300) // TotalXPForLevel(2) = 300
	if gained != 1 {
		t.Fatalf("AwardXPAndNotify = %d level(s), want 1", gained)
	}
	if c.Level != 2 || c.XP != 300 {
		t.Errorf("component after award = level %d, XP %d; want level 2, XP 300", c.Level, c.XP)
	}
	if c.UnspentPoints != 0 {
		t.Errorf("UnspentPoints = %d, want 0 (level 2 grants no point)", c.UnspentPoints)
	}
	mustSetComponent(t, store, id, c)

	// The store reflects the write-back.
	stored, _ := entity.GetComponent[stats.StatsComponent](store, id)
	if stored.Level != 2 {
		t.Errorf("store component Level = %d, want 2 after write-back", stored.Level)
	}

	// Exactly one level-up event with the final derived values: MaxHealth
	// 20+14*2+2*2 = 52, AP 4+1 = 5 (DEX unchanged by leveling).
	if n := recorder.count(stats.EventStatLevelUp); n != 1 {
		t.Fatalf("stats.level_up fired %d times, want 1", n)
	}
	ev, _ := firstEventOf(recorder, stats.EventStatLevelUp)
	if ev.Source != "stats" {
		t.Errorf("event Source = %q, want %q", ev.Source, "stats")
	}
	data, ok := ev.Data.(stats.LevelUpEventData)
	if !ok {
		t.Fatalf("event Data = %#v, want stats.LevelUpEventData", ev.Data)
	}
	if data.EntityID != id {
		t.Errorf("payload EntityID = %d, want %d", data.EntityID, id)
	}
	if data.NewLevel != 2 {
		t.Errorf("payload NewLevel = %d, want 2", data.NewLevel)
	}
	if data.NewMaxHealth != 52 {
		t.Errorf("payload NewMaxHealth = %d, want 52", data.NewMaxHealth)
	}
	if data.NewAP != 5 {
		t.Errorf("payload NewAP = %d, want 5", data.NewAP)
	}
}

// TestAwardXPBelowThresholdEmitsNoEvent covers the bridge's no-op path: an
// award below the next level threshold gains no levels and emits no event.
func TestAwardXPBelowThresholdEmitsNoEvent(t *testing.T) {
	store, _, bus := newTestStore()
	id := createTestEntity(t, store, entity.EntityTypeCharacter, "Elena", matrix.NewVec3(5, 5, 0))
	mustSetComponent(t, store, id, fixtureBaseline())

	recorder := &eventRecorder{}
	recorder.subscribe(bus, stats.EventStatLevelUp)

	c, _ := entity.GetComponent[stats.StatsComponent](store, id)
	gained := stats.AwardXPAndNotify(bus, id, &c, 299) // below TotalXPForLevel(2) = 300
	if gained != 0 {
		t.Errorf("AwardXPAndNotify = %d level(s), want 0", gained)
	}
	if c.Level != 1 || c.XP != 299 {
		t.Errorf("component after award = level %d, XP %d; want level 1, XP 299", c.Level, c.XP)
	}
	if n := recorder.count(stats.EventStatLevelUp); n != 0 {
		t.Errorf("stats.level_up fired %d times, want 0", n)
	}
}

// TestAwardXPAtLevelCapEmitsNoEvent covers the level-cap edge: XP continues
// to accumulate at LevelCap but no further level-ups occur, so no event is
// emitted (GDD Edge Cases).
func TestAwardXPAtLevelCapEmitsNoEvent(t *testing.T) {
	store, _, bus := newTestStore()
	atCap := fixtureBaseline()
	atCap.Level = stats.LevelCap
	atCap.XP = 57000 // TotalXPForLevel(20) = 150*20*19
	id := createTestEntity(t, store, entity.EntityTypeCharacter, "Elena", matrix.NewVec3(5, 5, 0))
	mustSetComponent(t, store, id, atCap)

	recorder := &eventRecorder{}
	recorder.subscribe(bus, stats.EventStatLevelUp)

	c, _ := entity.GetComponent[stats.StatsComponent](store, id)
	gained := stats.AwardXPAndNotify(bus, id, &c, 5000)
	if gained != 0 {
		t.Errorf("AwardXPAndNotify = %d level(s), want 0 at level cap", gained)
	}
	if c.Level != stats.LevelCap || c.XP != 62000 {
		t.Errorf("component after award = level %d, XP %d; want level 20, XP 62000 (XP preserved past cap)", c.Level, c.XP)
	}
	if n := recorder.count(stats.EventStatLevelUp); n != 0 {
		t.Errorf("stats.level_up fired %d times, want 0", n)
	}
}

// TestAwardXPNegativeOrZeroEmitsNoEvent covers the documented xp ≤ 0 no-op
// path: AwardXPAndNotify's doc states "0 when xp ≤ 0 … in which case no event
// is emitted." Both zero and negative awards must be no-ops that leave the
// component unchanged and emit nothing.
func TestAwardXPNegativeOrZeroEmitsNoEvent(t *testing.T) {
	store, _, bus := newTestStore()
	id := createTestEntity(t, store, entity.EntityTypeCharacter, "Elena", matrix.NewVec3(5, 5, 0))
	mustSetComponent(t, store, id, fixtureBaseline())

	recorder := &eventRecorder{}
	recorder.subscribe(bus, stats.EventStatLevelUp)

	// Zero award: no-op.
	c, _ := entity.GetComponent[stats.StatsComponent](store, id)
	gained := stats.AwardXPAndNotify(bus, id, &c, 0)
	if gained != 0 {
		t.Errorf("AwardXPAndNotify(xp=0) = %d level(s), want 0", gained)
	}
	if c.Level != 1 || c.XP != 0 {
		t.Errorf("component after zero award = level %d, XP %d; want level 1, XP 0", c.Level, c.XP)
	}
	if n := recorder.count(stats.EventStatLevelUp); n != 0 {
		t.Errorf("stats.level_up fired %d times for xp=0, want 0", n)
	}

	// Negative award: also a no-op (GDD has no XP-loss mechanic).
	c2, _ := entity.GetComponent[stats.StatsComponent](store, id)
	gained2 := stats.AwardXPAndNotify(bus, id, &c2, -100)
	if gained2 != 0 {
		t.Errorf("AwardXPAndNotify(xp=-100) = %d level(s), want 0", gained2)
	}
	if c2.Level != 1 || c2.XP != 0 {
		t.Errorf("component after negative award = level %d, XP %d; want level 1, XP 0 (unchanged)", c2.Level, c2.XP)
	}
	if n := recorder.count(stats.EventStatLevelUp); n != 0 {
		t.Errorf("stats.level_up fired %d times for xp=-100, want 0", n)
	}
}

// TestLevelUpMultiLevelJumpEmitsSingleAggregateEvent covers multi-level
// jumps: one award that crosses several thresholds gains multiple levels but
// emits a single aggregate event carrying the final values (the payload is
// the resulting state, not per-level increments).
func TestLevelUpMultiLevelJumpEmitsSingleAggregateEvent(t *testing.T) {
	store, _, bus := newTestStore()
	id := createTestEntity(t, store, entity.EntityTypeCharacter, "Elena", matrix.NewVec3(5, 5, 0))
	mustSetComponent(t, store, id, fixtureBaseline())

	recorder := &eventRecorder{}
	recorder.subscribe(bus, stats.EventStatLevelUp)

	c, _ := entity.GetComponent[stats.StatsComponent](store, id)
	gained := stats.AwardXPAndNotify(bus, id, &c, 1200) // crosses L2 (300) and L3 (900), not L4 (1800)
	if gained != 2 {
		t.Fatalf("AwardXPAndNotify = %d level(s), want 2", gained)
	}
	if c.Level != 3 {
		t.Errorf("component Level = %d, want 3", c.Level)
	}
	if n := recorder.count(stats.EventStatLevelUp); n != 1 {
		t.Fatalf("stats.level_up fired %d times for a 2-level jump, want 1", n)
	}
	ev, _ := firstEventOf(recorder, stats.EventStatLevelUp)
	data, ok := ev.Data.(stats.LevelUpEventData)
	if !ok {
		t.Fatalf("event Data = %#v, want stats.LevelUpEventData", ev.Data)
	}
	if data.NewLevel != 3 {
		t.Errorf("payload NewLevel = %d, want 3", data.NewLevel)
	}
	if data.NewMaxHealth != 54 { // 20 + 14*2 + 3*2
		t.Errorf("payload NewMaxHealth = %d, want 54", data.NewMaxHealth)
	}
	if data.NewAP != 5 {
		t.Errorf("payload NewAP = %d, want 5", data.NewAP)
	}
}
