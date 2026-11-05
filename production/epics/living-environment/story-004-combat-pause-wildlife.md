# Story 004: Combat Clock Pause & Ambient Wildlife

> **Epic**: Living Environment
> **Status**: Ready | **Layer**: Core | **Type**: Integration | **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09 | **Depends on**: Story 001

## Context
**GDD**: `design/gdd/living-environment.md` | **Req**: `TR-env-005`, CR-5 (Wildlife) | **ADR**: ADR-0007

## Acceptance Criteria

- [ ] GIVEN combat starts at night, WHEN the encounter lasts 5 real minutes, THEN game time has not advanced — clock is paused during combat
- [ ] GIVEN combat ends, WHEN the encounter resolves, THEN the clock resumes from the paused time
- [ ] GIVEN Day state at a forest edge, WHEN the player approaches within 8 units, THEN deer entities flee
- [ ] GIVEN Dawn state, WHEN the light state transitions, THEN birdsong audio begins and bird entities spawn
- [ ] GIVEN Night state, WHEN the clock advances, THEN diurnal wildlife despawns and nocturnal wildlife (fireflies, owls) appears

## Implementation Notes
From ADR-0007: Combat start → clock paused. Combat end → clock resumed. Wildlife: simple entities with diurnal/nocturnal spawn patterns. Flee behavior: move away from player at 2× player speed when within 8 units. No combat, no drops — atmosphere only.
