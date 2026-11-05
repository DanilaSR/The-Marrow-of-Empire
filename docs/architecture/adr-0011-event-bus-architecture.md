# ADR-0011: Event Bus Architecture

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Core |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | `kaiju/src/engine/` (events.Event), Entity Framework GDD, Persistent World State GDD |
| **Post-Cutoff APIs Used** | `events.Event` (Kaiju native) |
| **Verification Required** | `events.Event` dispatch model (sync vs. async), subscriber priority, event payload size limits |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — lifecycle events originate from entity system) |
| **Enables** | ADR-0010 (Thread Investigation — contradiction events), Persistent World State (world state change events) |
| **Blocks** | Any system that subscribes to entity lifecycle or world state events |
| **Ordering Note** | Must be Accepted before systems that produce/consume events are implemented |

## Context

### Problem Statement
22 systems must communicate without tight coupling. When an entity dies, Persistent World State must record it, Trust & Relationship must fire trust events, Combat UI must remove the entity from turn order, and Thread Investigation must check for broken thread chains. Each system cannot call every other system directly. How should cross-system events be dispatched?

### Constraints
- Events must be dispatched within the same frame (no delayed/queued events in normal operation)
- Systems must subscribe declaratively (not poll)
- Event payloads must be lightweight (no heap allocation in hot path)
- Must support Kaiju's native `events.Event` system
- Write arbitration for same-frame component modifications (Entity Framework CR-5)

### Requirements
- Entity lifecycle events: created, destroyed, suspended, reactivated
- Component mutation events: added, removed, field changed
- World state events: flag set, memory record added
- Trust events: trust increased, trust decreased, betrayed
- Combat events: combat started, turn changed, combat ended, entity died
- Thread events: thread created, contradiction detected, revelation threshold reached

## Decision

**Synchronous event dispatch via Kaiju `events.Event` with system priority ordering.** Events fire and all subscribers are notified in the same frame. Subscribers register with a priority value; higher-priority subscribers receive the event first. This ensures deterministic ordering for dependent systems (e.g., Persistent World State processes death before Trust & Relationship fires trust events).

### Architecture Diagram

```
┌──────────────────────────────────────────────────┐
│  EVENT PRODUCER (any system)                     │
│  eventBus.Emit(Event{Type: "entity_died", ...})  │
└──────────────────┬───────────────────────────────┘
                   │
┌──────────────────▼───────────────────────────────┐
│  EVENT BUS                                       │
│                                                  │
│  Subscribers by event type + priority:           │
│  "entity_died" → [                               │
│    {priority: 1, handler: PWS.onEntityDied},     │
│    {priority: 5, handler: Trust.onEntityDied},   │
│    {priority: 10, handler: CombatUI.onEntityDied} │
│  ]                                               │
│                                                  │
│  Dispatch: iterate subscribers in priority order  │
│  (1 = highest). Synchronous — same frame.         │
└──────────────────────────────────────────────────┘
```

### Key Interfaces

```go
// EventType is a namespaced string identifier.
type EventType string

const (
    EventEntityCreated    EventType = "entity.created"
    EventEntityDestroyed  EventType = "entity.destroyed"
    EventEntitySuspended  EventType = "entity.suspended"
    EventEntityReactivated EventType = "entity.reactivated"
    EventComponentAdded   EventType = "component.added"
    EventComponentRemoved EventType = "component.removed"
    EventComponentChanged EventType = "component.changed"
    EventWorldFlagSet     EventType = "world.flag_set"
    EventMemoryRecorded   EventType = "world.memory_recorded"
    EventTrustChanged     EventType = "trust.changed"
    EventCombatStarted    EventType = "combat.started"
    EventCombatEnded      EventType = "combat.ended"
    EventTurnChanged      EventType = "combat.turn_changed"
    EventThreadCreated    EventType = "thread.created"
    EventContradictionDetected EventType = "thread.contradiction"
    EventRevelationReady  EventType = "thread.revelation_ready"
)

// Event is a lightweight event payload.
type Event struct {
    Type      EventType
    Timestamp float64 // game time in minutes
    Source    string  // system that emitted the event
    Data      any     // type-specific payload — consumer casts
}

// EventBus is the central event dispatcher.
type EventBus interface {
    // Subscribe registers a handler for an event type with a priority.
    // Lower priority = higher precedence. Priority 1 dispatches first.
    Subscribe(eventType EventType, priority int, handler EventHandler)

    // Emit dispatches an event to all subscribers synchronously.
    Emit(event Event)
}

// EventHandler receives events.
type EventHandler func(event Event)
```

**System priority assignments (entity lifecycle):**

| Priority | System | Reason |
|----------|--------|--------|
| 1 | Entity Framework | Owns the lifecycle — processes the transition first |
| 2 | Persistent World State | Records the event in memory log before downstream systems react |
| 3 | Trust & Relationship | Fires trust events based on what happened |
| 4 | Thread Investigation | Checks for broken chains, contradiction triggers |
| 5 | Combat UI / Dialogue UI | Updates display (lowest — presentation is last) |

## Alternatives Considered

### Alternative 1: Synchronous dispatch with priority (CHOSEN)
- **Pros**: Deterministic ordering. All systems see the state change in the same frame. No stale state. Easy to reason about — events are function calls with extra indirection.
- **Cons**: Slow subscriber blocks all subsequent subscribers. Recursive event emission could cause infinite loops. Single-frame dispatch means all event handlers must complete within the frame budget (16.6ms).
- **Rejection Reason**: N/A — chosen.

### Alternative 2: Asynchronous event queue
- **Description**: Events are enqueued and processed on the next frame or a worker goroutine.
- **Pros**: No subscriber blocks the producer. Burst tolerance — queue absorbs spikes.
- **Cons**: Stale state — System B reads component data that System A hasn't updated yet because the event is queued. Non-deterministic ordering. Harder to debug. Requires queue management, backpressure, overflow handling.
- **Rejection Reason**: Non-deterministic ordering is unacceptable for game state. A death must be recorded BEFORE trust events fire. Synchronous dispatch guarantees this; async does not.

### Alternative 3: Direct method calls (no event bus)
- **Description**: Each system directly calls methods on dependent systems.
- **Pros**: No indirection. Compile-time type checking. Fastest possible dispatch.
- **Cons**: Tight coupling — Entity Framework must import Persistent World State, which imports Trust, which imports Thread Investigation. Circular dependency risk. Adding a new subscriber requires modifying the producer. Untestable in isolation.
- **Rejection Reason**: Tight coupling defeats the purpose of the layered architecture. The event bus decouples producers from consumers — Entity Framework doesn't know Persistent World State exists.

## Consequences

### Positive
- Loose coupling: producers and consumers know only the EventBus interface
- Deterministic ordering: priority system guarantees correct event sequence
- Testable: systems can be tested with a mock EventBus that records emitted events
- Extensible: new subscribers register without modifying existing code

### Negative
- Runtime overhead: map lookup + slice iteration per event (~100ns per subscriber)
- No compile-time safety: event Data is `any` — consumer must cast correctly
- Recursive event risk: handler that emits an event of the same type must be guarded

### Risks
- **Frame budget**: All synchronous handlers must complete within 16.6ms. Mitigation: handlers do minimal work — set flags, not recalculate. Heavy work deferred to system update.
- **Infinite recursion**: Handler A emits event that triggers Handler B which emits event that triggers Handler A. Mitigation: re-entrancy detection — if the same event type is emitted during its own dispatch, the recursive emission is suppressed with a warning.
- **Event payload allocation**: `any` boxing allocates on heap. Mitigation: use value types for hot-path events (entity lifecycle); limit heap-allocated events to <10% of event volume.

## GDD Requirements Addressed

| GDD System | Requirement | How This ADR Addresses It |
|------------|-------------|--------------------------|
| Entity Framework | CR-5: Lifecycle Events — broadcast to subscribed systems | Synchronous event dispatch via EventBus with priority ordering |
| Persistent World State | Event Processing Pipeline — validate, mutate, log, broadcast | PWS subscribes at priority 2 for entity lifecycle events |
| Trust & Relationship | Trust events triggered by thread resolution and combat outcomes | Trust subscribes at priority 3 to entity destroyed, world flag set, and thread resolved events |
| Thread Investigation | Thread creation, contradiction detection triggered by fact discovery | Thread subscribes to component.changed (PlayerKnowledge) and trust.changed events |

## Performance Implications
- **CPU**: Event dispatch: ~100ns per subscriber (map lookup + function call). With 5 subscribers per event type, total <1µs per event. Frame budget: all event handlers must complete within 2ms combined.
- **Memory**: EventBus: ~2KB for subscriber registry (22 event types × ~5 subscribers × ~20 bytes each). Event payloads: stack-allocated for value types; ~50 heap allocations per frame for complex events.
- **Load Time**: N/A — subscriber registration at init, <1ms.
- **Network**: N/A.

## Migration Plan
No migration — first implementation. All cross-system communication uses EventBus from the start. Direct system-to-system calls are permitted only for read-only queries (e.g., Combat reads Character Stats) — never for state mutations.

## Validation Criteria
- Entity death → all 5 lifecycle subscribers receive `entity.destroyed` in priority order within one frame
- Recursive event emission: handler A emits event X during dispatch of event X → suppressed with warning
- Mock EventBus: system under test emits expected events without depending on real subscribers
- Frame budget: all event handlers for a single event complete within 2ms

## Related Decisions
- ADR-0001: Entity Architecture — lifecycle events originate from entity system
- ADR-0002: Save/Load — events are not persisted (world state is the persistent record, not the event stream)
