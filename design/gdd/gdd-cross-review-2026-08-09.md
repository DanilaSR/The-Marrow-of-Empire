# Cross-GDD Review Report
**Date**: 2026-08-09
**GDDs Reviewed**: 22 (all systems)
**Reviewers**: systems-designer (Phase 2: Consistency), creative-director (Phase 3: Design Holism)
**Verdict**: 🔴 FAIL — 8 blocking issues must be resolved before architecture begins

---

## Consistency Issues

### 🔴 Blocking (5)

**B1. Quick-save allowed during combat in save-load, blocked in TCE.**
- GDDs: `save-load-system.md` vs `tactical-combat-engine.md`
- save-load CR-5: Combat row allows manual + quick-save. TCE: "Manual save only; no quick-save or auto-save during combat."
- Fix: TCE is authoritative. Change save-load CR-5 Combat row to Quick-save: Blocked.

**B2. Party interjection contract imposed on ADE, but ADE has no such dependency.**
- GDDs: `party-system.md` vs `ai-dialogue-engine.md`
- Party System CR-6 mandates a `companion_interject` interface as a Hard dependency. ADE has zero knowledge of interjections, companions, or multi-party conversation.
- Fix: ADE must add Party System dependency and define the interjection interface, or party-system downgrades to provisional.

**B3. LLM model identity mismatch across three documents.**
- GDDs: `ai-dialogue-engine.md` vs `game-concept.md` vs `entity-framework.md` vs `systems-index.md`
- ADE: Fiendish_LLAMA_3B / Q4_K_M / ~2.1GB. Game-concept & systems-index: ~3.8B / Q5_K_M / ~2.7GB. The memory budget in technical-preferences depends on which is correct.
- Fix: Pick one model spec and update all four documents.

**B4. ADE first-token latency AC internally unpassable.**
- GDD: `ai-dialogue-engine.md` (internal)
- AC-21 requires ≤500ms first-word latency. ADE's own latency math gives ~1.4s best case. AC cannot pass by the document's own numbers.
- Fix: Re-bound AC-21 to achievable value (≤2s cached / ≤6s cold), or split into cached vs cold ACs.

**B5. ADE timeout and throughput contradictions.**
- GDD: `ai-dialogue-engine.md` (internal)
- Timeout: 25s (knob), 15s (AC-17), >15s (Visual/Audio). Throughput: 10 tok/s (CR-5) vs 8 tok/s minimum (AC-16).
- Fix: Single timeout constant and single minimum-tok/s constant in Tuning Knobs, referenced by all ACs.

### ⚠️ Warning (19)

**W1. Dialogue UI stale against ADE typewriter spec.** dialogue-ui: 20 cps character-by-character. ADE CR-4: 45-60 cps word-chunk reveal. Fix: update dialogue-ui to ADE CR-4.

**W2. Faction price formula contradicts Hostile prose.** Hostile: "+100% or refuse service" vs formula `1.0 − standing/400` = 1.25×. Fix: pick one and make both agree.

**W3. Faction standing storage ownership split three ways.** faction-system: Faction component. PWS: `world.faction.*.standing`. EF component schema can only hold one faction_id per entity. Fix: single owner with multi-faction schema.

**W4. memory_log append ownership split.** EF: "framework appends." PWS: "PWS appends." Fix: EF append-only model; PWS submits entries.

**W5. "Suspended" entity semantics clash.** EF: distance-based. Living Environment: schedule-based, "not interactable." Fix: define trigger union/priority, single interaction rule.

**W6. Clock lacks day counter, but three systems need days.** LE serializes "minutes since midnight" — cannot distinguish day 2 from day 9. PWS needs `days_since()`, LE has 7-day wildlife respawn, trust has 72h freeze. Fix: add day counter to clock serialization.

**W7. Wildlife respawn contradicts EF serialization scope.** LE: wildlife entities "not persisted individually — they respawn." EF AC-7: serializes ALL Active + Suspended entities. Fix: entity-level persistence exclusion marker.

**W8. PWS broadcasts "EF event" not in EF catalog.** `on_world_state_changed` referenced as EF event but EF has no such event. Fix: add to EF catalog or reword as PWS-defined.

**W9. Camera references nonexistent PartyMember component.** Fix: Camera subscribes to `pc_controlled` tag or party-system membership event.

**W10. PlayerKnowledge schema drift between EF and Thread Investigation.** Fields differ. Fix: single schema; Thread Investigation is writer, EF is component owner.

**W11. Knowledge component field drift.** EF: `known_facts[]`. NPC KG: `facts[]`. Fix: unify field name.

**W12. Persuasion/Insight/Comprehension derived stats orphaned.** character-stats defines them; ADE removed its dependency; Thread Investigation never consumed them. Fix: re-home as journal/KG mechanics or delete. (Matches Phase 3 F18.)

**W13. 12 one-directional dependencies.** Including critical TCE→Party tier conflict: MVP-tier TCE hard-depends on VS-tier Party System. Fix: add reciprocal rows; resolve tier conflict.

**W14. 3 duplicate tuning knobs.** `max_trust_level`, `memory_log_max_entries`, `companion_death_enabled` each owned by two GDDs. Fix: single owner per knob.

**W15. Inventory write-ownership gap.** EF: "Inventory/Loot System — Not yet authored." character-inventory-ui writes to Inventory component. Fix: assign ownership or author Loot GDD.

**W16. Camera follow-speed bound references nonexistent movement GDD.** Fix: define exploration walk speed or convert to ratio.

**W17. Input Binding System referenced but GDD doesn't exist.** Main Menu CR-6 depends on `design/gdd/input-binding-system.md` — not in systems index. Fix: author or fold back into Main Menu.

**W18. Spatial audio listener contract unowned.** Fix: assign ownership (camera provides transform, audio owns listener).

**W19. Five GDDs violate the 8-section standard.** investigation-journal, dialogue-ui, combat-ui, map-system, audio-system missing Dependencies/Tuning Knobs/other sections. Fix: fill missing sections per design/CLAUDE.md.

---

## Game Design Issues

### 🔴 Blocking (3)

**F10. No economy exists — gold, loot, and equipment acquisition are unowned.**
- TCE references "enemy drops loot if applicable" — no Loot GDD exists among the 22 indexed systems. Equipment is the primary combat power curve (weapon base 2-20 dwarfs attribute mods −4..+4). The game's combat progression depends entirely on a system that does not exist. Gold display caps at 999,999 with no gameplay limit — unbounded accumulation with no sink.
- Fix: Author a Loot & Economy GDD covering gold sources/sinks, loot tables, equipment acquisition curve, shops.

**F13. Enemy scaling is undefined — the difficulty curve does not exist.**
- No GDD defines enemy stat blocks, health/damage growth by region or player level, or encounter budgets. Player curve is equipment-dominant + attribute-thin. Two of the three required curves (enemy, equipment) are missing.
- Fix: Add enemy scaling + encounter budget section to TCE GDD with per-region tiers.

**F18. Investigator skills are orphaned — "mind is your weapon" fantasy has no mechanics.**
- Persuasion/Insight/Comprehension defined in character-stats as signature investigation bonuses. ADE removed its Character Stats dependency; Thread Investigation has no INT/WIS checks. The game's core identity is mechanically disconnected.
- Fix: Re-home as journal/KG-side mechanics (Comprehension → auto-suggest linking facts; Insight → highlight contradiction candidates; Persuasion → modifier on trust-gain events). Update both GDDs' dependency tables.

### ⚠️ Warning (9)

**F1. XP curve demands ~200 thread chains for level 20 — unsustainable on indie content budget.** Fix: flatter curve plateau or re-weight revelations/trust milestones.

**F4. Investigation demands 5+ simultaneous tracked systems.** Highest cognitive load in the game. Fix: consolidate through diegesis; journal as only external memory.

**F5. max_active_threads=20 risks checklist-ification.** Anti-pillar: "not a completionist checklist." Fix: cap visible threads lower (7-9); auto-archive stale to cold case.

**F8. DEX is a triple-threat (AP, initiative, evasion, damage).** STR/CON underweight. Fix: move initiative dominant weight to WIS; cap evasion with armor interaction.

**F9. CON is a near-dump stat; CHA risks joining it.** Fix: give CON investigation role (fatigue, interrogation endurance); resolve barter question for CHA.

**F11. Health economy: rest-only healing under permadeath.** Coherent but unverified pacing. Fix: add consumable class or codify encounter budget between rests.

**F14. Player growth is so slow difficulty ramps must be authored, not derived.** Accept as deliberate Disco-adjacent choice and state explicitly.

**F15. Two parallel price systems (faction multiplier vs CHA barter).** Fix: single system — faction as merchant gate, CHA as per-transaction haggling.

**F20. Save/Load combat-save contradicts TCE.** Same as B1 — independently found by both phases.

---

## GDDs Flagged for Revision

| GDD | Reason | Priority |
|-----|--------|----------|
| ai-dialogue-engine.md | 3 internal contradictions, missing interjection interface, orphaned stat dependency | 🔴 Blocking |
| save-load-system.md | Combat quick-save contradiction with TCE | 🔴 Blocking |
| tactical-combat-engine.md | Enemy scaling undefined, loot references unowned | 🔴 Blocking |
| character-stats-progression.md | Orphaned derived stats, XP curve unsustainable | ⚠️ Warning |
| faction-system.md | Internal formula contradiction, standing ownership split | ⚠️ Warning |
| living-environment.md | Missing day counter, suspended semantics clash | ⚠️ Warning |
| dialogue-ui.md | Stale against ADE typewriter, missing sections | ⚠️ Warning |
| investigation-journal.md | Missing sections, no PWS mention | ⚠️ Warning |
| combat-ui.md | Missing sections | ⚠️ Warning |
| map-system.md | Missing sections, thread markers risk checklist-ification | ⚠️ Warning |
| audio-system.md | Missing sections | ⚠️ Warning |
| systems-index.md | 12 stale dependency rows, TCE→Party tier conflict | ⚠️ Warning |

---

## Verdict: 🔴 FAIL

8 blocking issues. The most critical three:
1. **No economy (F10)** — the entire equipment/combat progression chain has no owner
2. **No enemy scaling (F13)** — the difficulty curve literally does not exist
3. **Orphaned investigator skills (F18)** — the game's core fantasy has no mechanics

### Required Actions Before Architecture

1. Author a **Loot & Economy GDD** (or quick-spec) covering gold sources/sinks, loot tables, equipment curve, shops
2. Add **enemy scaling + encounter budget** to TCE GDD with per-region tiers
3. Re-home **Persuasion/Insight/Comprehension** as journal/KG mechanics (not dialogue rolls)
4. Resolve ADE's 3 internal contradictions (model identity, latency target, timeout constants)
5. Add Party System interjection interface to ADE
6. Fix save-load combat quick-save row to match TCE (Block quick-save and auto-save)
7. Resolve LLM model identity across all 4 referencing documents
8. Author Input Binding System GDD or fold keybinding spec back into Main Menu
