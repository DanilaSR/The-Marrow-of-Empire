# Story 002: Derived Stats

> **Epic**: Character Stats & Progression
> **Status**: Complete | **Last Updated**: 2026-08-11 | **Layer**: Core | **Type**: Logic | **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09 | **Depends on**: Story 001

## Context
**GDD**: `design/gdd/character-stats-progression.md` | **Req**: `TR-stats-003` | **ADR**: ADR-0006

## Acceptance Criteria

- [ ] GIVEN CON 14 (+2) at level 3, WHEN MaxHealth is calculated, THEN result is 20 + (14×2) + (3×2) = 54
- [ ] GIVEN DEX 16 (+3), WHEN AP is calculated, THEN result is 4 + 3 = 7
- [ ] GIVEN DEX 6 (-2), WHEN AP is calculated, THEN result is max(2, 4-2) = 2 (minimum 2)
- [ ] GIVEN DEX 14 (+2) and WIS 12 (+1), WHEN Initiative is calculated, THEN result is 2 + 1 = 3
- [ ] GIVEN DEX 12 (+1), WHEN Evasion is calculated, THEN result is 10 + 1 = 11
- [ ] GIVEN STR 16 (+3), WHEN Carry Capacity is calculated, THEN result is (16×10) + 30 = 190 lbs

## Implementation Notes
From ADR-0006: Derived stats computed from attributes + modifiers on read. No caching unless profiling shows hot path. Health: `20 + CON×2 + Level×CON_mod`. AP: `max(2, 4 + DEX_mod)`. Initiative: `DEX_mod + WIS_mod`. Evasion: `10 + DEX_mod`. Carry: `STR×10 + 30`.
