# Story 005: Serialization

> **Epic**: Entity Framework
> **Status**: Complete
> **Last Updated**: 2026-08-10
> **Layer**: Foundation
> **Type**: Integration
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 002 (Component Model), Story 003 (Spatial Index)
> **Out of Scope**: Save file I/O, version headers, CRC32 checksums, .bak backups, Steam Cloud integration, platform save paths, incremental/delta saves — those belong to the Save/Load System epic (Foundation #4). This story covers only the Entity Framework serialization boundary: component store registration with `pod.Register` and the `pod.Encode`/`pod.Decode` pass over all component stores plus the entity ID table.

## Context

**GDD**: `design/gdd/entity-framework.md`
**Requirement**: `TR-ef-006`

**ADR Governing Implementation**: ADR-0001 (Entity Architecture), ADR-0002 (Save/Load)
**ADR Decision Summary**: All component stores implement EntityData interface, registered via pod.Register. Full-state serialization via pod.Encode/Decode. Entity table (GameEntityID ↔ EntityId) included in save data.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Verify `pod.Register` reflection requirements, `EntityData` interface methods against `kaiju/src/pod/`.

**Control Manifest Rules**: Required: All component stores implement EntityData via pod.Register. Guardrail: Save <2s for 500 entities.

## Acceptance Criteria

- [ ] GIVEN 100 entities with various components, WHEN Save/Load requests serialization, THEN all Active and Suspended entities with full component state are encoded via pod.Encode
- [ ] GIVEN a serialized save blob, WHEN Save/Load requests deserialization, THEN entities are reconstructed with identical GameEntityIDs, positions, and component values
- [ ] GIVEN a component type has no entities, WHEN serialization runs, THEN the empty store encodes as zero bytes (no error)

## Implementation Notes

From ADR-0001/0002: Each component type registers with pod at init time. The entity table (GameEntityID → EntityId mapping) is serialized first, then each component store in registration order. On load: entity table restored → entities recreated → component data distributed to stores.

## Test Evidence

- **Location**: `tests/integration/entity-framework/serialization_test.go`
- **Type**: Integration test
- **Covers**: AC-1 (100-entity encode via pod.Encode), AC-2 (round-trip: identical GameEntityIDs, positions, component values), AC-3 (empty store → zero bytes, no error)
- **Additional**: Verify that Pending entities are excluded from the encode pass (per GDD CR-8: only Active and Suspended entities are serialized)

## QA Test Cases

- **AC-2 — Round-trip**:
  - Given: 50 Active entities with Health, Stats, and Visual components at known positions
  - When: pod.Encode all stores → pod.Decode into fresh stores → rebuild entities
  - Then: All 50 entities have identical GameEntityIDs, Health values, Stats attributes, and positions as before encode.

## Completion Notes
**Completed**: 2026-08-10
**Criteria**: 3/3 passing (+7 edge-case tests)
**Deviations**: StateFlags changed from map[string]any to StateFlagValue tagged union (pod limitation); EntityData adapted to actual pod API; empty stores skipped at encode time (pod can't round-trip empty top-level maps)
**Test Evidence**: `tests/integration/entity/serialization_test.go` — 10 test functions, all passing; 30 pre-existing tests also pass (zero regressions)
**Code Review**: Skipped (Lean mode — user confirmed review before sprint close-out)
