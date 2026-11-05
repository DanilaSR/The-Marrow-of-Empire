# Epic: Living Environment

> **Layer**: Core
> **GDD**: design/gdd/living-environment.md
> **Architecture Module**: Living Environment
> **Status**: Ready
> **Stories**: 4 stories

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Game Clock & Light States | Logic | Ready | ADR-0007 |
| 002 | Wait / Pass-Time Mechanic | Logic | Ready | ADR-0007 |
| 003 | NPC Schedules | Integration | Ready | ADR-0007 |
| 004 | Combat Clock Pause & Wildlife | Integration | Ready | ADR-0007 |

## Overview

Implement the game clock (24-hour, 10:1 ratio), five light states (Dawn/Day/
Dusk/Night/Deep Night) with smooth interpolation, wait/pass-time mechanic at
rest points, NPC schedules via the Schedule component, ambient wildlife
(diurnal/nocturnal patterns), and combat clock pause. Weather and seasons
deferred to VS/Alpha. MVP: day/night only.

## Governing ADRs

| ADR | Decision Summary | Engine Risk |
|-----|-----------------|-------------|
| ADR-0007: Game Clock | 10:1 time ratio, 5 light states, wait mechanic, NPC schedules | HIGH |

## GDD Requirements

| TR-ID | Requirement | ADR Coverage |
|-------|-------------|--------------|
| TR-env-001 | 24-hour clock, 10:1 ratio | ADR-0007 ✅ |
| TR-env-002 | Five light states with interpolation | ADR-0007 ✅ |
| TR-env-003 | Wait/pass-time mechanic | ADR-0007 ✅ |
| TR-env-004 | NPC schedules via Schedule component | ADR-0007 ✅ |
| TR-env-005 | Combat pauses clock | ADR-0007 ✅ |

## Definition of Done

- Game clock advances at 10:1 ratio during exploration
- Light states transition smoothly (Dawn→Day→Dusk→Night→Deep Night)
- Wait mechanic works at rest points (camp, inn)
- NPCs suspend outside active hours
- Combat freezes the clock

## Next Step

Run `/create-stories living-environment`
