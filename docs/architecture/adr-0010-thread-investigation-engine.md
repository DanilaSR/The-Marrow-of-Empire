# ADR-0010: Thread Investigation Engine

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Core / Scripting |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | Thread Investigation GDD, NPC Knowledge Graph GDD, Paper Prototype Report |
| **Post-Cutoff APIs Used** | None — pure game logic |
| **Verification Required** | None — data model and algorithms are engine-agnostic |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — PlayerKnowledge component), ADR-0005 (NPC Knowledge Graph — fact references and leads_to), ADR-0011 (Event Bus — contradiction and revelation events) |
| **Enables** | Investigation Journal (display) |
| **Blocks** | Thread Investigation implementation |
| **Ordering Note** | Must be Accepted before investigation system development |

## Context

### Problem Statement
The Marrow of Empire replaces the traditional quest log with player-driven thread tracking. Threads are created from NPC fact references, auto-chained by entity overlap, cross-referenced for contradictions, and resolved when enough connected facts point to a conclusion. How should threads be modeled, chained, and checked for revelations?

### Constraints
- Must create threads automatically from NPC fact `leads_to` hints and entity references
- Must detect when two facts from different NPCs reference the same entity
- Must detect contradictions (fact A contradicts fact B) and flag them
- Must detect when 3+ connected facts point to the same conclusion (revelation threshold)
- Must persist all thread state in the PlayerKnowledge component for save/load
- Paper prototype validated the 5-NPC thread chain design

### Requirements
- Thread: a lead the player can follow (ask X about Y, find location Z)
- Chain: a group of related threads sharing entity references
- Contradiction: two discovered facts where one has `contradicts: [other_fact_id]`
- Revelation: 3+ connected facts pointing to the same conclusion

## Decision

**In-memory thread graph with PlayerKnowledge component persistence.** Threads, chains, and discovered facts are maintained in a graph structure at runtime and serialized to the PlayerKnowledge component for save/load. Thread creation is event-driven: when a fact is discovered (via Question System), the Thread Investigation system checks for new threads, entity overlaps, contradictions, and revelation thresholds.

### Data Model

```go
type ThreadStatus string
const (
    ThreadActive      ThreadStatus = "active"
    ThreadPrioritized ThreadStatus = "prioritized"
    ThreadResolved    ThreadStatus = "resolved"
    ThreadDeadEnd     ThreadStatus = "dead_end"
)

type Thread struct {
    ID           string       // unique thread identifier
    Source       string       // which fact/NPC created this thread
    TargetEntity GameEntityID // who or what to investigate
    Topic        string       // what to ask about
    Status       ThreadStatus
    ChainID      string       // which investigation chain this belongs to
    CreatedAt    float64      // game timestamp
}

type Chain struct {
    ID      string   // unique chain identifier
    Summary string   // auto-generated: "Aldric's Death"
    Facts   []string // fact_ids discovered in this chain
    Threads []string // thread_ids in this chain
    Status  string   // "active", "revelation_ready", "resolved"
}

type PlayerKnowledge struct {
    DiscoveredFacts  []DiscoveredFact
    ActiveThreads    []Thread
    ResolvedThreads  []Thread
    Chains           []Chain
    KnownEntities    []KnownEntity
    Revelations      []Revelation
}

type DiscoveredFact struct {
    FactID     string
    SourceNPC  GameEntityID
    DiscoveredAt float64
}
```

### Auto-Chaining Algorithm

```
OnFactDiscovered(fact, sourceNPC):
  1. Record fact in PlayerKnowledge.DiscoveredFacts
  2. If fact.LeadsTo is non-empty:
       For each lead in fact.LeadsTo:
         CreateThread(target=lead.EntityID, topic=lead.TopicHint, source=fact.ID)
  3. Assign thread to a chain:
       If any existing chain contains a fact referencing the same entity:
         Assign to that chain
       Else:
         Create new chain
  4. If fact.Contradicts is non-empty:
       For each contradicted fact_id:
         If that fact is in PlayerKnowledge.DiscoveredFacts:
           CreateThread(type="contradiction", topic="Resolve: A says X, B says Y")
           Emit EventContradictionDetected
  5. CheckRevelation(chain):
       If chain has 3+ facts all pointing to the same target entity or conclusion:
         chain.Status = "revelation_ready"
         Emit EventRevelationReady
```

### Key Interfaces

```go
type InvestigationEngine interface {
    // OnFactDiscovered is called when the player learns a new fact.
    OnFactDiscovered(factID string, sourceNPC GameEntityID)

    // GetActiveThreads returns all active and prioritized threads.
    GetActiveThreads() []Thread

    // GetChain returns a specific investigation chain.
    GetChain(chainID string) (Chain, bool)

    // ResolveThread marks a thread as resolved (player followed the lead).
    ResolveThread(threadID string, resolution ThreadStatus)

    // GetRevelations returns all chains ready for revelation.
    GetRevelations() []Revelation
}
```

## Alternatives Considered

### Alternative 1: Event-driven thread graph (CHOSEN)
- **Pros**: Reactive — threads update automatically on fact discovery. Persistable — graph serializes to PlayerKnowledge. Extensible — new event types add new thread behaviors. Validated by paper prototype.
- **Cons**: Auto-chaining may create false connections (two facts reference the same entity but are unrelated). Edge cases in contradiction detection. Graph consistency must be maintained across save/load.
- **Rejection Reason**: N/A.

### Alternative 2: Manual thread management only
- **Description**: No auto-chaining or auto-detection. Player manually creates all threads and connections.
- **Pros**: No false connections. Player has complete control.
- **Cons**: High cognitive load — player must remember and manually connect everything. Missed connections = frustrated players. Contradiction detection left entirely to player memory.
- **Rejection Reason**: The game's core promise is "feel like a detective." Auto-detection of connections and contradictions is what makes the player feel smart — the system surfaces the clues, the player makes the deduction.

### Alternative 3: Quest log with auto-population
- **Description**: Traditional quest log that auto-populates objectives from NPC dialogue.
- **Pros**: Familiar. Predictable. Easy to implement.
- **Cons**: Exactly what the game's anti-pillars forbid — "NOT a completionist checklist." Quest markers and objective lists undermine the investigation fantasy.
- **Rejection Reason**: Violates the game's core design philosophy. The thread system IS the quest system replacement.

## Consequences

### Positive
- Automatic thread creation reduces player bookkeeping
- Entity overlap detection surfaces connections the player might miss
- Contradiction detection creates investigation tension
- Revelation threshold gates major story moments behind genuine investigation

### Negative
- Auto-chaining false positives: two facts about the same entity may be unrelated
- Graph complexity grows with discovered facts — chain maintenance overhead
- Player may feel led if auto-detection is too aggressive

### Risks
- **False chain merges**: Entity overlap may merge unrelated chains. Mitigation: chains can be manually split by the player (via Investigation Journal). Auto-merge can be disabled via tuning knob.
- **Revelation spam**: Too many facts pointing to the same conclusion may trigger multiple revelation prompts. Mitigation: revelation threshold is per chain; once a chain is "revelation_ready," additional facts don't trigger new prompts.

## GDD Requirements Addressed

| GDD System | Requirement | How This ADR Addresses It |
|------------|-------------|--------------------------|
| Thread Investigation | CR-1: Thread Creation — from leads_to, entity mentions, manual, contradictions | OnFactDiscovered creates threads from fact metadata |
| Thread Investigation | CR-2: Investigation Chains — grouped by entity overlap, auto-merge | Auto-chaining algorithm with entity reference overlap detection |
| Thread Investigation | CR-3: Cross-Referencing — entity overlap, contradiction detection, revelation threshold | Checked on every fact discovery; events emitted for contradictions and revelations |
| Thread Investigation | CR-4: Thread Resolution — discovered, confronted, dead_end, merged | ResolveThread with resolution type |
| Thread Investigation | CR-5: Player Knowledge Component — discovered facts, threads, chains, revelations | PlayerKnowledge struct persisted via Entity Framework |

## Performance Implications
- **CPU**: OnFactDiscovered: <5ms (thread creation + chain assignment + contradiction check across 100 discovered facts).
- **Memory**: ~2KB per active thread. ~50KB for PlayerKnowledge with 20 threads, 5 chains, 50 discovered facts.
- **Load Time**: PlayerKnowledge deserialization: <1ms.
- **Network**: N/A.

## Migration Plan
First implementation. No existing thread data to migrate.

## Validation Criteria
- NPC fact with leads_to → thread created automatically
- Two facts referencing same entity → assigned to same chain
- Fact with contradicts pointing to discovered fact → contradiction thread created + event emitted
- Chain with 3 connected facts pointing to same target → revelation_ready status
- Save/load round-trip: all threads, chains, and discovered facts restored identically

## Related Decisions
- ADR-0005: NPC Knowledge Graph — fact references and leads_to are the input to thread creation
- ADR-0011: Event Bus — contradiction and revelation events for downstream systems
