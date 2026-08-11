# Story 004: Revelation Threshold & Thread Resolution

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
- `TR-thread-004` — Revelation threshold: 3+ connected facts pointing to same conclusion → chain status = `revelation_ready`

**ADR Governing Implementation**: ADR-0010: Thread Investigation Engine
**ADR Decision Summary**: When 3+ facts in the same chain all point to the same conclusion (same `leads_to` target, or all contradict the same official story), the chain status becomes `revelation_ready` and `EventRevelationReady` is emitted. Thread resolution supports four types: `discovered`, `confronted`, `dead_end`, and `merged`. NPC death auto-resolves related threads as `dead_end`.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Pure game logic — no Kaiju APIs. NPC death detection listens for entity destruction events via EventBus (ADR-0011). No post-cutoff API risk.

**Control Manifest Rules (Feature Layer)**:
- Required: Event-driven architecture — `EventRevelationReady` emitted via EventBus for Investigation Journal to display
- Required: Synchronous event dispatch with system priority ordering
- Forbidden: Never use a traditional quest log — revelations are not quest completions, they are investigation milestones
- Guardrail: `OnFactDiscovered` (full algorithm): <5ms (ADR-0010)

---

## Acceptance Criteria

*From GDD `design/gdd/thread-investigation.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN 3 connected facts pointing to the same conclusion in one chain, WHEN the third fact is discovered, THEN the system suggests a revelation is available (chain status → `revelation_ready`, `EventRevelationReady` emitted).

- [ ] **AC-2**: GIVEN a chain with only 2 facts pointing to the same conclusion, WHEN the second fact is discovered, THEN the chain remains `active` (revelation threshold NOT met).

- [ ] **AC-3**: GIVEN a thread's target NPC dies, WHEN the death event fires, THEN the thread auto-resolves as `dead_end` with the reason recorded.

- [ ] **AC-4**: GIVEN the player discovers the fact a thread pointed to, WHEN `ResolveThread` is called with `discovered`, THEN the thread resolves to history and the fact is recorded as discovered via that thread.

- [ ] **AC-5**: GIVEN the player manually marks a thread as unresolvable, WHEN `ResolveThread` is called with `dead_end`, THEN the thread moves to history with a dead-end marker (may be reactivated if new evidence emerges).

---

## Implementation Notes

*Derived from ADR-0010 Implementation Guidelines:*

### Revelation Check Algorithm (Step 5)

```
OnFactDiscovered — Step 5 (builds on Steps 1–4):
  5. CheckRevelation(chain):
       a. Count facts in chain that share the same leads_to target entity
       b. OR count facts that all contradict the same fact_id
       c. If count >= revelation_threshold (default 3):
            chain.Status = "revelation_ready"
            Emit EventRevelationReady{
              ChainID:   chain.ID,
              FactCount: count,
              TargetEntity: sharedTarget,
              Summary:   chain.Summary
            }
       d. If chain is already "revelation_ready", skip — no duplicate events
```

### Thread Resolution

```go
func (e *InvestigationEngine) ResolveThread(threadID string, resolution ThreadStatus) {
    thread := e.findThread(threadID)
    switch resolution {
    case ThreadResolved:  // "discovered"
        thread.Status = ThreadResolved
        e.knowledge.ResolvedThreads = append(e.knowledge.ResolvedThreads, thread)
        e.removeActiveThread(threadID)

    case ThreadDeadEnd:
        thread.Status = ThreadDeadEnd
        e.knowledge.ResolvedThreads = append(e.knowledge.ResolvedThreads, thread)
        e.removeActiveThread(threadID)
        // Thread may be reactivated if new evidence emerges (CR-4)

    case ThreadPrioritized:
        thread.Status = ThreadPrioritized
        // Player marks thread as priority — stays in active list, sorted to top
    }
    Emit EventThreadResolved(threadID, resolution)
}
```

### NPC Death Auto-Resolution (GDD AC-5, CR-4)

```go
// Subscribe to entity destruction events on initialization
func (e *InvestigationEngine) OnEntityDestroyed(event EntityDestroyedEvent) {
    for _, thread := range e.knowledge.ActiveThreads {
        if thread.TargetEntity == event.EntityID {
            e.ResolveThread(thread.ID, ThreadDeadEnd)
            // Record the reason
            log.Infof("Thread %s auto-resolved: target entity %d destroyed (%s)",
                thread.ID, event.EntityID, event.Reason)
        }
    }
}
```

### Revelation Event

```go
type EventRevelationReady struct {
    ChainID      string
    FactCount    int
    TargetEntity GameEntityID // the entity/conclusion the facts point to
    Summary      string       // chain summary
}
```

### Tuning Knobs

| Parameter | Default | Range | Effect |
|-----------|---------|-------|--------|
| `revelation_threshold` | 3 facts | 2–5 | Lower = revelations trigger faster (GDD tuning knob) |

### Revelation Spam Prevention

Per ADR-0010 risk mitigation: once a chain is `revelation_ready`, additional facts discovered in that chain do NOT emit new `EventRevelationReady`. The revelation is a one-time signal. Additional facts may deepen the revelation (more evidence), but the signal fires once.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Thread data model and auto-creation
- Story 002: Auto-chaining and entity overlap
- Story 003: Contradiction detection
- Story 005: PlayerKnowledge persistence and save/load round-trip
- Investigation Journal: "amber glow" visual feedback for revelation_ready chains (Presentation layer)
- Confrontation: Acting on a revelation (the "confront" resolution type) — handled by Dialogue system

---

## QA Test Cases

*Test cases for automated unit testing — developer implements against these:*

- **AC-1**: Revelation threshold met (3 facts)
  - Given: A chain "chain_01" has 2 facts both pointing to entity 42; a third fact is discovered also pointing to entity 42
  - When: `OnFactDiscovered` runs and `CheckRevelation` is called
  - Then: `chain.Status = "revelation_ready"`; `EventRevelationReady` is emitted with `FactCount = 3`, `TargetEntity = 42`; revelation recorded in `PlayerKnowledge.Revelations`
  - Edge cases: Exactly 3 facts (boundary); 5+ facts (still fires once)

- **AC-2**: Revelation threshold NOT met (2 facts)
  - Given: A chain "chain_02" has 1 fact pointing to entity 55; a second fact pointing to entity 55 is discovered
  - When: `CheckRevelation` runs
  - Then: Chain status remains `active`; no `EventRevelationReady` emitted; no revelation recorded
  - Edge cases: 2 facts pointing to different entities in same chain (no revelation — need same target)

- **AC-3**: NPC death auto-resolution
  - Given: An active thread targets entity 42; `EntityDestroyedEvent{EntityID: 42, Reason: "combat_death"}` fires
  - When: The InvestigationEngine receives the event
  - Then: Thread resolves as `dead_end`; thread moves to `ResolvedThreads`; death reason recorded
  - Edge cases: Thread already resolved (no-op); NPC has multiple threads targeting them (all resolve)

- **AC-4**: Thread resolution — discovered
  - Given: An active thread with ID "thread_01" targeting entity 42 exists
  - When: `ResolveThread("thread_01", ThreadResolved)` is called
  - Then: Thread status = `resolved`; moved from `ActiveThreads` to `ResolvedThreads`; `EventThreadResolved` emitted
  - Edge cases: Resolve already-resolved thread (no-op, log warning); resolve non-existent thread ID (error)

- **AC-5**: Thread resolution — dead_end (manual)
  - Given: An active thread with ID "thread_02"
  - When: `ResolveThread("thread_02", ThreadDeadEnd)` is called
  - Then: Thread status = `dead_end`; moved to `ResolvedThreads`; dead-end marker set
  - Edge cases: Thread reactivation on new evidence (new fact references same entity → thread moves back to active)

---

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/thread_investigation/revelation_resolution_test.go` — must exist and pass

**Status**: [x] Created — `tests/unit/thread_investigation/revelation_resolution_test.go` (15 tests + benchmark, all PASS, ~842ns/op, go vet clean)

---

## Dependencies

- Depends on: Story 001 (Thread data model, OnFactDiscovered)
- Depends on: Story 002 (Auto-chaining — revelation check operates on chains)
- Depends on: Story 003 (Contradiction detection — contradiction facts contribute to revelation count)
- Depends on: EventBus (ADR-0011 — EntityDestroyedEvent subscription + EventRevelationReady emission)
- Unlocks: Story 005 (PlayerKnowledge persistence needs all data model fields populated)

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 5/5 passing
**Deviations**: ADVISORY — `OnEntityDestroyed(entityID, reason)` added to InvestigationEngine interface (event-driven pattern deferred to Story 005 EventBus wiring). Death reason in event, not Thread struct. `revelationGroup` refactored into 4 helpers for standards compliance.
**Test Evidence**: Logic — `tests/unit/thread_investigation/revelation_resolution_test.go` (17 tests + benchmark, all PASS, ~844ns/op). `go vet` clean.
**Code Review**: Complete — APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester). revelationGroup refactored in-review.
**Unlocks**: Story 005 (PlayerKnowledge Persistence & Integration)
