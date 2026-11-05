# Systems Index: The Marrow of Empire

> **Status**: Draft
> **Created**: 2026-08-08
> **Last Updated**: 2026-08-08
> **Source Concept**: design/gdd/game-concept.md

---

## Overview

The Marrow of Empire is an isometric tactical CRPG with AI-driven NPCs, thread-based
investigation, and persistent world state. Its systems span six categories: AI &
Knowledge (the core innovation — local LLM-powered NPCs with domain-limited knowledge
graphs), Investigation (replaces traditional quest log with player-driven thread
tracking), Combat (turn-based tactical with positioning and synergies), World
(persistent, living, reactive), Foundation (entity architecture, camera, assets, save/load),
and Presentation (UI, audio, menus). The 22 systems are layered from Foundation (zero
dependencies) through Presentation (wraps all gameplay systems), with 16 required for MVP.

---

## Systems Enumeration

| # | System Name | Category | Priority | Status | Design Doc | Depends On |
|---|-------------|----------|----------|--------|------------|------------|
| 1 | Entity Framework | Foundation | MVP | Approved | [entity-framework.md](entity-framework.md) | — |
| 2 | Camera System | Foundation | MVP | Approved | [camera-system.md](camera-system.md) | Entity Framework, Asset & Scene Pipeline |
| 3 | Asset & Scene Pipeline | Foundation | MVP | Designed | [asset-scene-pipeline.md](asset-scene-pipeline.md) | — |
| 4 | Save/Load System | Foundation | MVP | Needs Revision | [save-load-system.md](save-load-system.md) | — ⚠️ CR-4 camera persistence conflicts with Camera GDD |
| 5 | Character Stats & Progression | Combat | MVP | Needs Revision | [character-stats-progression.md](character-stats-progression.md) | Entity Framework |
| 6 | NPC Knowledge Graph | AI & Knowledge | MVP | Designed | [npc-knowledge-graph.md](npc-knowledge-graph.md) | Entity Framework |
| 7 | Living Environment | World | MVP | Needs Revision | [living-environment.md](living-environment.md) | Entity Framework, Save/Load |
| 8 | Persistent World State | World | VS | Designed | [persistent-world-state.md](persistent-world-state.md) | Entity Framework, Save/Load |
| 9 | AI Dialogue Engine | AI & Knowledge | MVP | Needs Revision | [ai-dialogue-engine.md](ai-dialogue-engine.md) | NPC Knowledge Graph |
| 10 | Question System | AI & Knowledge | MVP | Designed | [question-system.md](question-system.md) | AI Dialogue Engine, NPC Knowledge Graph |
| 11 | Trust & Relationship | AI & Knowledge | MVP | Designed | [trust-relationship.md](trust-relationship.md) | NPC Knowledge Graph, Persistent World State |
| 12 | Thread Investigation | Investigation | MVP | Designed | [thread-investigation.md](thread-investigation.md) | Question System, Trust & Relationship |
| 13 | Tactical Combat Engine | Combat | MVP | Needs Revision | [tactical-combat-engine.md](tactical-combat-engine.md) | Character Stats, Entity Framework |
| 14 | Party System | Combat | VS | In Review | [party-system.md](party-system.md) | Character Stats, Trust & Relationship |
| 15 | Faction System | World | Alpha | Needs Revision | [faction-system.md](faction-system.md) | Persistent World State, Trust & Relationship |
| 16 | Investigation Journal | Investigation | MVP | Needs Revision | [investigation-journal.md](investigation-journal.md) | Thread Investigation |
| 17 | Dialogue UI | Presentation | MVP | Needs Revision | [dialogue-ui.md](dialogue-ui.md) | Question System, AI Dialogue Engine |
| 18 | Combat UI | Presentation | MVP | Needs Revision | [combat-ui.md](combat-ui.md) | Tactical Combat Engine |
| 19 | Character & Inventory UI | Presentation | VS | Designed | [character-inventory-ui.md](character-inventory-ui.md) | Character Stats, Party System |
| 20 | Map System | Presentation | Alpha | Needs Revision | [map-system.md](map-system.md) | Living Environment, Persistent World State |
| 21 | Audio System | Presentation | VS | Needs Revision | [audio-system.md](audio-system.md) | Living Environment, Tactical Combat Engine |
| 22 | Main Menu & Settings | Presentation | MVP | Designed | [main-menu-and-settings.md](main-menu-and-settings.md) | Save/Load |
| 23 | Loot & Economy | Economy | MVP | Designed | [loot-economy.md](loot-economy.md) | Entity Framework, Character Stats, Faction System, Tactical Combat Engine |

---

## Categories

| Category | Description | Systems |
|----------|-------------|---------|
| **AI & Knowledge** | AI-driven NPC dialogue and knowledge systems — the core innovation | NPC Knowledge Graph, AI Dialogue Engine, Question System, Trust & Relationship |
| **Investigation** | Thread-based investigation replacing traditional quest log | Thread Investigation, Investigation Journal |
| **Combat** | Turn-based tactical combat and character systems | Character Stats & Progression, Tactical Combat Engine, Party System |
| **World** | Persistent, living world with environmental systems | Living Environment, Persistent World State, Faction System |
| **Foundation** | Zero-dependency infrastructure — everything builds on these | Entity Framework, Camera System, Asset & Scene Pipeline, Save/Load System |
| **Economy** | Loot, gold, shops, equipment acquisition — the resource layer between combat and inventory | Loot & Economy |
| **Presentation** | UI, audio, and player-facing information displays | Dialogue UI, Combat UI, Character & Inventory UI, Map System, Audio System, Main Menu & Settings |

---

## Priority Tiers

| Tier | Definition | Target Milestone | Systems |
|------|------------|------------------|---------|
| **MVP** | Required for the core loop. Without these, you can't test "is the AI NPC investigation loop fun?" | First playable prototype (1 region, ~20 NPCs) | 16 systems (see enumeration) |
| **Vertical Slice** | One region, production quality. Full world memory, full party, audio, inventory UI. | Polished demo (1 region) | 4 systems: Persistent World State, Party System, Character & Inventory UI, Audio System |
| **Alpha** | All features present. Multi-region scope. | Full borderland playable (5-7 regions) | 2 systems: Faction System, Map System |
| **Full Vision** | Polish, edge cases, nice-to-haves. | Beta / Release | — (to be defined as scope evolves) |

---

## Dependency Map

### Foundation Layer (no dependencies)

1. **Entity Framework** — Base entity/component architecture for all game objects (characters, NPCs, items, props, wildlife). Every other system creates or manipulates entities.
2. **Camera System** — Isometric camera: zoom, pan, tactical view. Pure rendering concern — independent of gameplay.
3. **Asset & Scene Pipeline** — KitBash3D Dark Fantasy import, texture/model management, scene composition. Pure infrastructure.
4. **Save/Load System** — State serialization framework. Defines the interface; gameplay systems register their state. Generic at this layer.

### Core Layer (depends on Foundation)

5. **Character Stats & Progression** — depends on: Entity Framework. Attributes, skills, leveling, build choices. The mechanical backbone of tactical mastery.
6. **NPC Knowledge Graph** — depends on: Entity Framework. Data model defining what each NPC knows: domain, facts, knowledge boundaries. The core innovation's data layer.
7. **Living Environment** — depends on: Entity Framework. Day/night cycle, weather, seasons, wildlife patterns. MVP: day/night only.

### Feature Layer (depends on Core)

8. **Persistent World State** — depends on: Entity Framework, Save/Load. Player action consequences — cleared ruins, NPC memory, faction shifts. VS tier.
9. **AI Dialogue Engine** — depends on: NPC Knowledge Graph. LLM integration: knowledge graph → prompt construction → response generation via local GGUF model.
10. **Question System** — depends on: AI Dialogue Engine, NPC Knowledge Graph. Player input method — how questions are asked and matched to NPC knowledge.
11. **Trust & Relationship** — depends on: NPC Knowledge Graph, Persistent World State. NPCs gate deep knowledge behind trust; relationships persist.
12. **Thread Investigation** — depends on: Question System, Trust & Relationship. Thread creation, tracking, cross-referencing, and connection detection.
13. **Tactical Combat Engine** — depends on: Character Stats, Entity Framework. Turn-based, grid/range-based, positioning, abilities, environmental interaction.
14. **Party System** — depends on: Character Stats, Trust & Relationship. Recruitment, party management, companion behavior. VS tier.
15. **Faction System** — depends on: Persistent World State, Trust & Relationship. Faction standing, reputation consequences. Alpha tier.

### Presentation Layer (depends on Features)

16. **Investigation Journal** — depends on: Thread Investigation. Player-facing investigation interface — tracks leads and discoveries, not quest markers.
17. **Dialogue UI** — depends on: Question System, AI Dialogue Engine. Conversation interface: question input, NPC response display, topic suggestions.
18. **Combat UI** — depends on: Tactical Combat Engine. Tactical interface: action menu, turn order, health bars, ability targeting.
19. **Character & Inventory UI** — depends on: Character Stats, Party System. Stats display, equipment, inventory management. VS tier.
20. **Map System** — depends on: Living Environment, Persistent World State. Local/region maps. Alpha tier.
21. **Audio System** — depends on: Living Environment, Tactical Combat Engine. Ambient nature soundscapes, adaptive music, SFX. VS tier.
22. **Main Menu & Settings** — depends on: Save/Load. Title screen, options, save/load UI, accessibility settings.

---

## Recommended Design Order

| Order | System | Priority | Layer | Est. Effort |
|-------|--------|----------|-------|-------------|
| 1 | Entity Framework | MVP | Foundation | M |
| 2 | Camera System | MVP | Foundation | S |
| 3 | Asset & Scene Pipeline | MVP | Foundation | M |
| 4 | Save/Load System | MVP | Foundation | M |
| 5 | Character Stats & Progression | MVP | Core | M |
| 6 | NPC Knowledge Graph | MVP | Core | L |
| 7 | Living Environment | MVP | Core | M |
| 8 | AI Dialogue Engine | MVP | Feature | L |
| 9 | Question System | MVP | Feature | L |
| 10 | Trust & Relationship | MVP | Feature | M |
| 11 | Thread Investigation | MVP | Feature | L |
| 12 | Tactical Combat Engine | MVP | Feature | L |
| 13 | Investigation Journal | MVP | Presentation | M |
| 14 | Dialogue UI | MVP | Presentation | M |
| 15 | Combat UI | MVP | Presentation | M |
| 16 | Main Menu & Settings | MVP | Presentation | S |
| 17 | Persistent World State | VS | Feature | L |
| 18 | Party System | VS | Feature | M |
| 19 | Character & Inventory UI | VS | Presentation | M |
| 20 | Audio System | VS | Presentation | M |
| 21 | Faction System | Alpha | Feature | M |
| 22 | Map System | Alpha | Presentation | S |

*Effort: S = 1 session, M = 2-3 sessions, L = 4+ sessions. A "session" is one focused design conversation producing a complete GDD.*

---

## Circular Dependencies

None found. The dependency graph is acyclic.

---

## High-Risk Systems

| System | Risk Type | Risk Description | Mitigation |
|--------|-----------|-----------------|------------|
| NPC Knowledge Graph | Design | Authoring pipeline for 100+ NPC knowledge domains is unproven. "Deep but fractured" must scale beyond 5 NPCs (paper prototype validated at small scale). | Prototype confirmed 5-NPC design works. Scale test: design knowledge graphs for 20 NPCs before committing to GDD. |
| AI Dialogue Engine | Technical | Local LLM inference (~3.8B GGUF) alongside game rendering introduces memory (2.7GB+) and latency constraints. Prompt engineering must keep model in character and within knowledge domain. | Prototype the LLM integration early — test response latency, memory footprint, and prompt constraint effectiveness on target hardware. |
| Thread Investigation | Design | Information density across 100+ NPCs with "deep but fractured" knowledge may overwhelm players. Thread tracking UX is unproven. | MVP validates with 15-25 NPCs and one complete chain. Journal UX must be tested with real players early. |
| Persistent World State | Technical | Tracking meaningful state changes across 5-7 regions with 100+ NPCs requires careful state architecture. Custom engine (Kaiju) adds complexity. | Design state serialization abstractly in Save/Load (Foundation). Prototype world state with one region before scaling. |

---

## Progress Tracker

| Metric | Count |
|--------|-------|
| Total systems identified | 23 |
| Design docs started | 23 |
| Design docs reviewed | 2 |
| Design docs approved | 2 |
| Design docs needs revision | 11 |
| MVP systems designed | 16 / 16 |
| Vertical Slice systems designed | 4 / 4 |
| Alpha systems designed | 2 / 2 |

---

## Next Steps

- [ ] Design MVP-tier systems first — start with Entity Framework (design order #1)
- [ ] Run `/design-system entity-framework` to author the first GDD
- [ ] Run `/design-review design/gdd/[system].md` after each GDD is authored
- [ ] Run `/prototype` spikes on high-risk systems (NPC Knowledge Graph, AI Dialogue Engine) before committing to full GDDs
- [ ] Run `/gate-check pre-production` when MVP systems are designed
