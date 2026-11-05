# Story 001: World Flag Store & Query API

> **Epic**: Persistent World State
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/persistent-world-state.md`
**Requirement**: `TR-pws-001`, `TR-pws-006`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0001: Entity Architecture
**ADR Decision Summary**: PWS owns the `world.*` namespace of the Entity Framework's `Persistent` component. Flags are typed key-value pairs (bool/int/float/string) stored per entity. PWS is the authoritative writer — other systems read via PWS queries.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Pure data structures — engine-agnostic. Reads/writes `Persistent` component via ADR-0001 component store API. Verify component store read/write patterns against `kaiju/src/`.

**Control Manifest Rules (Feature Layer)**:
- Required: Typed component stores: `map[GameEntityID]ComponentType` per component — ADR-0001
- Required: Write arbitration: per-frame delta accumulation, system priority resolution — ADR-0001
- Forbidden: Never define a parallel entity system — use the game-layer-over-Kaiju pattern — ADR-0001

---

## Acceptance Criteria

*From GDD `design/gdd/persistent-world-state.md`:*

- [ ] **AC-1**: GIVEN an entity with a `Persistent` component, WHEN PWS sets `world.ruin_cleared = true` on that entity, THEN `GetWorldFlag(entity_id, "world.ruin_cleared")` returns `(true, true)` — the value and a "found" indicator.
- [ ] **AC-2**: GIVEN `GetEntitiesWithFlag("world.ruin_cleared", region_id)`, WHEN three entities in the region have the flag set, THEN exactly three GameEntityIDs are returned. Entities in other regions are excluded.
- [ ] **AC-3**: GIVEN a flag write with a type that doesn't match the declared schema (e.g., writing a string to a bool-typed flag), WHEN PWS processes the write, THEN a warning is logged and the flag is not mutated.
- [ ] **AC-4**: GIVEN a system requests a flag that has never been set, WHEN `GetWorldFlag(entity_id, "unknown_flag")` is called, THEN it returns `(nil, false)` — no value, not found.
- [ ] **AC-5**: GIVEN a system writes a flag value, WHEN the same flag is queried via `GetWorldFlags(entity_id)`, THEN the flag appears in the returned map with its correct typed value.

---

## Implementation Notes

*Derived from ADR-0001:*

- PWS owns the `world.*` namespace of the `Persistent` component per EF's write-ownership table.
- Fluent flag API: `SetWorldFlag(entityID, flag string, value interface{})`, `GetWorldFlag(entityID, flag string) (interface{}, bool)`, `GetWorldFlags(entityID) map[string]interface{}`, `GetEntitiesWithFlag(flag string, regionID string) []GameEntityID`.
- Flag names are dotted strings scoped to entity and region: `world.ruin_cleared`, `world.thornwell.bridge_repaired`.
- Type schema per region: declare expected types for flags. On write, validate type matches schema. Mismatch → warning + skip (CR-1).
- Query indexing: `GetEntitiesWithFlag` must be indexed — maintain a reverse index `map[flagName]map[GameEntityID]bool` for fast lookup.
- Exported types: `WorldFlagStore` struct with constructor taking component store reference.
- Pure Go — no Kaiju engine APIs in the core logic. Component store integration via interface.

---

## Out of Scope

- **Story 002**: Event submission pipeline, SubmitWorldEvent, validation, broadcast
- **Story 003**: Memory log per entity
- **Story 004**: Clue-path protection on entity destruction
- **Story 005**: World evolution rules
- **Story 006**: Save/load serialization

---

## QA Test Cases

- **AC-1 (Flag set/get round-trip)**:
  - Given: An entity with a Persistent component, a WorldFlagStore instance
  - When: `SetWorldFlag(entityID, "world.ruin_cleared", true)` then `GetWorldFlag(entityID, "world.ruin_cleared")`
  - Then: Returns `(true, true)` — the bool value and found=true
  - Edge cases: Overwrite existing flag (last write wins), set flag on entity without Persistent component (error)

- **AC-2 (GetEntitiesWithFlag region-scoped)**:
  - Given: 5 entities total, 3 in region "A" with `world.ruin_cleared = true`, 1 in region "B" with same flag, 1 without
  - When: `GetEntitiesWithFlag("world.ruin_cleared", "A")`
  - Then: Returns exactly 3 GameEntityIDs (region A only)
  - Edge cases: Empty region (0 results), nil regionID (all regions), flag never set (0 results)

- **AC-3 (Type schema validation)**:
  - Given: Region schema declares `world.bandit_count: int`
  - When: `SetWorldFlag(entityID, "world.bandit_count", "many")` (string where int expected)
  - Then: Warning logged, flag value unchanged (or not set), function returns error
  - Edge cases: Flag not in schema (accept any type), schema declares float but int written (reject — strict typing)

- **AC-4 (Missing flag returns not-found)**:
  - Given: An entity with no flags set
  - When: `GetWorldFlag(entityID, "world.never_set")`
  - Then: Returns `(nil, false)`
  - Edge cases: Flag set then deleted/cleared

- **AC-5 (GetWorldFlags returns all flags)**:
  - Given: Entity with `world.a = true`, `world.b = 42`, `world.c = "hello"`
  - When: `GetWorldFlags(entityID)`
  - Then: Map contains all three with correct types and values
  - Edge cases: Entity with no flags (empty map, not nil), entity with 32+ flags (within max_flags_per_entity tuning knob)

---

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/persistent_world_state/flag_store_test.go` — must exist and pass
**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Entity Framework epic (Persistent component, component store API)
- Unlocks: Story 002 (Event Submission Pipeline)

---

## Completion Notes

**Completed**: 2026-08-12
**Criteria**: 5/5 passing — all automated via 18 unit tests
**Deviations**: ADVISORY — PersistentAccessor interface injection (not bare constructor), SetEntityRegion + ClearWorldFlag added (region scoping + deletion gaps), soft limit for max_flags_per_entity (warn + write)
**Test Evidence**: `tests/unit/persistent_world_state/flag_store_test.go` — 18 tests, all PASS (BLOCKING gate met)
**Code Review**: APPROVED — no issues
**Tech Debt**: None
**PWS Epic**: 1/6 Complete
