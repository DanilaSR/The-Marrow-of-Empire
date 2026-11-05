# The Marrow of Empire — Master Architecture

## Document Status
- **Version**: 1.0
- **Last Updated**: 2026-08-09
- **Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan, custom)
- **Risk Level**: HIGH — custom engine, no LLM training data
- **GDDs Covered**: 22 of 22 (all systems)
- **ADRs Referenced**: None yet
- **Technical Director Sign-Off**: Pending
- **Lead Programmer Feasibility**: Pending

## Engine Knowledge Gap Summary

Kaiju Engine is a custom Go/Vulkan engine. No LLM has training data on its APIs.
ALL architectural decisions must be verified against:
- `kaiju/AGENTS.md` — comprehensive engine subsystem documentation
- `kaiju/CLAUDE.md` — build commands, project structure
- `kaiju/src/` — engine source (module: `kaijuengine.com`, Go 1.26.0)
- `kaijuengine-game-dev` skill — entity/rendering/UI patterns

Every engine API reference in this document is flagged `[VERIFY: kaiju/src/...]`.

## System Layer Map

```
┌─────────────────────────────────────────────────────────────┐
│  PRESENTATION LAYER                                         │
│  Investigation Journal · Dialogue UI · Combat UI            │
│  Character & Inventory UI · Map System · Audio System       │
│  Main Menu & Settings                                       │
├─────────────────────────────────────────────────────────────┤
│  FEATURE LAYER                                              │
│  AI Dialogue Engine · Question System · Trust & Relationship│
│  Thread Investigation · Tactical Combat Engine              │
│  Persistent World State · Party System · Faction System     │
├─────────────────────────────────────────────────────────────┤
│  CORE LAYER                                                 │
│  Character Stats & Progression · NPC Knowledge Graph        │
│  Living Environment                                         │
├─────────────────────────────────────────────────────────────┤
│  FOUNDATION LAYER                                           │
│  Entity Framework · Camera System                           │
│  Asset & Scene Pipeline · Save/Load System                  │
├─────────────────────────────────────────────────────────────┤
│  PLATFORM LAYER                                             │
│  Kaiju Engine (Go/Vulkan) · Steamworks API · Linux/Windows  │
│  GGUF Inference Backend (llama.cpp)                         │
└─────────────────────────────────────────────────────────────┘
```

**Layer rules:**
- Layers only communicate downward through defined interfaces
- Foundation has zero game-logic dependencies
- Feature systems never talk directly to Platform — they go through Foundation
- Presentation reads from Feature/Core; never writes to them
- Cross-cutting concerns (save/load, events) are Foundation-layer services

## Architecture Principles

1. **Verify against source**: Every engine API decision must be checked against Kaiju source before implementation. No architectural decision is final until the relevant engine code has been read.

2. **Component over inheritance**: Game entities use composable components (Entity Framework). Systems operate on component data. No deep inheritance hierarchies.

3. **Downward-only dependencies**: Higher layers depend on lower layers; lower layers never know about higher layers. Presentation reads; never writes game state.

4. **Interface-first**: Every cross-layer boundary is defined by a Go interface. Systems depend on interfaces, not concrete types.

5. **Save everything**: All game state is serializable. No game state lives outside the Entity Framework's component stores. Save/Load orchestrates; systems register.

## Module Ownership

### Foundation Layer

| Module | Owns | Exposes | Engine APIs [VERIFY] |
|--------|------|---------|---------------------|
| **Entity Framework** | GameEntityID mapping, typed component stores, spatial index, lifecycle events, write arbitration | `CreateEntity()`, `GetComponent()`, `QueryEntities()`, `SubscribeToEvents()` | `engine.Entity`, `matrix.Transform`, `events.Event`, `pod.Register` |
| **Camera System** | Orthographic projection, zoom state (4u/10u/20u), follow target, camera bounds | `GetViewMatrix()`, `GetProjectionMatrix()`, `SetZoom()`, `GetCameraPosition()` | `cameras.NewStandardCameraOrthographic()` — [VERIFY: kaiju/src/engine/cameras/] |
| **Asset & Scene Pipeline** | Import pipeline, validation rules, scene format, material mapping tables | `ImportAsset()`, `LoadScene()`, `ValidateAsset()`, `GetMaterialMapping()` | Kaiju native mesh/texture formats — [VERIFY: kaiju/src/rendering/] |
| **Save/Load System** | Save format, versioning, migration, slot management, Steam Cloud sync | `Save(slot)`, `Load(slot)`, `ListSaves()`, `DeleteSave(slot)` | `pod.Register`, `EntityData` interface, Steamworks API |

### Core Layer

| Module | Owns | Exposes | Consumes |
|--------|------|---------|----------|
| **Character Stats** | 6-attribute model, modifier table, derived stats, XP/level (1-20) | `GetAttribute(entity, attr)`, `GetDerivedStat(entity, stat)`, `AwardXP(entity, xp)` | Entity Framework |
| **NPC Knowledge Graph** | Domains, facts (FACT/RUMOR/OPINION/SECRET), trust gates, .kg authoring format | `QueryFacts(npc, domain, trust_level)`, `GetDomains(npc)`, `GetPersonality(npc)` | Entity Framework |
| **Living Environment** | Game clock (10:1 ratio), 5 light states, NPC schedules, wildlife spawning | `GetGameTime()`, `GetLightState()`, `IsNPCActive(npc)`, `Wait(duration)` | Entity Framework |

### Feature Layer

| Module | Owns | Exposes | Key Dependencies |
|--------|------|---------|-----------------|
| **AI Dialogue Engine** | LLM inference (~3.8B GGUF), prompt construction, response pipeline, degraded fallback | `GenerateResponse(npc, question, trust)`, `GetModelStatus()` | NPC Knowledge Graph, GGUF backend |
| **Question System** | Free-text input, topic matching (keyword + entity detection), suggestion generation | `MatchTopic(question, npc)`, `GetSuggestions(npc)` | AI Dialogue Engine, NPC Knowledge Graph |
| **Trust & Relationship** | Trust levels (0-3), trust events, vouching, betrayal, trust freeze | `GetTrust(npc)`, `ModifyTrust(npc, event)`, `CanVouchFor(npc, target)` | NPC Knowledge Graph, Persistent World State |
| **Thread Investigation** | Thread creation, auto-chaining, cross-referencing, contradiction detection, revelation threshold (3 facts) | `CreateThread(fact)`, `GetActiveThreads()`, `CheckRevelation(chain)` | Question System, Trust & Relationship |
| **Tactical Combat Engine** | AP system, initiative, grid movement, attack resolution (d20), death/dying (3 rounds) | `StartCombat(participants)`, `EndTurn()`, `ResolveAttack(attacker, target, ability)` | Character Stats, Entity Framework |
| **Persistent World State** | World flags (typed), memory log, region state, evolution rules, clue protection | `GetWorldFlags(entity)`, `SubmitWorldEvent(event)`, `GetMemoryLog(entity, since)` | Entity Framework, Save/Load |
| **Party System** | Party roster (max 6), recruitment, companion management, formation | `GetParty()`, `Recruit(npc)`, `Dismiss(npc)`, `SetFormation()` | Character Stats, Trust & Relationship |
| **Faction System** | Faction definitions, standing values, reputation propagation | `GetStanding(faction)`, `ModifyStanding(faction, delta)`, `GetFactionNPCs(faction)` | Persistent World State, Trust & Relationship |

### Presentation Layer

| Module | Owns | Reads From |
|--------|------|-----------|
| **Investigation Journal** | Thread/chains/facts display, contradiction visualization, revelation prompts | Thread Investigation |
| **Dialogue UI** | Conversation panel, text input, topic chips, typewriter (45-60 cps), thinking indicator | Question System, AI Dialogue Engine |
| **Combat UI** | Turn order bar, AP bar, health bars, ability grid, targeting reticle, damage numbers | Tactical Combat Engine |
| **Character & Inventory UI** | Character sheet, equipment slots, inventory grid, weight bar, gold display | Character Stats, Party System |
| **Map System** | Local/region map, location pins, thread markers, ink-on-parchment rendering | Living Environment, Persistent World State |
| **Audio System** | Ambient layers, adaptive music, SFX, spatial audio (camera = listener) | Living Environment, Tactical Combat Engine |
| **Main Menu & Settings** | Title screen, save/load UI, settings panels, pause menu, Steam Cloud conflict dialog | Save/Load |

## Data Flow

### Frame Update Path
```
Input → Camera System → Entity Framework (spatial query) → Feature Systems (AI, Combat, Investigation) → Entity Framework (component writes) → Presentation (read-only display)
```

### Event / Lifecycle Path
```
Entity Framework (on_entity_created/destroyed) → Persistent World State (memory log) → Trust & Relationship (trust events) → Thread Investigation (thread updates) → Investigation Journal (display)
```

### Save Path
```
Save/Load trigger → Entity Framework (collect all component stores) → pod.Encode → write .sav file → Steam Cloud sync
```

### Load Path
```
Read .sav → verify checksum → check version → run migrations → pod.Decode → Entity Framework (rebuild entities + components) → Persistent World State (restore flags + logs) → Camera reposition
```

### Initialization Order
1. Kaiju Engine bootstrap → 2. Entity Framework → 3. Save/Load (check for saves) → 4. Asset & Scene Pipeline (load scene) → 5. Camera System → 6. Core systems (Stats, Knowledge Graph, Living Environment) → 7. Feature systems → 8. Presentation systems → 9. AI Dialogue Engine (async model load)

## API Boundaries

All cross-module communication uses Go interfaces defined in the consuming module. Key boundaries:

- **Entity Framework → All Systems**: `type EntityStore interface { CreateEntity(...); GetComponent[T](id GameEntityID) (T, bool); QueryEntities(Query) []GameEntityID }`
- **Save/Load → All Systems**: Systems implement `type Persistable interface { SaveState() []byte; LoadState([]byte) error }` — registered with Save/Load at init.
- **AI Dialogue Engine → GGUF Backend**: `type LLMBackend interface { Generate(prompt string) (string, error); Status() ModelStatus }` — swap implementation without touching dialogue logic.
- **NPC Knowledge Graph → AI Dialogue Engine**: `type KnowledgeStore interface { QueryFacts(npc GameEntityID, domain string, trust int) []Fact }`
- **Feature → Presentation**: All reads are through the owning Feature system's public interface. Presentation never writes game state.

## ADR Audit

**Existing ADRs**: None. All architectural decisions are captured in this document and need formal ADRs.

**Traceability**: The TR registry is empty. All 22 GDDs contain technical requirements that need TR-IDs and ADR coverage.

### Required New ADRs — Foundation Layer (must create before coding)

1. **ADR-0001: Entity Architecture** — GameEntityID mapping over Kaiju `engine.Entity`, typed component stores, spatial index strategy, lifecycle event bridge, write arbitration rules. Covers: Entity Framework GDD.
2. **ADR-0002: Save/Load Serialization** — Kaiju `pod` integration, save file format, versioning strategy, migration framework, Steam Cloud integration. Covers: Save/Load GDD.
3. **ADR-0003: Isometric Camera Setup** — Kaiju `StandardCameraOrthographic` configuration, zoom register mapping (4u/10u/20u), occlusion handling, aspect ratio strategy. Covers: Camera System GDD.
4. **ADR-0004: Asset Pipeline & KitBash3D Integration** — FBX→Kaiju conversion, material mapping tables, validation rules, scene format. Covers: Asset & Scene Pipeline GDD.

### Required New ADRs — Core Layer (must create before core system implementation)

5. **ADR-0005: NPC Knowledge Graph Data Model** — `.kg` file format, fact storage and querying, domain matching, trust gating at the data layer. Covers: NPC Knowledge Graph GDD.
6. **ADR-0006: Character Stats & Component Schema** — Six-attribute model, modifier calculation, derived stat formulas, XP table. Covers: Character Stats GDD.
7. **ADR-0007: Game Clock & Environment Systems** — Time ratio, light state machine, NPC schedule component, wait/pass-time. Covers: Living Environment GDD.

### Required New ADRs — Feature Layer (must create before feature implementation)

8. **ADR-0008: LLM Inference Integration** — GGUF backend selection (go-llama.cpp vs. IPC), prompt template, context window management, graceful degradation. Covers: AI Dialogue Engine GDD.
9. **ADR-0009: Combat Engine** — Grid type (square vs. hex), AP system, attack resolution (d20), initiative, death/dying state machine. Covers: Tactical Combat Engine GDD.
10. **ADR-0010: Thread Investigation Engine** — Thread data model, auto-chaining algorithm, contradiction detection, revelation threshold logic. Covers: Thread Investigation GDD.

### Required New ADRs — Cross-Cutting

11. **ADR-0011: Event Bus Architecture** — Lifecycle events, world state change events, trust events. Synchronous vs. asynchronous dispatch. Covers: Entity Framework + Persistent World State GDDs.
12. **ADR-0012: UI Framework** — Kaiju HTML/CSS-like UI system configuration, desk register implementation, font loading, accessibility hooks. Covers: All Presentation GDDs.

## Open Questions

| ID | Summary | Priority | Resolution Path |
|----|---------|----------|-----------------|
| QQ-01 | GGUF backend: go-llama.cpp bindings vs. separate llama.cpp process with IPC vs. custom CGo wrapper? | HIGH | ADR-0008 |
| QQ-02 | Combat grid: square tiles (simpler) or hex (more tactical depth)? | MEDIUM | ADR-0009 |
| QQ-03 | Kaiju native mesh/texture format — need to verify against engine source before ADR-0004 | HIGH | Read kaiju/src/rendering/ |
| QQ-04 | Does Kaiju's `pod` support delta/incremental serialization, or full-state only? | MEDIUM | Read kaiju/src/ + ADR-0002 |
| QQ-05 | Entity Framework: wrap Kaiju entities or define parallel game entity system? | HIGH | ADR-0001 (already resolved in reviewed Entity Framework GDD — game layer over Kaiju entities) |
