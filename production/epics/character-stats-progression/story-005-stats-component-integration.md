# Story 005: Stats Component Integration

> **Epic**: Character Stats & Progression
> **Status**: Complete | **Layer**: Core | **Type**: Integration | **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09 | **Last Updated**: 2026-08-11 | **Depends on**: Story 001-004

## Context
**GDD**: `design/gdd/character-stats-progression.md` | **Req**: All TR-stats-* | **ADR**: ADR-0006

## Acceptance Criteria

- [x] GIVEN a character entity, WHEN the Stats component is attached via Entity Framework, THEN all six attributes and derived stats are accessible via GetComponent[StatsComponent]
- [x] GIVEN a Save operation, WHEN the Stats component is serialized via pod, THEN all attributes, level, and XP are preserved in the save file
- [x] GIVEN a Load operation, WHEN the Stats component is deserialized, THEN attributes, level, and XP match the saved values
- [x] GIVEN a level-up during combat, WHEN XP is awarded, THEN the Combat UI updates to reflect new MaxHealth and AP values

## Implementation Notes
Stats component registered with pod.Register at init. Serialization includes: Attributes[6], Level, XP. Derived stats are NOT serialized — they are recomputed from attributes on load (idempotent).

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 4/4 passing
**Deviations**: None
**Test Evidence**: `tests/integration/stats/stats_integration_test.go` — 8 tests, all PASS
**Code Review**: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester, 2026-08-11). All 4 suggestions applied.
