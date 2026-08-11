# Story 001: Thread Data Model & Auto-Creation

> **Epic**: Thread Investigation
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (4–6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/thread-investigation.md`
**Requirements**:
- `TR-thread-001` — Thread creation: auto-created from fact `leads_to` hints, entity references, contradictions, and manual player input

**ADR Governing Implementation**: ADR-0010: Thread Investigation Engine
**ADR Decision Summary**: In-memory thread graph with PlayerKnowledge component persistence. Thread creation is event-driven — when a fact is discovered via the Question System, `OnFactDiscovered` checks for new threads, entity overlaps, contradictions, and revelation thresholds. Threads, chains, and discovered facts are maintained in a graph structure at runtime.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Pure game logic and data structures — no Kaiju APIs required. PlayerKnowledge component will use Entity Framework (ADR-0001) for registration; this story defines the data model. Post-cutoff risk is engine-agnostic.

**Control Manifest Rules (Feature Layer)**:
- Required: Event-driven architecture — thread creation triggered by fact discovery events from Question System
- Required: Synchronous event dispatch via EventBus with system priority ordering (ADR-0011)
- Forbidden: Never use a traditional quest log — violates anti-pillar "NOT a completionist checklist" (ADR-0010)
- Guardrail: `OnFactDiscovered`: <5ms (thread creation + chain assignment + contradiction check across 100 discovered facts, ADR-0010)

---

## Acceptance Criteria

*From GDD `design/gdd/thread-investigation.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN a fact with `leads_to: [{entity_id: orin_id, topic_hint: "Ley-touched metal"}]`, WHEN the player discovers that fact, THEN an active thread "Ask Orin about Ley-touched metal" is created with status `active`.

- [ ] **AC-2**: GIVEN a fact with multiple `leads_to` entries, WHEN the player discovers the fact, THEN one thread is created per lead.

- [ ] **AC-3**: GIVEN a fact with no `leads_to` entries, WHEN the player discovers the fact, THEN no thread is created (fact is still recorded in `DiscoveredFacts`).

- [ ] **Performance**: GIVEN PlayerKnowledge with 50 discovered facts, WHEN `OnFactDiscovered` is called, THEN thread creation step completes within 2ms.

---

## Implementation Notes

*Derived from ADR-0010 Implementation Guidelines:*

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
    ChainID      string       // which investigation chain this belongs to (set in Story 002)
    CreatedAt    float64      // game timestamp
}

type Chain struct {
    ID      string   // unique chain identifier
    Summary string   // auto-generated description
    Facts   []string // fact_ids discovered in this chain
    Threads []string // thread_ids in this chain
    Status  string   // "active", "revelation_ready", "resolved"
}

type DiscoveredFact struct {
    FactID       string
    SourceNPC    GameEntityID
    DiscoveredAt float64
}

type PlayerKnowledge struct {
    DiscoveredFacts []DiscoveredFact
    ActiveThreads   []Thread
    ResolvedThreads []Thread
    Chains          []Chain
    KnownEntities   []KnownEntity
    Revelations     []Revelation
}

type KnownEntity struct {
    EntityID     GameEntityID
    DiscoveredAt float64
    DiscoveredVia string // fact_id that introduced this entity
}

type Revelation struct {
    ID          string
    ChainID     string
    DiscoveredAt float64
    ConfrontedAt float64 // 0 = not yet confronted
}
```

### Core Interface

```go
type InvestigationEngine interface {
    // OnFactDiscovered is called when the player learns a new fact.
    OnFactDiscovered(factID string, sourceNPC GameEntityID)

    // GetActiveThreads returns all active and prioritized threads.
    GetActiveThreads() []Thread

    // GetChain returns a specific investigation chain.
    GetChain(chainID string) (Chain, bool)

    // ResolveThread marks a thread as resolved.
    ResolveThread(threadID string, resolution ThreadStatus)

    // GetRevelations returns all chains ready for revelation.
    GetRevelations() []Revelation
}
```

### OnFactDiscovered Algorithm (Step 1–2)

```
OnFactDiscovered(fact, sourceNPC):
  1. Record fact in PlayerKnowledge.DiscoveredFacts
  2. Record target entities in KnownEntities (if new)
  3. If fact.LeadsTo is non-empty:
       For each lead in fact.LeadsTo:
         CreateThread(
           target=lead.EntityID,
           topic=lead.TopicHint,
           source=fact.ID
         )
         Emit EventThreadCreated
```

This story implements steps 1–2 of the full algorithm. The `ChainID` field on `Thread` is populated in Story 002 (auto-chaining). Contradiction and revelation checks are implemented in Stories 003 and 004.

### Thread ID Generation

Thread IDs use the format: `thread_{source_fact_id}_{target_entity_id}`. This ensures idempotency — discovering the same fact twice does not create duplicate threads (the second `CreateThread` call detects the existing ID and returns the existing thread).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Auto-chaining — chain assignment and entity overlap detection
- Story 003: Contradiction detection — `contradicts` field checking and contradiction thread creation
- Story 004: Revelation threshold and thread resolution states (resolved, dead_end, etc.)
- Story 005: PlayerKnowledge component registration, pod serialization, save/load round-trip
- Manual thread creation (player types their own thread) — Story 002 provides the interface hook

---

## QA Test Cases

*Test cases for automated unit testing — developer implements against these:*

- **AC-1**: Thread creation from leads_to hint
  - Given: A fact with `leads_to: [{entity_id: 42, topic_hint: "Ley-touched metal"}]`, NPC entity ID 7
  - When: `OnFactDiscovered("fact_001", 7)` is called
  - Then: One active thread is created with `Topic = "Ask [entity 42] about Ley-touched metal"`, `Source = "fact_001"`, `Status = "active"`, `TargetEntity = 42`
  - Edge cases: Entity name resolution for topic text; empty topic_hint; target entity does not exist

- **AC-2**: Multiple leads_to entries
  - Given: A fact with `leads_to: [{entity_id: 10, topic_hint: "the old mine"}, {entity_id: 11, topic_hint: "Captain Veren"}]`
  - When: `OnFactDiscovered("fact_002", 3)` is called
  - Then: Two threads are created, one per lead; each has the correct target and topic

- **AC-3**: Fact with no leads_to
  - Given: A fact with empty `leads_to` (no hints)
  - When: `OnFactDiscovered("fact_003", 5)` is called
  - Then: No thread is created; the fact is recorded in `DiscoveredFacts` with `FactID = "fact_003"`, `SourceNPC = 5`

- **Performance**: Thread creation speed
  - Given: PlayerKnowledge with 50 discovered facts pre-loaded
  - When: `OnFactDiscovered` is called 1000 times under Go benchmark
  - Then: p95 thread creation step ≤ 2ms

---

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/thread_investigation/thread_creation_test.go` — must exist and pass

**Status**: [x] Created — `tests/unit/thread_investigation/thread_creation_test.go` (23 tests, all PASS, benchmark ~8µs/op, go vet clean)

---

## Dependencies

- Depends on: NPC Knowledge Graph (KG-001 — fact data model including `leads_to` field; KG-005 — Knowledge component integration)
- Depends on: Entity Framework (EF-001, EF-002 — GameEntityID type, component store registration pattern)
- Unlocks: Story 002 (Auto-Chaining needs Thread and Chain data model)

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 4/4 passing
**Deviations**: None — implementation follows ADR-0010 exactly, manifest versions match (2026-08-09), all Feature Layer control manifest rules respected.
**Test Evidence**: Logic — `tests/unit/thread_investigation/thread_creation_test.go` (23 tests, all PASS, benchmark ~8µs/op ~250× under 2ms budget, go vet clean)
**Code Review**: Complete — APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester, 2026-08-11). All 6 QA gaps closed, all engine suggestions implemented.
**Unlocks**: Story 002 (Auto-Chaining & Entity Overlap)
