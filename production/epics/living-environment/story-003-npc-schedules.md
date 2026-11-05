# Story 003: NPC Schedules

> **Epic**: Living Environment
> **Status**: Ready | **Layer**: Core | **Type**: Integration | **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09 | **Depends on**: Story 001

## Context
**GDD**: `design/gdd/living-environment.md` | **Req**: `TR-env-004` | **ADR**: ADR-0007

## Acceptance Criteria

- [ ] GIVEN an NPC with Schedule [{start: 8:00, end: 20:00, activity: "work"}], WHEN game time is 14:00, THEN the NPC is Active
- [ ] GIVEN the same NPC, WHEN game time is 22:00, THEN the NPC is Suspended (outside active hours)
- [ ] GIVEN game time crosses into an NPC's active hours, WHEN the clock advances, THEN the NPC transitions from Suspended to Active
- [ ] GIVEN an NPC suspended when the player talks to them, WHEN the player attempts interaction, THEN the NPC displays "away" state (not interactable)

## Implementation Notes
From ADR-0007: Schedule component on NPC entities. `schedule_entries[]` with start_time, end_time, activity, target_position. Active check: current game time falls within any entry's window. Suspended NPCs: Entity Framework Suspended state, not interactable.
