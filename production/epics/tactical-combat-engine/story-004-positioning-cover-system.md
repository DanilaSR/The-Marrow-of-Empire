# Story 004: Positioning & Cover System

> **Epic**: Tactical Combat Engine
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09

## Context

**GDD**: `design/gdd/tactical-combat-engine.md`
**Requirement**: `TR-combat-005`

**ADR Governing Implementation**: ADR-0009: Combat Engine
**ADR Decision Summary**: Flanking +2 accuracy, half cover +2 evasion, full cover blocks LoS, elevation +1 accuracy/range per level.

**Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN attacker and ally on opposite sides of defender, WHEN attacking, THEN flanking +2 accuracy
- [ ] GIVEN defender adjacent to half cover, WHEN attacked, THEN defender +2 evasion
- [ ] GIVEN solid wall between attacker and defender, WHEN targeting, THEN LoS blocked
- [ ] GIVEN attacker 2 levels above defender, WHEN attacking, THEN accuracy +2, range +2
- [ ] GIVEN attacker adjacent to defender, open ground, THEN no modifiers apply

## Implementation Notes

From ADR-0009: Flanking: 180° arc check on XZ grid. Half cover: obstacle adjacent to defender. Full cover: Bresenham line check. Elevation: +1 acc/range per level above. Modifiers additive — flanking + elevation = +3 accuracy.

## Out of Scope

- 3D raycasting (grid-level Bresenham only)
- Destructible cover
- Environmental hazards
- Combat UI indicators

## QA Test Cases

- **AC-1**: attacker(2,3), defender(3,3), ally(4,3) → flanking +2 acc.
- **AC-2**: defender adjacent to low wall → evasion +2.
- **AC-3**: solid wall between → LoS blocked.
- **AC-4**: attacker elev 3, defender elev 1 → +2 acc, +2 range.
- **AC-5**: adjacent, open ground → no modifiers.

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/combat/positioning_test.go`
**Status**: [ ] Not yet created

## Dependencies

- Depends on: Story 001, Story 002
- Unlocks: Story 006 (Combat Resolution)
