# Story 001: Entity Lifecycle

> **Epic**: Entity Framework
> **Status**: Complete
> **Layer**: Foundation
> **Type**: Logic
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-10

## Context

**GDD**: `design/gdd/entity-framework.md`
**Requirement**: `TR-ef-001`, `TR-ef-004`

**ADR Governing Implementation**: ADR-0001: Entity Architecture
**ADR Decision Summary**: GameEntityID (uint64) maps to Kaiju EntityId (string `"game_{ID}"`). Game-layer state machine (Pending/Active/Suspended/Destroyed) bridged to Kaiju lifecycle (OnActivate/OnDeactivate/OnDestroy). Lifecycle events broadcast via synchronous EventBus with system priority ordering.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Verify `engine.Entity` lifecycle hooks, `events.Event` dispatch model against `kaiju/src/engine/`.

**Control Manifest Rules (Foundation)**:
- Required: GameEntityID maps to Kaiju EntityId; Lifecycle bridge via EventBus
- Forbidden: Never reuse a GameEntityID; Never define a parallel entity system
- Guardrail: Entity creation <0.5ms

## Acceptance Criteria

- [ ] GIVEN a call to CreateEntity, WHEN the entity is assembled with type + display name + position, THEN a Kaiju engine.Entity is created with EntityId `"game_{GameEntityID}"` and the entity transitions from Pending to Active
- [ ] GIVEN an Active entity, WHEN DestroyEntity is called, THEN the entity transitions to Destroyed, all subscribed systems receive `on_entity_destroyed`, and the GameEntityID is never reused
- [ ] GIVEN an Active entity, WHEN the player leaves the region (distance > suspension_distance), THEN the entity transitions to Suspended and is hidden from update loops
- [ ] GIVEN a Suspended entity, WHEN the player returns to within suspension_distance, THEN the entity transitions back to Active with preserved state
- [ ] GIVEN entity lifecycle transitions, WHEN any transition fires, THEN all subscribed systems receive the event in priority order within the same frame

## Implementation Notes

From ADR-0001: GameEntityID is a uint64 monotonic counter. EntityId string format is `"game_{GameEntityID}"`. The mapping is stored in a `map[GameEntityID]string`. Use `host.EntityById(id)` for Kaiju entity access.

Lifecycle state machine: Pending (pre-activation assembly) → Active (in world, updating) → Suspended (off-screen, position preserved) → Destroyed (removed, ID retired). Suspended→Active on player return. Active→Destroyed on death/removal. Suspended→Destroyed on off-screen death.

Events: `on_entity_created`, `on_entity_destroyed`, `on_entity_suspended`, `on_entity_reactivated`. Dispatched synchronously via EventBus with priority ordering (ADR-0011). Entity Framework emits at priority 1 (highest).

## Out of Scope

- Story 002: Component attachment/detachment lifecycle
- Story 003: Spatial index updates on entity position change

## Dependencies

None — this is the root Foundation story. No other stories must complete before this one.

## QA Test Cases

- **AC-1 — Entity creation**:
  - Given: Entity Framework initialized with Kaiju host
  - When: CreateEntity(Character, "Player", Vec3{10, 20, 0})
  - Then: Returns valid GameEntityID > 0. Kaiju entity exists with EntityId `"game_{ID}"`. Entity state is Active.
  - Edge cases: Duplicate creation with same ID rejected. Max entities per region (500) exceeded → warning logged, entity still created.

- **AC-2 — Entity destruction**:
  - Given: Active entity with ID 42
  - When: DestroyEntity(42)
  - Then: Entity state is Destroyed. All subscribers received `on_entity_destroyed`. QueryEntities does not return ID 42. CreateEntity with next counter does not reuse 42.

- **AC-3 — Entity suspension**:
  - Given: Active entity at position P, player at position Q where distance(P,Q) > 100
  - When: Suspension check runs
  - Then: Entity transitions to Suspended. Component data preserved. Entity hidden from update loops but visible to spatial queries.

- **AC-4 — Entity reactivation**:
  - Given: Suspended entity at position P, player moves to position Q where distance(P,Q) < 100
  - When: Reactivation check runs
  - Then: Entity transitions to Active. `on_entity_reactivated` event fired. Component data unchanged from suspension.

## Test Evidence

**Test file**: `tests/unit/entity/entity_lifecycle_test.go`

| AC | Test function | What it verifies |
|----|--------------|------------------|
| AC-1 | `test_create_entity_maps_to_kaiju_entityid` | CreateEntity returns valid GameEntityID > 0, Kaiju entity exists with `"game_{ID}"` EntityId, state is Active |
| AC-2 | `test_destroy_entity_id_never_reused` | DestroyEntity transitions to Destroyed, `on_entity_destroyed` dispatched to all subscribers, ID removed from QueryEntities, counter does not reuse the ID |
| AC-3 | `test_suspend_on_distance_exceeded` | Distance > suspension_distance → Suspended, component data preserved, hidden from update loops, visible to spatial queries |
| AC-4 | `test_reactivate_on_player_return` | Player returns within suspension_distance → Active, `on_entity_reactivated` fired, component data unchanged |
| AC-5 | `test_lifecycle_events_in_priority_order` | All lifecycle transitions dispatch events to subscribers in priority order within the same frame |

## Completion Notes
**Completed**: 2026-08-10
**Criteria**: 5/5 passing
**Deviations**: EntityId uses engine.EntityId type (ADR-aligned); event names use dotted form per ADR-0011
**Test Evidence**: `src/tests/unit/entity/entity_lifecycle_test.go` — 5 test functions, all passing
**Code Review**: Skipped (Lean mode)
