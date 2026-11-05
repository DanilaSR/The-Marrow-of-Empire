# Story 007: Game-Layer Wiring — PWS Live in the Build

> **Epic**: Persistent World State
> **Status**: Ready
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-13

## Context

**GDD**: `design/gdd/persistent-world-state.md`
**Requirement**: TR-pws-* catalog (flag store, event pipeline, memory log, evolution rules) — lookup fresh in `docs/architecture/tr-registry.yaml` at dev time
**ADR Governing Implementation**: ADR-0013: Persistent World State (primary); ADR-0001/ADR-0002 (entity architecture, serialization)
**ADR Decision Summary**: World flags live on entities' `Persistent` component; events submit flag mutations + memory-log entries; YAML-authored evolution rules evaluate once per region load. Stories 001-004 are complete headless — this story wires the flag store, event pipeline, and region-load rule evaluation into the running game.
**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Pure Go wiring. The slice world (`src/game/world.go`) builds regions at new game — hook evolution-rule evaluation there; rule files live at `assets/world_rules/{region_id}.yaml` (create the slice region's file).

---

## Acceptance Criteria

- [ ] **AC-1** (events live): GIVEN an in-game action (e.g., the player talks to the slice miner), THEN a world event is submitted through the pipeline — flag mutation applied and a memory-log entry appended (queryable via debug hook).
- [ ] **AC-2** (region-load rules): GIVEN the slice region loads, THEN its evolution rules evaluate once (declaration order) and triggered events are submitted — no re-fire on repeat visits within the same session.
- [ ] **AC-3** (persistence): GIVEN a save/load cycle (INT-6), THEN world flags and memory log entries round-trip.
- [ ] **AC-4** (destroyed-entity protection): GIVEN a quest-relevant entity destroyed, THEN the clue-path protection flag is set and the memory log records it (live in the mine encounter).

---

## Implementation Notes

- Instantiate the Story 001-004 components in `src/game/`: flag store over the entity framework's `Persistent` components (the slice entities already carry components — verify which ones have `Persistent`), event pipeline, memory log, clue-path protection listener on the EF's `on_entity_destroyed` broadcast.
- Region-load hook: `world.go`'s region build calls rule evaluation with the current game time; author `assets/world_rules/borderland.yaml` (slice region) with one simple demonstrable rule (e.g., `IF world.ruin_cleared AND days_since(world.ruin_cleared_at) > 7 THEN location_repopulated`).
- Persistence: PWS state already lives on entities' Persistent components — INT-6's orchestrator serializes them; verify round-trip, do not duplicate serialization.
- Debug hook (console/ai_driver): submit a world event and query flags/log for live verification.

## Out of Scope

- Story 005 evolution-rule engine internals (complete), Story 006 save/load performance benchmarks
- Journal UI consumption of the memory log (investigation journal epic is complete — wiring its feed to the log is optional polish, not this story)

---

## QA Test Cases

**Automated (integration) — `tests/integration/persistent_world_state/game_wiring_test.go`:**
- Game harness: submit event → flag + log entry; region load → rules evaluate once, no re-fire
- Save → mutate flags → load → restored (with INT-6 harness)
- Destroyed quest-relevant entity → protection flag + log entry

**Manual/live (ai_driver, BLOCKING for this story):**
- Talk to miner → flag set (debug hook shows it); region rules fire on load
- Evidence: screenshots to `production/qa/evidence/`

**Estimated test count**: ~8 + 2 live checks

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/persistent_world_state/game_wiring_test.go` — must exist and pass
- Live: ai_driver screenshots in `production/qa/evidence/`

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Stories 001-004 (Complete), INT-6 (save-load-system/story-004-game-layer-wiring.md)
- Unlocks: World evolution visibility in the running game
