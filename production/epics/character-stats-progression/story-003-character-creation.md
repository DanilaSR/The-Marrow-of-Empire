# Story 003: Character Creation

> **Epic**: Character Stats & Progression
> **Status**: Complete | **Layer**: Core | **Type**: Logic | **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09 | **Last Updated**: 2026-08-11 | **Depends on**: Story 001, Story 002

## Context
**GDD**: `design/gdd/character-stats-progression.md` | **Req**: `TR-stats-004` | **ADR**: ADR-0006

## Acceptance Criteria

- [ ] GIVEN 30 starting points, WHEN the player distributes STR 16, DEX 12, CON 14, INT 10, WIS 10, CHA 8, THEN 22 points are spent (8+4+6+2+2+0) and 8 remain
- [ ] GIVEN all attributes start at 8, WHEN the player raises an attribute to N, THEN cost is N-8 points
- [ ] GIVEN the player attempts to raise an attribute above 16 at creation, THEN it is rejected (max starting attribute)
- [ ] GIVEN the player confirms creation, WHEN the Stats component is created, THEN all six attributes match the allocation and derived stats are computed

## Implementation Notes
From ADR-0006: All attributes start at 8. Linear cost: `N - 8`. Max starting: 16. 30 points total. Confirmation creates the Stats component on the player entity.

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 4/4 passing
**Deviations**: None
**Test Evidence**: `tests/unit/stats/creation_test.go` — 7 tests, all PASS
**Code Review**: Approved with suggestions (lead-programmer + qa-tester, 2026-08-11)
