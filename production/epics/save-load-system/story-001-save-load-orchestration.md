# Story 001: Save/Load Orchestration

> **Epic**: Save/Load System
> **Status**: Complete
> **Last Updated**: 2026-08-11
> **Layer**: Foundation
> **Type**: Integration
> **Estimate**: L (6-8 hours)
> **Manifest Version**: 2026-08-09

## Context

**GDD**: `design/gdd/save-load-system.md`
**Requirement**: `TR-save-001`, `TR-save-002`

**ADR**: ADR-0002 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN 100 entities with components, WHEN Save(slotID) is called, THEN all component stores are collected, encoded via pod, and written to .sav file within 2 seconds
- [ ] GIVEN a .sav file, WHEN Load(slotID) is called, THEN entities are reconstructed with identical GameEntityIDs, positions, and component values
- [ ] GIVEN a save from version N, WHEN version N+1 loads it with a registered migration, THEN the migration runs and data loads correctly
- [ ] GIVEN 20 manual save slots + 1 auto + 1 quicksave, WHEN the player saves, THEN slots are managed without exceeding max
- [ ] GIVEN F5 during exploration, WHEN quicksave completes, THEN "Game Saved" indicator appears for 1.5 seconds
- [ ] GIVEN F9 with valid quicksave, WHEN quickload completes, THEN game state matches the quicksave moment

## Implementation Notes

From ADR-0002: Save file format: [Header 16B] [Metadata] [Entity Table] [Component Stores] [World State] [Investigation]. Full-state pod serialization. Save version uint32 in header. Migration functions registered per version transition. 20 manual + 1 auto + 1 quicksave slots.

## QA Test Cases

> Back-filled by /qa-plan 2026-08-10. Source: story acceptance criteria + ADR-0002.

- **AC-1 — Save 100 entities**:
  - Given: 100 entities with components
  - When: Save(slotID) is called
  - Then: All component stores collected, encoded via pod, and written to .sav file within 2 seconds.

- **AC-2 — Load round-trip**:
  - Given: A .sav file
  - When: Load(slotID) is called
  - Then: Entities reconstructed with identical GameEntityIDs, positions, and component values.

- **AC-3 — Version migration**:
  - Given: A save from version N
  - When: Version N+1 loads it with a registered migration
  - Then: Migration runs and data loads correctly.

- **AC-4 — Slot management**:
  - Given: 20 manual save slots + 1 auto + 1 quicksave configured
  - When: Player saves
  - Then: Slots are managed without exceeding max.

- **AC-5 — Quicksave indicator**:
  - Given: F5 during exploration
  - When: Quicksave completes
  - Then: "Game Saved" indicator appears for 1.5 seconds.

- **AC-6 — Quickload**:
  - Given: F9 with valid quicksave
  - When: Quickload completes
  - Then: Game state matches the quicksave moment.

**Edge cases**:
- Entity with zero components: serialized and deserialized as componentless entity
- Maximum component count: entity with all 12+ component types serialized correctly
- Corrupted .sav file → checksum (CRC32) detects, .bak restored
- No quicksave exists → F9 → no-op, no crash

**Estimated test count**: ~6 integration tests
**Test file**: `tests/integration/save-load/save_load_test.go`

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 6/6 passing — Save/Load with pod serialization, version migration, slot management (20+1+1), quicksave/quickload, corrupt detection (CRC32)
**Deviations**: Header uses magic+CRC32 instead of magic-only (story QA edge case requires CRC32); slots 0-19 vs 1-20; QuickLoad returns ErrSaveNotFound (input layer handles as no-op)
**Test Evidence**: `tests/integration/save-load/save_load_test.go` — 10 test functions, all passing
**Code Review**: Skipped (Lean mode)
