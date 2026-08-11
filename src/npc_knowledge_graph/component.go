// KnowledgeGraph as an Entity Framework component (Story KG-005, ADR-0001,
// ADR-0005): the parsed .kg file is attached to an NPC's entity as the
// knowledge.KnowledgeGraph component value, so the AI Dialogue Engine can
// retrieve the NPC's full fact set at runtime and the save/load system
// round-trips all facts, domains, and boundaries (story AC-2, AC-3).
//
// Registration lives in this package, not in the entity package, because this
// package imports entity (Fact.References uses entity.GameEntityID) — entity
// cannot import knowledge without an import cycle. entity.RegisterComponent
// registers the type with pod and its typed store factory at init time, so
// the typed accessors (entity.GetComponent, entity.SetComponent,
// entity.RemoveComponent) and the EntityData serialization pass work for it
// exactly as for the entity package's own components (one descriptor-keyed
// map lookup, no reflection on the hot path).
//
// KnowledgeGraph is pure data — slices of strings, ints, and
// entity.GameEntityID only; no pointers, maps, or interfaces — which pod
// requires, so the typed store's EntityData implementation encodes it
// directly. The nested named types reachable from the encoded layout (Fact,
// Lead, FactType) are registered explicitly because pod requires every named
// type reachable from a serialized value to be registered
// (entity.GameEntityID is registered by the entity package's own init).
//
// Example usage:
//
//	graph, err := knowledge.ParseKGFile("assets/npc_knowledge/thornwell/greta_blacksmith.kg")
//	// ... handle err, fall back to FallbackPersonality per story AC-2 ...
//	entity.SetComponent(store, npcID, graph)
//	got, ok := entity.GetComponent[knowledge.KnowledgeGraph](store, npcID)
package knowledge

import "marrow-of-empire/entity"

// init registers KnowledgeGraph as an Entity Framework component and its
// nested named types with pod, so GetComponent/SetComponent/RemoveComponent
// and the save/load serialization pass work for it. Package init order
// guarantees the entity package's registrations (GameEntityID, its own
// components) run first: knowledge imports entity, so entity's init executes
// before this one.
func init() {
	entity.RegisterComponent[KnowledgeGraph]()
	entity.RegisterPodType(Fact{})
	entity.RegisterPodType(Lead{})
	entity.RegisterPodType(FactType(""))
}
