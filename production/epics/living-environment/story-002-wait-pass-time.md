# Story 002: Wait / Pass-Time Mechanic

> **Epic**: Living Environment
> **Status**: Ready | **Layer**: Core | **Type**: Logic | **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09 | **Depends on**: Story 001

## Context
**GDD**: `design/gdd/living-environment.md` | **Req**: `TR-env-003` | **ADR**: ADR-0007

## Acceptance Criteria

- [ ] GIVEN the player at a rest point, WHEN they interact, THEN the wait dialog opens with preset options (Dawn 6am, Noon 12pm, Dusk 6pm, Midnight 12am) + custom slider
- [ ] GIVEN "Wait until Dawn" selected at 22:00, WHEN confirmed, THEN time fast-forwards at 100× with fade-to-black transition and the light state reaches Dawn (6:00 AM)
- [ ] GIVEN an inn rest point, WHEN the player rests, THEN health is fully restored
- [ ] GIVEN a camp rest point, WHEN the player rests, THEN health is restored by 50%
- [ ] GIVEN combat active, WHEN the player attempts to wait, THEN wait is blocked

## Implementation Notes
From ADR-0007: Wait speed 100× (1 game hour = 36 real seconds). Fade to black 500ms, time advances, fade in 500ms. Rest point types: inn (full heal), camp (50% heal), unmarked (no heal, time only). Blocked during combat, conversation, cutscene.
