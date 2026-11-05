# Story 002: Grid Movement & AP Economy

> **Epic**: Tactical Combat Engine
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09

## Context

**GDD**: `design/gdd/tactical-combat-engine.md`
**Requirement**: `TR-combat-001`, `TR-combat-002`

**ADR Governing Implementation**: ADR-0009: Combat Engine, ADR-0006: Character Stats
**ADR Decision Summary**: Square grid, 1u=1m, diagonal 1 AP, AP costs: Move 1, Attack 2, Ability 1-3, Interact 1.

**Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN a square grid with 1u=1m tiles, WHEN a combatant moves diagonally, THEN cost is 1 AP (same as orthogonal)
- [ ] GIVEN a combatant with 7 AP, WHEN they move 5 tiles, THEN 5 AP consumed, 2 AP remain
- [ ] GIVEN a combatant at (3,5), WHEN movement range is calculated, THEN reachable tiles = all within Chebyshev distance ≤ AP
- [ ] GIVEN an occupied tile, WHEN pathfinding runs, THEN the tile is excluded
- [ ] GIVEN a combatant with 2 AP, WHEN they attack (2 AP), THEN AP reaches 0

## Implementation Notes

From ADR-0009: Integer grid (x, z) in XZ plane. Chebyshev distance: max(|dx|, |dz|). BFS pathfinding from current position constrained by AP budget. Blocked: occupied entities, wall/obstacle tags. AP costs: Move 1, Attack 2, Ability 1-3, Interact 1. AP does not carry over.

## Out of Scope

- Attack resolution (Story 003)
- Positioning bonuses (Story 004)
- Combat UI grid rendering
- AI pathfinding decisions

## QA Test Cases

- **AC-1**: Given (0,0) with 1 AP, When moving to (1,1), Then 1 AP consumed, position (1,1).
- **AC-2**: Given (0,0) with 7 AP, When moving 5 tiles, Then AP = 2.
- **AC-3**: Given (0,0) with 2 AP, When calculating range, Then all Chebyshev ≤ 2 included.
- **AC-4**: Given obstacle at (1,0), When pathfinding, Then (1,0) excluded.
- **AC-5**: Given 2 AP, When attacking, Then AP = 0.

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/combat/grid_movement_test.go`
**Status**: [ ] Not yet created

## Dependencies

- Depends on: Story 001 (State Machine), STAT-002 (Derived Stats)
- Unlocks: Story 003 (Attack Resolution), Story 004 (Positioning)
