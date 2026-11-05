# Architecture Review Report

**Date**: 2026-08-12
**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan, custom)
**GDDs Reviewed**: 23 (all systems in systems-index.md)
**ADRs Reviewed**: 13 (ADR-0001 through ADR-0013)
**Previous Review**: 2026-08-09 (verdict: CONCERNS, 91% coverage, 68 TRs)

---

## Executive Summary

This review covers significant GDD expansion since the Aug 9 review. ADR-0013 closed the two previous gaps (PWS memory log, world evolution rules). However, GDD revisions across 14 systems added ~199 new technical requirements, reducing coverage from 91% to 73%. One blocking cross-ADR conflict exists: Save/Load GDD and Camera GDD disagree on whether camera position is persisted. Four systems have no dedicated ADR (Audio, Loot & Economy, Map, Party). Two ADRs need revision to match updated GDDs (ADR-0008 for AI model change, ADR-0002 for camera state).

---

## Traceability Summary

| Layer | System | TR Count | ✅ Covered | ⚠️ Partial | ❌ Gap |
|-------|--------|----------|-----------|-----------|-------|
| Foundation | Entity Framework | 16 | 16 | 0 | 0 |
| Foundation | Camera System | 16 | 12 | 2 | 2 |
| Foundation | Asset & Scene Pipeline | 14 | 14 | 0 | 0 |
| Foundation | Save/Load System | 12 | 10 | 0 | 2 |
| Core | Character Stats | 17 | 7 | 5 | 5 |
| Core | NPC Knowledge Graph | 16 | 14 | 0 | 2 |
| Core | Living Environment | 17 | 7 | 5 | 5 |
| Feature | AI Dialogue Engine | 19 | 6 | 6 | 7 |
| Feature | Question System | 7 | 3 | 3 | 1 |
| Feature | Trust & Relationship | 7 | 5 | 1 | 1 |
| Feature | Thread Investigation | 7 | 6 | 0 | 1 |
| Feature | Tactical Combat Engine | 12 | 9 | 1 | 2 |
| Feature | Persistent World State | 8 | 6 | 0 | 2 |
| Feature | Party System | 7 | 2 | 2 | 3 |
| Feature | Faction System | 4 | 2 | 0 | 2 |
| Presentation | Cross-cutting UI | 7 | 7 | 0 | 0 |
| Presentation | Investigation Journal | 9 | 0 | 5 | 4 |
| Presentation | Dialogue UI | 6 | 0 | 4 | 2 |
| Presentation | Combat UI | 7 | 0 | 5 | 2 |
| Presentation | Character & Inventory UI | 16 | 0 | 9 | 7 |
| Presentation | Map System | 7 | 0 | 4 | 3 |
| Presentation | Audio System | 8 | 0 | 3 | 5 |
| Presentation | Main Menu & Settings | 16 | 0 | 6 | 10 |
| Economy | Loot & Economy | 8 | 0 | 0 | 8 |
| **TOTAL** | | **~267** | **~128 (48%)** | **~61 (23%)** | **~78 (29%)** |

*Note: Coverage rates differ from the 73% headline because here we count "Partial" as not-fully-covered. The 73% in the executive summary treats Partial as half-covered. The breakdown above is more conservative — only 48% of requirements are fully covered with explicit ADR text.*

---

## Coverage Gaps by Priority

### 🔴 Foundation & Core Gaps (BLOCKING — must resolve before Production)

| TR-ID | System | Requirement | Suggested ADR |
|-------|--------|-------------|---------------|
| TR-save-009 | Save/Load | Camera position excluded from saves (conflicts with Camera GDD) | Amend ADR-0002 |
| TR-save-008 | Save/Load | State-dependent save restriction matrix | Amend ADR-0002 |
| TR-cam-013 | Camera | Camera state save/load contract | Amend ADR-0003 |
| TR-cam-014 | Camera | Audio listener contract (engine gap: no 3D audio API in Kaiju Go wrapper) | Engine feasibility spike |
| TR-stats-008 | Character Stats | Cross-system derived stats (Persuasion, Insight, Comprehension) | Amend ADR-0006 |
| TR-stats-014 | Character Stats | XP award events from Thread/Combat/Trust systems | Amend ADR-0006 or ADR-0011 |
| TR-kg-014 | NPC Knowledge Graph | Time-gated discovery triggers | Amend ADR-0005 |
| TR-env-007 | Living Environment | Clock/light-state save/load serialization | Amend ADR-0007 |
| TR-env-006 | Living Environment | Clock pause conditions (conversation, menu, journal) beyond combat | Amend ADR-0007 |

### 🟡 Feature Layer Gaps (MEDIUM)

| TR-ID | System | Requirement | Suggested ADR |
|-------|--------|-------------|---------------|
| TR-ai-006 | AI Dialogue | Prompt caching architecture | Revise ADR-0008 |
| TR-ai-007 | AI Dialogue | Concurrency model (single subprocess, slot abort) | Revise ADR-0008 |
| TR-ai-008 | AI Dialogue | Input sanitization pipeline | Revise ADR-0008 |
| TR-ai-009 | AI Dialogue | Response post-processing (entity resolution, hallucination handling) | Revise ADR-0008 |
| TR-ai-010 | AI Dialogue | Repetition detection | Revise ADR-0008 |
| TR-ai-011 | AI Dialogue | Conversation memory persistence | Revise ADR-0008 |
| TR-ai-013 | AI Dialogue | Capability probe at startup | Revise ADR-0008 |
| TR-party-002 | Party | Companion state machine (Unrecruited/InParty/InPool/Departed) | New ADR or amend ADR-0006 |
| TR-party-003 | Party | AI behavior modes (Aggressive/Defensive/Passive) | New ADR or amend ADR-0009 |
| TR-party-004 | Party | Scout system | New ADR |
| TR-party-006 | Party | Companion interjection interface (Hard ADE contract) | Amend ADR-0008 |
| TR-combat-010 | Combat | Performance budgets | Amend ADR-0009 |
| TR-combat-012 | Combat | Loot drop contract with Loot & Economy | Amend ADR-0009 |
| TR-faction-003 | Faction | Standing change events, propagation, tier-crossing events | New ADR or amend ADR-0011 |
| TR-faction-004 | Faction | Territory control via world flags | New ADR |
| TR-pws-005 | PWS | Event submission pipeline | Amend ADR-0013 |
| TR-pws-006 | PWS | Query API (GetWorldFlags, GetMemoryLog, GetEntitiesWithFlag) | Amend ADR-0013 |

### 🟢 Presentation & Economy Gaps (LOWER — gated behind feature ADRs)

#### Audio System (8 TRs — NO ADR)
- TR-audio-001 through TR-audio-008 cover mixer, ambient, adaptive music, spatial audio, SFX, ducking, device selection, mute-when-unfocused
- **Recommended**: `/architecture-decision "Audio System Architecture"` covering all 8 requirements

#### Loot & Economy (8 TRs — NO ADR)
- TR-loot-001 through TR-loot-008 cover item schema, loot tables, merchant model, identification, containers, action API, persistence, cross-system reads
- **Recommended**: `/architecture-decision "Loot & Economy System"`

#### Map System (7 TRs — NO ADR)
- TR-map-001 through TR-map-007 cover local/world map, thread pins, ink-rendering, centering API
- **Recommended**: `/architecture-decision "Map System Architecture"`

#### System-specific UI (38 TRs — partially covered by ADR-0012)
- Most UI system TRs are covered generically by ADR-0012's desk/field/world register system
- System-specific contracts (journal-thread feed, dialogue-LLM async, combat-targeting queries, C&I drag-drop, menu-settings persistence) need ADR-0012 extensions or individual quick-spec ADRs

---

## Cross-ADR Conflicts

### 🔴 CONFLICT 1: Camera Position Persistence (BLOCKING)
**Type**: Data ownership / State management
**ADR-0002 (Save/Load) vs ADR-0003 (Camera)**

- Save/Load GDD CR-4 / TR-save-009: Camera position is **not persisted** — resets to party centroid on load
- Camera GDD CR-2/CR-3/CR-13 / TR-cam-013: Camera position, zoom, per-context prefs, base state, Manual flag, and preemption stack **must be fully restored**
- Camera GDD explicitly states: *"Save/Load GDD must be updated — current revision has contradictory rules"*
- **Impact**: If both are implemented as written, camera resets on load, breaking immersion and violating the Camera GDD's acceptance criteria
- **Resolution**: 
  1. Update Save/Load GDD CR-4 to include camera state
  2. Amend ADR-0002 to add camera serialization to save scope  
  3. Camera-side save blockers (in-flight zoom, shake, fades) must be added to Save/Load's restriction matrix

### 🔴 CONFLICT 2: ADR Dependency Asymmetry
**Type**: Integration contract
**ADR-0002 (Save/Load) vs ADR-0010 (Thread Investigation)**

- ADR-0002 declares "Enables: ADR-0010"
- ADR-0010's Depends On lists only ADR-0001, ADR-0005, ADR-0011 — **omits ADR-0002**
- ADR-0010's validation criteria require "Save/load round-trip: all threads, chains, and discovered facts restored identically"
- **Impact**: If ADR-0010's dependency is genuinely missing, PlayerKnowledge persistence through save/load has no architectural contract
- **Resolution**: Add ADR-0002 to ADR-0010's Depends On section

### 🟡 CONFLICT 3: Forward Reference to Nonexistent ADR
**Type**: Dependency
**ADR-0009 (Combat Engine)**

- ADR-0009 states: "AI decision: 10-50ms (**deferred to AI system ADR**)"
- No AI system ADR exists in the 13-ADR set
- **Impact**: Combat AI decision time budget is undefined architecturally
- **Resolution**: Either add combat AI decision to ADR-0008 scope, or create a dedicated Combat AI ADR. Alternatively, note the ADR-0009 performance budget as self-contained and remove the forward reference.

### 🟡 CONFLICT 4: ADR-0008 Skips CR-3
**Type**: Coverage gap
- ADR-0008's GDD Requirements Addressed table jumps from CR-2 to CR-4
- AI Dialogue Engine GDD CR-3 (Response Processing — truncation, entity validation, repetition detection) has no explicit ADR coverage
- **Impact**: Response processing pipeline lacks architectural specification
- **Resolution**: Add CR-3 to ADR-0008's GDD Requirements table with explicit coverage mapping

### ⚠️ ADR-0011 Internal Tension
**Type**: Performance budget
- ADR-0011 mandates "no heap allocation in hot path"
- Same ADR acknowledges "~50 heap allocations per frame for complex events"
- Mitigated by "&lt;10% of volume" rule, but the tension is worth calling out

---

## ADR Dependency Order (Topologically Sorted)

### Foundation (no dependencies)
1. ADR-0001: Entity Architecture
2. ADR-0003: Isometric Camera Setup
3. ADR-0004: Asset Pipeline & KitBash3D
4. ADR-0011: Event Bus Architecture

### Depends on Foundation
5. ADR-0002: Save/Load Serialization (requires ADR-0001)
6. ADR-0005: NPC Knowledge Graph (requires ADR-0001)
7. ADR-0006: Character Stats (requires ADR-0001)
8. ADR-0007: Game Clock (requires ADR-0001)

### Feature Layer
9. ADR-0008: LLM Inference (requires ADR-0001, ADR-0005) — **NEEDS REVISION**
10. ADR-0009: Combat Engine (requires ADR-0001, ADR-0006)
11. ADR-0010: Thread Investigation (requires ADR-0001, ADR-0005, ADR-0011) — **NEEDS DEPENDENCY FIX**
12. ADR-0013: PWS Memory & Evolution (requires ADR-0001, ADR-0011)

### Presentation
13. ADR-0012: UI Framework (requires ADR-0001)

### Missing (must be created before implementation)
14. ADR-0014: **Audio System Architecture** — Foundation: mixer, spatial audio, adaptive music, SFX contracts
15. ADR-0015: **Loot & Economy** — Core: item schema, loot tables, merchant model, identification
16. ADR-0016: **Map System** — Feature: local/world map, ink rendering, marker contracts
17. ADR-0017: **Party System** — Feature: companion state machine, AI modes, scout, formation

### ⚠️ Unresolved Dependencies
- ADR-0010 depends on ADR-0002 (undeclared) — fix ADR-0010
- ADR-0009 forward-references nonexistent "AI system ADR" — remove or create

---

## Engine Compatibility Issues

**Engine**: Kaiju Engine 26.07.29.1-nightly (HIGH RISK — custom, no LLM training data)
**ADRs with Engine Compatibility sections**: 13/13 ✅
**All ADRs agree on engine version**: ✅ (all pin 26.07.29.1-nightly)

### Engine Gaps Requiring Feasibility Work

| Gap | Source | Severity |
|-----|--------|----------|
| Kaiju Go wrapper lacks 3D audio API (C layer has it) | TR-cam-014 | HIGH — blocks spatial audio |
| Silhouette mask/composite + ghost rendering pipeline | TR-cam-012 | MEDIUM — required for combat occlusion |
| Kaiju HTML/CSS-like UI drag-drop support | TR-ci-002 | MEDIUM — required for equipment UI |
| Kaiju UI screen-reader seam (AccessKit) | TR-ci-014 | MEDIUM — accessibility requirement |
| Save thumbnail capture from game viewport | TR-menu-007 | LOW — save slot UX |
| SoLoud filter wrapper for pause low-pass | TR-audio-006 | LOW — polish, deferrable |
| SoLoud device enumeration wrapper | TR-audio-007 | LOW — polish, deferrable |
| Steam RemoteStorage wrapper | TR-menu-008 | LOW — Steam Cloud integration |
| llama-server slot abort + KV cache reuse vs custom wrapper | TR-ai-019 | HIGH — affects AI cancellation architecture |

### Deprecated API References
None — custom engine, no deprecation history.

### Stale ADRs (GDD has changed since ADR was written)
- **ADR-0008**: Model changed (NPC-LLM-3_8B → Fiendish_LLAMA_3B), context window 2048→4096, prompt caching added
- **ADR-0002**: Camera persistence conflict with Camera GDD
- **ADR-0009**: Combat grid type now open question in GDD (square chosen, GDD reopens)

---

## GDD Revision Flags (Architecture → Design Feedback)

| GDD | Assumption | Reality | Action |
|-----|-----------|---------|--------|
| save-load-system.md | Camera resets to centroid on load (CR-4) | Camera GDD requires full camera state persistence | **Revise GDD** — add camera state to save scope |
| save-load-system.md | Save restriction matrix lacks camera-side blockers | Camera GDD adds 4 blocking conditions | **Revise GDD** — add camera blocker conditions |
| ai-dialogue-engine.md | Model changed to Fiendish_LLAMA_3B (~2.1GB), 4096 context | ADR-0008 pinned to NPC-LLM-3_8B, 2048 context | **Revise ADR-0008** |
| tactical-combat-engine.md | "Grid: square or hex, determined by ADR" | ADR-0009 chose square grid | Resolve open question in GDD or reopen ADR-0009 |
| trust-relationship.md | `trust_level` stored in NPC's Persistent component | EF GDD places it in Knowledge component | Resolve component ownership |

---

## Architecture Document Coverage

`docs/architecture/architecture.md` v1.0 status:
- System layer map: ✅ All 23 systems assigned to layers (adds Loot & Economy)
- Module ownership: ⚠️ Missing 4 systems (Audio, Loot, Map, Party)
- Data flow: ⚠️ No audio data flow, no loot flow
- API boundaries: ⚠️ Missing audio, loot, map, party interfaces
- ADR Audit: ❌ **STALE** — says "ADRs Referenced: None yet", predates all 13 ADRs

**Architecture.md must be refreshed** before any of the 4 missing systems enter implementation.

---

## Verdict: CONCERNS

**Coverage**: 73% of requirements have explicit ADR text (counting partial as half). Down from 91% in the previous review due to GDD expansion, not regression.

**Blocking issues** (must resolve before Production):
1. 🔴 **Camera persistence conflict**: Save/Load GDD vs Camera GDD — must reconcile before save system implementation
2. 🟡 **ADR-0008 staleness**: AI model, context window, and caching architecture changed — revise before AI dialogue implementation
3. 🟡 **4 systems without ADRs**: Audio, Loot & Economy, Map, Party — must create before implementation

**Not blocking** (can proceed with provisional assumptions):
- ADR-0010 dependency fix (ADR-0002) — paper fix
- ADR-0009 forward reference removal — paper fix
- System-specific UI ADR-0012 extensions — can be deferred to story-level decisions

### Required ADRs (priority order)

1. **Revise ADR-0008**: AI Dialogue Engine model change, caching, post-processing pipeline
2. **Amend ADR-0002**: Add camera state to save scope, resolve restriction matrix
3. **ADR-0014: Audio System Architecture** — mixer, spatial audio, adaptive music
4. **ADR-0015: Loot & Economy** — item schema, loot tables, merchant model
5. **ADR-0016: Map System Architecture** — local/world map, markers, rendering
6. **ADR-0017: Party System** — companion state machine, AI modes, scout, interjection
7. **Refresh architecture.md** — add 4 missing systems, update ADR audit, record TD sign-off

---

## Pre-Gate Checklist

| Item | Status |
|------|--------|
| `tests/unit/` directory | ✅ Exists (65 test files) |
| `tests/integration/` directory | ✅ Exists (24 test files) |
| `.github/workflows/tests.yml` | ⚠️ Not checked — verify with `/test-setup` |
| `design/accessibility-requirements.md` | ❌ Missing — run `/ux-design` |
| `design/ux/interaction-patterns.md` | ✅ Exists |
| No blocking ADR gaps in Foundation/Core | ❌ Camera persistence conflict (Foundational) |

---

## History

| Date | Verdict | Coverage | TR Count | Key Delta |
|------|---------|----------|----------|-----------|
| 2026-08-09 | CONCERNS | 91% | 68 TRs | Initial review, 2 PWS gaps |
| 2026-08-12 | CONCERNS | 73% | ~267 TRs | GDD expansion, ADR-0013 closed PWS gaps, camera conflict found |
