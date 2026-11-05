# Story 001: Game Clock & Light States

> **Epic**: Living Environment
> **Status**: Complete | **Layer**: Core | **Type**: Logic | **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context
**GDD**: `design/gdd/living-environment.md` | **Req**: `TR-env-001`, `TR-env-002` | **ADR**: ADR-0007

## Acceptance Criteria

- [x] GIVEN game start, WHEN the clock initializes, THEN game time is 8:00 AM (Day)
- [x] GIVEN 6 real seconds elapsed, WHEN the clock updates, THEN 1 game minute has advanced (10:1 ratio)
- [x] GIVEN game time 4:00 AM, WHEN the clock advances, THEN the light state transitions from Deep Night to Dawn with smooth interpolation over 2-5 game minutes
- [x] GIVEN game time 17:00, WHEN the clock advances, THEN Dusk begins (amber hour) with smooth transition
- [x] GIVEN game time 20:00, WHEN the clock advances, THEN Night begins (cool indigo with warm life-pockets)
- [x] GIVEN conversation active, WHEN the clock would advance, THEN time is paused during conversation

## Scope

### In Scope
- 24-hour game clock with 10:1 compression (2.4 real hours = 1 game day)
- Five light states with smooth interpolation at boundaries
- Time pause during conversation
- Clock initialization at 8:00 AM Day

### Out of Scope
- **Wait/pass-time mechanic** (rest points, fast-forward) — story-002
- **NPC schedules** (Schedule component, start/end times) — story-003
- **Combat time pause integration** (clock freeze during combat) — story-004
- **Ambient wildlife** (cosmetic NPCs) — story-004
- **Weather and seasons** — post-MVP

## Dependencies

| Story | Status | Rationale |
|-------|--------|-----------|
| EF-001 (Entity Lifecycle) | Required | Entity creation/destruction events via EventBus needed for clock-aware entities. Must be Complete before this story starts. |
| EF-002 (Component Model) | Not required for this story | Schedule component integration happens in story-003. Clock and light states are self-contained math + renderer output. |

**Note**: No story depends on this one for its own completion. ENV-002 (Wait/Pass-Time) builds on the clock but is a Should Have and not blocking.

## Implementation Notes

From ADR-0007: Time ratio `game_minutes = real_seconds × 10 / 60`. Five light states with interpolation using smooth-step between boundaries. Light state affects ambient color/intensity fed to the renderer. Time pauses: conversation, menu, journal, combat.

### Engine Verification Required
- **Renderer ambient light API**: Verify the Kaiju renderer's ambient color/intensity
  API against `kaiju/src/rendering/` before wiring light states. The clock math
  (`game_minutes = real_seconds × 10 / 60`) is pure Go — engine risk is in the
  renderer integration only.
- **Smooth-step interpolation**: Pure math (no engine API). Use `math.SmoothStep`
  or equivalent for light state transitions at boundaries (2-5 game minute window).

### Control Manifest Guardrails (v2026-08-09)
From Core Layer rules (ADR-0007):
- **Required**: Clock update <0.01ms/frame
- **Required**: Light state interpolation <0.1ms during transitions
- **Forbidden**: Never advance time during combat, conversation, menus, or journal
- **Required**: Smooth interpolation at all five light state boundaries

## Performance Budget

| Metric | Target | Source |
|--------|--------|--------|
| Clock update per frame | <0.01ms | ADR-0007, Control Manifest Core Layer |
| Light state interpolation (during transitions) | <0.1ms | ADR-0007, Control Manifest Core Layer |
| Steady-state (no transition) | <0.005ms | Implied by <0.01ms clock budget — simple integer increment |

**Verification**: Run a 24-hour simulation test (2.4 real hours → 24 game hours in wall-clock fast-forward) and profile clock update + interpolation overhead.

## Test Evidence

**Type**: Logic — automated unit tests required.

**Test file**: `tests/unit/env/clock_test.go` (actual; the story-planned path was `tests/unit/environment/clock_light_test.go` — see Completion Notes)

| Test | What it verifies |
|------|-----------------|
| `test_clock_initializes_at_8am` | Clock starts at 8:00 AM, light state = Day |
| `test_time_advance_ratio_10_to_1` | 6 real seconds → 1 game minute advanced |
| `test_light_transition_deep_night_to_dawn` | At 4:00 AM, state transitions Deep Night → Dawn with interpolation over 2-5 game minutes |
| `test_light_transition_dusk` | At 17:00, state transitions to Dusk (amber hour) |
| `test_light_transition_night` | At 20:00, state transitions to Night (cool indigo) |
| `test_time_paused_during_conversation` | Clock does not advance while conversation flag is active |
| `test_full_day_cycle` | 24-hour cycle completes correctly, all 5 states visited in order |

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 6/6 passing
**Deviations**: Package lives at `src/env/` (single-word lowercase per naming convention) rather than the `src/environment/` working name; test suite lives at `tests/unit/env/` (repo `[system]_[feature]_test.go` convention) rather than the story-planned `tests/unit/environment/clock_light_test.go`; `Update` accepts `float64` dt for accumulator precision (float32 loses ~3e-5 precision at the 1440-minute scale). Ambient color values are placeholders pending art direction (art bible Section 3 mood targets), as noted in `src/env/light.go`.
**Test Evidence**: `tests/unit/env/clock_test.go` — 20 test functions covering AC-1..AC-6 plus edge cases (midnight wrap, zero/negative dt, full-day cycle, SetTime wrap, state table, transition range), all passing. `tests/unit/env/benchmark_test.go` — ADR-0007 guardrails verified: clock update p95 ~76ns (<0.01ms), light interpolation p95 ~99ns (<0.1ms).
**Code Review**: Pending
