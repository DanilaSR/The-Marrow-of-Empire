# Story 002: Event Submission Pipeline

> **Epic**: Persistent World State
> **Status**: Complete
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/persistent-world-state.md`
**Requirement**: `TR-pws-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0001: Entity Architecture (primary), ADR-0011: Event Bus (secondary — broadcast)
**ADR Decision Summary**: Systems submit world events to PWS via `SubmitWorldEvent`. PWS validates the event, applies flag mutations, and broadcasts `on_world_state_changed` on the EventBus. Processing is synchronous with priority ordering — same-frame guarantees for downstream consumers.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Event bus integration via ADR-0011. Verify synchronous event dispatch API. World flag store from Story 001 provides the mutation target. Pure data pipeline — engine-agnostic core.

**Control Manifest Rules (Feature Layer)**:
- Required: Synchronous event dispatch via EventBus with system priority ordering — ADR-0011
- Required: Write arbitration: per-frame delta accumulation, system priority resolution — ADR-0001
- Forbidden: Never use asynchronous event dispatch — synchronous with priority guarantees correct ordering — ADR-0011

---

## Acceptance Criteria

*From GDD `design/gdd/persistent-world-state.md`:*

- [ ] **AC-1**: GIVEN a system calls `SubmitWorldEvent(location_cleared, {world.ruin_cleared: true})`, WHEN the event is processed, THEN the target entity's `world.ruin_cleared` flag is set to `true` AND a memory log entry is appended with `event_type = location_cleared` and the current game timestamp.
- [ ] **AC-2**: GIVEN a system submits an event with a non-`world.*` flag (e.g., `trust.some_flag`), WHEN the event is processed, THEN the non-world flag is skipped (warning logged), the `world.*` flags are still applied, and the memory log entry is still written.
- [ ] **AC-3**: GIVEN two systems submit events for the same entity in the same frame, WHEN both are processed, THEN events are processed in submission order, both flag mutations are applied, and both memory log entries are appended. If they write the same flag, EF's write arbitration rules apply (last writer by priority wins).
- [ ] **AC-4**: GIVEN an event with an invalid target entity (destroyed or nonexistent), WHEN submitted, THEN the event is rejected with an error and no flag mutations or log entries are created.
- [ ] **AC-5**: GIVEN a valid event, WHEN processed, THEN `on_world_state_changed` is broadcast on the EventBus with the event summary — subscribers react in the same frame.

---

## Implementation Notes

*Derived from ADR-0001, ADR-0011:*

- `WorldEvent` struct: `EventType`, `ActorEntityID`, `TargetEntityID`, `RegionID`, `Details` (JSON string), `FlagMutations` (map[string]interface{}).
- `SubmitWorldEvent(event WorldEvent) error` — entry point for all systems.
- Pipeline: 1) Validate (actor exists, target exists, event_type valid, region_id valid). 2) Apply each flag in `FlagMutations`: if flag matches `world.*` AND type matches schema → apply; else skip (log warning). 3) Append MemoryEntry to target's MemoryLog (delegates to Story 003). 4) Broadcast `on_world_state_changed` on EventBus.
- Validation errors return immediately — no partial processing.
- Flag filtering is per-flag, not per-event: a single non-world flag doesn't block the event.
- Event bus priority: PWS broadcasts at priority 5 (after entity mutations, before UI consumers).
- Same-frame ordering: events processed in `SubmitWorldEvent` call order within the frame. Last writer wins for same-flag conflicts.
- Exported `WorldEvent` struct, `SubmitWorldEvent` method, `OnWorldStateChanged` event type constant.

---

## Out of Scope

- **Story 001**: Flag store implementation, query API
- **Story 003**: Memory log data structure, eviction logic
- **Story 004**: Clue-path protection on entity destruction
- **Story 005**: World evolution rules
- **Story 006**: Save/load

---

## QA Test Cases

- **AC-1 (Event → flag + log)**:
  - Given: Target entity with Persistent component, WorldFlagStore from Story 001, stub MemoryLog
  - When: `SubmitWorldEvent(WorldEvent{Type: "location_cleared", TargetID: e1, FlagMutations: {"world.ruin_cleared": true}})`
  - Then: `GetWorldFlag(e1, "world.ruin_cleared")` returns `(true, true)`; MemoryLog has 1 entry with EventType="location_cleared" and non-zero GameTimestamp
  - Edge cases: Event with 0 flag mutations (valid — log entry still written), event with 10+ mutations (all applied in order)

- **AC-2 (Non-world flag filtered)**:
  - Given: Event with `{"world.a": true, "trust.x": 5}`
  - When: `SubmitWorldEvent(...)`
  - Then: `world.a` set to true; `trust.x` skipped; warning logged; memory log entry still written; no error returned
  - Edge cases: All flags non-world (event still processed, log entry written, no flags mutated), mixed types (each flag independently validated)

- **AC-3 (Same-frame ordering)**:
  - Given: Two events A then B, both writing `world.count`
  - When: Submitted in order A(priority=3, value=1), B(priority=5, value=2)
  - Then: Both log entries appended in A→B order; `world.count = 2` (B wins by higher priority)
  - Edge cases: Same priority (registration order), A writes flag F and B writes flag G (no conflict, both applied)

- **AC-4 (Invalid target rejected)**:
  - Given: Target entity ID that doesn't exist
  - When: `SubmitWorldEvent(...)`
  - Then: Error returned; no flags mutated; no log entry appended; no broadcast
  - Edge cases: Target destroyed mid-frame before event submission, actor entity invalid

- **AC-5 (Broadcast on success)**:
  - Given: Event bus with a subscriber counting `on_world_state_changed` events
  - When: `SubmitWorldEvent(...)` succeeds
  - Then: Subscriber received exactly 1 event with matching EventType and TargetID
  - Edge cases: Event rejected (no broadcast), multiple subscribers (all receive)

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/persistent_world_state/event_pipeline_test.go` — must exist and pass
**Status**: [x] Created — `tests/integration/persistent_world_state/event_pipeline_test.go`

---

## Dependencies

- Depends on: Story 001 (World Flag Store & Query API) — provides flag mutation target
- Depends on: Story 003 (Memory Log) — provides log append target
- Depends on: Entity Framework epic (Persistent component, EventBus)
- Unlocks: Story 004 (Clue-Path Protection), Story 005 (World Evolution Rules)

---

## Completion Notes

**Completed**: 2026-08-12
**Criteria**: 5/5 passing — all automated via 14 integration tests
**Deviations**: ADVISORY — EventPipeline struct with injected deps (not free function), submission-order last-writer-wins (no per-event priority), clock injection via config
**Test Evidence**: `tests/integration/persistent_world_state/event_pipeline_test.go` — 14 tests, all PASS (BLOCKING gate met). Unit suite 31/31 PASS (no regressions).
**Code Review**: N/A (clean implementation, approved design)
**Tech Debt**: None
**PWS Epic**: 3/6 Complete
