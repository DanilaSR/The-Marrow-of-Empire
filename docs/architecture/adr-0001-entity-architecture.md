# ADR-0001: Entity Architecture

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan, custom) |
| **Domain** | Core |
| **Knowledge Risk** | HIGH — custom engine, no LLM training data |
| **References Consulted** | `kaiju/AGENTS.md`, `kaiju/CLAUDE.md`, `kaiju/src/engine/`, `kaijuengine-game-dev` skill |
| **Post-Cutoff APIs Used** | All Kaiju APIs — none exist in LLM training data |
| **Verification Required** | `engine.Entity` lifecycle hooks, `matrix.Transform` dirty-flag behavior, `pod.Register` reflection requirements, `events.Event` dispatch model |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | None |
| **Enables** | ADR-0002 (Save/Load), ADR-0005 (NPC Knowledge Graph), ADR-0006 (Character Stats), ADR-0007 (Living Environment), ADR-0009 (Combat), ADR-0010 (Thread Investigation), ADR-0011 (Event Bus) |
| **Blocks** | All Core and Feature layer implementation — no game code can be written without the entity model |
| **Ordering Note** | Must be Accepted before any other ADR implementation begins |

## Context

### Problem Statement
The Marrow of Empire needs a game-level entity system that supports composable components, spatial queries, lifecycle events, and full serialization. Kaiju Engine already provides `engine.Entity`, `matrix.Transform`, `events.Event`, and `pod` serialization. Should the game define its own entity system, wrap Kaiju's, or extend Kaiju's directly?

### Constraints
- Must support 500+ entities per region (Entity Framework GDD tuning knob: `max_entities_per_region = 500`)
- Must serialize all game state through Save/Load (Kaiju `pod` + `EntityData`)
- Must support component queries by type, tag, and spatial position
- Must integrate with Kaiju's rendering pipeline (`rendering.Drawing.Transform`)
- Must support lifecycle events for downstream systems (Persistent World State, Trust & Relationship, Thread Investigation)

### Requirements
- Game entities must be distinguishable from engine entities (separation of concerns)
- Component model must be type-safe (Go generics for component access)
- Spatial queries must complete within 1ms for 500 entities
- Entity creation must complete within 0.5ms
- GameEntityIDs must be stable across save/load and never reused

## Decision

**Game layer over Kaiju entities (Option A).** The game defines a typed component layer that wraps Kaiju's native entity system. Game entities are identified by `GameEntityID` (uint64, monotonic, never reused) and mapped to Kaiju `engine.Entity` via `EntityId` (string, format `"game_{GameEntityID}"`).

### Architecture Diagram

```
┌──────────────────────────────────────────────────┐
│  GAME LAYER (Entity Framework)                   │
│                                                  │
│  GameEntityID (uint64) ←──→ EntityId (string)   │
│  ┌─────────────────────────────────────────┐    │
│  │ Typed Component Stores                  │    │
│  │ map[GameEntityID]Health                 │    │
│  │ map[GameEntityID]Stats                  │    │
│  │ map[GameEntityID]Knowledge              │    │
│  │ map[GameEntityID]Persistent             │    │
│  │ ... (12+ component types)               │    │
│  └─────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────┐    │
│  │ Spatial Index (Uniform Grid)            │    │
│  │ map[[2]int32][]GameEntityID             │    │
│  └─────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────┐    │
│  │ Lifecycle Bridge                        │    │
│  │ OnActivate → on_entity_created          │    │
│  │ OnDeactivate → on_entity_suspended      │    │
│  │ OnDestroy → on_entity_destroyed         │    │
│  └─────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────┐    │
│  │ Write Arbitration                       │    │
│  │ Per-frame delta accumulation            │    │
│  │ System priority resolution              │    │
│  └─────────────────────────────────────────┘    │
└──────────────────┬───────────────────────────────┘
                   │ maps to
┌──────────────────▼───────────────────────────────┐
│  KAIJU ENGINE LAYER                              │
│                                                  │
│  engine.Entity                                   │
│  ├── EntityId (string)                           │
│  ├── matrix.Transform (position, rotation, scale)│
│  ├── rendering.Drawing (Transform reference)     │
│  ├── Parent/Children hierarchy                   │
│  └── OnActivate / OnDeactivate / OnDestroy       │
│                                                  │
│  events.Event (lifecycle dispatch)               │
│  pod.Register + EntityData (serialization)       │
└──────────────────────────────────────────────────┘
```

### Key Interfaces

```go
// GameEntityID is the universal game-layer entity reference.
// Stable across save/load, never reused after destruction.
type GameEntityID uint64

// EntityStore is the primary interface exposed by Entity Framework.
type EntityStore interface {
    // CreateEntity creates a Kaiju engine.Entity and returns its GameEntityID.
    CreateEntity(entityType EntityType, displayName string, position Vec3) (GameEntityID, error)

    // DestroyEntity marks an entity for destruction. Lifecycle events fire.
    DestroyEntity(id GameEntityID) error

    // GetComponent retrieves a typed component for an entity.
    GetComponent[T any](id GameEntityID) (T, bool)

    // SetComponent stores a typed component for an entity.
    SetComponent[T any](id GameEntityID, component T) error

    // QueryEntities returns GameEntityIDs matching the given query.
    QueryEntities(q Query) []GameEntityID
}

// Query defines a component + spatial query.
type Query struct {
    Components  []string  // required component type names
    Tags        []string  // required tags (any match)
    Position    Vec3      // center of spatial query
    Range       float64   // search radius in world units
}

// Persistable is implemented by component stores for save/load.
type Persistable interface {
    SaveState() ([]byte, error)
    LoadState([]byte) error
}
```

## Alternatives Considered

### Alternative 1: Game Layer over Kaiju (CHOSEN)
- **Description**: GameEntityID maps to Kaiju EntityId. Typed component stores in Go maps. Kaiju owns transform/lifecycle; game layer owns gameplay state.
- **Pros**: Clean separation. Kaiju handles rendering/transform/hierarchy. Game code never touches engine internals. Swap engine later by rewriting the mapping layer. Type-safe component access via Go generics.
- **Cons**: Double identity (GameEntityID + EntityId). Mapping overhead on entity lookup. Must bridge lifecycle events.
- **Rejection Reason**: N/A — this is the chosen approach.

### Alternative 2: Parallel Entity System
- **Description**: Define a completely separate game entity system with its own transform. Sync to Kaiju entities only for rendering.
- **Pros**: Complete independence from engine. No mapping overhead.
- **Cons**: Duplicates transform state (position sync bugs). Two hierarchies to maintain. Kaiju's rendering expects `Drawing.Transform = &entity.Transform` — parallel system breaks this link. More code, more bugs.
- **Rejection Reason**: Duplicates engine functionality without benefit. Kaiju's entity/transform system is performant and well-integrated with rendering.

### Alternative 3: Extend Kaiju Entities Directly
- **Description**: Embed game component data directly into Kaiju entity structs via Go composition or wrapper types.
- **Pros**: No mapping layer. Single identity. Direct access to engine APIs.
- **Cons**: Tight coupling to engine internals. Engine version upgrades may break game code. Cannot easily test game systems without a running engine. Component queries require iterating all engine entities.
- **Rejection Reason**: Tight coupling to a custom engine under active development is high-risk. The mapping layer cost (map lookup) is negligible (~50ns) compared to the safety of decoupling.

## Consequences

### Positive
- Game code depends on `EntityStore` interface, not Kaiju internals — testable with mock entity store
- Typed component access via Go generics — no interface{} casts, no reflection in hot paths
- Spatial index enables efficient range queries for combat targeting, AI perception, interaction
- Lifecycle bridge ensures all downstream systems are notified of entity state changes
- Write arbitration prevents component field conflicts between systems

### Negative
- Double identity: every game entity has both a GameEntityID and a Kaiju EntityId
- Mapping overhead: `host.EntityById(string)` call to get Kaiju entity from GameEntityID
- Lifecycle event translation: Kaiju events must be bridged to game-layer events
- Component stores must register with `pod.Register` for serialization — reflection requirement

### Risks
- **Kaiju API instability**: Engine under active development (nightly builds). Entity/Transform API may change. Mitigation: `EntityStore` interface abstracts engine — only the mapping layer changes.
- **Spatial index performance**: Uniform grid may degrade with highly clustered entities (e.g., town square with 100+ NPCs). Mitigation: bucket overload triggers linear scan with warning; tune `spatial_bucket_size` (default 32 units).
- **Memory overhead**: Per-component maps for 500 entities × 12 component types = maps with mostly-empty slots. Mitigation: only create component stores for components that actually exist on entities; nil maps cost nothing.

## GDD Requirements Addressed

| GDD System | Requirement | How This ADR Addresses It |
|------------|-------------|--------------------------|
| Entity Framework | CR-1: Entity Identity — unique GameEntityID, stable across save/load, never reused | uint64 monotonic counter, mapped to Kaiju EntityId string |
| Entity Framework | CR-2: Component Model — composable components, pure data containers | Typed `map[GameEntityID]ComponentType` stores, Go generics for type-safe access |
| Entity Framework | CR-3: Component Queries — by type, tag, position | `QueryEntities(Query)` with component type names, tag filter, spatial range |
| Entity Framework | CR-4: Entity Lifecycle — Pending/Active/Suspended/Destroyed | Game-layer state machine bridged to Kaiju OnActivate/OnDeactivate/OnDestroy |
| Entity Framework | CR-5: Lifecycle Events — broadcast to subscribed systems | `events.Event` bridge: Kaiju events → game-layer event dispatch |
| Entity Framework | CR-6: Spatial Organization — efficient range queries | Uniform grid spatial index: `map[[2]int32][]GameEntityID` with configurable bucket size |

## Performance Implications
- **CPU**: Entity creation <0.5ms (Kaiju entity + component store insertion). Range query across 500 entities <1ms (uniform grid). Entity ID lookup ~50ns (map access).
- **Memory**: ~2KB per entity for component data (12 component types, sparsely populated). ~50KB for spatial index (500 entities in 32-unit buckets).
- **Load Time**: Entity reconstruction on scene load dominated by `pod.Decode` — ~2 seconds for 200 entities.
- **Network**: N/A (single-player).

## Migration Plan
No migration needed — this is the first architectural decision. All game code will be written against the `EntityStore` interface from the start.

## Validation Criteria
- Entity creation benchmark: <0.5ms per entity at p95
- Spatial query benchmark: <1ms for 500 entities in range query
- Save/load round-trip: entity IDs, component values, and spatial positions identical after reload
- Lifecycle test: all subscribed systems receive `on_entity_created` within one frame of entity activation

## Related Decisions
- ADR-0002: Save/Load Serialization — depends on entity component registration with `pod`
- ADR-0005: NPC Knowledge Graph — depends on Knowledge component store
- ADR-0009: Combat Engine — depends on spatial queries for targeting
- ADR-0011: Event Bus Architecture — depends on lifecycle event bridge defined here
