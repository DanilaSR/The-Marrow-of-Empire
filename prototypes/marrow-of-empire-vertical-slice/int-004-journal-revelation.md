# INT-4 — Journal, Revelation & Rendering Fixes (Complete — playtest pending)

- **Date**: 2026-08-13
- **Status**: ✅ Code complete and verified. The human playtest (Session 1)
  and REPORT.md are the remaining slice steps.

## What Was Done

1. **UI rendering recipe found** (the slice's biggest technical discovery):
   - Working: markup + `position: absolute` + **px** top/left/width/height +
     z-index. Proven by a 3-variant repro screen (flow divs collapse to a
     1px line; % heights silently fail).
   - Working show/hide: **entity `Activate()`/`Deactivate()`** over the
     document's elements (the engine console's own mechanism). The CSS
     `display` toggle via `SetElementStyleProperty` does NOT re-apply
     layout in this engine version.
   - Runtime UI tree mutations (add/remove children) race the engine's UI
     update pass → `anyChildDirty` index-out-of-range panic. Fix: static
     label pools, text assignment only.
   - Markup `<label>` elements are panels wrapping an inner label;
     whitespace-only labels have no inner label (nil crash) — zero-width
     space content + `InnerLabel()` binding required.
2. **Dialogue panel fixed**: px-absolute markup + activation-based
   visibility + 8-label static log pool. Live-verified: 233k parchment
   pixels at the bottom third, in-character response in the log, zero
   panics.
3. **Slice journal view** (`src/game/journal_view.go`): parchment overlay
   (proven recipe), 14-line static pool, fed by the investigation journal
   package's pure display functions (BuildChainList / BuildThreadList /
   BuildRevelationPrompt). J key or the parchment tab toggles it. The full
   three-panel journal UI (imperative construction) needs a markup
   conversion — production story.
4. **Revelation proven headlessly**: `tests/unit/game/revelation_chain_test.go`
   replays Vek → Tomas → Mira through the real thread engine: threads open,
   the authored contradiction flags its facts, and the chain reaches
   `revelation_ready` — `BuildRevelationPrompt` fires. Deterministic, no
   input flakiness.
5. **Debug cleanup**: F1–F4 keys, position/telemetry logs, click-cycle
   navigation, and the repro screen removed. J + the journal tab remain
   (real features).

## Live-Verification Notes

- The dialogue panel + response pipeline verified live in one session
  (Vek's authored ore fact returned in character).
- The full three-conversation live walkthrough was repeatedly blocked by
  OS window-focus flakiness (AI Driver input drops when unfocused); the
  chain logic is instead proven by the headless test — the human playtest
  covers the live path.

## Findings Carried to Production

1. px-absolute markup + entity activation = the engine's proven UI pattern
   (applies to the journal epic, combat HUD, all UI stories).
2. Static label pools; never mutate UI trees at runtime in this engine.
3. `TryPlaneHit` uses camera-origin rays — screen projections computed
   from View/Projection matrices do not match it. Use the engine's own
   raycasts for picking; avoid custom world→screen math.
4. `ResolveAttack` returns outcomes; callers apply damage.
5. `entity.Stats` (EF component) vs `stats.StatsComponent` duplication
   needs a bridging story.
6. Local LLM decode 1.2–2.5 tok/s on dev hardware — the real-time dialogue
   budget is not met on this machine (text mode engaged by design).

## Remaining (playtest + report)

- Playtest Session 1 per `production/playtests/playtest-plan-2026-08-12.md`
- REPORT.md + `prototypes/index.md` verdict row (vertical-slice skill
  Phases 6–8)
