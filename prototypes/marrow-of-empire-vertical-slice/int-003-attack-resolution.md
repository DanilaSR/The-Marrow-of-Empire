# INT-3 — Attack Resolution & the Mine Encounter (Complete)

- **Date**: 2026-08-12 (verification completed 2026-08-13)
- **Status**: ✅ Complete — encounter fought and won in the running game

## Scope Correction

COMBAT-003 (d20 attack resolution) was **already implemented** when INT-3
started — `src/combat/attack_resolution.go` + tests (the story file's
test-evidence section was stale). INT-3 became the **game-layer wiring**:
compose the completed combat engine (stories 001–005) into the running game
and attach the mine encounter.

## What Was Built

`src/game/combat_wiring.go` — the composition root:

- **Entity framework activation**: `entity.NewEntityStore(host, bus, config)`
  at New Game; player + wolves as EF entities with `entity.Stats` + `entity.Health`
- **Combat composition**: `CombatManager` + `D20Attacker` (nil positioning =
  Story-003 behavior) + `BasicEnemyAI` + slice `EquipmentReader` (player blade
  acc 12 / dmg 4; wolf claws) + straight-line `Mover`
- **Encounter**: proximity trigger (4.0u from the mine) → `StartCombat`
  (player + 2 wolves); ambush outranks open dialogue panels
- **Player turn**: click wolf → AP spend → d20 resolution → caller applies
  damage → defeat removal → `EndTurn`; wolf AI moves/attacks automatically
- **Victory**: inscription cube drops at the mine (INT-4 chain material)
- **Defeat**: respawn at town, wolves cleared, encounter re-armed

Files: `src/game/combat_wiring.go` (new), `world.go` (combat input gating,
turn telemetry, wolf screen telemetry), `world_config.go` (enemy/inscription
colors, talk range), `game.go` (field). Tests:
`tests/unit/game/combat_wiring_test.go` (5 tests: trigger distance, stat
blocks).

## Live Verification (standalone + AI Driver)

Full loop in one running session: walk to the mine → encounter starts →
4 attacks (rolls 8, 11, 6, 17; 5 damage each = weapon 4 + STR_mod 1) →
both wolves defeated → **VICTORY** → inscription cube rendered at the mine.

Evidence: `production/qa/evidence/int3/victory-inscription.png` + run logs.

## Findings

1. **`ResolveAttack` returns the outcome; the caller applies the damage.**
   The first wiring forgot this — wolves never lost HP. The enemy AI applies
   internally; the game-side player attack now mirrors it.
2. **Two parallel stat types exist**: `entity.Stats` (EF component, read by
   combat) vs `stats.StatsComponent` (stats package). Combat validates
   `entity.Stats` — the wiring must set that one. Bridging them is a
   production story.
3. **Deactivated entities still render** in this engine version — defeated
   wolves kept drawing (and intercepting clicks). Use `DestroyEntity`.
4. **Dead enemies stayed clickable** — repeated clicks re-defeated the same
   corpse. `wolfAt` now skips defeated entries.
5. **Dialogue-vs-combat priority**: distant NPC clicks opened dialogue and
   froze the walk (click-anywhere slice simplification). Fixes: dialogue now
   requires conversation range (3.5u, CRPG convention), and the encounter
   trigger outranks an open dialogue panel (ambush interrupts conversation).
6. AI Driver input is dropped while the game window lacks OS focus —
   keyboard (F1/F2) and clicks both. Requires a manual window click;
   verification loops must poll for effect, not assume delivery.

## Velocity

One long session; ~2/3 of the iterations were input-delivery and
component-type mismatches (findings 1–2), not combat logic — the engine
itself worked on first composition.
