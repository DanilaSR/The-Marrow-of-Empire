# Story 002: Auto-Chaining & Entity Overlap

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
- `TR-thread-002` — Auto-chaining: threads sharing entity references grouped into investigation chains; chains auto-merge on entity overlap

**ADR Governing Implementation**: ADR-0010: Thread Investigation Engine
**ADR Decision Summary**: Threads that reference the same entities or share `leads_to` targets are grouped into investigation chains. A chain is a set of related threads that together form a coherent line of inquiry. Threads are auto-assigned to a chain when they share at least one entity reference with an existing thread in that chain. Chains can merge when two chains are discovered to be about the same entity.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Pure data model and set-intersection algorithms — no Kaiju APIs required. Entity reference comparison uses `GameEntityID` (uint64) equality from ADR-0001.

**Control Manifest Rules (Feature Layer)**:
- Required: Event-driven thread graph — auto-chaining triggered on thread creation, chain merge emits events
- Forbidden: Never use a traditional quest log — chains are not quest objectives
- Guardrail: Entity overlap check across 100 discovered facts <1ms (ADR-0010)

---

## Acceptance Criteria

*From GDD `design/gdd/thread-investigation.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN two discovered facts from different NPCs that both reference the same entity, WHEN the second fact is discovered, THEN the system flags the entity overlap: "Both [NPC A] and [NPC B] mentioned [entity]."

- [ ] **AC-2**: GIVEN a newly created thread (from Story 001) that references entity X, WHEN an existing chain already contains facts referencing entity X, THEN the thread is auto-assigned to that chain.

- [ ] **AC-3**: GIVEN a newly created thread that references an entity not in any existing chain, WHEN the thread is created, THEN a new chain is created containing that thread.

- [ ] **AC-4**: GIVEN two chains that both reference the same entity, WHEN the overlap is detected, THEN the chains auto-merge into one chain containing all facts and threads from both.

- [ ] **Performance**: GIVEN 100 discovered facts distributed across 5 chains, WHEN entity overlap is checked for a new fact, THEN the check completes within 1ms.

---

## Implementation Notes

*Derived from ADR-0010 Implementation Guidelines:*

### Auto-Chaining Algorithm (Step 3)

```
OnFactDiscovered — Step 3 (builds on Story 001 Steps 1–2):
  3. For each newly created thread:
       a. Collect all entity IDs referenced by the thread's source fact
       b. Scan existing chains for any facts referencing the same entities
       c. If match found:
            Assign thread to that chain
            Emit EventThreadChained(threadID, chainID)
       d. Else:
            Create new chain with this thread
            chain.Summary = autoGenerateSummary(fact)
            Emit EventChainCreated(chainID)
  3b. Chain merge check:
       For each pair of chains:
         If they share at least one entity reference:
           Merge chains (union of facts + threads)
           Emit EventChainsMerged(chainA_ID, chainB_ID, mergedChainID)
```

### Chain Summary Generation

```go
func autoGenerateSummary(fact Fact) string {
    // Derive a short description from the fact's topic
    // Example: fact about "Aldric's death" → "Aldric's Death"
    // Example: fact about "old mine ore" → "The Old Mine"
    return fact.TopicHint  // initial implementation: use the first fact's topic hint
}
```

### Entity Overlap Flagging

When a new fact's entities overlap with facts from a different NPC in the same chain:

```go
func checkEntityOverlap(newFact Fact, existingFacts []DiscoveredFact) []OverlapFlag {
    var flags []OverlapFlag
    for _, existing := range existingFacts {
        if existing.SourceNPC == newFact.SourceNPC {
            continue // skip same NPC
        }
        overlap := intersectEntities(newFact.EntityRefs, existing.EntityRefs)
        if len(overlap) > 0 {
            flags = append(flags, OverlapFlag{
                Entity:     overlap[0],
                NPC_A:      existing.SourceNPC,
                NPC_B:      newFact.SourceNPC,
                Message:    fmt.Sprintf("Both %s and %s mentioned %s.", npcNameA, npcNameB, entityName),
            })
        }
    }
    return flags
}
```

### Chain Merge Logic

When chain merge is detected:
1. Combine all facts from both chains (deduplicate by fact_id)
2. Combine all threads from both chains
3. Update all merged threads' `ChainID` to the surviving chain ID
4. Old chain is archived (not deleted — investigation history)
5. Recalculate chain summary to reflect merged scope

### Tuning Knobs

| Parameter | Default | Range | Effect |
|-----------|---------|-------|--------|
| `auto_chain_merge` | `true` | bool | Whether chains auto-merge on entity overlap (GDD tuning knob) |

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Thread data model and auto-creation from `leads_to`
- Story 003: Contradiction detection — `contradicts` field checking
- Story 004: Revelation threshold and thread resolution
- Story 005: PlayerKnowledge persistence and save/load
- Manual chain splitting (player can split false merges via Investigation Journal) — Presentation layer

---

## QA Test Cases

*Test cases for automated unit testing — developer implements against these:*

- **AC-1**: Entity overlap flagging
  - Given: `DiscoveredFacts` contains fact "fact_001" from NPC 7 referencing entity 42, and a new fact "fact_002" from NPC 3 also references entity 42
  - When: `OnFactDiscovered("fact_002", 3)` is called
  - Then: An overlap flag is returned: "Both [NPC 7] and [NPC 3] mentioned [entity 42]"; both NPCs are listed
  - Edge cases: Same NPC (skip — no overlap flag for same source); fact references multiple entities with overlap on one

- **AC-2**: Thread auto-assigned to existing chain
  - Given: A chain "chain_01" exists containing facts that reference entity 42; a new thread is created referencing entity 42
  - When: The auto-chaining step runs
  - Then: The thread's `ChainID = "chain_01"`; `EventThreadChained` is emitted
  - Edge cases: Thread references multiple entities, some in chain — still assigned

- **AC-3**: New chain for unmatched entity
  - Given: No existing chain contains facts referencing entity 99; a new thread is created referencing entity 99
  - When: The auto-chaining step runs
  - Then: A new chain is created; `EventChainCreated` is emitted; the thread is the chain's first thread

- **AC-4**: Chain auto-merge
  - Given: Chain A has facts referencing entity 42 and 43; Chain B has facts referencing entity 43 and 44
  - When: A new fact referencing entity 43 is discovered, connecting both chains
  - Then: Chains A and B merge into one; all threads' `ChainID` updated to surviving chain; `EventChainsMerged` emitted
  - Edge cases: Chain A and B already merged (idempotent); more than 2 chains share entity

- **Performance**: Entity overlap check speed
  - Given: 100 discovered facts distributed across 5 chains
  - When: Entity overlap is checked for a new fact (1000 iterations under benchmark)
  - Then: p95 check time ≤ 1ms

---

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/thread_investigation/auto_chaining_test.go` — must exist and pass

**Status**: [x] Created — `tests/unit/thread_investigation/auto_chaining_test.go` (15 tests, all PASS, go vet clean)

---

## Dependencies

- Depends on: Story 001 (Thread data model and `OnFactDiscovered` Steps 1–2 — this story extends the algorithm with Step 3)
- Unlocks: Story 003 (Contradiction detection needs chain assignment to be working)

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 5/5 passing
**Deviations**: ADVISORY — `threadEntitySet` includes lead target entity in addition to fact References (superset of story note 3a, justified by AC-2 requirement). ADVISORY — `ChainEvent.Data` uses `any` (sealed interface deferred to Story 003/005).
**Test Evidence**: Logic — `tests/unit/thread_investigation/auto_chaining_test.go` (16 tests) + legacy `thread_creation_test.go` (22 tests) = 38 total, all PASS. 2 benchmarks healthy (~11µs, ~1µs). `go vet` clean.
**Code Review**: Complete — APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester, 2026-08-11). GAP-1 (reversed-pair dedup) fixed, mergeChains split, regression test added.
**Unlocks**: Story 003 (Contradiction Detection)
