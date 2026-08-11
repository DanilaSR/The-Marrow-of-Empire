// Package thread_investigation implements the Thread Investigation data
// model and auto-creation logic (Story 001: Thread Data Model &
// Auto-Creation, TR-thread-001, ADR-0010).
//
// The Thread Investigation system replaces the traditional quest log with
// a player-driven web of leads (anti-pillar: NOT a completionist
// checklist). Threads are auto-created when the player discovers a fact
// that carries `leads_to` hints, chained by entity overlap (Story 002),
// checked for contradictions (Story 003), and promoted to revelations when
// enough connected facts point to the same conclusion (Story 004). All
// state persists in PlayerKnowledge, registered as an Entity Framework
// component in Story 005 (TR-thread-005, ADR-0001).
//
// This package is pure game logic — engine-agnostic, no Kaiju APIs
// (Engine Notes, Story 001).
package thread_investigation

import "marrow-of-empire/entity"

// ThreadStatus is the lifecycle state of an investigation thread
// (ADR-0010). Story 004 owns the full resolution state machine; this
// story only creates threads in ThreadActive.
type ThreadStatus string

const (
	// ThreadActive means the lead is open and worth following.
	ThreadActive ThreadStatus = "active"
	// ThreadPrioritized means the player pinned the thread as important.
	ThreadPrioritized ThreadStatus = "prioritized"
	// ThreadResolved means the player followed the lead to its end.
	ThreadResolved ThreadStatus = "resolved"
	// ThreadDeadEnd means the lead went nowhere (NPC unavailable, etc.).
	ThreadDeadEnd ThreadStatus = "dead_end"
)

// Thread is a single lead the player can follow — "Ask X about Y".
// Threads are auto-created from fact `leads_to` hints (TR-thread-001,
// ADR-0010) and record the target entity and the topic worth pursuing.
type Thread struct {
	// ID is the unique thread identifier, format
	// "thread_{source_fact_id}_{target_entity_id}" — deterministic and
	// idempotent: discovering the same fact twice never duplicates.
	ID string
	// Source is the fact ID that created this thread.
	Source string
	// TargetEntity is the GameEntityID of who or what to investigate.
	TargetEntity entity.GameEntityID
	// Topic is the player-facing lead text, e.g.
	// "Ask [entity 42] about Ley-touched metal".
	Topic string
	// Status is the thread lifecycle state.
	Status ThreadStatus
	// ChainID is the investigation chain this thread belongs to;
	// populated by Story 002 (auto-chaining).
	ChainID string
	// CreatedAt is the game timestamp of creation (ADR-0007 game clock).
	CreatedAt float64
}

// Chain groups threads that share entity references (Story 002,
// TR-thread-002). Defined here because Story 001 ships the data model;
// chain assignment logic is Story 002.
type Chain struct {
	// ID is the unique chain identifier, format "chain_{N}" where N is a
	// monotonically incrementing counter (Story 002).
	ID string
	// Summary is the auto-generated description, e.g. "Aldric's Death".
	Summary string
	// Facts are the fact_ids discovered in this chain.
	Facts []string
	// Threads are the thread_ids in this chain.
	Threads []string
	// Status is "active", "revelation_ready", "resolved", or "archived"
	// (merged away — kept for investigation history, ADR-0010).
	Status string
}

// ChainStatusArchived marks a chain that was merged into a surviving chain
// on entity overlap (Story 002 Step 3b). Archived chains are not deleted —
// they stay in PlayerKnowledge.Chains as investigation history, and their
// facts/threads are unioned into the survivor (ADR-0010 merge rule 4).
const ChainStatusArchived = "archived"

// ChainStatusRevelationReady marks a chain whose facts reached the
// revelation threshold — 3+ facts pointing to the same conclusion (Story
// 004, TR-thread-004, ADR-0010 Step 5). It tells the Investigation Journal
// that a revelation is available. The status is terminal for the signal:
// additional facts in the chain do not re-emit EventRevelationReady
// (ADR-0010 spam mitigation).
const ChainStatusRevelationReady = "revelation_ready"

// OverlapFlag records that two facts from different NPCs both reference the
// same entity (Story 002 AC-1, TR-thread-002). The system surfaces these so
// the journal can show connections the player might miss: "Both [NPC A] and
// [NPC B] mentioned [entity]." Flags are deduplicated per (entity, NPC pair)
// on the Engine and read via GetOverlapFlags.
type OverlapFlag struct {
	// Entity is the GameEntityID both facts reference.
	Entity entity.GameEntityID
	// NPC_A is the NPC of the fact that was discovered first.
	NPC_A entity.GameEntityID
	// NPC_B is the NPC of the fact that was discovered second (the new one).
	NPC_B entity.GameEntityID
	// Message is the player-facing overlap text, e.g. "Both Orin and Greta
	// mentioned the old mine."
	Message string
}

// DiscoveredFact records that the player learned a fact (OnFactDiscovered
// step 1, ADR-0010). The full fact content lives in the NPC Knowledge
// Graph; PlayerKnowledge only tracks discovery.
type DiscoveredFact struct {
	// FactID is the fact identifier from the NPC Knowledge Graph
	// (ADR-0005), e.g. "greta_ley_metal_001".
	FactID string
	// SourceNPC is the GameEntityID of the NPC who revealed the fact.
	SourceNPC entity.GameEntityID
	// DiscoveredAt is the game timestamp of discovery.
	DiscoveredAt float64
}

// KnownEntity records an entity the player learned about, and which fact
// introduced it (OnFactDiscovered step 2, Story 001).
type KnownEntity struct {
	// EntityID is the GameEntityID of the entity the player met or heard
	// of.
	EntityID entity.GameEntityID
	// DiscoveredAt is the game timestamp of first discovery.
	DiscoveredAt float64
	// DiscoveredVia is the fact_id that introduced this entity.
	DiscoveredVia string
}

// Revelation records a chain that reached the revelation threshold
// (3+ connected facts, Story 004, TR-thread-004). Defined here because
// Story 001 ships the data model; detection logic is Story 004.
type Revelation struct {
	// ID is the unique revelation identifier.
	ID string
	// ChainID is the chain this revelation belongs to.
	ChainID string
	// DiscoveredAt is the game timestamp of the triggering fact.
	DiscoveredAt float64
	// ConfrontedAt is the game timestamp of confrontation; 0 = not yet
	// confronted.
	ConfrontedAt float64
}

// PlayerKnowledge is the player's full investigation state (ADR-0010,
// TR-thread-005): discovered facts, threads, chains, known entities, and
// revelations. It is persisted as an Entity Framework component in Story
// 005; until then it lives in memory on the Engine.
//
// Engine-owned: all mutations go through the InvestigationEngine
// implementation so its runtime indexes stay consistent with the slices.
type PlayerKnowledge struct {
	// DiscoveredFacts are the facts the player has learned, in discovery
	// order.
	DiscoveredFacts []DiscoveredFact
	// ActiveThreads are open leads (active or prioritized).
	ActiveThreads []Thread
	// ResolvedThreads are leads the player followed to an end (Story 004
	// refines resolution semantics).
	ResolvedThreads []Thread
	// Chains are investigation chains (Story 002).
	Chains []Chain
	// KnownEntities are entities the player has learned about.
	KnownEntities []KnownEntity
	// ContradictedFacts are the fact_ids flagged as part of a detected
	// contradiction (Story 003, ADR-0010 Step 4b). Every detection flags
	// both facts, deduplicated, in detection order; latent contradictions
	// (the other fact undiscovered) flag nothing.
	ContradictedFacts []string
	// Revelations are chains ready for confrontation (Story 004).
	Revelations []Revelation
}
