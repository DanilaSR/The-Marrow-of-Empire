# Architecture Review Report

**Date**: 2026-08-09
**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan, custom)
**GDDs Reviewed**: 22
**ADRs Reviewed**: 12

---

## Traceability Summary

Total requirements extracted: 68
✅ Covered: 62 (91%)
⚠️ Partial: 4 (6%)
❌ Gaps: 2 (3%)

### Foundation Layer (ADR-0001 through ADR-0004)

| TR-ID | GDD | Requirement | ADR | Status |
|-------|-----|-------------|-----|--------|
| TR-ef-001 | entity-framework.md | GameEntityID mapping to Kaiju EntityId | ADR-0001 | ✅ |
| TR-ef-002 | entity-framework.md | Typed component stores with Go generics | ADR-0001 | ✅ |
| TR-ef-003 | entity-framework.md | Uniform grid spatial index | ADR-0001 | ✅ |
| TR-ef-004 | entity-framework.md | Lifecycle event bridge (Kaiju → game events) | ADR-0001, ADR-0011 | ✅ |
| TR-ef-005 | entity-framework.md | Write arbitration by system priority | ADR-0001 | ✅ |
| TR-ef-006 | entity-framework.md | pod.Register serialization | ADR-0001, ADR-0002 | ✅ |
| TR-cam-001 | camera-system.md | Orthographic projection 30°/45° | ADR-0003 | ✅ |
| TR-cam-002 | camera-system.md | Three zoom registers (4u/10u/20u) | ADR-0003 | ✅ |
| TR-cam-003 | camera-system.md | Occlusion handling (fade, roof removal) | ADR-0003 | ✅ |
| TR-cam-004 | camera-system.md | Aspect ratio strategy (16:9 ref, 21:9 supported) | ADR-0003 | ✅ |
| TR-asset-001 | asset-scene-pipeline.md | FBX → Kaiju native conversion | ADR-0004 | ✅ |
| TR-asset-002 | asset-scene-pipeline.md | KitBash3D scale/rotation normalization | ADR-0004 | ✅ |
| TR-asset-003 | asset-scene-pipeline.md | YAML material mapping tables | ADR-0004 | ✅ |
| TR-asset-004 | asset-scene-pipeline.md | Asset validation rules (poly, texture, naming) | ADR-0004 | ✅ |
| TR-asset-005 | asset-scene-pipeline.md | Scene format and entity placement | ADR-0004, ADR-0001 | ✅ |
| TR-save-001 | save-load-system.md | pod full-state serialization | ADR-0002 | ✅ |
| TR-save-002 | save-load-system.md | Save versioning with migration functions | ADR-0002 | ✅ |
| TR-save-003 | save-load-system.md | Steam Cloud integration | ADR-0002 | ✅ |
| TR-save-004 | save-load-system.md | Platform-standard save paths | ADR-0002 | ✅ |
| TR-save-005 | save-load-system.md | Checksum + backup corruption recovery | ADR-0002 | ✅ |

### Core Layer (ADR-0005 through ADR-0007)

| TR-ID | GDD | Requirement | ADR | Status |
|-------|-----|-------------|-----|--------|
| TR-stats-001 | character-stats-progression.md | Six-attribute model (STR/DEX/CON/INT/WIS/CHA, 3-18) | ADR-0006 | ✅ |
| TR-stats-002 | character-stats-progression.md | Modifier formula: floor((A-10)/2) | ADR-0006 | ✅ |
| TR-stats-003 | character-stats-progression.md | Derived stats (HP, AP, Initiative, Evasion) | ADR-0006 | ✅ |
| TR-stats-004 | character-stats-progression.md | Point-buy creation (30 pts, max 16 start) | ADR-0006 | ✅ |
| TR-stats-005 | character-stats-progression.md | XP/level progression (1-20, linear thresholds) | ADR-0006 | ✅ |
| TR-kg-001 | npc-knowledge-graph.md | Knowledge domains per NPC | ADR-0005 | ✅ |
| TR-kg-002 | npc-knowledge-graph.md | Fact types (FACT/RUMOR/OPINION/SECRET) | ADR-0005 | ✅ |
| TR-kg-003 | npc-knowledge-graph.md | Trust gating (0-3 levels) | ADR-0005 | ✅ |
| TR-kg-004 | npc-knowledge-graph.md | YAML .kg authoring format | ADR-0005 | ✅ |
| TR-kg-005 | npc-knowledge-graph.md | Entity references via GameEntityID | ADR-0005, ADR-0001 | ✅ |
| TR-kg-006 | npc-knowledge-graph.md | Knowledge boundaries (knows_not) | ADR-0005 | ✅ |
| TR-env-001 | living-environment.md | 24-hour game clock, 10:1 ratio | ADR-0007 | ✅ |
| TR-env-002 | living-environment.md | Five light states with interpolation | ADR-0007 | ✅ |
| TR-env-003 | living-environment.md | Wait/pass-time mechanic | ADR-0007 | ✅ |
| TR-env-004 | living-environment.md | NPC schedules via Schedule component | ADR-0007, ADR-0001 | ✅ |
| TR-env-005 | living-environment.md | Combat pauses clock | ADR-0007 | ✅ |

### Feature Layer (ADR-0008 through ADR-0011)

| TR-ID | GDD | Requirement | ADR | Status |
|-------|-----|-------------|-----|--------|
| TR-ai-001 | ai-dialogue-engine.md | LLM inference via GGUF (~3.8B, Q5_K_M) | ADR-0008 | ✅ |
| TR-ai-002 | ai-dialogue-engine.md | Prompt template (SYSTEM/FACTS/HISTORY/QUESTION) | ADR-0008 | ✅ |
| TR-ai-003 | ai-dialogue-engine.md | Graceful degradation (text fallback) | ADR-0008 | ✅ |
| TR-ai-004 | ai-dialogue-engine.md | <2s response target, non-blocking | ADR-0008 | ✅ |
| TR-ai-005 | ai-dialogue-engine.md | Separate process IPC (stdin/stdout JSON) | ADR-0008 | ✅ |
| TR-q-001 | question-system.md | Free-text input + topic matching | — | ⚠️ Partial |
| TR-q-002 | question-system.md | Domain keyword scoring | — | ⚠️ Partial |
| TR-q-003 | question-system.md | Topic suggestion generation | — | ⚠️ Partial |
| TR-trust-001 | trust-relationship.md | Trust levels (0-3) per NPC | ADR-0005 | ✅ |
| TR-trust-002 | trust-relationship.md | Trust events (demonstrated_investigation, etc.) | ADR-0011 | ✅ |
| TR-trust-003 | trust-relationship.md | Vouching and trust propagation | ADR-0011 | ⚠️ Partial |
| TR-thread-001 | thread-investigation.md | Thread creation from fact references | ADR-0010 | ✅ |
| TR-thread-002 | thread-investigation.md | Auto-chaining by entity overlap | ADR-0010 | ✅ |
| TR-thread-003 | thread-investigation.md | Contradiction detection | ADR-0010 | ✅ |
| TR-thread-004 | thread-investigation.md | Revelation threshold (3+ facts) | ADR-0010 | ✅ |
| TR-thread-005 | thread-investigation.md | PlayerKnowledge component persistence | ADR-0010, ADR-0001 | ✅ |
| TR-combat-001 | tactical-combat-engine.md | Square grid, 1 unit = 1 meter | ADR-0009 | ✅ |
| TR-combat-002 | tactical-combat-engine.md | AP system (4 + DEX_mod) | ADR-0009, ADR-0006 | ✅ |
| TR-combat-003 | tactical-combat-engine.md | d20 attack resolution | ADR-0009 | ✅ |
| TR-combat-004 | tactical-combat-engine.md | Death/dying (3-round stabilization) | ADR-0009 | ✅ |
| TR-combat-005 | tactical-combat-engine.md | Flanking, cover, elevation bonuses | ADR-0009 | ✅ |
| TR-pws-001 | persistent-world-state.md | World flag store (typed, per-entity) | ADR-0001 | ✅ |
| TR-pws-002 | persistent-world-state.md | Memory log (append-only, capped) | — | ❌ Gap |
| TR-pws-003 | persistent-world-state.md | World evolution rules | — | ❌ Gap |
| TR-pws-004 | persistent-world-state.md | Clue-path protection on entity destroy | ADR-0001 | ✅ |
| TR-party-001 | party-system.md | Party roster, recruitment, dismissal | ADR-0006, ADR-0011 | ✅ |
| TR-faction-001 | faction-system.md | Faction standing values, reputation | ADR-0011 | ✅ |

### Presentation Layer

| TR-ID | GDD | Requirement | ADR | Status |
|-------|-----|-------------|-----|--------|
| TR-ui-001 | All Presentation GDDs | Desk register (parchment overlay) | ADR-0012 | ✅ |
| TR-ui-002 | All Presentation GDDs | Field register (umber combat chrome) | ADR-0012 | ✅ |
| TR-ui-003 | All Presentation GDDs | Pen-and-ink icons | ADR-0012 | ✅ |
| TR-ui-004 | All Presentation GDDs | Paper-slide motion, no 3D/bounce | ADR-0012 | ✅ |
| TR-ui-005 | All Presentation GDDs | Kaiju HTML/CSS-like UI | ADR-0012 | ✅ |
| TR-ui-006 | All Presentation GDDs | Minimum 1280×720, ref 1920×1080 | ADR-0012 | ✅ |
| TR-ui-007 | All Presentation GDDs | Accessibility: tab order, focus ring, typewriter settings | ADR-0012 | ✅ |

---

## Coverage Gaps

### ❌ TR-pws-002: Persistent World State — Memory Log
**GDD**: persistent-world-state.md
**Requirement**: Structured memory log per entity (append-only, capped at 50 entries, protected events never evicted)
**Suggested ADR**: ADR-0011 covers event bus but doesn't specify the log storage format or eviction policy. Extend ADR-0011 or create dedicated ADR.
**Domain**: Core
**Engine Risk**: LOW (pure data structure)

### ❌ TR-pws-003: Persistent World State — World Evolution Rules
**GDD**: persistent-world-state.md
**Requirement**: Autonomous world evolution rules (condition → event, evaluated on region load)
**Suggested ADR**: `/architecture-decision "World Evolution Rules"`
**Domain**: Core
**Engine Risk**: MEDIUM (rule evaluation timing vs. frame budget)

### ⚠️ TR-q-001/002/003: Question System
**GDD**: question-system.md
**Requirements**: Free-text topic matching, domain keyword scoring, topic suggestion generation
**Coverage**: ADR-0005 defines the knowledge graph data model and query interface, but the Question System's topic matching algorithm (keyword overlap + entity reference detection + relevance threshold) is not explicitly addressed by any ADR.
**Suggested**: Extend ADR-0005 to cover the query matching algorithm, or create a dedicated ADR.

---

## Cross-ADR Conflicts

**No conflicts detected.** All 12 ADRs form a clean dependency DAG rooted at ADR-0001. Data ownership is unambiguous:
- Entity Framework owns GameEntityID space and component stores (ADR-0001)
- NPC Knowledge Graph owns fact data and domain definitions (ADR-0005)
- Save/Load orchestrates serialization; systems register (ADR-0002)
- Event Bus owns cross-system event dispatch (ADR-0011)
- Combat Engine owns combat state; reads Character Stats (ADR-0009)
- Each ADR's "Owns/Exposes/Consumes" is consistent with others

### ADR Dependency Order (topologically sorted)

**Foundation (no dependencies):**
1. ADR-0001: Entity Architecture
2. ADR-0003: Isometric Camera Setup
3. ADR-0004: Asset Pipeline & KitBash3D
4. ADR-0011: Event Bus Architecture

**Depends on Foundation:**
5. ADR-0002: Save/Load Serialization (requires ADR-0001)
6. ADR-0005: NPC Knowledge Graph (requires ADR-0001)
7. ADR-0006: Character Stats (requires ADR-0001)
8. ADR-0007: Game Clock (requires ADR-0001)

**Feature layer:**
9. ADR-0008: LLM Inference (requires ADR-0001, ADR-0005)
10. ADR-0009: Combat Engine (requires ADR-0001, ADR-0006)
11. ADR-0010: Thread Investigation (requires ADR-0001, ADR-0005, ADR-0011)

**Presentation:**
12. ADR-0012: UI Framework (requires ADR-0001)

---

## Engine Compatibility Issues

**Engine**: Kaiju Engine 26.07.29.1-nightly (HIGH RISK — custom)
**ADRs with Engine Compatibility sections**: 12/12 ✅

All 12 ADRs correctly flag "Knowledge Risk: HIGH" and list "Verification Required" items against specific Kaiju source paths. No deprecated API references found (custom engine, no deprecation history to check).

**Unresolved engine verifications** (all ADRs carry these — structural, not a failure):
- ADR-0001: `engine.Entity` lifecycle hooks, `matrix.Transform` dirty-flag behavior, `pod.Register` reflection requirements
- ADR-0002: `pod` delta serialization support, `EntityData` interface requirements
- ADR-0003: `StandardCameraOrthographic` API stability, aspect ratio handling
- ADR-0004: Kaiju native mesh/texture formats (QQ-03)
- ADR-0008: llama.cpp IPC latency, memory footprint with game running
- ADR-0012: Kaiju HTML/CSS-like UI capabilities, AccessKit support

These are resolved by the planned engine feasibility spike (Milestone 0).

---

## Architecture Document Coverage

`docs/architecture/architecture.md` v1.0 covers:
- System layer map: ✅ All 22 systems assigned to layers
- Module ownership: ✅ Owns/Exposes/Consumes defined for all modules
- Data flow: ✅ Frame update, event, save, load, initialization paths
- API boundaries: ✅ Go interfaces for key module boundaries
- ADR Audit: ❌ **STALE** — says "Existing ADRs: None" and "ADRs Referenced: None yet". Predates the 12 ADRs. Must be refreshed.

**Orphaned architecture**: None. All modules in architecture.md have corresponding GDDs.

---

## GDD Revision Flags

No GDD assumptions conflict with verified engine behaviour. The 12 ADRs follow GDD specifications faithfully. The Entity Framework and Camera System GDDs were revised through multiple review cycles and now correctly specify Kaiju-specific implementation details.

---

## Verdict: CONCERNS

91% coverage (62/68 requirements covered). Two genuine gaps (PWS memory log eviction, world evolution rules) and one partial (Question System topic matching). No cross-ADR conflicts. Clean dependency DAG. Architecture document needs refresh to reference the 12 ADRs.

### Required Actions

1. **Refresh `architecture.md`** — update ADR Audit to list the 12 ADRs, update Document Status (ADRs Referenced), record TD sign-off
2. **Close coverage gaps** — extend ADR-0011 for memory log storage, create ADR for world evolution rules, extend ADR-0005 for topic matching
3. **Engine feasibility spike** — verify all ADR "Verification Required" items against `kaiju/src/`

### Blocking Issues

None. The 2 gaps are in VS/Alpha tier systems (Persistent World State). No Foundation or Core layer gaps.
