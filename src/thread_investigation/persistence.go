// Persistence integration for Story 005 (TR-thread-005, ADR-0001, ADR-0002):
// the load-side bridge between the Entity Framework's component store and the
// Engine's runtime state.
//
// PlayerKnowledge is persisted as an Entity Framework component on the player
// entity (component.go). The Engine is the runtime owner of the same state —
// its mutation methods keep the runtime indexes consistent with the slices.
// After a save/load round-trip the component holds deserialized state that no
// Engine has touched; RestoreKnowledge replaces the Engine's state with it and
// RebuildIndexes reconstructs every runtime index from the restored slices.
package thread_investigation

import (
	"strconv"
	"strings"

	"marrow-of-empire/entity"
	"marrow-of-empire/save"
)

// RestoreKnowledge replaces the Engine's PlayerKnowledge with deserialized
// state (Story 005 AC-1) and rebuilds all runtime indexes. It is the
// load-side integration point for the Save/Load system:
//
//	pk, ok := entity.GetComponent[thread_investigation.PlayerKnowledge](store, playerID)
//	if ok {
//	    engine.RestoreKnowledge(pk)
//	}
//
// Contract: every fact referenced by the restored knowledge must be
// registered via RegisterFact before this method runs — the NPC Knowledge
// Graph loader re-registers all facts on region load (KG-001, ADR-0005) —
// because the entity→chain index derives chain entity sets from fact
// References (collectChainEntities). Facts registered after the rebuild are
// picked up by subsequent discoveries; only the entity→chain index entries of
// the restored chains depend on the registration order.
//
// Pending event queues (DrainEvents/DrainChainEvents) and overlapFlags are
// engine-runtime state, not part of PlayerKnowledge, and are not restored: a
// deserialized engine starts with empty queues and no overlap flags, exactly
// like NewEngine. Overlap flags are intentionally not persisted (Story 005
// AC-1 field list); they are re-raised as new facts are discovered.
func (e *Engine) RestoreKnowledge(pk PlayerKnowledge) {
	e.knowledge = pk
	e.RebuildIndexes()
}

// RebuildIndexes reconstructs the Engine's runtime indexes from the current
// PlayerKnowledge state (Story 005 AC-1): factIndex from DiscoveredFacts,
// threadIndex from ActiveThreads and ResolvedThreads, knownIndex from
// KnownEntities, contradictedIndex from ContradictedFacts, and the
// entity→chain index from every non-archived chain's entity set (the
// References of its facts plus the TargetEntity of its threads, via
// collectChainEntities). Archived chains are excluded — they are never
// reachable through the index after a merge — and chainCounter is set above
// the highest existing chain number so post-load chain creation never
// collides ("chain_{N}" IDs are monotonic, Story 002).
//
// It is called by RestoreKnowledge after deserialization; the same contract
// applies: facts must be registered before the rebuild so chain entity sets
// resolve (see RestoreKnowledge). Calling it on a fresh engine is a no-op
// over already-consistent indexes.
func (e *Engine) RebuildIndexes() {
	e.factIndex = make(map[string]struct{}, len(e.knowledge.DiscoveredFacts))
	for _, d := range e.knowledge.DiscoveredFacts {
		e.factIndex[d.FactID] = struct{}{}
	}

	e.threadIndex = make(map[string]struct{}, len(e.knowledge.ActiveThreads)+len(e.knowledge.ResolvedThreads))
	for _, t := range e.knowledge.ActiveThreads {
		e.threadIndex[t.ID] = struct{}{}
	}
	for _, t := range e.knowledge.ResolvedThreads {
		e.threadIndex[t.ID] = struct{}{}
	}

	e.knownIndex = make(map[entity.GameEntityID]struct{}, len(e.knowledge.KnownEntities))
	for _, k := range e.knowledge.KnownEntities {
		e.knownIndex[k.EntityID] = struct{}{}
	}

	e.contradictedIndex = make(map[string]struct{}, len(e.knowledge.ContradictedFacts))
	for _, fid := range e.knowledge.ContradictedFacts {
		e.contradictedIndex[fid] = struct{}{}
	}

	e.entityChainIndex = make(map[entity.GameEntityID]map[string]struct{})
	for i := range e.knowledge.Chains {
		chain := &e.knowledge.Chains[i]
		if chain.Status == ChainStatusArchived {
			continue // archived chains are never in the index (merge cleanup)
		}
		e.indexAdd(chain.ID, e.collectChainEntities(chain.ID))
	}

	e.chainCounter = 1
	for _, c := range e.knowledge.Chains {
		if n, err := strconv.Atoi(strings.TrimPrefix(c.ID, "chain_")); err == nil && n >= e.chainCounter {
			e.chainCounter = n + 1
		}
	}
}

// migrationRegistrar is the subset of the Save/Load system's migration API
// that PlayerKnowledge migrations need (save.SaveManagerImpl implements it).
// The structural interface keeps the caller decoupled from the concrete
// manager while failing at compile time if the migration API changes. The
// parameter is save.MigrationFunc — not the underlying func type — because
// Go interface satisfaction matches named types exactly.
type migrationRegistrar interface {
	RegisterMigration(fromVersion, toVersion uint32, fn save.MigrationFunc)
}

// RegisterMigrations registers the PlayerKnowledge save-format migration
// functions with a Save/Load manager (Story 005 AC-3, ADR-0002 CR-3).
//
// The v1→v2 step is a no-op template: the v2 layout is identical to v1, so a
// save from version 1 loads on a version 2 build via this registered step.
// Future schema changes to PlayerKnowledge (or to any serialized component)
// replace this template with a real transformation: decode the payload, apply
// the field-level changes, re-encode. The migration runs on the whole save
// payload — the Save/Load system's MigrationFunc contract (TR-save-002) — so
// it must keep every other part of the payload intact.
//
// The game calls this once at startup for every SaveManager, before any load:
//
//	thread_investigation.RegisterMigrations(saveManager)
func RegisterMigrations(m migrationRegistrar) {
	m.RegisterMigration(1, 2, func(payload []byte) ([]byte, error) {
		// No-op template: PlayerKnowledge v1 and v2 share the same layout.
		// Future schema changes add the transformation here.
		return payload, nil
	})
}
