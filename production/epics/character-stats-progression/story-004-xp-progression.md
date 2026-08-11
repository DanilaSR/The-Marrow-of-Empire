# Story 004: XP & Level Progression

> **Epic**: Character Stats & Progression
> **Status**: Complete | **Last Updated**: 2026-08-11 | **Layer**: Core | **Type**: Logic | **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09 | **Depends on**: Story 001, Story 002

## Context
**GDD**: `design/gdd/character-stats-progression.md` | **Req**: `TR-stats-005` | **ADR**: ADR-0006

## Acceptance Criteria

- [ ] GIVEN level 1 (0 XP), WHEN 300 XP is awarded, THEN character reaches level 2
- [ ] GIVEN level 4 (1800 XP), WHEN 1200 more XP is awarded (total 3000), THEN character reaches level 5
- [ ] GIVEN level-up, WHEN processed, THEN MaxHealth increases by CON_mod (minimum +1)
- [ ] GIVEN level 4, 8, 12, 16, WHEN level-up occurs, THEN the player gains +1 attribute point to allocate
- [ ] GIVEN level 20, WHEN XP is awarded, THEN XP accumulates but no further level-up occurs (cap)
- [ ] GIVEN CON 6 (-2) leveling from 1 to 2, WHEN health is recalculated, THEN MaxHealth increases by at least +1 (minimum gain rule)

## Implementation Notes
From ADR-0006: `XP_to_next = Level × 300`. `Total_XP_for_N = 150 × N × (N-1)`. Health on level-up: `max(1, CON_mod)`. Attribute point every 4 levels. Level cap: 20.

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 6/6 passing — XP formulas, level-up health/attribute gains, level 20 cap, minimum +1 health gain
**Deviations**: AttributePoints field added to StatsComponent (needed for save persistence); level 20 excluded from attribute point grant per story AC-4
**Test Evidence**: `tests/unit/stats/xp_test.go` — 21 test functions, all passing
**Code Review**: Skipped (Lean mode)
