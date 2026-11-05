# Architecture Review Report

**Date**: 2026-08-12 (Review #2)
**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan, custom)
**GDDs Reviewed**: 23 (all systems)
**ADRs Reviewed**: 15 (ADR-0001 through ADR-0015)
**Previous Review**: 2026-08-12 Review #1 (verdict: CONCERNS, 48% full coverage, 13 ADRs)

---

## Executive Summary

This is the second review on 2026-08-12, reflecting two new ADRs and two amendments since the morning review. ADR-0014 (Audio) and ADR-0015 (Loot & Economy) were created — both Proposed, converting 16 TRs from ❌ Gap to ⚠️ Partial. ADR-0002 was amended to resolve the blocking Save/Load vs. Camera conflict. ADR-0008 was revised to cover prompt caching, post-processing, conversation memory, and capability probe — but the TR registry has not yet been updated to reflect this coverage.

Coverage stands at **84.4%** (222/263 TRs covered by Accepted ADRs), with 40 gaps and 1 stale conflict marker. The remaining gaps are concentrated in: AI Dialogue Engine (8 TRs — ADR-0008 revision pending registry update), Audio (8 TRs — ADR-0014 Proposed), Loot & Economy (8 TRs — ADR-0015 Proposed), Party System (6 TRs — no dedicated ADR), Faction System (3 TRs — no dedicated ADR), Map System (2 TRs — no dedicated ADR), Persistent World State (2 TRs — ADR-0013 gap), Camera (1 TR — engine feasibility spike), Character Stats (1 TR — ADR-0006 extension needed), Combat (1 TR — loot contract).

---

## Traceability Summary

| Layer | System | TR Count | ✅ Covered | ❌ Gap |
|-------|--------|----------|-----------|-------|
| Foundation | Entity Framework | 16 | 16 | 0 |
| Foundation | Camera System | 16 | 15 | 1 |
| Foundation | Asset & Scene Pipeline | 14 | 14 | 0 |
| Foundation | Save/Load System | 12 | 11 | 0 (1 stale conflict) |
| Core | Character Stats | 17 | 16 | 1 |
| Core | NPC Knowledge Graph | 16 | 16 | 0 |
| Core | Living Environment | 17 | 17 | 0 |
| Feature | AI Dialogue Engine | 19 | 11 | 8 |
| Feature | Question System | 7 | 7 | 0 |
| Feature | Trust & Relationship | 7 | 7 | 0 |
| Feature | Thread Investigation | 7 | 7 | 0 |
| Feature | Tactical Combat Engine | 12 | 11 | 1 |
| Feature | Persistent World State | 8 | 6 | 2 |
| Feature | Party System | 7 | 1 | 6 |
| Feature | Faction System | 4 | 1 | 3 |
| Presentation | Cross-cutting UI | 7 | 7 | 0 |
| Presentation | Investigation Journal | 9 | 9 | 0 |
| Presentation | Dialogue UI | 6 | 6 | 0 |
| Presentation | Combat UI | 7 | 7 | 0 |
| Presentation | Character & Inventory UI | 16 | 16 | 0 |
| Presentation | Map System | 7 | 5 | 2 |
| Presentation | Audio System | 8 | 0 | 8 |
| Presentation | Main Menu & Settings | 16 | 16 | 0 |
| Economy | Loot & Economy | 8 | 0 | 8 |
| **TOTAL** | | **263** | **222 (84.4%)** | **40 (15.2%)** |

> **Note on ADR-0008 gaps**: ADR-0008's header claims revision coverage for TR-ai-006 through TR-ai-011 and TR-ai-013 (prompt caching, concurrency, sanitization, post-processing, repetition detection, conversation memory, capability probe). The TR registry hasn't been updated. If the ADR body substantively covers these, coverage rises to ~90.5% (238/263).

> **Note on ADR-0014/ADR-0015**: Both are **Proposed** (not yet Accepted). Audio and Loot requirements have architectural decisions drafted but not formally accepted. Until Accepted, these 16 TRs remain at risk — stories referencing Proposed ADRs are auto-blocked per project standards.

> **Note on TR-save-009**: Registry still shows "ADR-0002 — CONFLICT with ADR-0003 (camera)" but ADR-0002 was amended 2026-08-12 to include camera state in save scope. This conflict is **resolved** — registry is stale.

---

## Coverage Gaps by Priority

### 🔴 Foundation & Core Layer Gaps (BLOCKING — must resolve before Production)

| TR-ID | System | Requirement | Suggested Action |
|-------|--------|-------------|-----------------|
| TR-cam-014 | Camera | Audio listener: 3D spatial audio via Kaiju C layer (Go wrapper missing) | Engine feasibility spike — verify `soloud_c.h` 3D API wrapping effort |
| TR-stats-008 | Character Stats | Cross-system derived stats: Persuasion/Insight/Comprehension from Thread/NPC interaction counters | Amend ADR-0006 |
| TR-pws-002 | Persistent World State | Memory log: append-only structured event log per entity, capped at 50 entries | Amend ADR-0013 |
| TR-pws-003 | Persistent World State | World evolution rules: condition→event evaluated on region load, authored per region | Amend ADR-0013 |

### 🟡 Feature Layer Gaps (MEDIUM — must resolve before feature implementation)

| TR-ID | System | Requirement | Suggested Action |
|-------|--------|-------------|-----------------|
| TR-ai-006 | AI Dialogue | Prompt caching: multi-slot LRU, ~448 MiB KV cache per slot | Revise ADR-0008 (claimed in header, verify body) |
| TR-ai-007 | AI Dialogue | Concurrency: single shared subprocess, slot abort, queue depth 1 | Revise ADR-0008 (claimed in header, verify body) |
| TR-ai-008 | AI Dialogue | Input sanitization: strip chat tokens, injection detection, entity whitelist | Revise ADR-0008 (claimed in header, verify body) |
| TR-ai-009 | AI Dialogue | Response post-processing: streaming, entity resolution, hallucination handling | Revise ADR-0008 (claimed in header, verify body) |
| TR-ai-010 | AI Dialogue | Repetition detection: Jaccard >85% → re-query with penalty | Revise ADR-0008 (claimed in header, verify body) |
| TR-ai-011 | AI Dialogue | Conversation memory: rolling summary + disclosed-fact IDs via PWS | Revise ADR-0008 (claimed in header, verify body) |
| TR-ai-013 | AI Dialogue | Capability probe at startup: ~150-token benchmark | Revise ADR-0008 (claimed in header, verify body) |
| TR-ai-019 | AI Dialogue | Inference architecture: llama-server vs custom thin wrapper | New ADR or resolve in ADR-0008 |
| TR-combat-012 | Combat | Loot drop contract with Loot & Economy | Covered by ADR-0015 once Accepted |
| TR-party-002 | Party | Companion state machine (Unrecruited/InParty/InPool/Departed) | New Party ADR |
| TR-party-003 | Party | Companion AI behavior modes (Aggressive/Defensive/Passive) | New Party ADR or amend ADR-0009 |
| TR-party-004 | Party | Scout system: detection range, opposed check, staged alert | New Party ADR |
| TR-party-005 | Party | Formation: diamond, trailing arc, front/back rank config | New Party ADR |
| TR-party-006 | Party | Companion knowledge contribution + interjection interface | New Party ADR + amend ADR-0008 |
| TR-party-007 | Party | Rest-point gating for party management | New Party ADR |
| TR-faction-002 | Faction | Faction data model: Faction component + YAML definitions | New Faction ADR |
| TR-faction-003 | Faction | Standing change events, propagation, tier-crossing events | New Faction ADR or amend ADR-0011 |
| TR-faction-004 | Faction | Territory control via world.faction.* flags | New Faction ADR |

### 🟢 Presentation & Economy Layer Gaps (LOWER — gated behind feature ADRs)

| TR-ID | System | Requirement | Suggested Action |
|-------|--------|-------------|-----------------|
| TR-audio-001 through -008 | Audio | All 8 audio requirements (mixer, ambient, music, spatial, SFX, ducking, device, mute) | Accept ADR-0014 |
| TR-loot-001 through -008 | Loot & Economy | All 8 loot requirements (item schema, loot tables, merchants, identification, containers, action API, persistence, cross-system reads) | Accept ADR-0015 |
| TR-map-001 | Map | Local map data: discovered locations, NPC last-seen, cleared markers | New Map ADR |
| TR-map-002 | Map | World map (Alpha): region state summaries, travel routes, faction overlays | New Map ADR |

---

## Cross-ADR Conflicts

### Resolved Since Previous Review

| Conflict | Resolution |
|----------|-----------|
| **ADR-0002 vs ADR-0003 — Camera state persistence** | ADR-0002 amended 2026-08-12: camera position, zoom, per-context prefs, and preemption stack now included in save scope. TR-save-009 registry entry still has stale CONFLICT marker — update needed. |

### Remaining Conflicts

**None detected.** All 15 ADRs are internally consistent. Previous review's flagged concerns are resolved or tracked as known gaps.

### Known Conflict from consistency-failures.md (still open)

| Date | Domain | Status |
|------|--------|--------|
| 2026-08-09 | AI Dialogue / Party — interjection interface | Party System GDD defines interjection contract; AI Dialogue Engine GDD is single-speaker. Tracked as design gap T0-2. |
| 2026-08-09 | Combat / Party — knockout state | Party System specifies VS knockout; TCE has no knockout state. Tracked as design gap T0-3. |
| 2026-08-12 | Dependency — ADR-0002 vs ADR-0010 | ADR-0002 declares "Enables: ADR-0010" but ADR-0010 omits ADR-0002 from Depends On. Fix: add ADR-0002 to ADR-0010's Depends On. |
| 2026-08-12 | AI Decision Budget — ADR-0009 | ADR-0009 defers "AI decision: 10-50ms" to nonexistent "AI system ADR". Fix: add to ADR-0008 or resolve in ADR-0009. |

---

## ADR Dependency Ordering

### Recommended ADR Implementation Order (topologically sorted)

**Foundation (no dependencies):**
1. ADR-0001: Entity Architecture ✅ Accepted
2. ADR-0003: Isometric Camera Setup ✅ Accepted
3. ADR-0004: Asset Pipeline & KitBash3D ✅ Accepted

**Depends on Foundation:**
4. ADR-0002: Save/Load Serialization ✅ Accepted (requires ADR-0001, ADR-0003)
5. ADR-0011: Event Bus Architecture ✅ Accepted (requires ADR-0001)

**Core Layer:**
6. ADR-0005: NPC Knowledge Graph ✅ Accepted (requires ADR-0001)
7. ADR-0006: Character Stats ✅ Accepted (requires ADR-0001)
8. ADR-0007: Game Clock & Environment ✅ Accepted (requires ADR-0001)

**Feature Layer:**
9. ADR-0008: LLM Inference Integration ✅ Accepted (requires ADR-0001, ADR-0005)
10. ADR-0009: Combat Engine ✅ Accepted (requires ADR-0001, ADR-0006)
11. ADR-0010: Thread Investigation ✅ Accepted (requires ADR-0001, ADR-0005, ADR-0011)
12. ADR-0013: Persistent World State ✅ Accepted (requires ADR-0001, ADR-0011)

**Pending Acceptance:**
13. ADR-0014: Audio System ⚠️ Proposed (requires ADR-0001, ADR-0003, ADR-0007, ADR-0011)
14. ADR-0015: Loot & Economy ⚠️ Proposed (requires ADR-0001, ADR-0006, ADR-0009, ADR-0011)

**Not Yet Created:**
15. Party System ADR (requires ADR-0006, ADR-0011)
16. Faction System ADR (requires ADR-0013, ADR-0011)
17. Map System ADR (requires ADR-0007, ADR-0013)

### Unresolved Dependencies

- ⚠️ ADR-0014 (Audio, Proposed) depends on ADR-0003, ADR-0007, ADR-0011 — all Accepted. ADR-0014 itself must be Accepted before Audio implementation.
- ⚠️ ADR-0015 (Loot, Proposed) depends on ADR-0001, ADR-0006, ADR-0009, ADR-0011 — all Accepted. ADR-0015 itself must be Accepted before Loot implementation.
- ⚠️ ADR-0009 references a nonexistent "AI system ADR" for combat AI decision budgets — needs resolution.

### Dependency Cycles

None detected. The dependency graph remains acyclic.

---

## Engine Compatibility Issues

### Engine Audit Results

| Metric | Value |
|--------|-------|
| Engine | Kaiju Engine 26.07.29.1-nightly |
| ADRs with Engine Compatibility section | 15 / 15 (100%) |
| Deprecated API References | 0 |
| Stale Version References | 0 |

### Post-Cutoff API Summary

| ADR | Post-Cutoff APIs Used | Verification Status |
|-----|----------------------|---------------------|
| ADR-0001 | `engine.Entity`, `matrix.Transform`, `pod.Register`, `events.Event` | Verified against kaiju/src/ |
| ADR-0002 | `pod.Register`, `EntityData` interface | Verified |
| ADR-0003 | `cameras.NewStandardCameraOrthographic()` | Verified |
| ADR-0004 | Kaiju native mesh/texture formats | Verified |
| ADR-0008 | llama.cpp GGUF inference (external) | Verified |
| ADR-0011 | `events.Event` (Kaiju native) | Verified |
| ADR-0012 | Kaiju HTML/CSS-like UI system | Verified |
| ADR-0014 | Kaiju SoLoud wrapper, C-layer SoLoud API | Verified — 3D spatial API declared in C, unwrapped in Go |

### Engine Gap Flags

| Gap | Location | Severity | Impact |
|-----|----------|----------|--------|
| 3D spatial audio Go wrapper | ADR-0014, TR-cam-014 | MEDIUM | Camera cannot act as spatial audio listener without Go-level API. C-layer `soloud_c.h` has `Soloud_play3d`, `Soloud_set3dListenerParameters` — need CGo wrappers. |
| Kaiju UI accessibility seam | TR-ci-014 | LOW | Screen reader announcements require accessibility hook in Kaiju UI — deferred engine work. |
| Kaiju UI drag-and-drop | TR-ci-002 | LOW | Equipment drag-drop needs Kaiju UI support — engine gap. |
| Save thumbnail capture | TR-menu-007 | LOW | 320×180 PNG from viewport requires engine screenshot/framebuffer capability. |
| Steam RemoteStorage wrapper | TR-menu-008 | LOW | Deferred engine work for Steam Cloud integration. |
| VSync reconfig at runtime | TR-menu-013 | LOW | Deferred engine capability — greyed with "Coming Soon" at MVP. |

---

## GDD Revision Flags (Architecture → Design Feedback)

### Flag 1: TR-save-009 Registry Staleness
- **GDD**: save-load-system.md CR-4
- **Assumption**: Camera position resets to centroid on load
- **Reality**: ADR-0002 amended 2026-08-12 — camera state IS persisted
- **Action**: Update Save/Load GDD CR-4 to reflect amendment. Update TR-save-009 registry entry to remove CONFLICT marker.

### Flag 2: AI Dialogue Engine GDD Revisions Not Reflected in Registry
- **GDD**: ai-dialogue-engine.md (major revision)
- **Assumption**: 7 new technical requirements (TR-ai-006 through TR-ai-013, TR-ai-019) are gaps
- **Reality**: ADR-0008 was revised 2026-08-12 claiming coverage of prompt caching, post-processing, conversation memory, capability probe, etc.
- **Action**: Verify ADR-0008 body substantively covers these TRs. If yes, update TR registry. If partial coverage, note which TRs remain gaps.

No other GDD assumptions conflict with verified engine behaviour.

---

## Architecture Document Coverage

`docs/architecture/architecture.md` (v1.0, 2026-08-09) maps all 23 systems across 5 layers. Coverage assessment:

| Check | Status |
|-------|--------|
| All systems from systems-index.md appear in architecture layers | ✅ 23/23 |
| Data flow covers all cross-system communication | ✅ |
| API boundaries support integration requirements | ✅ |
| Orphaned architecture (systems with no GDD) | None |
| ADR audit section | ⚠️ Outdated — lists "None" for existing ADRs, recommends 12 ADRs (15 now exist) |

**Action**: `architecture.md` ADR audit section needs refresh. It lists 12 "Required New ADRs" — all 12 have been created plus 3 more (ADR-0013, ADR-0014, ADR-0015).

---

## Verdict: CONCERNS

**Coverage**: 84.4% (222/263) — improved from 48% in previous review due to reclassification of Presentation-layer TRs as covered by ADR-0012.

**Rationale**: No blocking cross-ADR conflicts (the camera save conflict is resolved). Foundation and Core layers are solid (95%+ coverage). The CONCERNS verdict is driven by:

1. **16 TRs gated behind Proposed ADRs** (ADR-0014 Audio, ADR-0015 Loot) — stories referencing these are auto-blocked
2. **6 Party System TRs without any ADR** — Party is a VS-tier system with no architectural coverage
3. **3 Faction System TRs without any ADR** — Faction is Alpha-tier but its data model and standing mechanics need architecture before Party (which depends on faction standing)
4. **8 AI Dialogue TRs** — registry shows gaps but ADR-0008 claims coverage; needs verification
5. **TR registry staleness** — TR-save-009 conflict marker and AI TR gaps may be out of date

### Blocking Issues (must resolve before PASS)

None are technically blocking for Pre-Production (Foundation/Core ADRs are all Accepted). However, the following block Production entry:

1. **Accept ADR-0014 (Audio)** — required before Audio System stories can be implemented
2. **Accept ADR-0015 (Loot & Economy)** — required before Loot stories; Combat loot contract (TR-combat-012) depends on this
3. **Create Party System ADR** — 6 uncovered TRs, VS-tier system
4. **Verify and update TR registry** for ADR-0008 revision coverage of AI TRs

### Required ADRs (priority order)

| Priority | ADR | Covers | Effort |
|----------|-----|--------|--------|
| 1 | Accept ADR-0014 | Audio System (8 TRs) | Review existing Proposed ADR |
| 2 | Accept ADR-0015 | Loot & Economy (8 TRs) | Review existing Proposed ADR |
| 3 | Party System ADR | Party System (6 TRs: TR-party-002 through -007) | New ADR |
| 4 | Faction System ADR | Faction System (3 TRs: TR-faction-002 through -004) | New ADR |
| 5 | Map System ADR | Map System (2 TRs: TR-map-001, TR-map-002) | New ADR |
| 6 | Amend ADR-0013 | PWS memory log + evolution rules (2 TRs) | ADR revision |
| 7 | Amend ADR-0006 | Cross-system derived stats (TR-stats-008) | ADR revision |
| 8 | Engine feasibility spike | Camera 3D audio listener (TR-cam-014) | Technical spike |

---

## Pre-Gate Checklist

| Item | Status |
|------|--------|
| `tests/unit/` and `tests/integration/` directories | ✅ Present |
| `.github/workflows/tests.yml` | ❌ Not found — run `/test-setup` |
| `design/accessibility-requirements.md` | ✅ Present |
| `design/ux/interaction-patterns.md` | ❌ Not found — run `/ux-design` |

---

## Comparison with Previous Review (2026-08-12 Review #1)

| Metric | Review #1 | Review #2 | Delta |
|--------|-----------|-----------|-------|
| ADRs reviewed | 13 | 15 | +2 (ADR-0014, ADR-0015) |
| Total TRs | ~267 | 263 | −4 (consolidation) |
| Covered | ~128 (48%) | 222 (84.4%) | +94 (+36.4pp) |
| Gaps | ~78 (29%) | 40 (15.2%) | −38 (−13.8pp) |
| Blocking conflicts | 1 (Camera save) | 0 | −1 |
| Proposed ADRs | 0 | 2 | +2 |

> The large coverage increase is primarily due to reclassification: Review #1 counted Presentation-layer TRs as "not covered" because ADR-0012 is a framework ADR that enables all Presentation systems but doesn't enumerate each TR. Review #2 counts TRs covered when the relevant ADR(s) exist and are Accepted, which better reflects the actual architectural guidance available to implementers.

---

*Report generated by /architecture-review (full mode)*
*Next review: after ADR-0014 and ADR-0015 are Accepted, or after Party/Faction/Map ADRs are created*
