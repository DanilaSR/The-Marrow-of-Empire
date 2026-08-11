# Story 001: Attribute Model & Modifiers

> **Epic**: Character Stats & Progression
> **Status**: Complete
> **Last Updated**: 2026-08-11
> **Layer**: Core
> **Type**: Logic
> **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: EF-001 (Entity Lifecycle — Stats component attaches to entities via EF component store)

## Context

**GDD**: `design/gdd/character-stats-progression.md`
**Requirement**: `TR-stats-001`, `TR-stats-002`

**ADR**: ADR-0006 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: N/A — Stats component is pure Go math with no Kaiju engine API dependency. The component attaches to entities via EF-001 (already implemented).
**Control Manifest**: Core Layer — Required: six-attribute model STR/DEX/CON/INT/WIS/CHA, modifier = floor((A-10)/2) clamped [-4,+4]. Performance: attribute modifier <0.01ms (ADR-0006).
**Performance**: Attribute modifier calculation <0.01ms per ADR-0006. No heap allocations — Attributes is a fixed-size [6]int array.

## Acceptance Criteria

- [ ] GIVEN a Stats component, WHEN attributes are set to STR 16, DEX 12, CON 14, INT 10, WIS 10, CHA 8, THEN all six are stored and retrievable
- [ ] GIVEN attribute 14, WHEN modifier is calculated, THEN result is +2 (floor((14-10)/2))
- [ ] GIVEN attribute 18, WHEN modifier is calculated, THEN result is +4
- [ ] GIVEN attribute 3, WHEN modifier is calculated, THEN result is -4
- [ ] GIVEN attribute 20 (above cap), WHEN modifier is calculated, THEN result is clamped to +4
- [ ] GIVEN attribute 1 (below floor), WHEN modifier is calculated, THEN result is clamped to -4
- [ ] GIVEN attribute 10, WHEN modifier is calculated, THEN result is 0

## Implementation Notes

From ADR-0006: `Attributes [6]int` array in StatsComponent. `func (s StatsComponent) Modifier(attr int) int { return clamp((s.Attributes[attr]-10)/2, -4, 4) }`. Integer division truncates toward zero in Go — matches the floor semantic for positive values. For negative values, use `(s.Attributes[attr]-10)/2` with integer division (Go truncates toward zero, which matches floor for negative even differences; for odd negative differences, adjust with a conditional).

## Out of Scope

- Point-buy character creation (story 003) — this story only defines the data model and modifier formula
- Derived stats calculation (STAT-002)
- XP, leveling, and attribute point allocation on level-up (STAT-003)
- Save/load serialization (SAVE-001)

## Test Evidence

**Test file**: `tests/unit/stats/attribute_test.go`
**Type**: Logic — unit test required and blocking
**Tests**: 7 (one per acceptance criterion)

## QA Test Cases

- **AC-2 — Modifier +2**: Given attribute 14, When modifier calculated, Then result = +2.
- **AC-4 — Modifier -4**: Given attribute 3, When modifier calculated, Then result = -4.
- **AC-5 — Clamp**: Given attribute 20, When modifier calculated, Then result clamped to +4 (not +5).
