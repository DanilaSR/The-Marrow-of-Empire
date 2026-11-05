# Entity Framework

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Review**: MAJOR REVISION NEEDED → revised 2026-08-09 → NEEDS REVISION → revised 2026-08-09 (see review-log)
> **Implements Pillar**: All four — entities are the substrate for NPC knowledge (Pillar 1), world memory (Pillar 2), tactical combat (Pillar 3), and investigation (Pillar 4)

## Summary

The Entity Framework is the game-level component and query layer for The Marrow
of Empire. It does **not** define its own entity system — Kaiju Engine already
provides `engine.Entity`, `matrix.Transform`, `events.Event`, and `pod`/`EntityData`
serialization. The framework wraps these: it provides typed game-component storage
keyed by game entity ID, a uniform-grid spatial index, a defined memory model for
write resolution, and the serialization orchestrator through which all game state
is saved and loaded.

Every game object (characters, NPCs, items, props, wildlife) is backed by a Kaiju
`engine.Entity`. Game-specific state — health, knowledge, inventory, faction,
combat participation — lives in typed per-component stores managed by the framework.
Systems query entities by component type and spatial position, operate on their
data, and react to lifecycle events bridged from engine entity events.

> **Quick reference** — Layer: `Foundation` · Priority: `MVP` · Key deps: `Kaiju Engine (engine.Entity, matrix.Transform, events, pod)`

## Overview

The Entity Framework is the foundational game-data layer for The Marrow of Empire.
It defines what a game object *is* at the gameplay level — characters, NPCs,
items, props, wildlife, environmental objects — and provides the common interface
through which every other system reads and manipulates game state.

The framework **extends** Kaiju Engine's native entity system rather than
replacing it. Every game entity is backed by a Kaiju `engine.Entity`, which
owns the `matrix.Transform` (position, rotation, scale), the parent/child
hierarchy, and the native lifecycle hooks (`OnActivate`, `OnDeactivate`,
`OnDestroy`). The game layer adds typed components, spatial indexing,
write arbitration, and serialization orchestration.

The framework provides:

- **Entity identity**: Every game object has a unique `GameEntityID` (uint64)
  mapped to a Kaiju `engine.Entity` via its `EntityId` string. Systems reference
  entities by `GameEntityID`. IDs are stable across save/load and never reused
  after destruction.
- **Component model**: Entities gain gameplay capabilities through composable
  components stored in typed per-component maps (`map[GameEntityID]ComponentType`).
  A system that needs "things with health" queries the Health store regardless of
  whether the entity is a character, a destructible prop, or wildlife.
- **Lifecycle management**: Game-level lifecycle hooks bridge Kaiju's native
  `OnActivate`/`OnDeactivate`/`OnDestroy` events. The framework adds a `Pending`
  (pre-activation assembly) state and a `Suspended` (reduced-fidelity update) state
  as game-layer concerns.
- **Spatial registry**: A uniform-grid spatial index maps world-space positions to
  entity IDs for efficient range queries. Systems query "what entities are within
  range X of position Y" for combat targeting, AI perception, wildlife spawning,
  and interaction range.
- **Serialization boundary**: Game components implement Kaiju's `EntityData`
  interface and register with `pod.Register` for automatic reflection-based
  serialization. The framework orchestrates the save/load pass across all
  component stores.

Without the Entity Framework: every system would need its own object model,
cross-system references would be fragile, save/load would require custom
serialization per system, and there would be no single answer to "what
exists in the world right now?"

## Player Fantasy

The Entity Framework has no direct player fantasy. It is pure infrastructure —
players never see it, touch it, or think about it. The fantasies it serves are
the ones built on top of it:

- **Every Voice Holds a Piece** (Pillar 1): NPCs exist as entities with knowledge
  components, making every character a potential witness. Party members carry
  their own knowledge fragments.
- **The World Remembers** (Pillar 2): Entities carry persistent state with
  namespaced flags and structured memory records, making the world's memory of
  player actions possible beyond simple booleans.
- **Tactical Depth** (Pillar 3): Combat entities with stat components, spatial
  positions, movement resources, and lifecycle hooks make tactical encounters work.
- **Earned Revelation** (Pillar 4): The player entity carries a dedicated
  `PlayerKnowledge` component tracking discovered facts with source attribution,
  making the thread system possible.

The Entity Framework's success metric is that no player ever notices it exists.
If a player says "I talked to the blacksmith and she remembered me from yesterday,"
the framework did its job. If a player says "the entity component system feels
responsive," something has gone wrong.

## Detailed Design

### Architecture: Game Layer over Kaiju Engine

The Entity Framework is a **typed component layer** over Kaiju Engine's native
entity system. It does not define a parallel entity, transform, or event system.

| Concern | Owned By | Mechanism |
|---------|----------|-----------|
| Entity identity & hierarchy | Kaiju `engine.Entity` | `EntityId string`, `Parent`/`Children`, `host.EntityById(id)` |
| World position, rotation, scale | Kaiju `matrix.Transform` | `entity.Transform`, dirty-flag system, WorkGroup recalculation |
| Rendering link | Kaiju `rendering.Drawing` | `Drawing.Transform = &entity.Transform` |
| Native lifecycle | Kaiju `OnActivate`/`OnDeactivate`/`OnDestroy` | `events.Event` on `engine.Entity` |
| Game component storage | Entity Framework | Typed `map[GameEntityID]ComponentType` per component |
| Spatial queries | Entity Framework | Uniform grid index: `map[[2]int32][]GameEntityID` |
| Write arbitration | Entity Framework | Frame-end staged writes with delta accumulation |
| Serialization | Kaiju `pod` + `EntityData` | `pod.Register`, `RegisterEntityData`, reflection-based encode/decode |
| Game lifecycle (Pending/Suspended) | Entity Framework | Game-layer state machine riding on engine events |

### CR-1: Entity Identity

Every game entity has:
- A unique **GameEntityID** (`uint64`, monotonic counter, never reused after destruction)
- A mapped Kaiju **EntityId** (`string`, format `"game_{GameEntityID}"` — see ADR required before implementation)
- An **Entity Type** classification: `Character`, `NPC`, `Item`, `Prop`, `Wildlife`, `Structure`, `Trigger`, `Zone`
- A **Display Name** (localizable string for UI and dialogue references)
- A **World Position** (derived from `entity.Transform.Position()` — logical world space, NOT isometric screen-space. Isometric is a camera projection; the data layer stores world coordinates.)
- A set of **Tags** for lightweight categorization (e.g., `friendly`, `hostile`, `imperial`, `quest-relevant`)

The GameEntityID is the universal reference. Every system references entities by
GameEntityID, never by pointer. The mapping to Kaiju `EntityId` is internal to
the framework.

**The Player Entity** is explicitly defined as a `Character`-type entity carrying:
`Health`, `Stats`, `Inventory`, `PlayerKnowledge`, `Faction`, `Persistent`,
`Visual`, and `Audio` components. The player is the entity the camera follows,
the investigation state owner, and the anchor of spatial queries.

### CR-2: Component Model

Entities gain capabilities through composable game components. A component is a
pure data container — it holds state; it does not hold logic. Systems operate on
components. Components are stored in typed per-component maps for O(1) access
and no reflection overhead on reads.

**Component storage pattern** (Go):
```go
// Per-component typed stores — no interface boxing, no reflection on read
var healthStore = make(map[GameEntityID]Health)
var statsStore  = make(map[GameEntityID]Stats)
// ... one map per component type

// Read path (hot):
func GetHealth(id GameEntityID) (Health, bool) {
    h, ok := healthStore[id]
    return h, ok
}
```

| # | Component | Data Fields | Attaches To | Purpose |
|---|-----------|------------|-------------|---------|
| 1 | `Health` | current, max, temporary | Characters, NPCs, Wildlife, Structures | Damageable entities |
| 2 | `Stats` | attributes (str, dex, int, etc.), derived_values | Characters, NPCs | RPG stat block |
| 3 | `Inventory` | items[], equipment_slots[], gold | Characters, NPCs | Item carrying capacity |
| 4 | `Container` | contents[], capacity, is_locked | NPCs, Props, Structures | Lootable containers (chests, corpses, shelves) |
| 5 | `Knowledge` | domain[], known_facts[], trust_threshold | Characters, NPCs | NPC/companion knowledge graph data |
| 6 | `PlayerKnowledge` | discovered_facts[] (fact_id, source_entity, source_npc_name, thread_id, discovered_at), connection_links[] | Player Character only | Player investigation state with source attribution |
| 7 | `Faction` | faction_id, standing | Characters, NPCs | Faction membership |
| 8 | `Dialogue` | greeting, topics[], personality_profile | Characters, NPCs | Conversation data for AI dialogue |
| 9 | `Combatant` | initiative, action_points, movement_points, movement_speed, actions[], status_effects[], reaction_available | Characters, NPCs | Combat participation |
| 10 | `Interactable` | interaction_type, prompt_text, range, is_once_only | Characters, NPCs, Items, Props, Triggers | Player interaction target |
| 11 | `Readable` | content_ref, language, is_translated, discovered_at | Items, Props, Structures | Documents, inscriptions, books — the investigation's primary props |
| 12 | `Schedule` | schedule_entries[] (start_time, end_time, activity, target_position), current_activity | NPCs | NPC daily schedule for Living Environment system |
| 13 | `Persistent` | state_flags (namespaced: `trust.*`, `world.*`, `investigation.*`, `thread.*`), memory_log[], modified_at | Characters, NPCs, Wildlife, Structures, Props | World state tracking and entity memory |
| 14 | `Visual` | model_ref, animation_state, tint, visibility_flag, render_layer | Visible entities | Rendering data — maps to Kaiju `rendering.Drawing` fields |
| 15 | `Audio` | ambient_sound, footstep_type, occlusion_flag | Entities with sound | Audio attachment data for Kaiju audio system |

**`InteractionType` enumeration** (required for `Interactable`):

| Value | Owner System | Description |
|-------|-------------|-------------|
| `talk` | AI Dialogue Engine | Initiate conversation with NPC |
| `loot` | Inventory / Loot System | Open container or corpse inventory |
| `read` | Investigation Journal | Read a document, inscription, or book |
| `examine` | Investigation Journal | Look closely at an object (description + potential clue) |
| `open` | Living Environment | Open a door, chest, or gate |
| `close` | Living Environment | Close a door or gate |
| `use` | Context-dependent | Pull lever, light torch, ring bell — system-dispatched |
| `pick_up` | Inventory | Take an item from the world |
| `search` | Investigation | Search a container, body, or area for hidden items/clues |

**`status_effects[]` element schema** (Combatant component, detailed design deferred to Tactical Combat Engine GDD):
```go
type StatusEffect struct {
    EffectID    string        // Unique effect type identifier
    SourceID    GameEntityID  // Who applied this effect
    Duration    int           // Remaining rounds (0 = instantaneous)
    StackCount  int           // Current stacks
    Parameters  map[string]float64 // Effect-specific values (e.g., damage_per_tick)
}
```

**Component composition examples:**

A Goblin is a GameEntity on a Kaiju `engine.Entity` with components: Health + Stats + Combatant + Faction + Container + Visual + Audio.
A Door is a GameEntity with components: Interactable + Visual (+ Persistent if its state survives).
The blacksmith is a GameEntity with components: Health + Stats + Knowledge + Faction + Dialogue + Interactable + Schedule + Persistent + Visual + Audio.
A document on a table is a GameEntity with components: Interactable + Readable + Visual.
The player character is a GameEntity with components: Health + Stats + Inventory + PlayerKnowledge + Faction + Persistent + Visual + Audio.

**Component restriction rule**: Restrict *content*, never *schema*. The schema
supports `Knowledge` and `Dialogue` on Characters (including party members) from
day one. Party content (companions with deep backstories) arrives at Vertical
Slice — no schema migration needed.

### CR-3: Component Queries

Systems query entities by component type, tag, position, or combination:

- "All entities with Health AND Combatant within 10 units of position P" → combat targeting
- "All entities with Knowledge where faction_id = 'thornwell'" → NPC dialogue discovery
- "All entities with Interactable within 2 units of player position" → interaction prompts
- "All entities with Persistent where modified_at > last_save_time" → incremental save
- "All entities with Schedule where current_activity = 'working' within 50 units" → ambient NPC behavior

Queries return `[]GameEntityID` via a **caller-provided scratch buffer**:
`RangeQuery(dst []GameEntityID, pos, radius, filter) []GameEntityID`. When the
caller's buffer has capacity ≥ result count, the query performs zero heap
allocations (append into pre-allocated capacity). Systems that don't control
allocation use a nil buffer and accept the allocation. The querying system then
reads the specific components it needs from the typed stores.

**Spatial query semantics**:
- Distance metric: 2D Euclidean in the XZ plane (isometric gameplay happens on the ground plane; Y is elevation and ignored for range checks)
- Boundary: inclusive (an entity at exactly `range` units is included)
- Suspended entities: returned by default; systems use a `LiveOnly` filter flag to exclude them

### CR-4: Entity Lifecycle

Every game entity transitions through a defined lifecycle. The framework maps
game-level states to Kaiju engine states:

| Game State | Kaiju Engine State | Meaning | System Visibility |
|------------|-------------------|---------|-------------------|
| `Pending` | `engine.Entity` created, not yet activated | Being assembled — components being attached | Hidden from all game queries |
| `Active` | `entity.IsActive() == true` | In the world, updating at full rate, visible to all queries | Fully visible |
| `Suspended` | `entity.IsActive() == true` (engine-active, game-suspended) | In the world, updating at reduced tick rate, position+state preserved | Visible to spatial queries (LiveOnly filter excludes); hidden from full-rate update loops |
| `Destroyed` | `entity.IsDestroyed() == true` | Removed from world; engine entity destroyed | Hidden from all queries; GameEntityID never reused |

**Suspended state — reduced-fidelity tick model:**

`Suspended` entities update at a configurable coarse tick rate (default: 1 tick
per second, vs. 60 fps full rate). This means:
- Schedules advance (an NPC suspended at the forge at noon will be home by evening when the player returns — the Living Environment promise is kept)
- World state mutations from active systems (Trust changes, faction shifts) are applied to suspended entities normally
- Position and component state are preserved and potentially updated by the reduced tick

**Suspension guards** (entities that CANNOT be suspended):
- Entities in active combat (referenced by Tactical Combat Engine turn state)
- Entities in active dialogue (referenced by AI Dialogue Engine)
- Entities flagged `quest-relevant` (tag-based)
- The player entity (never suspended)

**Suspended → Destroyed**: An entity may be destroyed while suspended (world
state change during player absence — e.g., an NPC killed by faction conflict).
The framework requires that destruction of a quest-relevant or clue-holding
entity logs a structured event to the region's world-state record so the player
can discover what happened. No clue path permanently closes without an
authored discovery path.

**Transitions:**
- `Pending → Active`: Components attached, entity is fully assembled. `engine.Entity.Activate()` called. All systems receive `on_entity_created`.
- `Active → Suspended`: Player moved beyond `suspension_distance`. Entity shifted to reduced tick rate. Position and state preserved.
- `Suspended → Active`: Player returned within `suspension_distance`. Entity restored to full tick rate. Systems receive `on_entity_reactivated`.
- `Active → Destroyed`: Entity removed (killed, picked up, demolished). `host.DestroyEntity(entity)` scheduled at frame end. Systems receive `on_entity_destroyed` *before* the engine entity is destroyed (while queries still return the entity).
- `Suspended → Destroyed`: Entity destroyed while suspended. Same event sequence as Active → Destroyed.

**Invalid transitions** (rejected with error): `Destroyed → any`, `Active → Pending`.

### CR-5: Lifecycle Events

The framework bridges game-level lifecycle events to Kaiju's native event system.
Kaiju provides `events.Event` (no-arg callback) and `events.EventWithArg[T]`
(single-arg generic). Multi-field events use payload structs.

**Event catalog:**

| Event | Kaiju Mechanism | Payload | When |
|-------|----------------|---------|------|
| `on_entity_created` | `EventWithArg[EntityCreatedPayload]` | `{GameEntityID, EntityType, ComponentTypes[]}` | After `Pending → Active` transition |
| `on_entity_destroyed` | `EventWithArg[EntityDestroyedPayload]` | `{GameEntityID, EntityType}` | Before engine entity destruction; entity still queryable |
| `on_entity_reactivated` | `EventWithArg[EntityReactivatedPayload]` | `{GameEntityID}` | After `Suspended → Active` transition |
| `on_entity_suspended` | `EventWithArg[EntitySuspendedPayload]` | `{GameEntityID}` | After `Active → Suspended` transition |
| `on_component_added` | `EventWithArg[ComponentChangedPayload]` | `{GameEntityID, ComponentType}` | After component attached to entity |
| `on_component_removed` | `EventWithArg[ComponentChangedPayload]` | `{GameEntityID, ComponentType}` | After component detached from entity |
| `on_component_changed` | `EventWithArg[ComponentChangedPayload]` | `{GameEntityID, ComponentType, Fields []string}` | After frame-end write resolution (see CR-7); fires once per changed component per entity per frame, with all changed field names in `Fields`. Does NOT fire for Transform changes (engine dirty flags cover those). |

**Event delivery guarantees:**
- Lifecycle events (`created`, `destroyed`, `reactivated`, `suspended`) are
  **never dropped**. They are delivered synchronously during the frame in which
  the transition occurs.
- `on_component_changed` events are dispatched synchronously at frame-end after
  write resolution. They are batched per (entity, component) per frame. If three
  fields of `Health` change in one frame, one event fires with `Fields: ["current",
  "temporary", "max"]` — not three separate events.
- **Payload construction is gated on subscriber count.** If no systems subscribe
  to `on_component_changed`, the `Fields` slice is never allocated. The framework
  checks `event.SubscriberCount() == 0` before building the payload. When
  subscribers exist, the `Fields` slice is drawn from a per-frame pool and cleared
  at frame end.
- **Per-frame change event cap**: If more than `max_change_events_per_frame`
  (default 500) entities have component changes in a single frame, the framework
  logs a warning with the count (once per frame, not per event) but dispatches
  all events. No events are dropped — the cap is a diagnostic signal, not a
  drop threshold. A sustained high count indicates a system is writing too broadly.
- Subscriber order is deterministic: registration order within a priority band.
- Subscribers receive events synchronously during frame-end dispatch.
- A subscriber that queries a destroyed entity during `on_entity_destroyed`
  receives the entity's last-known state — the entity is hidden from queries
  only *after* all subscribers have run.

**Example — Goblin death event chain:**
1. Health.current reaches 0 → Combat system initiates destruction
2. Framework broadcasts `on_entity_destroyed(goblin_id, Wildlife)` synchronously
3. World State system records kill in goblin's `Persistent.memory_log`
4. Loot system reads goblin's `Container` component, generates drops
5. Combat UI removes goblin from turn order
6. Visual system plays death animation
7. `host.DestroyEntity(goblin_entity)` scheduled at frame end
8. Kaiju engine destroys the entity, unregisters its ID

### CR-6: Spatial Organization

Entities exist in logical world space (X = east-west, Y = elevation, Z = north-south).
The framework maintains a **uniform grid spatial index** for efficient range queries:

- **Grid**: World space divided into cells of `spatial_bucket_size` × `spatial_bucket_size` units in the XZ plane. Y (elevation) is ignored for bucketing.
- **Bucket key**: `[2]int32{int32(floor(x / bucketSize)), int32(floor(z / bucketSize))}`
- **Bucket value**: `[]GameEntityID` — entity IDs in that cell
- **Storage**: `map[[2]int32][]GameEntityID` — idiomatic Go, trivially fast
- **Update**: Write-through on position change. When an entity's Transform position changes (detected via engine dirty flag or position-set hook), its spatial bucket is updated.
- **Range query**: Compute the bucket span covered by `(position ± range)` in both X and Z axes, scan those buckets, filter by squared 2D distance. At 500 entities and 32-unit buckets, the 1ms AC is met with orders of magnitude to spare. An R-tree is unjustified at this scale.
- **Overload**: If a single bucket exceeds a warning threshold (default 200 entities), log a warning. Correctness is preserved — all entities in the bucket are scanned. This is a content-authoring signal, not a runtime failure.

**Single-level grid only**: The two-level "region and sub-region" from the
original draft is removed. At the target scale (2500 entities across 5 regions),
a one-level uniform grid is sufficient and simpler. If streaming-based loading
requires a region hierarchy later, the Map System and Living Environment systems
own that concern, not the Entity Framework.

### CR-7: Memory Model (Write Semantics)

This section defines when writes become visible, how concurrent writes to the
same field are resolved, and when events fire. Every consumer system depends on
these rules.

**Write staging**: Component writes are staged during the frame and applied at
frame end. The staging mechanism uses a **per-entity staged-field bitmask** —
one bit per (entity, component, field) combination — stored persistently (not
recreated per frame) and cleared at frame end via Go 1.21+ `clear()`. A read
checks the bitmask first: if the bit is set for the field being read, the staged
value is returned; otherwise the component store value is returned. This
interposes a single bitmask lookup (cache-friendly, ~50KB for 2500 entities ×
10 fields) on every read, not a second map lookup.

**Read coherence**: A system that writes a field and then reads it within the
same frame sees its staged value (read-your-writes). A system that did NOT write
the field sees the pre-write value until frame-end resolution — regardless of
whether another system wrote it earlier in the same frame. This ensures
deterministic intra-frame behavior: systems are isolated from each other's
in-progress writes.

**Two write types, two resolution rules:**

1. **Absolute assignment** (`SetHealth(current, 100)`, `SetFaction(thornwell)`):
   The writer asserts a definitive value. If two systems write the same field
   with different absolute values in the same frame, the higher-priority
   (lower number) system wins. The losing write is discarded with an error-level
   log and debug-overlay notification. This is a last-resort arbitration —
   the real fix is declared write-ownership per field.

2. **Delta mutation** (`ModifyHealth(current, -10)`, `ModifyHealth(current, +5)`):
   The writer requests a relative change. All deltas to the same field
   accumulate within the frame. Same-frame damage (-10) and heal (+5) net to -5.
   **Deltas accumulate in system registration order** for float determinism —
   this guarantees identical results across runs given identical input ordering.
   No delta is discarded during accumulation; the net delta is applied after all
   absolute assignments for that field have been resolved.

   **Bounded-field clamping**: After the accumulated delta is applied, the
   resulting value is clamped to the field's defined bounds (e.g., `Health.current`
   clamped to `[0, Health.max]`). The clamp is applied by the framework, not
   the writing system, and is performed after delta application and before
   event dispatch. If clamping reduces the magnitude of the accumulated delta,
   the event payload carries the clamped value; the pre-clamp net delta is
   logged at debug level. The "no delta discarded" guarantee covers the
   accumulation phase only — clamping is semantically a constraint, not a discard.

**Priority model**: Systems register a priority value. **1 = highest priority**
(definitive). For absolute assignment conflicts: the lowest-numbered writer wins;
all other writers receive an error log and debug notification. This is consistent
across the GDD, the acceptance criteria, and the tuning knob table. (The tuning
knob's "Effect of Increase" column has been corrected — see Tuning Knobs.)

**Write ordering within equal priority**: Registration order breaks ties
deterministically. Systems registered earlier win. Go map iteration order is
NOT used — the framework maintains an ordered registration list.

**Field write-ownership** (declared, not arbitrated):

| Component.Field | Owner System | Notes |
|-----------------|-------------|-------|
| `Health.current` | Tactical Combat Engine | Damage, healing, death |
| `Health.max` | Character Stats & Progression | Changed on level-up or equipment |
| `Health.temporary` | Tactical Combat Engine | Temporary HP from abilities; absorbed before `current` |
| `Combatant.initiative` | Tactical Combat Engine | Rolled at combat start |
| `Combatant.action_points` | Tactical Combat Engine | Reset at round start; spent by player/AI |
| `Combatant.movement_points` | Tactical Combat Engine | **Composite**: round-reset clears and sets to max; AI pathfinder and player input spend via delta; reset takes priority over spends |
| `Combatant.status_effects` | Tactical Combat Engine | Single owner |
| `Combatant.reaction_available` | Tactical Combat Engine | Set on turn start, consumed on reaction use |
| `Stats.*` (all fields) | Character Stats & Progression | Single owner |
| `Inventory.*` (all fields) | Inventory/Loot System | Not yet authored; defer to that GDD |
| `Container.*` (all fields) | Inventory/Loot System | Not yet authored |
| `Knowledge.*` (all fields) | NPC Knowledge Graph | Not yet authored |
| `PlayerKnowledge.*` (all fields) | Thread Investigation | Not yet authored |
| `Faction.faction_id` | Faction System | Assigned at entity creation, immutable after |
| `Faction.standing` | Faction System | Single owner |
| `Dialogue.*` (all fields) | AI Dialogue Engine | Not yet authored |
| `Interactable.*` (all fields) | Respective owner per `InteractionType` | Dispatched by type (talk→Dialogue, loot→Inventory, etc.) |
| `Readable.*` (all fields) | Investigation Journal | Not yet authored |
| `Schedule.*` (all fields) | Living Environment | Not yet authored |
| `Persistent.state_flags[trust.*]` | Trust & Relationship | Namespaced — no conflict |
| `Persistent.state_flags[world.*]` | Persistent World State | Namespaced — no conflict |
| `Persistent.state_flags[investigation.*]` | Thread Investigation | Namespaced — no conflict |
| `Persistent.memory_log` | Entity Framework (append-only) | Systems submit entries; framework appends |
| `Visual.model_ref` | Asset & Scene Pipeline | Set at entity creation |
| `Visual.animation_state` | **Composite — animation stack** | Multiple systems push/pop states (combat: hit-react, Living Environment: working, Dialogue: talking). Highest-priority animation wins. Analogous to tint stack (see below). |
| `Visual.tint` | Tactical Combat Engine (damage flash) | See composite rule below |
| `Visual.visibility_flag` | **Composite**: stealth (stealth system) > destruction (EF) > default (visible). Highest priority = lowest number (1=stealth-hide, 2=destroyed-hide, 3=visible) |
| `Visual.render_layer` | Asset & Scene Pipeline | Set at entity creation |
| `Audio.*` (all fields) | Audio System | Not yet authored |

**Composite fields**: `Visual.tint` has multiple legitimate writers (damage
flash, status-effect tint, faction reaction glow). The framework supports a
*tint stack* — writers push/pop tint modifiers with a priority, and the final
tint is the stack's resolved value. Details deferred to the Visual/Combat GDDs.

**Concurrency rule**: All component writes occur on the main thread (Kaiju's
`host.Updater`). Systems run sequentially within a frame — there is no
goroutine-level race on component stores. The NPC LLM inference runs
asynchronously (separate goroutine, 2.7GB model) and marshals results to the
main thread via a thread-safe queue. LLM results are applied on the main
thread during the next frame's write-staging phase.

**`on_component_changed` delivery**: Events fire at frame end, after all
staged writes have been resolved and applied. A subscriber to
`on_component_changed` always sees the final resolved value — never an
intermediate or discarded value.

**Re-entrancy guard**: If an `on_component_changed` handler modifies the same
(component, field) pair that triggered it, the second modification is applied
but does NOT trigger another `on_component_changed` event for that pair in
the same frame. This prevents infinite recursion. The guard scope is
per-(component, field) per frame.

### CR-8: Serialization Boundary

Game components serialize through Kaiju Engine's `pod` encoding system:

1. Each game component struct implements the `engine.EntityData` interface:
   ```go
   type EntityData interface {
       Init(entity *engine.Entity, host *engine.Host)
   }
   ```
2. Components register with `engine.RegisterEntityData(componentInstance)` —
   this calls `pod.Register` internally, building the reflection-based
   encode/decode layout.
3. The framework's save orchestrator collects the full save set from:
   - **All typed component stores** — Active and Suspended entities' component data
   - **Kaiju Transform** — `entity.Transform.Position()`, rotation, and scale for
     every saved entity (lives on the engine entity, not in a game component store;
     the framework reads it via `entity.Transform`)
   - **Entity hierarchy** — parent `GameEntityID` for each entity that has a parent
     (Kaiju's `entity.Parent()` handles transform inheritance; the framework records
     the game-level parent reference for rebuild on load)
   - **Tags** — the set of string tags per entity (`pc_controlled`, `quest-relevant`,
     etc.) from CR-1
   - **GameEntityID counter** — the current monotonic counter value, so post-load
     entity creation does not collide with restored IDs
   The full save set is serialized via `pod` as a single whole-world blob (amortized
   header cost; per-entity blobs would be 2-4× more expensive due to repeated
   type/field-name tables).
4. On load: existing engine state is cleared, engine entities are recreated with
   their saved Transforms, game components are deserialized and inserted into typed
   stores, entity IDs are re-registered with the Kaiju host, parent/child hierarchy
   and `Drawing.Transform = &entity.Transform` links are rebuilt, Tags are restored,
   and the ID counter is restored.
5. **Lifecycle events are suppressed during load/teardown.** The clear-existing-state
   phase destroys entities without firing `on_entity_destroyed`, and the
   recreate-entities phase creates them without firing `on_entity_created`.
   Consumer systems are notified via a single `on_world_loaded` event after the
   full state is restored. This prevents subscriber storms (2500 entities × N
   subscribers per entity) during load.

**ID re-registration on load**: Kaiju strips entity IDs on destroy
(`unregisterEntityId` zeroes `entity.id`). The save/load system must explicitly
re-register restored GameEntityIDs with the Kaiju host via the framework's
`RegisterEntity` call after deserialization.

**Serialization budget**: Full save of 2500 entities (Active + Suspended,
all components) must complete within 100ms. Per-component encode cost is
measurable with Go benchmarks using `pod`.

**Float equality on round-trip**: Floats are serialized as `float32` via
`pod`. Round-trip equality uses a tolerance of `1e-5` for position/rotation
fields — exact bit-identical round-trip is not guaranteed for floating-point
values.

**Transient fields**: Component fields tagged `pod:"-"` are excluded from
serialization. Caches, derived values (recomputed on load), and runtime-only
state are marked transient.

**Pending entities**: Entities in `Pending` state are NOT serialized. Only
`Active` and `Suspended` entities are included in the save pass.

## Formulas

The Entity Framework has no mathematical formulas in the traditional sense. It
is a structural system. Its "calculations" are lifecycle transitions, spatial
queries, and write arbitration, none of which involve arithmetic.

**Structural constants** (cross-system facts):

| Constant | Value | Source | Notes |
|----------|-------|--------|-------|
| Priority direction | 1 = highest | CR-7 | Lower number = higher priority |
| Spatial distance metric | 2D Euclidean (XZ plane) | CR-6 | Y (elevation) ignored for range |
| Spatial bucket key | `[2]int32{x_bucket, z_bucket}` | CR-6 | Uniform grid |
| Range boundary | Inclusive at exact distance | CR-3 | Entity at range N is included |
| Delta accumulation | All deltas netted per field per frame | CR-7 | No delta is discarded |
| Float round-trip tolerance | `1e-5` | CR-8 | For save/load AC |

**Required invariants** (framework validates at config load):

| Invariant | Validation | Rationale |
|-----------|-----------|-----------|
| `suspension_distance >= spatial_bucket_size` | Warn if violated | Defensive check — CR-6's span-based range queries are correct at any bucket size, but bucket sizes larger than the reactivation scan radius cause the scan to cover unnecessary cells |
| `suspension_tick_rate_ms >= 100` | Warn if smaller | Below 100ms, suspension is nearly full-rate and defeats its purpose |

## Edge Cases

- **Entity destroyed while a system holds its GameEntityID**: The component
  store query returns `(ZeroValue, false)` for that ID. Systems must handle
  "entity no longer exists" gracefully. This is expected behavior — the
  framework guarantees the query returns cleanly without panic or hang.

- **Same-frame absolute assignment conflict**: The higher-priority (lower
  number) system's value is applied. All other writers receive an error-level
  log entry and a debug-overlay notification. A test must verify that the
  losing system can observe its write was discarded (via error return or
  log capture).

- **Same-frame delta accumulation**: All deltas to the same field accumulate.
  Same-frame damage (-10) and heal (+5) net to -5. A unit test must verify
  this exact scenario (the "potion bug" test).

- **Component removed while a system is actively reading it**: The typed-store
  read completes with the data that existed at the moment the read began.
  Subsequent reads return `(ZeroValue, false)`. Systems should check
  component existence before each read in hot paths.

- **Entity created with duplicate GameEntityID (save/load restoring an
  entity that already exists)**: The framework detects the collision via
  the monotonic counter and the ID registry check. Creation is rejected
  with an error. The Save/Load system is responsible for clearing existing
  state before restoring.

- **Spatial query spans multiple grid buckets**: The framework transparently
  scans all buckets within the query radius and filters by exact distance.
  An entity on a bucket boundary appears in exactly one bucket (its position
  determines the bucket key); it cannot be double-counted. A unit test
  with a boundary-positioned entity must verify this.

- **Zero-component entity**: Remains valid. A componentless entity has
  identity and position (via its Kaiju `engine.Entity`) but is invisible to
  all component queries. It remains in the spatial index. Valid for
  placeholder entities.

- **Spatial bucket exceeds warning threshold**: Correctness is preserved
  — all entities in the overloaded bucket are scanned linearly. The
  framework logs a warning. Performance degrades but does not break.
  This is a content-authoring signal, not a runtime failure.

- **Re-entrant `on_component_changed` handler**: The framework tracks
  per-(component, field) events within the current frame's dispatch.
  A second modification to the same (component, field) by a handler
  is applied but does not trigger another event. The guard scope is
  per-frame — a modification in frame N+1 triggers a normal event.

- **Entity suspended mid-combat**: Rejected by the suspension guard.
  Entities referenced by active combat state cannot be suspended.

- **Entity suspended mid-dialogue**: Rejected by the suspension guard.
  Entities referenced by active dialogue state cannot be suspended.

- **Quest-critical entity destroyed while suspended**: The framework
  requires that destruction of a `quest-relevant` entity logs a structured
  event to the Persistent World State system. The player can discover
  what happened. No clue path permanently closes without an authored
  discovery path.

- **Player oscillating at suspension boundary**: A hysteresis margin
  (default 10% of `suspension_distance`) prevents Active↔Suspended
  flapping. The entity must move `suspension_distance + hysteresis`
  away to suspend and `suspension_distance - hysteresis` closer to
  reactivate.

- **Change event volume exceeds diagnostic cap**: If more than
  `max_change_events_per_frame` entities have component changes, the framework
  logs a warning with the count (once per frame, rate-limited). All events
  are dispatched — no events are dropped. The cap is a diagnostic for
  over-broad writes, not a drop threshold. Lifecycle events are unaffected.

## Dependencies

The Entity Framework depends on Kaiju Engine's native systems:

| Dependency | Nature | Why |
|-----------|--------|-----|
| `engine.Entity` | **Hard** | Every game entity is backed by a Kaiju entity. Transform, parenting, and native lifecycle live here. |
| `matrix.Transform` | **Hard** | Position, rotation, scale for all game entities. The render path requires `&entity.Transform` on every Drawing. |
| `events.Event` / `events.EventWithArg[T]` | **Hard** | Lifecycle event dispatch. |
| `pod` / `engine.EntityData` / `engine.RegisterEntityData` | **Hard** | Reflection-based component serialization. |
| `engine.Host` | **Hard** | Entity creation, destruction, ID registry, update loop. |

All gameplay systems depend on the Entity Framework:

| System | Direction | Nature of Dependency |
|--------|-----------|---------------------|
| Camera System | Depends on EF | Subscribes to EF lifecycle events (`on_entity_created`, `on_entity_destroyed`) to maintain a cached set of tracked entity IDs (party, combatants). Reads `Transform.Position()` directly each frame for the tracked set (≤ ~10 entities — O(1) per entity, not a range query). Performance contract: no per-frame range queries; zero-alloc read path. Contract must be kept consistent with `design/gdd/camera-system.md`. |
| Character Stats & Progression | Depends on EF | Reads/Writes `Stats` component on entities |
| NPC Knowledge Graph | Depends on EF | Reads/Writes `Knowledge` component on NPC entities |
| Living Environment | Depends on EF | Creates wildlife entities; reads spatial index for spawning; reads `Schedule` component |
| Persistent World State | Depends on EF | Reads/Writes `Persistent` component flags and memory_log |
| AI Dialogue Engine | Depends on EF | Reads `Knowledge`, `Dialogue`, `Faction` components |
| Question System | Depends on EF | Reads `Knowledge` component via AI Dialogue Engine |
| Trust & Relationship | Depends on EF | Reads/Writes `Persistent` relationship flags (namespaced `trust.*`) |
| Thread Investigation | Depends on EF | Reads/Writes `PlayerKnowledge` and `Persistent` investigation state |
| Tactical Combat Engine | Depends on EF | Queries entities by component + spatial; reads/writes `Health`, `Combatant` |
| Party System | Depends on EF | Reads/Writes `Stats`, `Faction` on party entities |
| Faction System | Depends on EF | Reads/Writes `Faction` component |
| Save/Load System | Depends on EF | Serializes/deserializes all entity and component state via `pod` |
| Investigation Journal | Depends on EF | Reads `PlayerKnowledge` and `Persistent` investigation flags |
| Dialogue UI | Depends on EF | Reads `Dialogue`, `Knowledge` for display |
| Combat UI | Depends on EF | Reads `Health`, `Combatant`, `Stats` for display |
| Map System | Depends on EF | Spatial queries for entity positions |
| Audio System | Depends on EF | Reads `Audio` component for sound attachment |

All relationships are **hard dependencies** — none of these systems can function
without entities to operate on.

## Tuning Knobs

| Parameter | Default | Safe Range | Effect of Increase | Effect of Decrease |
|-----------|---------|------------|-------------------|-------------------|
| `spatial_bucket_size` (world units) | 32 | 16–128 | Larger buckets → slower queries (more entities per bucket), less memory overhead | Smaller buckets → faster queries, more bucket management overhead |
| `suspension_distance` (world units) | 100 | 50–300 | Entities stay active farther from player → more CPU, more immersive | Entities suspend closer → less CPU, more load-in visible to player |
| `suspension_tick_rate_ms` | 1000 | 100–5000 | Coarser ticks → less CPU for suspended entities, schedules advance less granularly | Finer ticks → more CPU, schedules advance more smoothly (approaching full rate below 200ms) |
| `suspension_hysteresis_margin` (fraction of suspension_distance) | 0.1 | 0.05–0.25 | Larger margin → less flapping, but boundary feels wider | Smaller margin → tighter boundary, more potential for flapping |
| `max_change_events_per_frame` | 500 | 100–2000 | Higher threshold → fewer warnings; more event work can go unnoticed | Lower threshold → more diagnostic signals; earlier detection of over-broad writes |
| `component_field_modification_priority` (per system) | 10 | 1–100 | **LOWER number = HIGHER priority.** System's absolute-assignment writes take precedence over higher-numbered systems. Delta mutations are unaffected by priority (all deltas accumulate). | Higher number = lower priority. System's absolute assignments are overridden by lower-numbered systems. |
| `memory_log_max_entries` (per entity) | 50 | 10–200 | More memory per entity; richer NPC memory for dialogue prompts | Less memory; older events discarded sooner |
| `spatial_bucket_warn_threshold` (entities per bucket) | 200 | 50–500 | Warning triggers later → denser content before content-authoring signal | Warning triggers sooner → more frequent content-authoring signals |

## Visual/Audio Requirements

**Visual**: The `Visual` component carries data consumed by Kaiju's rendering
pipeline. Its fields map to `rendering.Drawing`:

| Visual Field | Maps To | Notes |
|-------------|---------|-------|
| `model_ref` | `Drawing.Mesh` + `Drawing.Material` | Resolved via host caches |
| `animation_state` | `Drawing.ShaderData` | Animation pose name |
| `tint` | `Drawing.ShaderData` (color uniform) | RGB tint, damage flash, status effect |
| `visibility_flag` | Drawing added/removed from `host.Drawings` | Toggle rendering |
| `render_layer` | `Drawing.RenderLayer` | Layer mask for render views |

The critical rendering link — `Drawing.Transform = &entity.Transform` — is
maintained by the framework when creating the Kaiju entity. Game systems do
not manage this link directly.

**Audio**: The `Audio` component carries data consumed by Kaiju's audio system:
ambient sound loop reference, footstep material type, and audio occlusion flag.
The framework does not play audio — it provides the data the audio system reads.

**Same-frame visibility**: A `Visual.tint` change (e.g., damage flash) must be
visible in the next rendered frame. An `Audio.ambient_sound` change must be
audible within one audio buffer cycle.

## UI Requirements

The Entity Framework has no player-facing UI. It provides entity data to UI systems:

- Entity `Display Name` is read by Dialogue UI for conversation headers and by
  Combat UI for turn order displays.
- Entity `Tags` are read by UI systems for filtering and categorization.
- Component data (`Health`, `Stats`, `PlayerKnowledge`, etc.) is read by Combat
  UI, Character UI, and Investigation Journal for display.

**Editor/debug UI** (development only, not shipped): An entity inspector showing
active entities, their components, and component field values. Kaiju Engine
provides `DebugEntityDataRegistry` (populated by `RegisterEntityData` in debug
builds) — the inspector can leverage this for free component-type discovery.
This belongs in the editor/tools backlog, not the game's UI spec.

## Open Questions

| Question | Owner | Deadline | Resolution |
|----------|-------|----------|-----------|
| Entity ID format ADR: uint64 counter + Kaiju `EntityId` string mapping? Recommended: `"game_{GameEntityID}"`. Reject region-prefixed strings (couples identity to geography). | Technical Director | Before implementation (ADR) | — |
| Entity integration model ADR: confirm game-component layer wraps Kaiju entities (this GDD's premise). Resolve: component storage layout (typed per-component maps), lifecycle bridge details, Transform ownership. | Lead Programmer | Before implementation (ADR) | — |
| Serialization ADR: confirm `pod` + `EntityData`/`RegisterEntityData` as the mechanism. Resolve: save/load time budget, float precision, transient field conventions, ID re-registration on load. | Lead Programmer | Before implementation (ADR) | — |
| Write semantics ADR: confirm staged writes at frame-end, delta accumulation, absolute assignment priority resolution, event delivery timing. This GDD specifies the model; the ADR formalizes it. | Lead Programmer + Systems Designer | Before implementation (ADR) | — |
| Suspension policy ADR: confirm reduced-fidelity tick model, guard conditions (combat, dialogue, quest-relevant), hysteresis margin, clue-loss protection, region-level event logging for off-screen destruction. | Game Designer + Lead Programmer | Before implementation (ADR) | — |
| Component schema ADR: confirm the 15-component set with field types, write-ownership table, namespaced Persistent flags, PlayerKnowledge vs Knowledge separation, InteractionType enumeration, status_effects schema. | Game Designer + Systems Designer | Before implementation (ADR) | — |
| Entity templates/prefabs: defined in Go code (EntityData composition), JSON data files, or Kaiju scene format? Affects NPC Knowledge Graph and content authoring pipeline. Kaiju has template instantiation and `EntityDescription`/`EntityDescriptionJson` — evaluate these first. | Lead Programmer | Before first content authoring | — |
| Maximum component count per entity? With map-based storage there is no practical ceiling — 15+ components is fine. The question is reframed as per-entity-type authoring budgets (catch "door with Health" at authoring time). | Lead Programmer | Before content authoring | — |
| Does the framework support entity parenting/attachment (e.g., a torch attached to a character's hand)? Kaiju's `entity.SetParent()` handles transform inheritance natively. Game-level attachment semantics (inventory slot → child entity) are deferred to the Inventory/Equipment GDD. | Gameplay Programmer | Before vertical slice | Resolved at engine level; game semantics TBD |

## Acceptance Criteria

**Test methodology**: All performance targets are measured on the developer
workstation (the CI runner may differ; CI budgets are 2× the values below).
Measurements use Go `testing.B` benchmarks. Reported values are p95 over
100 iterations with a warm-up of 10 iterations. GC is triggered before each
benchmark (`runtime.GC()`). Build tag: `debug` (development build; release
builds are expected to be faster). All ACs are unit-testable in
`tests/unit/entity-framework/` unless marked Integration.

### Functional ACs

- **AC-1 — Entity creation and event delivery**: GIVEN the framework receives
  a `CreateEntity` call with type `NPC` and components `[Health, Knowledge]`,
  WHEN the entity transitions from `Pending` to `Active`, THEN all test
  subscribers registered before the create call receive `on_entity_created`
  with correct `GameEntityID`, `EntityType=NPC`, and `ComponentTypes` list
  (order-independent comparison). The entity is queryable by its
  `GameEntityID` immediately after the event dispatch completes.

- **AC-2 — Entity destruction and stale-ID safety**: GIVEN an Active entity
  with a `Health` component, WHEN the framework processes a `DestroyEntity`
  call for its `GameEntityID`, THEN: (a) all subscribers receive
  `on_entity_destroyed` before the entity is hidden from queries, (b) a
  subscriber querying the entity during `on_entity_destroyed` receives the
  entity's last-known component data, (c) subsequent queries for that
  `GameEntityID` after all subscribers have run return `(ZeroValue, false)`,
  and (d) destroying an already-destroyed ID returns an error without
  panicking or double-firing events.

- **AC-3a — Range query correctness**: GIVEN a fixture of exactly 200 entities
  distributed as: 49 with Health+Combatant strictly within 10 units of position
  P (distance 1.0–9.9u), 1 with Health+Combatant at exactly 10.0u on the
  boundary, 50 with Health+Combatant at 10.1–20u from P, 50 with Health only
  within 10u, and 50 with Combatant only within 10u, WHEN the framework executes
  a spatial query "all entities with Health AND Combatant within 10 units of P,"
  THEN exactly 50 entities are returned (49 strictly-in + 1 boundary-inclusive;
  inclusive boundary means the boundary entity counts as "within 10"). Each
  result is verified: distance from P ≤ 10.0u and both Health and Combatant
  components present.

- **AC-3b — LiveOnly filter**: GIVEN the AC-3a fixture plus 3 additional
  Suspended entities with Health+Combatant within 10u of P (total 203 entities),
  WHEN the same query runs with the `LiveOnly` filter, THEN exactly 50 entities
  are returned (the 3 Suspended entities are excluded).

- **AC-4 — Write priority and delta accumulation (the "potion bug" test)**:
  GIVEN an entity with `Health{current: 50, max: 100}`, WHEN System A
  (priority 5) calls `ModifyHealth(current, -10)` and System B (priority 15)
  calls `ModifyHealth(current, +5)` in the same frame, THEN at frame-end
  resolution `Health.current` equals 45 (50 - 10 + 5). No delta is discarded.
  The `on_component_changed` event fires once with `{ComponentType: Health,
  FieldName: current}` and the final value visible to the subscriber is 45.

- **AC-5 — Absolute assignment conflict**: GIVEN an entity with
  `Faction{faction_id: "neutral", standing: 0}`, WHEN System A (priority 5)
  calls `SetFaction(standing, 100)` and System B (priority 15) calls
  `SetFaction(standing, -50)` in the same frame, THEN at frame-end resolution
  `Faction.standing` equals 100 (System A has higher priority — lower number).
  System B receives an error-level log entry indicating its write was discarded.
  A test subscriber observing the error channel or log capture confirms the
  discard notification.

- **AC-6 — Suspension reactivation with state preservation**: GIVEN an entity
  with `Health{current: 30, max: 100}` and `Schedule{current_activity: "working"}`
  suspended at position P, WHEN the framework's suspension check runs with
  a simulated player position moved to within `suspension_distance` of P
  (no actual player system required — the test drives the check directly),
  THEN: (a) the entity transitions from Suspended to Active, (b) all test
  subscribers receive `on_entity_reactivated`, (c) `Health.current` is still
  30, (d) `Health.max` is still 100, (e) the `Schedule` component's values
  have advanced by the reduced tick (at least one tick fired during
  suspension). **Note**: This AC is framework-isolated — the test harness
  provides a synthetic position update, not a real player entity.

- **AC-7 — Save/load round-trip** [Integration]: GIVEN 100 Active entities
  with diverse component combinations (at least one entity with every
  component type), parent/child relationships (at least one parent with 2
  children), and entity tags including `pc_controlled` and `quest-relevant`,
  plus 20 Suspended entities, plus 3 Pending entities in mid-assembly at
  save time, WHEN the Save/Load System triggers a full serialization pass,
  THEN: (a) all 120 Active+Suspended entities are present in the serialized
  output, (b) after deserialization into a clean world, all 120 GameEntityIDs
  are queryable, (c) for each entity, every component field matches its
  pre-save value — int/string/bool/enum fields exactly equal; float32 fields
  within `max(1e-5, 1e-5 × |pre-save value|)` (magnitude-relative tolerance;
  at world position 320u, tolerance = 3.2e-3) — (d) Transform position
  matches pre-save position within the same float32 tolerance, (e) rotation
  and scale match, (f) parent/child hierarchy is reconstructed correctly
  (child entity's parent GameEntityID matches), (g) Tags are restored
  (entity tagged `pc_controlled` pre-save is tagged post-load), (h) Suspended
  entities are still Suspended, (i) Active entities are Active, (j) the 3
  Pending entities are NOT present in the save output, (k) the GameEntityID
  counter is restored such that the next entity created after load receives
  an ID > all restored IDs, (l) `Drawing.Transform` links on restored entities
  point to the correct `&entity.Transform` (verified by moving an entity and
  checking its Drawing transform follows).
  **Dependency**: Requires the Save/Load System (Foundation #4). Test location:
  `tests/integration/entity-framework/`.

- **AC-8 — Component removal**: GIVEN an entity with `Health` and `Combatant`
  components, WHEN a system removes the `Health` component from the entity,
  THEN: (a) a test subscriber receives `on_component_removed` with correct
  `GameEntityID` and `ComponentType=Health`, (b) subsequent reads of
  `Health` on that entity return `(ZeroValue, false)`, (c) reads of
  `Combatant` on the same entity still succeed, (d) re-adding `Health`
  to the entity fires `on_component_added` and the new component is queryable.

### State Machine & Edge Case ACs

- **AC-9 — State machine validity**: GIVEN the framework's lifecycle state
  machine, WHEN any of the following transitions are attempted, THEN an error
  is returned and no state change occurs: `Destroyed → any state`,
  `Active → Pending`, `Pending → Destroyed` (bypassing Active).

- **AC-10 — ID never reused**: GIVEN an entity with `GameEntityID = N`
  that is destroyed, WHEN 10,000 subsequent entities are created and
  destroyed, THEN `GameEntityID = N` is never assigned to any new entity.

- **AC-11 — Zero-component entity**: GIVEN an entity created with zero
  components, WHEN queried by `GameEntityID`, THEN the entity is found
  (identity exists), it appears in the spatial index at its position, and
  all component-type queries return `(ZeroValue, false)`.

- **AC-12 — Cross-bucket boundary deduplication**: GIVEN an entity positioned
  exactly at the corner where four spatial buckets meet, WHEN a range query
  covers that corner, THEN the entity appears exactly once in the result set.

- **AC-13 — Suspension guard prevents combat suspension**: GIVEN an entity
  registered as an active combat participant, WHEN the suspension check
  runs with the player beyond `suspension_distance`, THEN the entity
  remains Active (suspension is rejected).

- **AC-14 — Suspension guard prevents quest-critical suspension**: GIVEN an
  entity tagged `quest-relevant`, WHEN the suspension check runs with the
  player beyond `suspension_distance`, THEN the entity remains Active.

- **AC-15a — Hysteresis prevents Active→Suspended flapping**: GIVEN an Active
  entity at exactly `suspension_distance` (100u) from the player, with default
  hysteresis margin (0.1 × 100u = 10u), WHEN the player position alternates
  between `suspension_distance + 5%` (105u — does NOT reach the 110u suspend
  threshold) and `suspension_distance - 5%` (95u — well below 100u) for 10
  consecutive checks at the sequence [105, 95, 105, 95, 105, 95, 105, 95, 105, 95],
  THEN the entity remains Active for all 10 checks.

- **AC-15b — Hysteresis prevents Suspended→Active flapping**: GIVEN a Suspended
  entity at exactly `suspension_distance` (100u) from the player, WHEN the same
  10-position sequence [105, 95, 105, 95, ...] is applied, THEN the entity
  remains Suspended for all 10 checks (95u never reaches the 90u reactivate
  threshold, 105u never triggers a suspend — already suspended).

- **AC-16 — Duplicate ID creation rejected**: GIVEN an Active entity with
  `GameEntityID = 42`, WHEN the framework receives a `CreateEntity` call that
  would produce `GameEntityID = 42` (e.g., a save/load restore without clearing
  existing state), THEN the creation is rejected with an error, no entity is
  created, and the existing entity with ID 42 remains unchanged.

- **AC-17 — Re-entrancy guard prevents recursion**: GIVEN a subscriber to
  `on_component_changed` for `Health.current` that, in its handler, writes
  `ModifyHealth(current, -5)`, WHEN the handler fires and performs the write,
  THEN the second modification is applied to the staged value but does NOT
  trigger another `on_component_changed` event for `(entity, Health)` in the
  same frame. The guard scope is per-(entity, component) per frame. A
  modification to a *different* component from the same handler DOES trigger
  its own `on_component_changed` normally.

- **AC-18 — Dialogue suspension guard**: GIVEN an entity registered as an active
  dialogue participant (via `SetSuspensionGuard(id, DialogueParticipation)`),
  WHEN the suspension check runs with the player at 2 × `suspension_distance`,
  THEN the entity remains Active and no `on_entity_suspended` event fires.

- **AC-19 — Quest-critical destruction logging**: GIVEN a Suspended entity
  tagged `quest-relevant`, WHEN the framework processes a `DestroyEntity` call
  for that entity, THEN: (a) a structured destruction event is logged containing
  `{GameEntityID, EntityType, Tags, DestroyReason, Timestamp}`, (b) the entity
  is destroyed normally, and (c) the destruction event is queryable by the
  Thread Investigation system via the Persistent World State system.

- **AC-20 — Change event volume diagnostic**: GIVEN a frame where 600 entities
  have component changes (exceeding `max_change_events_per_frame` = 500), WHEN
  frame-end dispatch runs, THEN: (a) all 600 change events are dispatched (no
  events dropped), (b) exactly one warning is logged containing the count (600),
  rate-limited to once per frame, and (c) a subsequent frame where the count
  drops below 500 produces no warning.

- **AC-21 — Write-semantics bundle** (three sub-scenarios):
  - **(a) Read-your-writes**: GIVEN System A writes `SetHealth(current, 80)`
    on entity E, WHEN System A reads `GetHealth(E)` in the same frame, THEN
    the returned `current` is 80 (staged value). WHEN System B (which did
    NOT write Health) reads `GetHealth(E)` in the same frame, THEN the
    returned `current` is the pre-write value (50).
  - **(b) Equal-priority tie-break by registration order**: GIVEN System A
    (priority 10, registered first) and System B (priority 10, registered
    second) both call `SetFaction(standing, X)` on the same entity in the
    same frame, THEN at frame-end the value from System A (registered first)
    is applied. System B receives a warn-level log (not error — this is a
    designed degradation signaling the ownership table needs a row).
  - **(c) Multi-field event batching**: GIVEN a frame where `Health.current`,
    `Health.max`, and `Health.temporary` all change on entity E, WHEN frame-end
    dispatch fires `on_component_changed`, THEN exactly one event fires for
    `(E, Health)` with `Fields` containing all three field names (order-
    independent comparison). No separate field-level events fire.

### Performance ACs

- **AC-P1 — Entity creation**: Creating an entity with 1 component (Health)
  completes within 0.5ms at p95. Creating an entity with 9 components
  (the blacksmith profile) completes within 2ms at p95.

- **AC-P2 — Range query performance**: Spatial query for entities with 2
  component filters (Health + Combatant) across a 2500-entity world (entities
  evenly distributed across a 320×320 unit area) with a query radius of 100u
  completes within 1ms at p95. The degenerate case — all 2500 entities in a
  single 32×32 bucket (market square scenario, radius 100u) — also completes
  within 1ms at p95.

- **AC-P3 — Entity destruction**: Destroying an entity with 10 subscribed
  systems (each with a no-op handler) completes within 2ms at p95. Measurement
  scope: the synchronous `on_entity_destroyed` dispatch (all subscriber handlers
  fire within the measured window). Kaiju's deferred teardown (ID unregistration,
  Drawing removal) occurs in the next frame and is excluded from this AC —
  the deferred cost is captured separately in the engine profiler baseline.

- **AC-P4 — Save round-trip**: Full serialization of 2500 entities (Active +
  Suspended, all 15 component types represented) completes within 100ms at
  p95. Full deserialization into a clean world completes within 200ms at p95.

- **AC-P5 — Zero heap allocation with caller-provided buffer**: GIVEN the
  range+component query API with a caller-provided scratch buffer signature
  (`Query(dst []GameEntityID, pos, radius, filter) []GameEntityID`), WHEN a
  spatial query with 2 component filters returns exactly 50 results and the
  caller provides a buffer with capacity ≥ 50, THEN `testing.B.ReportAllocs()`
  reports 0 allocs/op and 0 B/op. The returned slice shares the caller's
  buffer backing array — no heap escape. (If the result count exceeds the
  buffer capacity, Go allocates a new backing array — that allocation is
  measured and reported separately, budgeted at ≤1 alloc/op for the overflow
  case.)
