package thread_investigation

import (
	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
)

// EventThreadCreatedType is the EventBus event type for thread creation
// (ADR-0011). It follows the namespaced "system.action" convention used by
// the Entity Framework ("entity.created"). EventBus wiring is a later
// story: the Engine queues EventThreadCreated values (drained via
// DrainEvents) and the Question System integration story emits them on the
// bus as Event{Type: EventThreadCreatedType, Source: "thread_investigation"}.
const EventThreadCreatedType entity.EventType = "thread.created"

// EventThreadCreated is emitted when OnFactDiscovered auto-creates a
// thread from a fact `leads_to` hint (TR-thread-001, ADR-0010). It is a
// value type so the hot path avoids heap allocation beyond the interface
// boxing in Event.Data (ADR-0011).
type EventThreadCreated struct {
	// ThreadID is the created thread's ID.
	ThreadID string
	// SourceFactID is the fact that created the thread.
	SourceFactID string
	// TargetEntity is the entity the thread points to.
	TargetEntity entity.GameEntityID
	// Topic is the player-facing lead text.
	Topic string
	// CreatedAt is the game timestamp of creation.
	CreatedAt float64
}

// Chain event types (Story 002 auto-chaining, ADR-0010 Step 3/3b). These
// follow the namespaced "system.action" convention of the EventBus
// (ADR-0011) and are queued on the Engine, drained via DrainChainEvents,
// then emitted on the bus as Event{Type: ..., Source: "thread_investigation"}
// by the Question System integration story.
const (
	// EventThreadChainedType fires when a thread is auto-assigned to an
	// existing investigation chain (ADR-0010 Step 3c).
	EventThreadChainedType entity.EventType = "thread.chained"
	// EventChainCreatedType fires when a new chain is created for a thread
	// whose entities match no existing chain (ADR-0010 Step 3d).
	EventChainCreatedType entity.EventType = "chain.created"
	// EventChainsMergedType fires when two or more chains are discovered to
	// share an entity reference and auto-merge (ADR-0010 Step 3b).
	EventChainsMergedType entity.EventType = "chain.merged"
)

// EventThreadChained is emitted when a newly created thread is assigned to
// an existing chain (Story 002 AC-2). Payloads are value types so the hot
// path avoids heap allocation beyond the interface boxing in Event.Data
// (ADR-0011).
type EventThreadChained struct {
	// ThreadID is the assigned thread's ID.
	ThreadID string
	// ChainID is the chain the thread was assigned to.
	ChainID string
}

// EventChainCreated is emitted when a new chain is created (Story 002
// AC-3). The creating thread is the chain's first thread.
type EventChainCreated struct {
	// ChainID is the new chain's ID, format "chain_{N}".
	ChainID string
	// Summary is the auto-generated chain summary (first topic hint or
	// source fact ID).
	Summary string
	// FirstThreadID is the thread whose discovery created the chain.
	FirstThreadID string
}

// EventChainsMerged is emitted when chains auto-merge on entity overlap
// (Story 002 AC-4). The surviving chain keeps its ID and absorbs all facts
// and threads of the archived chains; archived chains stay in
// PlayerKnowledge with Status "archived" for investigation history.
type EventChainsMerged struct {
	// SurvivingChainID is the chain that absorbed the others (the oldest).
	SurvivingChainID string
	// ArchivedChainIDs are the merged-away chains, in creation order.
	ArchivedChainIDs []string
	// MergedSummary is the recalculated summary covering the merged scope.
	MergedSummary string
}

// EventContradictionDetectedType is the EventBus event type for detected
// fact contradictions (ADR-0010 Step 4, TR-thread-003). It follows the
// namespaced "system.action" convention of the EventBus (ADR-0011) and is
// queued on the Engine, drained via DrainChainEvents, then emitted on the
// bus as Event{Type: ..., Source: "thread_investigation"} by the Question
// System integration story.
const EventContradictionDetectedType entity.EventType = "contradiction.detected"

// EventContradictionDetected is emitted when a newly discovered fact
// contradicts a previously discovered fact and a contradiction thread is
// created (Story 003 AC-1, ADR-0010 Step 4). Contradictions are clues, not
// bugs — they create investigation tension; downstream systems
// (Investigation Journal, Trust & Relationship) react via the EventBus.
// Payloads are value types so the hot path avoids heap allocation beyond
// the interface boxing in Event.Data (ADR-0011).
type EventContradictionDetected struct {
	// FactA is the newly discovered fact.
	FactA knowledge.Fact
	// FactB is the previously discovered fact it contradicts.
	FactB knowledge.Fact
	// SourceNPC_A is the GameEntityID of the NPC who revealed FactA.
	SourceNPC_A entity.GameEntityID
	// SourceNPC_B is the GameEntityID of the NPC who revealed FactB.
	SourceNPC_B entity.GameEntityID
	// Confidence is "high", or "low" when either fact is a RUMOR (ADR-0005
	// fact type) — the rumor may be false (Story 003 CR-3).
	Confidence string
	// ThreadID is the contradiction thread created for this detection.
	ThreadID string
}

// EventRevelationReadyType is the EventBus event type for chains that
// reached the revelation threshold (ADR-0010 Step 5, TR-thread-004). It
// follows the namespaced "system.action" convention of the EventBus
// (ADR-0011) and is queued on the Engine, drained via DrainChainEvents, then
// emitted on the bus as Event{Type: ..., Source: "thread_investigation"} by
// the Question System integration story. The Investigation Journal listens
// for it to suggest a revelation is available ("amber glow").
const EventRevelationReadyType entity.EventType = "revelation.ready"

// EventRevelationReady is emitted once per chain when the revelation
// threshold is met: 3+ facts in the chain share the same leads_to target
// entity, or all contradict the same fact (Story 004 AC-1, ADR-0010 Step 5).
// The signal fires exactly once per chain — additional facts after
// revelation_ready do not re-emit it (spam prevention). Payloads are value
// types so the hot path avoids heap allocation beyond the interface boxing
// in Event.Data (ADR-0011).
type EventRevelationReady struct {
	// ChainID is the chain that reached the threshold.
	ChainID string
	// FactCount is the number of facts sharing the conclusion.
	FactCount int
	// TargetEntity is the shared leads_to target entity the facts point to;
	// 0 when the revelation came from a contradiction group (the conclusion
	// is a fact, not an entity).
	TargetEntity entity.GameEntityID
	// Summary is the chain's auto-generated summary for display.
	Summary string
}

// EventThreadResolvedType is the EventBus event type for thread resolution
// (ADR-0010 CR-4, Story 004). It follows the namespaced "system.action"
// convention of the EventBus (ADR-0011). The Trust & Relationship system
// already listens for the same "thread.resolved" type string
// (EventGameplayThreadResolved, trust_relationship package) — the values
// deliberately agree.
const EventThreadResolvedType entity.EventType = "thread.resolved"

// EventThreadResolved is emitted when a thread changes state via the Story
// 004 resolution state machine: ThreadResolved (discovered) and
// ThreadDeadEnd (manual or NPC-death auto-resolution) move the thread to
// ResolvedThreads; ThreadPrioritized pins it in ActiveThreads. The thread
// itself is found by ThreadID in ResolvedThreads or ActiveThreads.
type EventThreadResolved struct {
	// ThreadID is the resolved thread's ID.
	ThreadID string
	// Resolution is the new ThreadStatus: ThreadResolved, ThreadDeadEnd, or
	// ThreadPrioritized.
	Resolution ThreadStatus
	// Reason is the resolution reason: the entity destruction reason for
	// NPC-death auto-resolution (e.g. "combat_death"), or "" for manual
	// resolution via ResolveThread.
	Reason string
}

// ChainEvent is the envelope for chain events queued on the Engine (Story
// 002). It pairs an EventType constant with its typed payload, so the
// Question System integration story can forward it to the EventBus
// (ADR-0011) as Event{Type: ev.Type, Source: "thread_investigation",
// Data: ev.Data}. The Timestamp is the game clock captured at queue time
// (ADR-0007); thread creation events carry their timestamp in the payload
// instead, matching the Story 001 pattern.
type ChainEvent struct {
	// Type identifies the event; one of EventThreadChainedType,
	// EventChainCreatedType, EventChainsMergedType,
	// EventContradictionDetectedType, EventRevelationReadyType, or
	// EventThreadResolvedType.
	Type entity.EventType
	// Timestamp is the game time when the event was queued.
	Timestamp float64
	// Data is the type-specific payload: EventThreadChained,
	// EventChainCreated, EventChainsMerged, EventContradictionDetected,
	// EventRevelationReady, or EventThreadResolved.
	Data any
}
