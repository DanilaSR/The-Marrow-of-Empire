// Integration tests for Story KG-005: Knowledge Component Integration.
//
// Evidence path: production/epics/npc-knowledge-graph/story-005-knowledge-component-integration.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_kg_parse_populates_npc_knowledge  -> TestAttachKnowledgeGraphToNPC
//	AC-2/3 test_save_load_preserves_knowledge     -> TestSaveLoadRoundTripPreservesKnowledge
//	AC-4  (KnowledgeStore interface)              -> deferred to the AI Dialogue Engine story
//
// Component lifecycle coverage beyond the story ACs: retrieval isolation
// across NPCs (TestGetComponentReturnsGraphOnlyWhenPresent) and explicit
// removal with the component.removed event
// (TestRemoveComponentFiresLifecycleEvent).
//
// The mock KaijuHost, eventRecorder, and newTestStore helpers are mirrored
// from tests/integration/entity (Go cannot import test helpers across
// packages). KnowledgeGraph is registered as an EF component by the knowledge
// package init (component.go), so the typed accessors and the serialization
// pass treat it like any other component.
package knowledge_test

import (
	"reflect"
	"slices"
	"testing"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"

	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
)

// gretaKG is the ADR-0005 authoring-format baseline fixture (mirrored from
// tests/unit/npc_knowledge_graph): the Thornwell blacksmith Greta with a
// full-coverage fact (references, discovery triggers, contradicts, leads_to)
// and a minimal fact relying on the trust_required=0 default.
func gretaKG() string {
	return `npc_id: "greta_blacksmith"
npc_name: "Greta"
domains: ["craft_metallurgy", "trade_commerce"]
personality: "Practical, observant, proud of her craft."
speech_style: "Direct, uses smithing metaphors."
knows_not: ["old_empire_lore", "faction_politics"]
knows_not_entities: []
facts:
  - fact_id: "greta_ley_metal_001"
    domain: "craft_metallurgy"
    type: FACT
    content: "This fragment... it's an alloy I've never seen. Pale silver, warm to the touch."
    trust_required: 0
    references: [7, 12]
    discovery_triggers: ["player_has_item:ley_metal_fragment"]
    contradicts: ["orin_ley_metal_003"]
    leads_to:
      - entity_id: 9
        topic_hint: "Ask the hermit about Ley-touched metal"
  - fact_id: "greta_trade_002"
    domain: "trade_commerce"
    type: RUMOR
    content: "The caravan from Velden has been buying scrap metal at three times market price."
`
}

// orinKG is a second, distinct NPC graph for cross-talk and round-trip
// isolation checks: different domains, a trust-gated SECRET fact, and a
// non-empty knows_not_entities boundary.
func orinKG() string {
	return `npc_id: "orin_hermit"
npc_name: "Orin"
domains: ["old_empire_lore", "herbalism"]
personality: "Reclusive, speaks in riddles."
speech_style: "Soft, cryptic."
knows_not: ["craft_metallurgy", "faction_politics"]
knows_not_entities: ["veren_secret_identity"]
facts:
  - fact_id: "orin_ley_metal_003"
    domain: "old_empire_lore"
    type: SECRET
    content: "The pale silver alloy was forged in the old empire's foundries."
    trust_required: 3
    references: [7]
    leads_to:
      - entity_id: 9
        topic_hint: "Ask the blacksmith about Ley-touched metal"
`
}

// mockKaijuHost is a test double for entity.KaijuHost (mirrored from
// tests/integration/entity). It registers engine entities by EntityId and
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
// tests/integration/entity).
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
// from tests/integration/entity). Config zero values fall back to the GDD
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

func mustParseKG(t *testing.T, data string) knowledge.KnowledgeGraph {
	t.Helper()
	graph, err := knowledge.ParseKG([]byte(data))
	if err != nil {
		t.Fatalf("ParseKG error: %v", err)
	}
	return graph
}

// sameGraph compares two KnowledgeGraphs for the save/load contract (story
// AC-3: facts, domains, and boundaries match the saved state). It is
// nil/empty-insensitive on every slice field because pod's binary format
// encodes slice length only: an empty-but-non-nil slice (e.g.
// "knows_not_entities: []" in a .kg file) decodes as nil. len() and iteration
// behave identically, so the data is preserved; reflect.DeepEqual is too
// strict for round-trip comparison.
func sameGraph(a, b knowledge.KnowledgeGraph) bool {
	return a.NPCID == b.NPCID &&
		a.NPCName == b.NPCName &&
		slices.Equal(a.Domains, b.Domains) &&
		a.Personality == b.Personality &&
		a.SpeechStyle == b.SpeechStyle &&
		slices.Equal(a.KnowsNot, b.KnowsNot) &&
		slices.Equal(a.KnowsNotEntities, b.KnowsNotEntities) &&
		sameFacts(a.Facts, b.Facts)
}

// sameFacts compares fact lists in authoring order with the same
// nil/empty-insensitive slice semantics as sameGraph.
func sameFacts(a, b []knowledge.Fact) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].FactID != b[i].FactID ||
			a[i].Domain != b[i].Domain ||
			a[i].Type != b[i].Type ||
			a[i].Content != b[i].Content ||
			a[i].TrustRequired != b[i].TrustRequired ||
			!slices.Equal(a[i].References, b[i].References) ||
			!slices.Equal(a[i].DiscoveryTriggers, b[i].DiscoveryTriggers) ||
			!slices.Equal(a[i].Contradicts, b[i].Contradicts) ||
			!sameLeads(a[i].LeadsTo, b[i].LeadsTo) {
			return false
		}
	}
	return true
}

// sameLeads compares lead lists with the same nil/empty-insensitive slice
// semantics as sameGraph.
func sameLeads(a, b []knowledge.Lead) bool {
	return slices.EqualFunc(a, b, func(x, y knowledge.Lead) bool {
		return x.EntityID == y.EntityID && x.TopicHint == y.TopicHint
	})
}

// TestAttachKnowledgeGraphToNPC is AC-1: GIVEN a parsed .kg file, WHEN the
// KnowledgeGraph is attached to an NPC entity via the Entity Framework, THEN
// GetComponent retrieves it intact and component.added fires with the graph.
func TestAttachKnowledgeGraphToNPC(t *testing.T) {
	store, _, bus := newTestStore()
	graph := mustParseKG(t, gretaKG())
	id := createTestEntity(t, store, entity.EntityTypeNPC, "Greta", matrix.NewVec3(5, 5, 0))

	recorder := &eventRecorder{}
	recorder.subscribe(bus, entity.EventComponentAdded)

	if err := entity.SetComponent(store, id, graph); err != nil {
		t.Fatalf("SetComponent error: %v", err)
	}

	got, ok := entity.GetComponent[knowledge.KnowledgeGraph](store, id)
	if !ok {
		t.Fatal("GetComponent did not find the attached KnowledgeGraph")
	}
	if !reflect.DeepEqual(got, graph) {
		t.Errorf("retrieved graph mismatch:\n got %+v\nwant %+v", got, graph)
	}

	ev, ok := firstEventOf(recorder, entity.EventComponentAdded)
	if !ok {
		t.Fatal("component.added did not fire on attachment")
	}
	data, ok := ev.Data.(entity.ComponentEventData)
	if !ok || data.EntityID != id || data.ComponentType != "KnowledgeGraph" {
		t.Errorf("component.added payload = %+v, want entity %d, type KnowledgeGraph", ev.Data, id)
	}
	if !reflect.DeepEqual(data.NewValue, graph) {
		t.Errorf("component.added NewValue mismatch:\n got %+v\nwant %+v", data.NewValue, graph)
	}
	if data.OldValue != nil {
		t.Errorf("component.added OldValue = %+v, want nil", data.OldValue)
	}
}

// TestGetComponentReturnsGraphOnlyWhenPresent covers retrieval isolation:
// each NPC retrieves its own graph with no cross-talk, an entity without a
// graph reports (zero, false), and the type-erased GetComponentData path
// returns the same value as the typed path.
func TestGetComponentReturnsGraphOnlyWhenPresent(t *testing.T) {
	store, _, _ := newTestStore()
	greta := mustParseKG(t, gretaKG())
	orin := mustParseKG(t, orinKG())
	gretaID := createTestEntity(t, store, entity.EntityTypeNPC, "Greta", matrix.NewVec3(1, 0, 0))
	orinID := createTestEntity(t, store, entity.EntityTypeNPC, "Orin", matrix.NewVec3(2, 0, 0))
	bareID := createTestEntity(t, store, entity.EntityTypeProp, "Bare", matrix.NewVec3(3, 0, 0))
	mustSetComponent(t, store, gretaID, greta)
	mustSetComponent(t, store, orinID, orin)

	got, ok := entity.GetComponent[knowledge.KnowledgeGraph](store, gretaID)
	if !ok || !reflect.DeepEqual(got, greta) {
		t.Errorf("Greta's graph mismatch:\n got %+v ok=%v\nwant %+v", got, ok, greta)
	}
	got, ok = entity.GetComponent[knowledge.KnowledgeGraph](store, orinID)
	if !ok || !reflect.DeepEqual(got, orin) {
		t.Errorf("Orin's graph mismatch:\n got %+v ok=%v\nwant %+v", got, ok, orin)
	}

	// An entity without a graph reports (zero, false) on both access paths.
	if _, ok := entity.GetComponent[knowledge.KnowledgeGraph](store, bareID); ok {
		t.Error("bare entity reported a KnowledgeGraph")
	}
	if got, ok := store.GetComponentData(bareID, "KnowledgeGraph"); ok || got != nil {
		t.Errorf("bare entity erased lookup = %#v ok=%v, want nil/false", got, ok)
	}

	// The erased path returns the same value as the typed path.
	raw, ok := store.GetComponentData(gretaID, "KnowledgeGraph")
	if !ok {
		t.Fatal("erased GetComponentData did not find Greta's graph")
	}
	typed, ok := raw.(knowledge.KnowledgeGraph)
	if !ok || !reflect.DeepEqual(typed, greta) {
		t.Errorf("erased lookup mismatch:\n got %#v\nwant %+v", raw, greta)
	}
}

// TestSaveLoadRoundTripPreservesKnowledge is AC-2/AC-3: GIVEN NPCs with
// KnowledgeGraph components, WHEN the store is serialized and deserialized
// into a fresh store, THEN all facts, domains, and boundaries match the saved
// state, the restored graph still answers QueryFacts identically, and no
// graph leaks onto entities that never had one.
func TestSaveLoadRoundTripPreservesKnowledge(t *testing.T) {
	source, _, _ := newTestStore()
	greta := mustParseKG(t, gretaKG())
	orin := mustParseKG(t, orinKG())
	gretaID := createTestEntity(t, source, entity.EntityTypeNPC, "Greta", matrix.NewVec3(1, 1, 0))
	orinID := createTestEntity(t, source, entity.EntityTypeNPC, "Orin", matrix.NewVec3(2, 2, 0))
	bareID := createTestEntity(t, source, entity.EntityTypeProp, "Bare", matrix.NewVec3(3, 3, 0))
	mustSetComponent(t, source, gretaID, greta)
	mustSetComponent(t, source, orinID, orin)

	blob, err := source.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}

	restored, _, _ := newTestStore()
	if err := restored.Deserialize(blob); err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}
	if got := restored.QueryEntities(entity.Query{}); len(got) != 3 {
		t.Fatalf("restored %d entities, want 3", len(got))
	}

	// All knowledge data survives the round-trip: identity, domains,
	// personality, speech style, boundaries, and the full fact set.
	for _, tc := range []struct {
		id   entity.GameEntityID
		want knowledge.KnowledgeGraph
	}{
		{gretaID, greta},
		{orinID, orin},
	} {
		got, ok := entity.GetComponent[knowledge.KnowledgeGraph](restored, tc.id)
		if !ok {
			t.Errorf("entity %d: KnowledgeGraph missing after round-trip", tc.id)
			continue
		}
		if !sameGraph(got, tc.want) {
			t.Errorf("entity %d: KnowledgeGraph mismatch after round-trip:\n got %+v\nwant %+v", tc.id, got, tc.want)
		}
	}

	// The restored graph is usable: QueryFacts filters it exactly like the
	// original (trust gating, domain match, discovery triggers).
	cfg := knowledge.QueryConfig{
		TrustLevel:        2,
		Domain:            "craft_metallurgy",
		SatisfiedTriggers: []string{"player_has_item:ley_metal_fragment"},
	}
	gotGreta, _ := entity.GetComponent[knowledge.KnowledgeGraph](restored, gretaID)
	wantGreta, _ := entity.GetComponent[knowledge.KnowledgeGraph](source, gretaID)
	gotFacts := knowledge.QueryFacts(&gotGreta, cfg)
	wantFacts := knowledge.QueryFacts(&wantGreta, cfg)
	if !sameFacts(gotFacts, wantFacts) {
		t.Errorf("QueryFacts on restored graph mismatch:\n got %+v\nwant %+v", gotFacts, wantFacts)
	}
	if len(gotFacts) == 0 {
		t.Error("QueryFacts on restored graph returned no facts, want the trigger-satisfied fact")
	}

	// The componentless entity picks up no graph.
	if _, ok := entity.GetComponent[knowledge.KnowledgeGraph](restored, bareID); ok {
		t.Error("KnowledgeGraph leaked onto the componentless entity after load")
	}
}

// TestRemoveComponentFiresLifecycleEvent covers explicit detachment: GIVEN an
// NPC with a KnowledgeGraph, WHEN RemoveComponent is called, THEN the
// component is gone, component.removed fires once with the removed graph, and
// a second removal is an idempotent no-op that fires no event.
func TestRemoveComponentFiresLifecycleEvent(t *testing.T) {
	store, _, bus := newTestStore()
	graph := mustParseKG(t, gretaKG())
	id := createTestEntity(t, store, entity.EntityTypeNPC, "Greta", matrix.NewVec3(4, 4, 0))
	mustSetComponent(t, store, id, graph)

	recorder := &eventRecorder{}
	recorder.subscribe(bus, entity.EventComponentRemoved)

	if err := entity.RemoveComponent[knowledge.KnowledgeGraph](store, id); err != nil {
		t.Fatalf("RemoveComponent error: %v", err)
	}
	if _, ok := entity.GetComponent[knowledge.KnowledgeGraph](store, id); ok {
		t.Error("component still present after RemoveComponent")
	}

	ev, ok := firstEventOf(recorder, entity.EventComponentRemoved)
	if !ok {
		t.Fatal("component.removed did not fire on removal")
	}
	data, ok := ev.Data.(entity.ComponentEventData)
	if !ok || data.EntityID != id || data.ComponentType != "KnowledgeGraph" {
		t.Errorf("component.removed payload = %+v, want entity %d, type KnowledgeGraph", ev.Data, id)
	}
	if !reflect.DeepEqual(data.OldValue, graph) {
		t.Errorf("component.removed OldValue mismatch:\n got %+v\nwant %+v", data.OldValue, graph)
	}
	if data.NewValue != nil {
		t.Errorf("component.removed NewValue = %+v, want nil", data.NewValue)
	}

	// Removal is idempotent: removing an absent component is a no-op and
	// fires no event (nil error, no state change).
	if err := entity.RemoveComponent[knowledge.KnowledgeGraph](store, id); err != nil {
		t.Fatalf("second RemoveComponent error: %v", err)
	}
	if n := recorder.count(entity.EventComponentRemoved); n != 1 {
		t.Errorf("component.removed fired %d times after idempotent removal, want still 1", n)
	}
}
