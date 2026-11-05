# ADR-0007: Game Clock & Environment Systems

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Core |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | Living Environment GDD (approved) |
| **Post-Cutoff APIs Used** | None — pure game logic |
| **Verification Required** | Time tracking integration with game loop, NPC schedule component in Entity Framework |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — Schedule component, wildlife entities) |
| **Enables** | Audio System (time-driven ambience), Map System (day/night tint), NPC schedules |
| **Blocks** | Living Environment implementation |
| **Ordering Note** | Must be Accepted before environment and NPC schedule systems |

## Context

The Marrow of Empire has a living world with a 24-hour game clock (10:1 compression ratio — 2.4 real hours per full day), five light states (Dawn/Day/Dusk/Night/Deep Night), NPC schedules, ambient wildlife, and a wait/pass-time mechanic. Combat pauses the clock. Weather and seasons deferred to later milestones.

## Decision

**Continuous 24-hour clock at 10:1 ratio, driven by frame delta time.** Time advances during exploration only; pauses during conversation, menus, journal, and combat. Five discrete light states with smooth interpolation at boundaries (2-5 game minutes per transition). Wait mechanic fast-forwards time at 100× speed.

### Clock Parameters

| Parameter | Value |
|-----------|-------|
| Time ratio | 1 real minute = 10 game minutes |
| Full cycle | 2.4 real hours = 24 game hours |
| Start time (new game) | 8:00 AM (Day) |
| Time advance | Exploration only |
| Time pause | Conversation, menu, journal, combat |

### Light States

| State | Game Time | Transition Duration |
|-------|-----------|-------------------|
| Dawn | 4:00-7:00 | 2-5 game minutes (12-30 real seconds) |
| Day | 7:00-17:00 | — |
| Dusk | 17:00-20:00 | 2-5 game minutes |
| Night | 20:00-2:00 | — |
| Deep Night | 2:00-4:00 | — |

### Wait / Pass Time
Interacting with a rest point opens wait dialog. Options: Dawn (6am), Noon (12pm), Dusk (6pm), Midnight (12am), or custom hours. Time fast-forwards at 100× (1 game hour = 36 real seconds) with fade-to-black transition. Rest at inn: full HP restore. Rest at camp: 50% HP restore.

### NPC Schedules
NPCs carry an optional Schedule component: `schedule_entries[]` with `start_time`, `end_time`, `activity`, `target_position`. NPC is active when game time falls within any entry; suspended otherwise.

## GDD Requirements Addressed
Living Environment GDD: CR-1 (Game Clock), CR-2 (Light States), CR-3 (Wait/Pass Time), CR-4 (NPC Time Awareness), CR-5 (Ambient Wildlife).

## Performance Implications
Clock update: <0.01ms per frame (delta accumulation). Light state interpolation: <0.1ms during transitions (every 2-5 game minutes). Wait fast-forward: ~500ms for full day skip (fade + time advance + state update). NPC schedule check: <0.1ms for 20 NPCs on region load.
