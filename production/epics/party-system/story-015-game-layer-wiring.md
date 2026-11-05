# Story 015: Game-Layer Wiring — Party Live in the Build

> **Epic**: Party System
> **Status**: Ready
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: M (5-7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-13

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**: `TR-party-001` — party roster, companion combat participation
**ADR Governing Implementation**: ADR-0006 (component schema, primary); ADR-0011 (EventBus, secondary); ADR-0002 (save serialization)
**ADR Decision Summary**: PartyState is one store per save; `pc_controlled` tagging is party-owned; roster changes emit synchronous events. Story 001 (complete) provides the storage/capacity/locking layer — this story instantiates it in the game and wires persistence, rest points, and the combat roster feed.
**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Pure Go wiring — no new Kaiju APIs. The combat-roster snapshot contract (`ActiveRoster()`, read-only) is Story 007; this story wires the feed only for the existing encounter (attack resolution already live in the slice).

---

## Acceptance Criteria

- [ ] **AC-1** (store live): GIVEN a new game, THEN the party store is instantiated with the PC registered as the first active member (PC slot locked).
- [ ] **AC-2** (persistence): GIVEN a save/load cycle (INT-6), THEN the roster (active order + pool + rest point) round-trips through pod serialization.
- [ ] **AC-3** (rest point hook): GIVEN the player reaches the slice's rest point, THEN `SetRestPoint` records it and pool companions are present there (CR-1).
- [ ] **AC-4** (combat feed): GIVEN a combat encounter starts, THEN the encounter roster is built from `ActiveRoster()` — PC plus active companions — via the read-only contract (no direct store mutation from combat).
- [ ] **AC-5** (tag live): GIVEN a companion in the active party, THEN the `pc_controlled` component is set on their entity (debug-overlay verifiable); pool members do not carry it.

---

## Implementation Notes

- Instantiate the Story 001 store in `src/game/` (game-owned, one per save); hook creation at new-game and load.
- Persistence: the store's `EntityData`/`pod.Register` wiring (Story 001) means the save orchestrator (INT-6) serializes it — verify the registration is picked up by the orchestrator; if the EF serialization pass only collects per-entity stores (Story 001's noted caveat), wire the singleton store into the save manifest explicitly, following the PWS tombstone precedent.
- Rest point: the slice has one rest point entity — hook `SetRestPoint` on arrival.
- Combat feed: the existing `src/game/combat_wiring.go` consumes the roster snapshot at encounter start; keep it read-only.
- No recruitment UI in this story (Story 003 territory) — a debug hook (console/ai_driver) may add a companion to prove AC-4/AC-5 live; do not build player-facing recruitment.

## Out of Scope

- Story 002 state machine, Story 003 recruitment validation, Story 004 swap/dismissal, Story 007 TCE-wide roster consumption
- Party management screen (Story 014)

---

## QA Test Cases

**Automated (integration) — `tests/integration/party/game_wiring_test.go`:**
- New game → PC registered, slot locked, roster read-only snapshot correct
- Save → mutate roster → load → restored (with INT-6 harness)
- Rest point hook: SetRestPoint updates pool presence
- Combat feed: roster snapshot at encounter start equals ActiveRoster; combat cannot mutate the store

**Manual/live (ai_driver, BLOCKING for this story):**
- Debug add companion → pc_controlled visible in overlay; encounter roster includes companion
- Evidence: screenshots to `production/qa/evidence/`

**Estimated test count**: ~8 + 2 live checks

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/party/game_wiring_test.go` — must exist and pass
- Live: ai_driver screenshots in `production/qa/evidence/`

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Complete), INT-6 (save-load-system/story-004-game-layer-wiring.md)
- Unlocks: Combat roster consumption at scale (Story 007)
