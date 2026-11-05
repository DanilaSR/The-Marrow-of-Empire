# Persistent World State

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 2 (The World Remembers — player actions echo across sessions)

## Summary

[To be designed]

> **Quick reference** — Layer: `Feature` · Priority: `VS` · Key deps: `Entity Framework`, `Save/Load System` · MVP note: `Trust & Relationship (MVP) depends on PWS with provisional get_flags/set_flags interface`

## Overview

The Persistent World State system is the memory of The Marrow of Empire — the
data layer that records what the player has changed so those changes survive
across sessions and are legible to every other system. It owns the `world.*`
namespace of the Entity Framework's `Persistent` component: a key-value flag
store per entity that tracks discrete world facts (ruin cleared, door unlocked,
NPC rescued, bridge repaired) and a structured event log (`memory_log[]`) that
records what happened, when, and who was involved. Systems do not modify world
state directly — they submit events to PWS, which resolves them into flag
mutations and appends to the entity's memory. Downstream systems (Trust &
Relationship, Faction System, Map System, Investigation Journal) query PWS to
answer "what does the world know about this entity?" and "what happened here
while I was gone?" Without PWS, every session is a fresh start — the world has
no memory of the player, and the player's actions have no permanence.

> **Quick reference** — Layer: `Feature` · Priority: `VS` · Key deps: `Entity Framework` (reads/writes `Persistent` component), `Save/Load System` (serializes world state section) · MVP note: `Trust & Relationship (MVP) depends on PWS with a provisional get_flags/set_flags interface — MVP may store trust in Knowledge component directly`

## Player Fantasy

The Persistent World State delivers the promise at the heart of Pillar 2:
**your actions matter, and the world proves it.**

This is the system that makes a player return to a ruin they cleared three
sessions ago and find it still empty — or, better, find that something new has
moved in. It's what makes an NPC say "I heard what you did at Thornwell"
without a quest script telling them to. The player's actions leave footprints,
and those footprints are visible to anyone paying attention.

**The world has a memory, and you can read it.** When the player investigates
a location, talks to an NPC, or checks their journal, they are reading the
world's memory — a record of what happened, who was involved, and what changed.
This isn't a quest log; it's the world's autobiography, written in the player's
actions.

**You are not the only thing that changes.** The world changes on its own —
factions shift, wildlife moves, ruins are reclaimed. The player's actions set
events in motion, but the world continues to evolve around them. A cleared ruin
may attract bandits. A rescued NPC may become a faction leader. The player is
the most important agent of change, but not the only one.

**Reference**: Divinity Original Sin 2 (world reactivity — NPCs acknowledge
player actions across acts), The Legend of Zelda: Breath of the Wild (persistent
world state — opened chests stay open), Baldur's Gate 2 (stronghold quests where
player decisions reshape a location permanently).

## Detailed Design

### Core Rules

**CR-1: World Flag Store**

Persistent World State owns the `world.*` namespace of the Entity Framework's
`Persistent` component. Every game entity with a `Persistent` component can
carry world state flags — key-value pairs that record what the player has changed.

Flags are typed: `bool` (ruin cleared, door unlocked), `int` (bandit occupation
count, days since last visit), `float` (faction influence in region, 0.0–1.0),
`string` (current occupant entity ID, last visitor name). Flag names are dotted
strings scoped to the entity and region: `world.ruin_cleared`,
`world.thornwell.bridge_repaired`, `world.old_mine.bandit_count`.

PWS is the authoritative writer of `world.*` flags per EF's write-ownership table.
Other systems read flags via PWS queries and request changes via PWS event
submission. No system writes `world.*` flags directly.

**CR-2: World Event Submission**

Systems notify PWS of world-significant events by calling `SubmitWorldEvent(event)`.
PWS resolves each event into flag mutations and a memory log entry.

Event types:

| Event Type | Triggering System(s) | Example Flag Mutations |
|------------|---------------------|----------------------|
| `location_cleared` | Tactical Combat Engine | `world.ruin_cleared = true`, `world.[location].cleared_at = timestamp` |
| `structure_changed` | Living Environment, Interaction | `world.[door_id].unlocked = true`, `world.[bridge_id].repaired = true` |
| `npc_state_changed` | Tactical Combat Engine, AI Dialogue Engine | `world.[npc_id].alive = false`, `world.[npc_id].recruited = true` |
| `item_world_event` | Inventory System | `world.[item_id].taken = true`, `world.[item_id].placed_at = position` |
| `faction_event` | Faction System | `world.faction.[faction_id].standing = value` |
| `discovery_event` | Thread Investigation | `world.[location].discovered = true`, `world.secret.[id].found = true` |

Each event carries: `event_type`, `actor_entity_id` (who did it — may be the player),
`target_entity_id` (what was affected), `region_id` (where it happened), `details`
(JSON blob for system-specific context), and `flag_mutations` (map of flag name →
new value — the submitting system proposes mutations; PWS validates and applies).

**CR-3: Memory Log**

Every world event is recorded in the affected entity's `Persistent.memory_log[]` —
an append-only structured log. Each entry:

| Field | Type | Description |
|-------|------|-------------|
| `event_id` | uint64 | Monotonic, globally unique |
| `event_type` | enum | From the CR-2 event type table |
| `game_timestamp` | float | Game time in minutes since midnight of day 0 |
| `actor_entity_id` | GameEntityID | Who performed the action |
| `target_entity_id` | GameEntityID | Primary entity affected |
| `region_id` | string | Which region |
| `details` | string | JSON blob — system-specific context |

The memory log is capped at `memory_log_max_entries` per entity (default 50, EF
tuning knob). When the cap is reached, the oldest entry is evicted. Critical
events (quest-relevant, unique discoveries) are marked `protected: true` and are
never evicted — they can only be removed by a designer-authored world-state migration.

**CR-4: State Queries**

PWS exposes a query interface for downstream systems:

| Query | Returns | Use |
|-------|---------|-----|
| `GetWorldFlags(entity_id)` | `map[string]interface{}` — all `world.*` flags | Trust & Relationship, Dialogue UI |
| `GetWorldFlag(entity_id, flag)` | Single typed value + `bool` found | Combat UI, Map System |
| `GetMemoryLog(entity_id, limit, since)` | `[]MemoryEntry` — recent events, newest first | Investigation Journal, NPC dialogue context |
| `GetEntitiesWithFlag(flag, region_id?)` | `[]GameEntityID` | Map System ("Show cleared ruins") |
| `GetRecentWorldEvents(region_id, since)` | `[]MemoryEntry` across all entities in region | NPC knowledge ("I heard what happened...") |

All queries are read-only. Performance: queries filter on indexed fields (entity_id,
flag name prefix, region_id). Memory log queries support `limit` and `since_timestamp`
for incremental reads.

**CR-5: World Evolution (VS scope — expanded in Alpha)**

The world changes autonomously based on time elapsed and player actions. Evolution
rules are authored per region and evaluated on region load (not continuously).

A rule is: `condition → world_event`. Example:
`IF world.ruin_cleared == true AND game_time - world.ruin_cleared_at > 7_days
THEN SubmitWorldEvent(location_repopulated, {occupant: "bandit_camp"})`

At VS scope, the rule set is: one rule per cleared location (repopulation), and
one rule per killed/rescued NPC (faction propagation). Full autonomous
faction/wildlife simulation is Alpha-tier.

**CR-6: Quest-Critical Entity Protection**

When a quest-relevant or clue-holding entity is destroyed (e.g., an NPC killed
while the player is away), PWS logs a structured event that the player can
discover. EF CR-4 requires PWS to receive a `clue_path_protection` event on
destruction of any `quest-relevant` entity. PWS records the event and flags the
entity as `world.[entity_id].destroyed_with_clues = true` so the Investigation
Journal can surface it. No clue path permanently closes without an authored
discovery path.

### States and Transitions

Persistent World State is stateless from the player's perspective — it does not
have gameplay states. Its internal state is the set of world flags and memory log
entries, which change only via event submission.

**Event Processing Pipeline:**
1. System calls `SubmitWorldEvent(event)`.
2. PWS validates the event (actor exists, target exists, event_type is valid,
   region_id is valid).
3. PWS applies flag mutations to the target entity's `Persistent.state_flags[world.*]`
   namespace (validated: flag name is in `world.*` namespace, type matches).
4. PWS appends a memory log entry to the target entity's `Persistent.memory_log[]`,
   evicting oldest unprotected entry if cap is reached.
5. PWS broadcasts `on_world_state_changed` (EF event) with the event summary —
   subscribers (Trust & Relationship, Map System, Investigation Journal) react in
   the same frame.

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Entity Framework | Mutual | PWS reads/writes `Persistent` component `world.*` flags and `memory_log[]`. PWS subscribes to `on_entity_destroyed` for clue-protection events. PWS broadcasts `on_world_state_changed` for downstream systems. |
| Save/Load System | PWS depends on Save/Load | World state section of save file: all `world.*` flags + all memory log entries (protected + recent). On load, PWS restores flags and logs, then re-evaluates world evolution rules for the elapsed game time. |
| Trust & Relationship | T&R depends on PWS | T&R reads `world.*` flags via `GetWorldFlags(entity_id)`. T&R writes trust state to `trust.*` namespace (T&R-owned); PWS provides the query interface. Provisional: MVP may bypass PWS and store trust directly in the Knowledge component until PWS is implemented. |
| Faction System | Faction depends on PWS | Faction System submits `faction_event` on standing changes. PWS stores faction flags and propagates faction shifts to related NPCs via evolution rules. |
| Investigation Journal | Journal depends on PWS | Journal queries `GetMemoryLog` and `GetEntitiesWithFlag` to surface world changes: "This ruin was cleared 3 days ago" or "The hermit hasn't been seen since the raid." |
| Map System | Map depends on PWS | Map reads `world.*` flags to reflect world state: cleared locations marked, destroyed structures shown, faction territories updated. |
| Living Environment | Coordinates | LE's wildlife spawning reads world state flags (cleared ruin → different wildlife table). LE submits `structure_changed` events for environmental interactions. |
| Tactical Combat Engine | TCE depends on PWS | TCE submits `location_cleared` and `npc_state_changed` events on combat resolution. |

## Formulas

The Persistent World State is primarily a data system. Its "formulas" are
structural rules rather than arithmetic calculations.

**Event Processing Pipeline**

```
SubmitWorldEvent(event) →
  1. Validate: actor exists, target exists, event_type valid, region_id valid
     → valid | rejected (error returned to caller)
  2. For each (flag, value) in event.flag_mutations:
       if flag matches world.* AND type matches registered schema:
         Persistent.state_flags[flag] = value
       else:
         skip (log warning, event still processed)
  3. entry = MemoryEntry{next_event_id, event.event_type, current_game_time,
         event.actor_entity_id, event.target_entity_id, event.region_id,
         event.details}
  4. Persistent.memory_log = append(Persistent.memory_log, entry)
  5. if len(Persistent.memory_log) > memory_log_max_entries:
       evict oldest entry where protected != true
  6. broadcast(on_world_state_changed, entry)
```

**World Evolution Rule Evaluation** (on region load):

```
For each rule in region.world_evolution_rules:
  if evaluate(rule.condition, world_flags, game_time):
    SubmitWorldEvent(rule.event)
```

Condition operators: `==`, `!=`, `>`, `<`, `AND`, `OR`, `days_since(flag_name)`,
`has_flag(entity_id, flag_name)`.

**Memory Log Eviction**: Unprotected entries evicted oldest-first. Protected
entries never evicted by the cap. Removing a protected entry requires a
designer-authored world-state migration (see CR-6).

**Flag Type Schema**: Flag types are declared per region:
```
world.ruin_cleared: bool
world.bandit_count: int
world.faction_influence: float
world.current_occupant: string
```
Type mismatch on write → logged warning, flag not mutated.

## Edge Cases

- **If a system submits an event for a destroyed entity**: PWS validates target
  existence. If the target entity has been destroyed, the event is rejected with
  an error. Destroyed entity memory logs are preserved in the save file and
  queryable by entity ID (IDs are never reused per EF CR-1).

- **If an event's flag mutations include non-world.* flags**: PWS silently skips
  non-`world.*` flags (logs a warning). The event is still processed and the
  memory log entry is still written. Other systems own their own namespaces per
  EF's Persistent flag model.

- **If the memory log cap is reached and all entries are protected**: The new
  entry is still appended (cap is temporarily exceeded, warning logged). The cap
  is a soft limit for memory budgeting, not a hard correctness constraint. On
  the next non-protected entry, eviction catches up.

- **If two systems submit events for the same entity in the same frame**: Events
  are processed in submission order. Both flag mutations are applied — if they
  write the same flag, EF's write arbitration rules apply (last writer by
  priority wins). Both memory log entries are appended.

- **If the game clock moves backwards** (save loaded from earlier time, or debug
  time travel): PWS does not validate temporal monotonicity. Events are appended
  with whatever `game_timestamp` they carry. Memory log queries with
  `since_timestamp` may return already-known events — consumers are responsible
  for deduplication by `event_id`.

- **If a world evolution rule creates a circular dependency**: Rules are
  evaluated once per region load in declaration order. No re-evaluation loop.
  Designers are responsible for avoiding circular logic; PWS logs a warning if
  a rule modifies a flag read by an earlier rule in the same evaluation pass.

- **If an entity is destroyed while it has pending world evolution rules**: Rules
  targeting destroyed entities are skipped during evaluation (warning logged).
  The rule stays in the region's rule set and continues to skip until a designer
  removes it.

- **If the player loads a save from a much older game version**: The save
  version header triggers a world-state migration pass before PWS restores flags.
  Migration functions are version-specific and authored per release. If no
  migration exists for a version gap, PWS loads flags matching the current
  schema and skips unknown flags (logged). Memory log entries with unknown
  `event_type` values are preserved but marked `legacy: true`.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Entity Framework | PWS depends on EF | Reads/writes `Persistent` component (`world.*` flags, `memory_log[]`). Subscribes to `on_entity_destroyed` for clue-protection. Broadcasts `on_world_state_changed`. **Hard.** |
| Save/Load System | PWS depends on Save/Load | World state serialized into save file's World State section. On load: flags restored, logs restored, evolution rules re-evaluated for elapsed time. **Hard.** |
| Living Environment | Coordinates with PWS | LE reads `world.*` flags for wildlife spawning tables. LE submits `structure_changed` events. Soft. |
| Trust & Relationship | T&R depends on PWS | T&R reads `GetWorldFlags(entity_id)`. T&R owns `trust.*` namespace. Provisional: MVP may bypass PWS and store trust directly in the Knowledge component. Soft. |
| Faction System | Faction depends on PWS | Faction submits `faction_event`. PWS propagates faction shifts via evolution rules. Soft. |
| Investigation Journal | Journal depends on PWS | Journal queries `GetMemoryLog` and `GetEntitiesWithFlag` for display. **Hard downstream** — Journal reads PWS data. |
| Map System | Map depends on PWS | Map reads `world.*` flags for location state and faction territories. Soft. |
| Tactical Combat Engine | TCE depends on PWS | TCE submits `location_cleared` and `npc_state_changed` events on combat resolution. Soft. |

## Tuning Knobs

| Parameter | Default | Safe Range | Effect of Increase | Effect of Decrease |
|-----------|---------|------------|-------------------|-------------------|
| `memory_log_max_entries` (per entity) | 50 | 10–200 | Richer NPC memory for dialogue context; more save file size | Older events discarded sooner; less memory |
| `world_evolution_enabled` | true | true/false | Autonomous world changes active | World static — player actions still recorded, but no autonomous repopulation. Debug/testing toggle. |
| `protected_event_types` | `[npc_state_changed, discovery_event]` | any subset of CR-2 event types | More protected entries → fewer evicted by cap | Fewer protected entries → important events may be evicted |
| `max_flags_per_entity` | 32 | 8–64 | More state tracked per entity | Less memory per entity in Persistent component; earlier warnings |
| `event_validation_strict` | false | true/false | Invalid flag mutations reject the entire event | Invalid flag mutations are skipped with a warning; event still processed |
| `evolution_max_rules_per_region` | 20 | 5–100 | More autonomous world reactivity | Less CPU on region load; simpler world behavior |

## Visual/Audio Requirements

Persistent World State produces no visual or audio output of its own — it is a
pure data system. The visual and audio expression of world state comes from the
systems that consume PWS data:

- **Map System**: Cleared ruins marked, destroyed structures shown, faction
  territories updated. The map IS the visual representation of world state.
- **Investigation Journal**: Memory log entries surfaced as discoverable facts
  ("The ruin was cleared 3 days ago").
- **Dialogue UI / AI Dialogue Engine**: NPCs reference world events ("I heard
  what you did at Thornwell").
- **Living Environment**: World state affects ambient spawning (cleared ruin →
  different wildlife table; repopulated ruin → bandit camp entities).

These systems own their visual/audio requirements. PWS provides the data; it
does not dictate presentation.

## UI Requirements

Persistent World State has no player-facing UI. It provides data to UI systems
(Investigation Journal, Map System, Dialogue UI) via the CR-4 query interface.

**Debug UI** (development only, not shipped): A world-state inspector showing
per-entity `world.*` flags with values and timestamps, per-entity memory log
with filtering by event_type/region/time, active evolution rules per region,
and a recent event submission log across all systems.

## Acceptance Criteria

- **GIVEN** an entity with a `Persistent` component, **WHEN** PWS sets
  `world.ruin_cleared = true` on that entity, **THEN**
  `GetWorldFlag(entity_id, "world.ruin_cleared")` returns `(true, true)` —
  the value and a "found" indicator.

- **GIVEN** a system calls `SubmitWorldEvent(location_cleared,
  {world.ruin_cleared: true})`, **WHEN** the event is processed, **THEN** the
  target entity's `world.ruin_cleared` flag is set to `true` AND a memory log
  entry is appended with `event_type = location_cleared` and the current game
  timestamp.

- **GIVEN** a system submits an event with a non-`world.*` flag (e.g.,
  `trust.some_flag`), **WHEN** the event is processed, **THEN** the non-world
  flag is skipped (warning logged), the `world.*` flags are still applied, and
  the memory log entry is still written.

- **GIVEN** an entity with 50 memory log entries (at cap, none protected),
  **WHEN** a new event is submitted, **THEN** the oldest entry is evicted and
  the new entry is appended — total count remains 50.

- **GIVEN** an entity with 50 memory log entries all marked `protected: true`,
  **WHEN** a new event is submitted, **THEN** the new entry is appended
  (total = 51), a warning is logged, and the soft cap is temporarily exceeded.

- **GIVEN** an entity with `world.ruin_cleared = true` and
  `world.ruin_cleared_at = [7 game days ago]`, **WHEN** the region is loaded,
  **THEN** the evolution rule `IF world.ruin_cleared AND
  days_since(world.ruin_cleared_at) > 7 THEN location_repopulated` fires and a
  new world event is submitted.

- **GIVEN** a `quest-relevant` entity is destroyed, **WHEN** EF broadcasts
  `on_entity_destroyed`, **THEN** PWS logs a `clue_path_protection` event and
  sets `world.[entity_id].destroyed_with_clues = true`.

- **GIVEN** a game with 500 world flags and 2000 memory log entries across 100
  entities, **WHEN** saved and loaded, **THEN** all flags and log entries are
  restored to pre-save values — GameEntityIDs, flag names, flag values, and
  memory log `event_id` fields all match within the save's tolerance.

- **GIVEN** a save from game version N loaded in version N+1, **WHEN** a
  world-state migration exists for N→N+1, **THEN** the migration runs before
  flags are restored and the loaded flags match the migrated schema. Unknown
  flags from version N are skipped with a log entry.

- **GIVEN** `GetEntitiesWithFlag("world.ruin_cleared", region_id)`, **WHEN**
  three entities in the region have the flag set, **THEN** exactly three
  GameEntityIDs are returned. Entities in other regions are excluded.

- **PERF**: `SubmitWorldEvent` completes within 0.5ms p95 for an event with up
  to 5 flag mutations and a target entity with 50 existing memory log entries.
  `GetWorldFlags` completes within 0.1ms p95 per entity. Measured on reference
  hardware (TBD by technical-director) with Go benchmark instrumentation.

## Open Questions

| Question | Owner | Deadline | Notes |
|----------|-------|----------|-------|
| Trust & Relationship (MVP) assumes a provisional `get_flags`/`set_flags` interface from PWS (VS). Should MVP ship with trust stored directly in the Knowledge component, deferring PWS integration to VS? | Game Designer + Producer | Before MVP sprint planning | Recommended: yes — MVP bypasses PWS for trust. PWS integration happens at VS alongside Party System and Persistent World State. |
| Should world evolution rules be authored in YAML (designer-friendly, region data files) or Go code (type-safe, testable)? | Lead Programmer + Game Designer | Before VS implementation | YAML recommended for designer iteration speed. Go code for complex Alpha-tier autonomous simulation. |
| Should PWS expose a "world state diff" API for the Investigation Journal to show "what changed while you were away"? | Game Designer | Before VS implementation | Recommended: yes — `GetRecentWorldEvents(region_id, since)` already supports this. Journal calls it with `since = last_visit_timestamp`. |
| How are world evolution rules tested? A rule that fires after 7 game days is impractical for real-time QA. | QA Lead + Game Designer | Before VS implementation | Recommended: debug time-advance command for QA, and unit tests with mock game clock injection. |
