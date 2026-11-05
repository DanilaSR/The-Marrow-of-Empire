# Story 001: Party Roster & Active Slots

> **Epic**: Party System
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: S (2–4 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-13

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-001` — Party roster (max 6), recruitment via trust/dialogue, dismissal with confirmation, companion combat participation

**ADR Governing Implementation (primary)**: ADR-0006: Character Stats & Component Schema
**ADR (secondary)**: ADR-0011: Event Bus Architecture — roster changes are notified via EventBus so downstream systems (TCE, UI) update in the same frame
**ADR Decision Summary**: ADR-0006 establishes the typed component-store pattern (`map[GameEntityID]T`, Go generics, `EntityData` interface) that the PartyState store follows. ADR-0011 requires synchronous event dispatch with priority ordering and forbids direct cross-system state mutation — roster changes emit events; consuming systems never write the roster.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Pure Go data model — no Kaiju-specific APIs. Persistence rides the Entity Framework component-store pattern (ADR-0001): the PartyState store must implement `EntityData` and register via `pod.Register` (verify pod reflection requirements against `kaiju/src/`).

**Control Manifest Rules (Feature Layer)**:
- Required: Typed component stores — `map[GameEntityID]ComponentType`, accessed via Go generics (ADR-0001)
- Required: All state mutations go through the owning system's interface, with events for notification (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Forbidden: Never reuse a GameEntityID after entity destruction (ADR-0001)
- Guardrail: Event dispatch (5 subscribers): <1µs (ADR-0011)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-1 (Party Size and Composition), scoped to this story:*

- [ ] **AC-1** (AC-PS-13, PC slot locked): GIVEN any party state, WHEN the player attempts to remove or swap out the player character, THEN the action is rejected. The PC slot is always locked.

- [ ] **AC-2** (size cap, data-driven): GIVEN a roster at capacity (`vs_party_size` 4 at VS, `full_vision_party_size` 6 at Full Vision — external config, never hardcoded), WHEN an add-active-member request arrives, THEN the request is rejected with a full-roster result and the member must enter the pool instead.

- [ ] **AC-3** (roster visibility): GIVEN a companion recruited into the active party, WHEN the party screen renders its active slots, THEN the companion appears in the active slots list (AC-PS-01 roster part).

- [ ] **AC-4** (`pc_controlled` tagging): GIVEN a companion in the active party, WHEN inspected via the debug overlay, THEN the companion entity carries the `pc_controlled` tag; companions in the pool do NOT.

- [ ] **AC-5** (pool relocation): GIVEN companions in the recruitment pool, WHEN the party reaches a new rest point, THEN the pool is recorded as present at that rest point (CR-1 — pool companions automatically relocate; they are always accessible for management when the party rests).

- [ ] **Edge case — solo party**: GIVEN the only non-PC active member departs mid-expedition, THEN the party size drops to 1 and remains a valid state; the player can return to camp to bring companions from the pool.

---

## Implementation Notes

*Derived from GDD CR-1 and ADR-0006/ADR-0011:*

### PartyState Store

The Party System owns a single roster store — one `PartyState` per save, not per entity:

```go
type PartyState struct {
    PCID       GameEntityID    // player character — fixed at creation, never removable
    Active     []GameEntityID  // ordered: PC first, then companions in marching order
    Pool       []GameEntityID  // recruited companions waiting at the current rest point
    RestPoint  GameEntityID    // current rest point pool companions are present at
}
```

- Max active size read from external config (`vs_party_size` 4 / `full_vision_party_size` 6). Never hardcode the number.
- Implement `EntityData` and register via `pod.Register` per ADR-0001/ADR-0002 so Save/Load serializes the roster.
- The player character's `GameEntityID` is fixed at character creation; `Remove`/`Swap` operations on `PCID` return an error — no UI should ever reach them (defense in depth for AC-1).

### `pc_controlled` Tag Ownership

- The Party System is the authoritative writer of the `pc_controlled` tag on companion entities (write-ownership per ADR-0001 field write-table convention).
- Set the tag on `InParty` transition, clear it on `InPool`/`Departed`. Other systems (TCE, UI) read it — they never write it.

### Event Contract

- Emit roster notifications on the EventBus (synchronous, ADR-0011): `party.member_added`, `party.member_removed`, `party.pool_changed`.
- ⚠️ ADR-0011's EventType catalog predates the Party System and has no `party.*` types. Register these types in the catalog following the existing namespaced-string convention. If the new Party ADR (see Story 002 blocker) defines a different mechanism, it supersedes this note.
- Handlers must stay within the combined <2ms/frame event budget — roster handlers set flags, never recalculate.

### Story Ordering Note

The four companion states (`Unrecruited`/`InParty`/`InPool`/`Departed`) are owned by Story 002 (currently blocked on the Party ADR). This story builds the storage, capacity, and locking rules only — state transition semantics live in Story 002.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Companion state machine — the four states, transitions, persisted `cause`
- Story 003: Recruitment validation (trust threshold, story condition, slot assignment)
- Story 004: Swap and dismissal operations at rest points
- Story 007: How TCE consumes the active roster for combat encounters

---

## QA Test Cases

> Generated by /qa-plan sprint-6 (2026-08-13).

**Automated (unit) — `tests/unit/party/roster_test.go`:**
- **AC-1**: remove/swap on PC ID rejected in every party state (PC slot locked)
- **AC-2**: add at capacity (config `vs_party_size` 4 / `full_vision_party_size` 6, never hardcoded) → rejected with full-roster result; member goes to pool
- **AC-3/AC-4**: recruited companion in active slots + `pc_controlled` tag; pool members untagged
- **AC-5**: pool relocates to new rest point (CR-1)
- **Edge cases**: solo party (size 1) valid; double-remove idempotent; removed member's GameEntityID not reused

**Estimated test count**: ~8
## Test Evidence

**Story Type**: Logic
**Required evidence**:
- Logic: `tests/unit/party/roster_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: None (reads Entity Framework + Character Stats, both complete)
- Unlocks: Story 002, Story 003, Story 007, Story 008, Story 010, Story 011, Story 013

---

## Completion Notes

**Completed**: 2026-08-13
**Criteria**: 5/5 passing + solo-party edge (automated). Package: `src/party/` (party_state.go, errors.go, roster.go, events.go, component.go), tests `tests/unit/party/roster_test.go` (42 tests).
**Deviations**: ADVISORY — (1) ADR-0016 (Proposed) conflicts with this story's committed contract (storage shape, field names, event naming); resolution owned by PRC-001 — recommended: revise the ADR to adopt this storage + dot-namespaced event names; (2) roster save-snapshot wiring not yet hooked into the save manifest — owned by INT-6/INT-7; (3) coordinator maintenance outside story scope: perf-flake threshold fixes (entity, ai_dialogue tests) + unused-import removal in src/game/game.go.
**Test Evidence**: `tests/unit/party/roster_test.go` — 42 tests pass (`-count=1`); full sweep 37/37 green; build + vet clean.
**Code Review**: Complete — `/code-review` run 2026-08-13: lead-programmer + qa-tester; self-swap guard fixed + 10 test items applied; final APPROVED WITH SUGGESTIONS.

