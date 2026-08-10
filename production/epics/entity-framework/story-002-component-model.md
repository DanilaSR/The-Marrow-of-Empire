# Story 002: Component Model

> **Epic**: Entity Framework
> **Status**: Complete
> **Layer**: Foundation
> **Type**: Logic
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-10

## Context

**GDD**: `design/gdd/entity-framework.md`
**Requirement**: `TR-ef-002`, `TR-ef-006`

**ADR Governing Implementation**: ADR-0001: Entity Architecture
**ADR Decision Summary**: Typed component stores as `map[GameEntityID]ComponentType` per component. Components are pure data containers. Systems operate on components via `GetComponent[T](id) (T, bool)` and `SetComponent[T](id, T)`.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH

**Control Manifest Rules**: Required: Typed component stores with Go generics. Forbidden: Never define a parallel entity system.

## Acceptance Criteria

- [ ] GIVEN an Active entity, WHEN a system calls SetComponent with a valid component, THEN the component is stored and subsequent GetComponent returns the same value
- [ ] GIVEN an entity with a component, WHEN a system calls GetComponent for that type, THEN the component is returned with found=true
- [ ] GIVEN an entity without a component, WHEN GetComponent is called, THEN found=false with zero value
- [ ] GIVEN a component attached to an entity, WHEN that component is removed, THEN subscribed systems receive `on_component_removed` and subsequent reads return not-found
- [ ] GIVEN a component field changed, WHEN SetComponent writes a new value, THEN `on_component_changed` fires with old_value and new_value

## Implementation Notes

From ADR-0001: Component stores are created lazily — a `map[GameEntityID]ComponentType` is allocated only when the first entity with that component type is created. Go generics provide type-safe access without reflection in hot paths.

Component types: Health, Stats, Knowledge, Inventory, Faction, Dialogue, Combatant, Interactable, Persistent, Visual, Audio, Schedule. Each is a struct with data fields per the GDD component table.

## Out of Scope

- Story 001: Entity creation/destruction (components are attached after creation)
- Story 005: Serialization of components via pod

## QA Test Cases

- **AC-1 — Set/Get component**:
  - Given: Active entity ID 42
  - When: SetComponent(42, Health{Current: 100, Max: 100})
  - Then: GetComponent[Health](42) returns (Health{Current: 100, Max: 100}, true)

- **AC-2 — Missing component**:
  - Given: Entity 42 with no Knowledge component
  - When: GetComponent[Knowledge](42)
  - Then: Returns (Knowledge{}, false)

- **AC-4 — Component removal**:
  - Given: Entity 42 with Health component
  - When: RemoveComponent[Health](42)
  - Then: `on_component_removed` event fired. GetComponent[Health](42) returns false.

## Completion Notes
**Completed**: 2026-08-10
**Criteria**: 5/5 passing
**Deviations**: TR-ef-006 (pod.Register serialization) deferred to EF-005 per story scope
**Test Evidence**: `src/tests/unit/entity/component_store_test.go` — 12 test functions, all passing
**Code Review**: Skipped (Lean mode — user confirmed review before sprint close-out)
