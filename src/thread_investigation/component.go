// PlayerKnowledge as an Entity Framework component (Story 005, TR-thread-005,
// ADR-0001, ADR-0002): the player's full investigation state is attached to
// the player entity as the thread_investigation.PlayerKnowledge component
// value, so the Save/Load system round-trips all facts, threads, chains,
// known entities, contradictions, and revelations (story AC-1, AC-2).
//
// Registration lives in this package, not in the entity package, because this
// package imports entity (Thread.TargetEntity uses entity.GameEntityID) —
// entity cannot import thread_investigation without an import cycle.
// entity.RegisterComponent registers the type with pod and its typed store
// factory at init time, so the typed accessors (entity.GetComponent,
// entity.SetComponent, entity.RemoveComponent) and the EntityData
// serialization pass work for it exactly as for the entity package's own
// components (one descriptor-keyed map lookup, no reflection on the hot
// path). This is the same pattern as the NPC Knowledge Graph component
// (npc_knowledge_graph/component.go, Story KG-005).
//
// PlayerKnowledge is pure data — slices of structs containing strings,
// entity.GameEntityID, and float64 only; no pointers, maps, or interfaces —
// which pod requires, so the typed store's EntityData implementation encodes
// it directly. The nested named types reachable from the encoded layout
// (DiscoveredFact, Thread, Chain, KnownEntity, Revelation, ThreadStatus) are
// registered explicitly because pod requires every named type reachable from
// a serialized value to be registered (entity.GameEntityID is registered by
// the entity package's own init). OverlapFlag is engine-runtime state, not
// part of PlayerKnowledge, and is not serialized (Story 005 AC-1 field list).
//
// Example usage:
//
//	// Save side: copy the engine's runtime state into the component.
//	entity.SetComponent(store, playerID, engine.Knowledge())
//
//	// Load side: restore the component into a fresh engine (persistence.go).
//	pk, ok := entity.GetComponent[thread_investigation.PlayerKnowledge](store, playerID)
//	engine.RestoreKnowledge(pk)
package thread_investigation

import "marrow-of-empire/entity"

// init registers PlayerKnowledge as an Entity Framework component and its
// nested named types with pod, so GetComponent/SetComponent/RemoveComponent
// and the save/load serialization pass work for it. Package init order
// guarantees the entity package's registrations (GameEntityID, its own
// components) run first: thread_investigation imports entity, so entity's
// init executes before this one.
func init() {
	entity.RegisterComponent[PlayerKnowledge]()
	entity.RegisterPodType(DiscoveredFact{})
	entity.RegisterPodType(Thread{})
	entity.RegisterPodType(Chain{})
	entity.RegisterPodType(KnownEntity{})
	entity.RegisterPodType(Revelation{})
	entity.RegisterPodType(ThreadStatus(""))
}
