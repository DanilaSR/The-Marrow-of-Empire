# Story 003: Memory Log

> **Epic**: Persistent World State
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (3-5 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/persistent-world-state.md`
**Requirement**: `TR-pws-002`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0013: Persistent World State — Memory Log & Evolution Rules
**ADR Decision Summary**: Append-only structured memory log per entity. Soft cap at 50 entries — oldest unprotected entry evicted when cap exceeded. Protected entries (npc_state_changed, discovery_event, quest_relevant) never evicted by cap. If all entries are protected, cap is temporarily exceeded with a warning.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: LOW
**Engine Notes**: Pure data structure — engine-agnostic. No Kaiju APIs. Part of the `Persistent` component per ADR-0001.

**Control Manifest Rules (Feature Layer)**:
- Required: Memory log: append-only per entity, soft cap 50, protected entries never evicted — ADR-0013
- Guardrail: Memory log append: <0.1ms — ADR-0013

---

## Acceptance Criteria

*From GDD `design/gdd/persistent-world-state.md`:*

- [ ] **AC-1**: GIVEN an entity with 50 memory log entries (at cap, none protected), WHEN a new event is submitted, THEN the oldest entry is evicted and the new entry is appended — total count remains 50.
- [ ] **AC-2**: GIVEN an entity with 50 memory log entries all marked `protected: true`, WHEN a new event is submitted, THEN the new entry is appended (total = 51), a warning is logged, and the soft cap is temporarily exceeded.
- [ ] **AC-3**: GIVEN an entity with 40 unprotected entries and 10 protected entries (50 total), WHEN a new unprotected event is submitted, THEN the oldest unprotected entry is evicted (not a protected one), and total count remains 50.
- [ ] **AC-4**: GIVEN a memory log with entries, WHEN `GetMemoryLog(entityID, since, limit)` is queried, THEN entries newer than `since` are returned, newest first, up to `limit` entries.
- [ ] **AC-5**: GIVEN a memory log, WHEN queried, THEN each entry contains: event_id (monotonic, globally unique), event_type, game_timestamp, actor_entity_id, target_entity_id, region_id, and details (JSON string).

---

## Implementation Notes

*Derived from ADR-0013:*

- `MemoryEntry` struct per ADR-0013: EventID (uint64, monotonic globally unique), EventType (string), GameTimestamp (float64), ActorID (GameEntityID), TargetID (GameEntityID), RegionID (string), Details (string, JSON blob), Protected (bool).
- `MemoryLog` struct: `Entries []MemoryEntry` (oldest first), `MaxSize int` (default 50).
- Append: `log.Entries = append(log.Entries, entry)`.
- Eviction: if `len(log.Entries) > MaxSize`, find oldest entry where `Protected == false` and remove it. If none found, keep all (soft cap exceeded, log warning).
- Globally unique event IDs: use an atomic counter `nextEventID uint64` at the PWS system level.
- `GetMemoryLog(entityID GameEntityID, since float64, limit int) []MemoryEntry` — filter entries where `GameTimestamp > since`, sort newest first, apply limit.
- Exported: `MemoryEntry`, `MemoryLog`, `NewMemoryLog(maxSize int)`, `Append(entry) (evicted bool)`, `GetMemoryLog(since, limit)`.
- Pure Go data structure — no engine dependencies. Testable with table-driven tests.

---

## Out of Scope

- **Story 001**: Flag store
- **Story 002**: Event submission pipeline (calls MemoryLog.Append)
- **Story 004**: Clue-path protection
- **Story 005**: World evolution rules
- **Story 006**: Save/load serialization

---

## QA Test Cases

- **AC-1 (Cap eviction — no protected)**:
  - Given: MemoryLog with MaxSize=50, 50 unprotected entries
  - When: Append a 51st entry
  - Then: Oldest entry evicted, total = 50, new entry at end, evicted=true returned
  - Edge cases: MaxSize=10 (small cap), MaxSize=1 (evicts immediately), append to empty log

- **AC-2 (Cap exceeded — all protected)**:
  - Given: MemoryLog with MaxSize=50, all 50 entries Protected=true
  - When: Append a 51st entry
  - Then: Total = 51, warning logged, no entry evicted, evicted=false
  - Edge cases: Cap already exceeded (52→53 still appends), all-protected at MaxSize=1

- **AC-3 (Mixed eviction — skip protected)**:
  - Given: MemoryLog with 10 protected (positions 0, 2, 4, 6, 8, 10, 12, 14, 16, 18) and 40 unprotected at cap=50
  - When: Append unprotected entry
  - Then: Oldest unprotected entry evicted (position 1), protected entries untouched, total = 50
  - Edge cases: Only 1 unprotected entry at cap (that one gets evicted), all entries at positions 49 are protected (evicts position 0 if unprotected)

- **AC-4 (Query with since + limit)**:
  - Given: Log with entries at timestamps 1.0, 2.0, 3.0, 4.0, 5.0 (oldest→newest)
  - When: `GetMemoryLog(since=2.5, limit=2)`
  - Then: Returns [{timestamp=5.0}, {timestamp=4.0}] — newest first, after 2.5, max 2 entries
  - Edge cases: since=0 (all entries), since=10 (empty result), limit=0 (empty result), limit > len (all matching)

- **AC-5 (Entry field completeness)**:
  - Given: A MemoryEntry with all fields populated
  - When: Retrieved via GetMemoryLog
  - Then: All fields (EventID, EventType, GameTimestamp, ActorID, TargetID, RegionID, Details, Protected) match input values
  - Edge cases: Empty Details string, zero GameTimestamp, zero entity IDs (valid for system-generated events)

---

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/persistent_world_state/memory_log_test.go` — must exist and pass
**Status**: [x] Created — `tests/unit/persistent_world_state/memory_log_test.go`

---

## Dependencies

- Depends on: Story 001 (World Flag Store — provides entity context)
- Unlocks: Story 002 (Event Submission Pipeline — appends to MemoryLog)

---

## Completion Notes

**Completed**: 2026-08-12
**Criteria**: 5/5 passing — all automated via 13 unit tests
**Deviations**: ADVISORY — `GetEntries(since, limit)` instead of story's `GetMemoryLog(entityID, since, limit)` (entityID scoping belongs at PWS system level in Story 002); `limit < 0` = unlimited (not specified in story)
**Test Evidence**: `tests/unit/persistent_world_state/memory_log_test.go` — 13 tests, all PASS (BLOCKING gate met). Story 001 tests 18/18 PASS (no regressions).
**Code Review**: N/A (simple data structure, verified via tests)
**Tech Debt**: None
**PWS Epic**: 2/6 Complete
