# INT-2 — The Talk Loop (Complete)

- **Date**: 2026-08-12
- **Status**: ✅ Complete — pipeline verified live end-to-end (visual panel fix deferred to INT-4)

## Scope

3 NPCs with authored `.kg` content (blacksmith → miner → scholar investigation
chain), click-NPC → dialogue, question → topic match → KG facts → prompt →
generation → processed response → thread creation.

**Cut**: topic chips, trust indicator, portrait art, conversation memory
persistence, warmup UI, walk-then-auto-talk, panel slide animation (deferred).

## Files

| File | Action | Content |
|------|--------|---------|
| `assets/npc_knowledge/borderland/{blacksmith,miner,scholar}.kg` | new | first game content — 3 NPCs, 9 facts, one chain, one authored contradiction (`mira_smelting_formula` ↔ `vek_forgemaster_oath`) |
| `src/ai_dialogue/companion_codec.go` | new | stdin JSON ↔ llama-server `/v1/completions` wire codec (testable, no model needed) |
| `src/ai_dialogue/companion/main.go` | new | companion binary: spawns llama-server, bridges game JSON protocol |
| `src/dialogue_ui/panel.go` | rewritten | imperative → **markup** panel (engine's proven UI path) |
| `src/dialogue_ui/messages.go` | new | message log (template duplication), text input, typewriter reveal |
| `src/game/talk.go` | new | talkSystem: .kg loading, NPC registry, `onQuestion` pipeline, thread registration |
| `src/game/world.go` / `world_config.go` / `game.go` | edit | NPC cubes, click routing, pick radius, debug aids |
| tests | new | companion codec (11), PickNPC (5), WrapLines (5), .kg content validation (2) |

## Live Verification (standalone + AI Driver)

- 3 NPCs loaded from `.kg` (log: "talk: NPC loaded" × 3)
- LLM backend spawned the companion → llama-server (real model available)
- Click NPC → "talk: dialogue opened" → F2 test question → full pipeline →
  **attributed in-character response** (see evidence)

> **Question**: "What do you know about the ore?"
> **Response**: "I told the smith my ore had gone bad. I didn't tell him what
> I found. He'd hammer it into nails — and I don't think the dead want their
> words used that way." — Tomas the Miner

**Evidence**: `production/qa/evidence/int2/` (response JSON + screenshot)

## Findings

1. **LLM speed gates the real model on this hardware.** Capability probe:
   decode 1.2–1.4 tok/s vs 10 tok/s minimum → session falls to text mode.
   The real-model path itself works (companion e2e test produced a correct
   in-character completion) but at ~1 tok/s a 100-token response ≈ 70s —
   outside the GDD's real-time budget (25s timeout). **Slice report item:
   either faster hardware, a smaller/faster model, or accept the fallback
   as the primary experience on low-end CPUs.**
2. **Markup absolute-positioned panels misrender.** The dialogue panel
   (`position: absolute; top/bottom` + px/percent sizes) renders at the
   top-left at ~half window size, backgrounds unfilled — text-only. Working
   references: the title screen (flow layout only) and the engine console
   (`top: 0`, `width: 100%`, `height: 50%`, `z-index: 100`). Deferred to
   INT-4 as a targeted fix; pipeline verified headlessly via logs.
3. **NPC pick radius must absorb the click-ray ground offset.** The ground
   hit through a cube's visual body lands ~1.7u beyond its base
   (30° pitch × cube height) — `worldNPCClickRadius` 1.5 → **3.0**.
4. **Matcher relevance threshold was too strict.** With 8–9 keywords per
   domain, one keyword scores ~0.12 — under the 0.3 threshold, so ordinary
   single-topic questions never matched (empty fallback "…"). Slice tuning:
   threshold **0.1** (within GDD range 0.1–0.6), via
   `MatchTopicsWithContext` + history.
5. **Engine panic fixed**: the dialogue panel's portrait `Image` was never
   `Init`'d before `AddChild` (nil layout backpointer → SIGSEGV). Also:
   markup text elements are panels — use `element.InnerLabel().SetText()`,
   not `elm.UI.ToLabel()`.
6. **Dual ui.Manager**: elements created under a second `ui.Manager` on one
   host do not render. The panel uses the game's shared manager
   (`character_sheet` pattern).

## Debug Aids (removed in INT-4)

- F1: teleport player to the miner; F2: submit the test question through the
  live pipeline; per-second player/NPC screen-position logs (NPC ground
  points projected through the camera's view-projection matrices).

## Velocity

One long working session: content authoring, companion binary (+ live
llama-server contract check), panel framework markup rewrite, orchestration,
tests, and ~10 live-verification iterations — most of them consumed by the
panel rendering investigation and AI-Driver navigation calibration, not by
the pipeline itself.
