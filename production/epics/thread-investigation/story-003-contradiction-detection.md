# Story 003: Contradiction Detection

> **Epic**: Thread Investigation
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: S (2–4 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/thread-investigation.md`
**Requirements**:
- `TR-thread-003` — Contradiction detection: when discovered fact contradicts another discovered fact → create contradiction thread, emit event

**ADR Governing Implementation**: ADR-0010: Thread Investigation Engine
**ADR Decision Summary**: When a discovered fact has `contradicts: [other_fact_id]` and the player has also discovered that other fact, the system creates a contradiction thread and flags both facts. Contradictions are clues, not bugs — they create investigation tension. Events are emitted via EventBus (ADR-0011) for downstream systems (Investigation Journal, Trust & Relationship).

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Pure comparison logic — no Kaiju APIs. Event emission uses EventBus (ADR-0011) with synchronous dispatch and system priority ordering.

**Control Manifest Rules (Feature Layer)**:
- Required: Event-driven architecture — contradiction events emitted via EventBus (ADR-0011)
- Required: Synchronous event dispatch with system priority ordering — death must be recorded BEFORE trust events fire
- Forbidden: Never use direct cross-system calls for state mutations — all mutations go through the owning system's interface, with events for notification
- Guardrail: `OnFactDiscovered` (full algorithm including contradiction check): <5ms (ADR-0010)

---

## Acceptance Criteria

*From GDD `design/gdd/thread-investigation.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN a discovered fact with `contradicts: [other_fact_id]` where the other fact is also discovered, WHEN the second fact is discovered, THEN a contradiction thread is created and both facts are flagged.

- [ ] **AC-2**: GIVEN a discovered fact with `contradicts: [other_fact_id]` where the other fact has NOT been discovered, WHEN the fact is discovered, THEN no contradiction thread is created (the contradiction is latent — it will trigger if/when the other fact is discovered).

- [ ] **AC-3**: GIVEN a contradiction thread is created, WHEN the contradiction is detected, THEN `EventContradictionDetected` is emitted via EventBus containing both fact IDs, both source NPC IDs, and the contradiction topic.

- [ ] **AC-4**: GIVEN one of the contradicting facts is a RUMOR (fact type from ADR-0005), WHEN the contradiction is detected, THEN the contradiction is flagged as lower confidence and the event includes `confidence: "low"`.

---

## Implementation Notes

*Derived from ADR-0010 Implementation Guidelines:*

### Contradiction Detection Algorithm (Step 4)

```
OnFactDiscovered — Step 4 (builds on Story 001 Steps 1–2 + Story 002 Step 3):
  4. If fact.Contradicts is non-empty:
       For each contradicted fact_id:
         If that fact is in PlayerKnowledge.DiscoveredFacts:
           a. Create contradiction thread:
              Thread{
                Type: "contradiction",
                Topic: "Resolve: [NPC A] says [X], but [NPC B] says [Y]",
                TargetEntity: null (open-ended — player decides who to believe),
                Source: fact.ID + "↔" + contradicted_fact.ID
              }
           b. Flag both facts as contradictory in PlayerKnowledge
           c. Determine confidence:
              If either fact is RUMOR → confidence = "low"
              Else → confidence = "high"
           d. Emit EventContradictionDetected{
                FactA:        fact,
                FactB:        contradictedFact,
                SourceNPC_A:  sourceNPC,
                SourceNPC_B:  contradictedFact.SourceNPC,
                Confidence:   confidence
              }
```

### Event Structure

```go
type EventContradictionDetected struct {
    FactA       Fact         // the newly discovered fact
    FactB       Fact         // the previously discovered fact it contradicts
    SourceNPC_A GameEntityID // who revealed Fact A
    SourceNPC_B GameEntityID // who revealed Fact B
    Confidence  string       // "high" or "low" (low if either fact is RUMOR)
    ThreadID    string       // the contradiction thread created
}
```

### Confidence Rules (CR-3)

Per GDD edge case: "If a contradiction is detected but one of the contradicting facts is a RUMOR: The contradiction is flagged as lower confidence. The rumor may be false."

Detection: check `fact.Type == FactTypeRumor` on either fact. Event carries `Confidence: "low"` so downstream systems (Investigation Journal) can display appropriately.

### Contradiction Thread Format

The contradiction thread's topic follows the GDD format:
- High confidence: "Resolve contradiction: [NPC A] claims [fact A summary]. [NPC B] claims [fact B summary]."
- Low confidence: "Possible contradiction: [NPC A] claims [fact A summary], but [NPC B] says [fact B summary] — one account may be a rumor."

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Thread data model and auto-creation from `leads_to`
- Story 002: Auto-chaining and entity overlap detection
- Story 004: Revelation threshold — contradiction threads may contribute to revelation if they share entities with other facts in a chain
- Story 005: PlayerKnowledge persistence
- Investigation Journal: How contradictions are displayed to the player (Presentation layer)

---

## QA Test Cases

*Test cases for automated unit testing — developer implements against these:*

- **AC-1**: Contradiction detected — both facts discovered
  - Given: `DiscoveredFacts` contains "fact_A" from NPC 5; a new fact "fact_B" from NPC 8 has `contradicts: ["fact_A"]`
  - When: `OnFactDiscovered("fact_B", 8)` is called
  - Then: A contradiction thread is created; both "fact_A" and "fact_B" are flagged; `EventContradictionDetected` is emitted with `Confidence = "high"`
  - Edge cases: Fact contradicts itself (reject); circular contradictions (A contradicts B, B contradicts A — second still fires)

- **AC-2**: Contradiction latent — other fact not yet discovered
  - Given: `DiscoveredFacts` does NOT contain "fact_C"; a new fact "fact_D" has `contradicts: ["fact_C"]`
  - When: `OnFactDiscovered("fact_D", 4)` is called
  - Then: No contradiction thread is created; no event is emitted; "fact_D" is recorded normally
  - Edge cases: Multiple contradicts entries, some discovered and some not

- **AC-3**: Event emission on contradiction
  - Given: Both contradicting facts are discovered (AC-1 scenario)
  - When: Contradiction is detected
  - Then: `EventContradictionDetected` is dispatched via EventBus with priority 1; event contains both fact IDs, both NPC IDs, and the contradiction thread ID
  - Edge cases: EventBus not initialized (skip gracefully); subscriber error (log, don't crash)

- **AC-4**: RUMOR lowers confidence
  - Given: "fact_E" is a RUMOR from NPC 2; "fact_F" is a FACT from NPC 9 with `contradicts: ["fact_E"]`
  - When: `OnFactDiscovered("fact_F", 9)` is called
  - Then: Contradiction detected with `Confidence = "low"`; event carries low confidence; thread topic includes "one account may be a rumor"
  - Edge cases: Both facts are RUMORs → still low; neither is RUMOR → high

---

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/thread_investigation/contradiction_test.go` — must exist and pass

**Status**: [x] Created — `tests/unit/thread_investigation/contradiction_test.go` (12 tests + benchmark, all PASS, go vet clean)

---

## Dependencies

- Depends on: Story 001 (Thread data model, OnFactDiscovered Steps 1–2)
- Depends on: Story 002 (Auto-chaining — contradiction threads need chain assignment)
- Depends on: NPC Knowledge Graph (KG-001 — fact type enum including RUMOR)
- Depends on: EventBus (ADR-0011 — event dispatch infrastructure)
- Unlocks: Story 004 (Revelation threshold — contradiction threads may contribute facts to chains)

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 4/4 passing
**Deviations**: ADVISORY — ContradictedFacts added to PlayerKnowledge (extends ADR-0010 illustrative model). Three defensive index fixes applied during code review (TargetEntity=0 skip, archived chain exclusion, entityChainIndex full scan).
**Test Evidence**: Logic — `tests/unit/thread_investigation/contradiction_test.go` (12 tests + benchmark, all PASS). Total suite: 36+ tests. `go vet` clean.
**Code Review**: Complete — APPROVED (lead-programmer + qa-tester, 2026-08-11). 3 chain/index warnings fixed in-review.
**Unlocks**: Story 004 (Revelation Threshold & Resolution)
