# Story 004: Clue-Path Protection

> **Epic**: Persistent World State
> **Status**: Complete
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/persistent-world-state.md`
**Requirement**: `TR-pws-004`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0001: Entity Architecture
**ADR Decision Summary**: When a quest-relevant entity is destroyed, PWS receives a `clue_path_protection` event via the EF lifecycle bridge (`on_entity_destroyed`). PWS records the event in the entity's memory log and flags the entity as `world.[entity_id].destroyed_with_clues = true` so the Investigation Journal can surface it. No clue path permanently closes without an authored discovery path.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Subscribes to `on_entity_destroyed` via ADR-0011 EventBus. Entity destruction lifecycle is Kaiju-managed; verify `OnDestroy` event payload includes `quest_relevant` flag from the Persistent component.

**Control Manifest Rules (Feature Layer)**:
- Required: Lifecycle bridge: Kaiju OnActivate/OnDeactivate/OnDestroy → game-layer events via EventBus — ADR-0001, ADR-0011
- Required: Synchronous event dispatch via EventBus with system priority ordering — ADR-0011

---

## Acceptance Criteria

*From GDD `design/gdd/persistent-world-state.md`:*

- [ ] **AC-1**: GIVEN a `quest-relevant` entity is destroyed, WHEN EF broadcasts `on_entity_destroyed`, THEN PWS logs a `clue_path_protection` event and sets `world.[entity_id].destroyed_with_clues = true`.
- [ ] **AC-2**: GIVEN a non-quest-relevant entity is destroyed, WHEN `on_entity_destroyed` fires, THEN PWS does NOT log a clue_path_protection event and does NOT set the destroyed_with_clues flag.
- [ ] **AC-3**: GIVEN a clue-path-protected entity's flags, WHEN `GetWorldFlag(entity_id, "world.[entity_id].destroyed_with_clues")` is called, THEN it returns `(true, true)` — the Investigation Journal can discover it.
- [ ] **AC-4**: GIVEN the entity's memory log after clue-path protection, WHEN queried, THEN the log contains an entry with `event_type = "clue_path_protection"` including which entity was destroyed, the region, and the game timestamp.

---

## Implementation Notes

*Derived from ADR-0001, ADR-0011:*

- Subscribe to `on_entity_destroyed` event on the EventBus at priority 2 (early — before consumers that might query the destroyed entity's state).
- On event: check if the destroyed entity's `Persistent` component has a `quest_relevant` flag set.
- If quest-relevant: create a WorldEvent with EventType="clue_path_protection", set flag `world.[entity_id].destroyed_with_clues = true`, submit via `SubmitWorldEvent` (Story 002 pipeline).
- The destroyed entity's GameEntityID is preserved (IDs are never reused per ADR-0001) — the memory log and flag remain queryable by entity ID.
- Non-quest-relevant entities: no action. The event is observed but not acted upon.
- Exported: `CluePathProtector` struct with `Start(eventBus)` and `Stop()` methods.

---

## Out of Scope

- **Story 001-003**: Flag store, event pipeline, memory log
- **Story 005**: World evolution rules
- **Story 006**: Save/load
- Investigation Journal display of clue-path events (owned by Journal epic)

---

## QA Test Cases

- **AC-1 (Quest-relevant destruction → clue event)**:
  - Given: Entity with Persistent component, `quest_relevant = true`, stub EventBus
  - When: `on_entity_destroyed` event fired for that entity
  - Then: `GetWorldFlag(entityID, "world.[entityID].destroyed_with_clues")` returns `(true, true)`; memory log has entry with EventType="clue_path_protection"
  - Edge cases: Entity destroyed while game clock is paused (timestamp frozen), entity with no Persistent component (skip gracefully)

- **AC-2 (Non-quest-relevant destruction → no action)**:
  - Given: Entity with Persistent component, `quest_relevant = false` (or not set)
  - When: `on_entity_destroyed` fired
  - Then: No `destroyed_with_clues` flag set; no clue_path_protection log entry; no error
  - Edge cases: Entity without Persistent component at all, entity where quest_relevant flag was removed before destruction

- **AC-3 (Destroyed entity flag queryable)**:
  - Given: Entity destroyed with clue-path protection active
  - When: `GetWorldFlag(entityID, "world.[entityID].destroyed_with_clues")` called after destruction
  - Then: Returns `(true, true)` — flag persists on the destroyed entity's record
  - Edge cases: Query flag for entity that was never destroyed (false), query for non-existent entity ID (not found)

- **AC-4 (Memory log preserves destruction context)**:
  - Given: Clue-path protection event processed
  - When: `GetMemoryLog(entityID, 0, 10)` called
  - Then: Entry exists with EventType="clue_path_protection", RegionID set, GameTimestamp set, ActorID = destroyer (or 0 for environmental)
  - Edge cases: Multiple destruction events for same entity (shouldn't happen — entity already destroyed; second event rejected)

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/persistent_world_state/clue_path_test.go` — must exist and pass
**Status**: [x] Created and passing — 17/17 tests, fresh `go test -count=1` run 2026-08-12

---

## Dependencies

- Depends on: Story 002 (Event Submission Pipeline) — uses SubmitWorldEvent
- Depends on: Entity Framework epic (on_entity_destroyed event, Persistent component)
- Unlocks: Investigation Journal (surfaces clue-path events)

---

## Completion Notes

**Completed**: 2026-08-12
**Criteria**: 4/4 passing — AC-1..AC-4 auto-verified via `tests/integration/persistent_world_state/clue_path_test.go` (17 tests, fresh run passes)
**Deviations**:
- PWS-owned quest-relevance registry (`MarkQuestRelevant`) instead of reading the Persistent component — user-approved; EF purges component stores before `entity.destroyed` fires
- Tombstone flag storage + `RecordDestructionEvent` destruction-time pipeline path — user-approved; Story 006 serializes tombstones
- `clue_path_protection` entries marked `Protected: true` — ADR-0013 submitting-system mechanism; event type not yet in ADR's named protected list (INFO)
**Test Evidence**: Integration — `tests/integration/persistent_world_state/clue_path_test.go` (17 tests, passing)
**Code Review**: Complete — `/code-review` APPROVED WITH SUGGESTIONS (3 warnings + 5 QA coverage gaps, logged to `docs/tech-debt-register.md`)
