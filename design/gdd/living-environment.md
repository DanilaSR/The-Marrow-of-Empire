# Living Environment

> **Status**: Approved
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Review**: APPROVED (2026-08-09 — lean mode, 4 blockers resolved)
> **Implements Pillar**: Pillar 2 (The World Remembers — environment changes are visible memory)

## Summary

The Living Environment drives the passage of time in The Marrow of Empire —
a 24-hour clock at 10:1 compression with five light states (Dawn/Day/Dusk/
Night/Deep Night), a wait/pass-time mechanic at rest points, ambient wildlife
that exists for atmosphere, and NPC schedules tied to time of day. Combat
pauses the clock. Weather and seasons deferred to later milestones.

> **Quick reference** — Layer: `Core` · Priority: `MVP` · Key deps: `Entity Framework` · MVP scope: `day/night only`

## Overview

The Living Environment system makes the world of The Marrow of Empire feel
alive. It drives the passage of time — the day/night cycle, the golden hours
of dawn and dusk, the cool indigo of night — and populates the world with
the ambient life that makes a forest feel like a forest and a ruin feel
abandoned.

MVP scope — day/night only:

- **Time system**: A continuous 24-hour clock running at a compressed ratio
  (1 real minute = 10 game minutes → 2.4 hours per full day-night cycle).
  Time advances during exploration; it pauses during conversations and
  menus. The clock drives the light state machine.
- **Light states**: Five discrete light states that interpolate smoothly:
  Dawn (warm gold rising, 4am–7am), Day (full sunlight, 7am–5pm), Dusk
  (amber hour, the most beautiful time, 5pm–8pm), Night (cool indigo with
  warm life-pockets, 8pm–4am), and Deep Night (coldest, darkest hour,
  2am–4am). These map directly to the art bible's mood targets (Section 3).
- **Ambient wildlife**: Passive creatures — birds, deer, insects — that
  exist as entities in the world. They don't fight, don't give quests,
  don't drop loot. They exist to make the forest feel inhabited. Wildlife
  follows diurnal/nocturnal patterns driven by the time system.

Deferred to later milestones:
- **Weather** (VS): Rain, fog, storms, clear skies. Weather affects
  visibility, audio ambience, and NPC schedules. Fog muffles the gold;
  storms edge toward threat (art bible Section 3 layering note).
- **Seasons** (Alpha): Seasonal palette shifts — autumn gold richness,
  winter pale cold. Seasonal wildlife migration and behavior changes.

Without the Living Environment: the borderland is frozen at a single time
of day. There are no sunsets to watch, no lanterns flickering on at dusk,
no birdsong at dawn. The world is a diorama, not a place.

## Player Fantasy

The Living Environment delivers the player's deepest emotional anchor:
**the world is real, and you are in it.**

This is the system that makes a player stop moving and just look. The sun
setting behind a ruined archway, amber light filtering through Enchanted
leaves, lanterns flickering on in a border town as night falls — these are
not cutscenes. They happen because the clock advanced, the light state
changed, and the world responded. The player was there to see it.

**Time is visible.** The position of the sun, the color of the light, the
length of shadows — the player can read the time of day from the world
itself without a UI clock. Dawn feels like dawn. Dusk is the golden hour
the art bible promises. Night is cool indigo punctured by warm lantern
light. The passage of time is a visual experience, not a number on a HUD.

**The world doesn't need you.** Wildlife goes about its business whether
the player is there or not. Birds sing at dawn. Deer graze at the forest
edge. The world existed before the player arrived, and it will exist after
they leave. This is the isometric camera's promise made environmental:
you are an observer in a world that is complete without you.

**Reference**: The Long Dark (time and light as core gameplay — the feeling
of dusk approaching), Red Dead Redemption 2 (wildlife that exists for its
own sake, not as a resource), Baldur's Gate 1 (day/night cycles that make
the world feel lived-in).

## Detailed Design

### Core Rules

**CR-1: Game Clock**

A continuous 24-hour clock drives all environmental changes.

| Parameter | Value |
|-----------|-------|
| Time ratio | 1 real minute = 10 game minutes |
| Full cycle | 2.4 real hours = 24 game hours |
| Start time (new game) | 8:00 AM (Day) |
| Time advance condition | During exploration only |
| Time pause conditions | Conversation active, menu open, journal open, combat active |

Combat pauses time: the clock freezes when combat starts and resumes when
combat ends. Combat at night stays night for the duration of the encounter —
no jarring dawn transitions mid-fight.

**CR-2: Light States**

Five discrete states with smooth interpolation at boundaries:

| State | Game Time | Light Character | Art Bible Ref |
|-------|-----------|----------------|---------------|
| Dawn | 4:00–7:00 | Warm gold rising from horizon; shadows shorten; fog burns off | State 1 (Exploration — golden adventure) |
| Day | 7:00–17:00 | Full sunlight, warm amber baseline; high ambient warmth | State 1 (day signature) |
| Dusk | 17:00–20:00 | Amber hour — deepest gold, longest shadows; the most beautiful time | State 1 (dusk) |
| Night | 20:00–2:00 | Cool indigo baseline with warm life-pockets (lanterns, windows, campfires) | State 1 (night) |
| Deep Night | 2:00–4:00 | Coldest, darkest hour; minimal warm pockets; wildlife silent | State 1 (night extreme) |

Transitions between light states last 2–5 game minutes (12–30 real seconds)
with smooth interpolation of ambient light color, shadow length, and sky color.

**CR-3: Wait / Pass Time**

The player can advance time manually from a rest point (camp, inn, or safe
location — any interactable entity tagged `rest_point`):

- Interacting with a rest point opens the wait dialog: "Rest until when?"
  Options: Dawn (6am), Noon (12pm), Dusk (6pm), Midnight (12am). Plus a
  "Wait [N] hours" slider for custom duration.
- Time fast-forwards at 100× speed (1 game hour = 36 real seconds) with a
  fade-to-black transition (500ms out, time advances, 500ms in).
- During wait: all NPC schedules update, wildlife despawns/respawns,
  light state transitions to match new time.
- Wait is blocked during combat, conversation, and cutscenes.
- Resting at an inn restores health (full restore). Resting at a camp
  restores partial health (50%). Resting at an unmarked safe location
  restores no health but advances time.

**CR-4: NPC Time Awareness**

NPCs respond to time of day via the Entity Framework `Schedule` component
(EF component #12). The Schedule component defines per-NPC time windows:
`schedule_entries[]` with `start_time`, `end_time`, `activity`, and
`target_position` fields. The Living Environment reads Schedule components
and determines NPC availability: an NPC is **active** when the current
game time falls within any of its schedule entries; outside all entries,
the NPC is **Suspended** (Entity Framework CR-4 — reduced-fidelity update,
not interactable, visually present but faded or in an "away" pose).

Typical schedules for MVP:
- Shopkeepers: `[{start: 8:00, end: 20:00, activity: "work"}]`
- Guards: shift-based — `[{start: 6:00, end: 14:00, activity: "patrol"}]`
  or `[{start: 14:00, end: 22:00, activity: "patrol"}]`
- Night NPCs (tavern keeper, night watch): active during night hours —
  e.g., `[{start: 18:00, end: 2:00, activity: "work"}]`
- Most NPCs: `[{start: 6:00, end: 22:00, activity: "idle"}]`
- Hermits and nocturnal characters may use inverted or irregular schedules.

The `activity` and `target_position` fields are defined by other systems
(e.g., AI Dialogue Engine for conversation availability, Tactical Combat
Engine for guard alert states). The Living Environment owns only the
**time → active/suspended** determination.

**CR-5: Ambient Wildlife**

Wildlife entities exist for atmosphere, not gameplay:
- **Birds**: Small entities with simple flight paths. Diurnal — active
  during Day, roost during Night. Ambient birdsong during Dawn and Day.
- **Deer/small mammals**: Grazing at forest edges. Diurnal, flee from
  player proximity (flee radius: 8 units). No combat, no drops.
- **Insects**: Particle-based. Fireflies at Dusk/Night near water.
  Crickets audio at Night.
- **Predators** (deferred to VS): Wolves, bears — rare, dangerous, may
  attack if player gets too close. These ARE combat entities.

**Performance budget**: Maximum 30 wildlife entities per region at any time.
Wildlife entities beyond 60u from the camera (roughly 1.5× the Far zoom
viewport depth) are culled — their Visual component is deactivated and
audio is muted. The `wildlife_density` knob (default 0.3, range 0.1–0.8)
controls what fraction of available spawn points are populated; spawn point
counts per region must be authored such that `spawn_points × density_max ≤ 30`.
Wildlife entities use simplified meshes (≤ 200 tris) and do not cast shadows.

### States and Transitions

The environment's state is the current light state (Dawn/Day/Dusk/Night/
Deep Night). Transitions are time-driven or triggered by Wait. No player
action other than Wait directly changes the state.

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Entity Framework | Creates/manages wildlife entities | Wildlife spawned as entities with Transform + Visual + Audio components |
| Camera System | Environment lighting affects camera view | Ambient light color and intensity fed to renderer |
| Audio System | Audio reads time/light state for ambience | Dawn birdsong, night crickets, weather ambience |
| NPC Knowledge Graph | NPCs may have time-gated facts | "Come back at night" — facts with `discovery_triggers` keyed to time |
| Map System | Reads time for day/night map tint | Map reflects current light state |
| Tactical Combat Engine | Combat pauses time | Clock frozen during combat; light state held constant |

## Formulas

The time ratio formula:

`game_minutes = real_seconds × time_ratio / 60`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| real_seconds | Rs | float | any | Real time elapsed since last tick |
| time_ratio | Tr | int | 5–30 | Game minutes per real minute |
| game_minutes | Gm | float | any | Game time to advance |

**Output Range:** N/A — continuous accumulation
**Example:** 6 real seconds at ratio 10 → 6 × 10 / 60 = 1 game minute advanced.

Light interpolation uses smooth-step: `t = (current_time − state_start) / state_duration`, clamped [0,1].

## Edge Cases

- **If the player enters conversation during a light state transition**: The transition pauses and resumes when conversation ends. The light state catches up over 1 real second — no jarring jump.
- **If an NPC's active hours end while the player is talking to them**: The NPC finishes the conversation normally, then transitions to Suspended. They may mention "It's getting late" as a natural end.
- **If combat starts during a light state transition**: The transition completes to the current target state immediately at combat start. No mid-combat transitions.
- **If the player waits past an NPC's active hours**: The NPC is Suspended. The player must wait again to reach the NPC's next active period, or find them elsewhere.
- **If the player opens the main menu, inventory, or journal**: The game clock pauses. Time resumes when the UI closes. This prevents "menu penalty" where browsing inventory costs game hours.
- **If the player activates Wait during a light state transition**: The current transition completes to its target state first (instant completion, same as combat start), then the fast-forward begins. The wait duration is measured from the completed transition time.
- **If a region has no NPCs with active schedules at the current time** (e.g., Deep Night in a rural area): All NPCs in that region are Suspended. The region is functionally empty of interactable characters — ambient wildlife, lighting, and audio remain active. Rest points still function.
- **If the game is saved and loaded**: The game clock restores to the saved time (±1 game minute tolerance). The light state is derived from the restored time. Wildlife spawns are refreshed (entities are not persisted individually — they respawn based on time + spawn points). NPC schedule state is restored from serialized Schedule component data (EF-owned).

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Entity Framework | Living Environment depends on Entity Framework | Creates wildlife entities; reads NPC entities for Schedule component. Hard. |
| Save/Load System | Living Environment depends on Save/Load (serialization) | Game clock time, current light state, and per-NPC schedule progression must survive save/load. The clock serializes as a single game-time value (minutes since midnight). On load, the light state is derived from the restored time. Schedule state (which schedule entry an NPC is in) is owned by EF; LE re-derives active/suspended from the Schedule component after load. Hard. |
| Camera System | Camera reads ambient light from environment | Light state feeds ambient color/intensity to renderer. Soft. |
| Audio System | Audio reads time/light state | Ambient audio driven by time of day. Soft. |
| Tactical Combat Engine | Combat pauses environment clock | Time frozen during combat. Soft. |
| Map System | Map reads light state | Map tint reflects day/night. Soft. |

## Tuning Knobs

| Parameter | Default | Safe Range | Effect |
|-----------|---------|------------|--------|
| `time_ratio` (game min / real min) | 10 | 5–30 | Faster = shorter day cycle, more sunsets per session |
| `dawn_duration` (game minutes) | 180 (3 hrs) | 60–240 | Longer dawn = more golden morning light |
| `dusk_duration` (game minutes) | 180 (3 hrs) | 60–240 | Longer dusk = more amber hour |
| `night_light_state_count` | 2 (Night + Deep Night) | 1–3 | More gradations = richer night atmosphere |
| `wildlife_density` | 0.3 (30% of spawn points) | 0.1–0.8 | More wildlife = livelier world, more entity overhead |
| `wait_speed_multiplier` | 100× | 50–500× | Faster wait = quicker time skip, less immersive |

## Visual/Audio Requirements

📌 **Asset Spec** — Level/world Visual/Audio required.

| Event | Visual Feedback | Audio Feedback |
|-------|----------------|---------------|
| Dawn transition | Sky warms from indigo to gold; shadows shorten; mist dissipates | Birdsong begins; night ambience fades |
| Dusk transition | Sky deepens to amber; shadows lengthen; golden light pools | Birdsong fades; crickets begin; evening ambience |
| Night | Cool indigo ambient; warm lantern light entities activate; stars | Night ambience; crickets; distant owl; wind |
| Wildlife spawn | Bird entities appear; deer at forest edges | Birdsong, rustling leaves |
| Player near wildlife | Deer flee (quick movement away) | Startled rustle, hoofbeats |
| Wait/rest | Fade to black (500ms), time-lapse sky, fade in (500ms) | Clock ticking sped up (soft), ambient crossfade |

## Acceptance Criteria

- **GIVEN** a new game starts, **WHEN** the first frame of Exploration renders, **THEN** the game clock reads 8:00 AM and the light state is Day.

- **GIVEN** the game clock at ratio 10, **WHEN** 2.4 real hours have elapsed in exploration (no pauses), **THEN** 24 game hours have passed — the clock has completed one full day/night cycle and returned to the same time ± 1 game minute.

- **GIVEN** Day state (7:00–17:00), **WHEN** the clock advances to 17:00, **THEN** the light state transitions from Day to Dusk with smooth-step interpolation (`t = (current_time − state_start) / state_duration`, clamped [0,1]) over 2–5 game minutes (12–30 real seconds at ratio 10).

- **GIVEN** Deep Night state (2:00–4:00), **WHEN** the player is in any region, **THEN** the ambient light is the coldest value on the day/night curve, wildlife entities are despawned (zero active wildlife), and warm light-pocket entities (lanterns, windows) are reduced to ≤ 25% of their Night-state count.

- **GIVEN** the game clock is advancing during exploration, **WHEN** the player opens the main menu, inventory, or journal, **THEN** the game clock pauses and resumes when the UI is closed. No game time advances while any fullscreen UI is open.

- **GIVEN** Night state, **WHEN** the player enters a town, **THEN** lantern and window light entities are active, creating warm life-pockets against the cool indigo ambient.

- **GIVEN** Day state at a forest edge, **WHEN** the player approaches within 8 units, **THEN** deer entities flee from the player.

- **GIVEN** an NPC with a Schedule component containing `[{start: 8:00, end: 20:00}]`, **WHEN** game time is 22:00, **THEN** the NPC entity is Suspended and not interactable.

- **GIVEN** the player at a camp rest point, **WHEN** they choose "Wait until Dawn," **THEN** time fast-forwards at 100× speed with a 500ms fade-out/fade-in transition and the light state reaches Dawn at 6:00 AM.

- **GIVEN** the player at an inn rest point, **WHEN** they wait, **THEN** the party's health is restored to full. Given a camp rest point, health is restored to 50%. Given an unmarked safe location, health is unchanged.

- **GIVEN** combat at night, **WHEN** the encounter lasts 5 real minutes, **THEN** the game clock has not advanced — time is paused during combat.

- **GIVEN** conversation during a Dusk→Night transition, **WHEN** conversation ends, **THEN** the light state catches up to current time smoothly over 1 real second.

- **GIVEN** the game is saved at 14:30 (Day) and loaded, **WHEN** the load completes, **THEN** the game clock reads 14:30 ± 1 game minute and the light state is Day.

## Open Questions

| Question | Owner | Deadline |
|----------|-------|----------|
| Should waiting at a camp have a risk of ambush (random encounter), or is rest always safe in MVP? | Game Designer | Before MVP implementation |
| Should different regions have different time ratios (e.g., deeper ruins = time feels slower)? | Game Designer | Before vertical slice |
| Is game time global (all regions share one clock) or per-region? MVP is single-region so this is moot, but it affects VS+ architecture. | Technical Director | Before vertical slice (multi-region) |
