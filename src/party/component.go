// The pc_controlled tag (AC-4) and its Entity Framework integration.
//
// The Entity Framework has no tag mechanism yet (its GDD describes a
// per-entity tag set; the code implements component stores only), so the
// tag lives on a party-owned component, PCControlled, registered with the
// Entity Framework via entity.RegisterComponent — the same pattern as
// stats.StatsComponent and knowledge.KnowledgeGraph. The component-store
// registration gives the tag two things for free:
//
//   - Entity Framework save/load: the EF's serialization pass (EntityData
//     through the typed store) persists PCControlled per entity, satisfying
//     the GDD requirement "entity tagged pc_controlled pre-save is tagged
//     post-load" (entity-framework.md CR-8).
//   - Debug-overlay inspection via entity.GetComponent[party.PCControlled]
//     (GDD party AC-PS-01: "verifiable via debug overlay").
//
// The Party System is the tag's authoritative writer (ADR-0001
// write-ownership): it sets the tag on add-to-active and clears it on
// remove / move to pool. Other systems (TCE, UI, camera) read it — they
// never write it. Production writes go through the Entity Framework's write
// arbitration (EFTagAdapter -> entity.SetComponentArbitrated), per the
// story's implementation note.
package party

import "marrow-of-empire/entity"

func init() {
	// Registers PCControlled with pod and its typed store factory
	// (entity.RegisterComponent registers the component type and the nested
	// field types automatically; the field is a pod-registered
	// GameEntityID).
	entity.RegisterComponent[PCControlled]()
}

// PCControlled marks an entity as player-controlled (AC-4, GDD CR-1 tag
// `pc_controlled`). The Party System is the authoritative writer: active
// companions carry it; pool members and departed companions never do. Pure
// data and pod-safe (a single registered field).
type PCControlled struct {
	// PCID is the player character whose party the entity belongs to. The
	// debug overlay can read it to show who controls the entity.
	PCID entity.GameEntityID
}

// PCTagAccessor is the Entity Framework surface the party store requires
// for the pc_controlled tag (story note: tag writes go through the
// framework's write arbitration). The production implementation is
// EFTagAdapter; unit tests use an in-memory fake — the same
// dependency-injection pattern as persistent_world_state.PersistentAccessor
// (ADR-0001: game code depends on interfaces, not engine internals).
type PCTagAccessor interface {
	// SetPCControlled tags the entity as controlled by pcID.
	SetPCControlled(id entity.GameEntityID, pcID entity.GameEntityID) error
	// ClearPCControlled removes the tag. Clearing an untagged entity is a
	// no-op success (the Entity Framework's RemoveComponent semantics).
	ClearPCControlled(id entity.GameEntityID) error
}

// partyWritePriority is the party system's write-arbitration priority
// (ADR-0001: 1 = highest). The party system is the sole writer of the
// PCControlled slot, so the value only matters if a future system writes
// the same slot in the same frame; 5 places it after the entity framework
// (1) and the trust/thread systems (3-4), before presentation (10).
const partyWritePriority = 5

// EFTagAdapter is the production PCTagAccessor: it writes PCControlled
// through the Entity Framework's write arbitration (ADR-0001) and clears it
// through the typed component removal. arbiter must wrap store — a
// NewWriteArbiter(store) result — because the arbiter is the only path
// that applies same-frame conflict resolution, while the removal path
// (which has no arbitrated variant) needs the store.
type EFTagAdapter struct {
	store   entity.EntityStore
	arbiter *entity.WriteArbiter
}

// NewEFTagAdapter creates the adapter. arbiter must wrap store (see
// EFTagAdapter).
func NewEFTagAdapter(store entity.EntityStore, arbiter *entity.WriteArbiter) *EFTagAdapter {
	return &EFTagAdapter{store: store, arbiter: arbiter}
}

// SetPCControlled implements PCTagAccessor: an arbitrated write of the tag
// with system name "party" at partyWritePriority, whole-component slot
// (field ""). An uncontested write applies immediately with zero arbitration
// overhead (ADR-0001 AC-3); a write to a destroyed or unknown entity
// propagates the framework's error, which aborts the roster mutation
// (AddActive).
func (a *EFTagAdapter) SetPCControlled(id entity.GameEntityID, pcID entity.GameEntityID) error {
	return entity.SetComponentArbitrated(a.arbiter, "party", partyWritePriority, id, PCControlled{PCID: pcID}, "")
}

// ClearPCControlled implements PCTagAccessor: removes the tag component.
// Removing an absent tag is a no-op success (Entity Framework
// RemoveComponent semantics); a destroyed entity returns the framework's
// error, which the store treats as best-effort (the framework already
// purged the tag with the entity).
func (a *EFTagAdapter) ClearPCControlled(id entity.GameEntityID) error {
	return entity.RemoveComponent[PCControlled](a.store, id)
}
