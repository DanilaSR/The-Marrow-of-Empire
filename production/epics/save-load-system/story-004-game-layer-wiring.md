# Story 004: Game-Layer Wiring — Save/Load Live in the Build

> **Epic**: Save/Load System
> **Status**: Ready
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-13

## Context

**GDD**: `design/gdd/save-load-system.md`
**Requirement**: TR-sl-* catalog — lookup fresh in `docs/architecture/tr-registry.yaml` at dev time
**ADR Governing Implementation**: ADR-0002: Save/Load Serialization (primary); ADR-0001: Entity Architecture (component stores)
**ADR Decision Summary**: All game state lives in component stores; the Save/Load orchestrator serializes them via the pod system with a version header and per-version migrations. The save-load epic's orchestration package is complete and headless — this story wires it into the running game.
**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: File I/O is plain Go — no Kaiju APIs. Verify the orchestration package's public surface (`src/save_load/` or equivalent) before writing the adapter; registered stores (`pod.Register`) serialize automatically once the orchestrator is driven.

---

## Acceptance Criteria

- [ ] **AC-1** (manual save live): GIVEN the game running (exploration state), THEN a manual save writes a save file to the save directory with the version header, world state (player position/rotation, entity stores, world flags from registered stores).
- [ ] **AC-2** (load live): GIVEN a save file, THEN loading restores the player position, world flags, and journal/thread state to the saved values (live-verified: save → move away → load → position restored).
- [ ] **AC-3** (combat save rules): GIVEN combat active, THEN manual save succeeds and quick-save/auto-save are blocked (GDD combat save rule).
- [ ] **AC-4** (corruption handled): GIVEN a corrupted save file (checksum mismatch), THEN loading surfaces the error path — the game does not crash or hang; title remains reachable.
- [ ] **AC-5** (migration path runs): GIVEN a save from version N loaded in N+1 with a registered migration, THEN the migration runs before restore (headless-testable; live path wired).

---

## Implementation Notes

- Wire the existing save/load orchestration package into `src/game/`: a save manager owned by the game, invoked from the pause/menu path (manual save) and boot (load).
- Save directory: injectable path (tests use temp dirs); follow the package's existing error contract (per-pod errors, checksum).
- Player position/rotation: the slice world stores the player transform in the game layer — route it through the entity framework or a registered store per ADR-0002 ("save everything" — all game state in component stores).
- Version header + migration: reuse the epic's existing migration framework (story 002/003 complete) — this story only wires the invocation.
- PWS and party stores register via `pod.Register` in their own packages (INT-7/INT-8) — this story must not duplicate their registration, only drive the orchestrator.

## Out of Scope

- Steam Cloud paths (epic story 002 — done headless; live cloud sync is a platform task)
- Save/Load screens' UI behavior (main-menu-settings epic)
- PWS/party store internals (their wiring stories)

---

## QA Test Cases

**Automated (integration) — `tests/integration/save-load/game_wiring_test.go`:**
- Save → mutate world state → load → state restored (with a minimal in-memory/temp-dir game harness)
- Corrupt file → error surfaced, no crash
- Version N save + migration → N+1 load applies migration
- Combat-state save rules: quick-save blocked, manual save allowed

**Manual/live (ai_driver, BLOCKING for this story):**
- In-game: move to a distinct position → manual save → move away → load → position restored
- Evidence: screenshots to `production/qa/evidence/` (before/after load)

**Estimated test count**: ~8 + 2 live checks

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/save-load/game_wiring_test.go` — must exist and pass
- Live: ai_driver screenshots in `production/qa/evidence/`

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: save-load-system stories 001-003 (Complete), entity framework (Complete), slice game layer
- Unlocks: INT-7 (party persistence), INT-8 (PWS persistence)
