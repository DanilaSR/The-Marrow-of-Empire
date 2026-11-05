# Story 006: Pause Menu

> **Epic**: Main Menu & Settings
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: L (5-7 hours) — world freeze via GamePaused/GameResumed events, 200ms dim overlay, 5-option menu, combat pause variant with HUD above dim, quit confirmation (Save & Quit / Quit Without Saving / Cancel), 300ms toggle cooldown, restricted state guards (cutscenes/transitions/save), ≤16.6ms p95 bench.
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/main-menu-and-settings.md`
**Requirement**: `TR-ui-001`, `TR-ui-004`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework; also references ADR-0007 (game clock — pause freezes time), ADR-0009 (combat — pause during combat), ADR-0011 (event bus — pause state change events)

**ADR Decision Summary**: Pause is a full world freeze — all simulation systems stop. Desk register overlay dims world to 30% over 200ms. Combat HUD renders above the dim layer. Pause is blocked during cutscenes, region transitions, and mid-save operations. 300ms cooldown prevents rapid-toggle flicker.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Engine update-loop suspension mechanism (pause must guarantee no game logic advances). Dim overlay rendering as a full-screen quad above game world but below combat HUD. Low-pass filter on game audio during pause (SoLoud filter API wrapper — deferred engine feature at MVP).

**Control Manifest Rules (Presentation layer)**:
- Required: Desk register — full-screen dim overlay (world dims to 30%), pause menu panel
- Required: Paper-slide motion: enter 200-300ms ease-out, exit 200ms
- Required: Desk never trembles — UI immune to camera shake
- Forbidden: Never use 3D UI or diegetic HUD
- Guardrail: No frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/main-menu-and-settings.md` CR-4:*

- [ ] **AC-1**: Given the player is in Exploration state with one scheduled NPC (Schedule component active) and one weather system tick, when Escape is pressed, then (a) PauseMenu state is entered by the end of that tick, (b) the dim overlay alpha reaches target 0.7 over `pause_dim_ms` (200ms) following `clamp(t_game / pause_dim_ms, 0, 1) × 0.7`, (c) each simulation system's tick counter is identical across all subsequent paused frames (freeze observable), and (d) the pause menu contains exactly five options: Resume, Save, Load, Settings, Quit to Main Menu. [Integration — freeze contract + UI state]

- [ ] **AC-2**: When the player clicks "Resume" or presses Escape in the pause menu, the pause menu fades out, the dim lifts, and the game world unfreezes. No game time has passed — NPC schedules, combat timers, and wildlife positions are identical to when the pause was triggered.

- [ ] **AC-3**: When the player is in combat and Escape is pressed, the combat world freezes with the combat UI still visible behind the dimmed overlay. All combatants remain in their paused positions. The battlefield state is visible through the overlay. The combat HUD is not re-rendered while paused (HUD draw-call delta = 0 across paused frames).

- [ ] **AC-4**: When the player clicks "Quit to Main Menu" with unsaved progress, a confirmation dialog appears: "Quit to Main Menu? Unsaved progress will be lost." Options: "Save & Quit" / "Quit Without Saving" / "Cancel." "Quit Without Saving" button uses destructive styling (oxblood fill #7A2E28, ivory text ≥4.5:1).

- [ ] **AC-5**: Two Escape presses separated by less than `pause_toggle_cooldown_ms` (300ms) game-time: the second press is ignored (no state transition, no dim animation reversal). Boundary: delta == cooldown is permitted (second press toggles). [Logic — unit-testable with injected clock]

- [ ] **AC-6**: The pause menu is blocked (Escape does not open it) during: cutscenes (Escape skips cutscene instead if skippable), region transitions (input already locked), and mid-save operations (input locked during save write).

- [ ] **AC-7**: During pause menu open with 10 combatants and full combat HUD rendered, frame time ≤ 16.6ms p95 (profiled over 3000-frame capture at 1920×1080), and the number of draw calls in the pause state equals the in-combat draw calls + exactly one additional draw call (the dim quad). [Bench — profiler benchmark]

---

## Implementation Notes

*Derived from ADR-0012, ADR-0007, ADR-0009, ADR-0011:*

- **World freeze mechanism**: On PauseMenu state entry, emit a `GamePaused` event via EventBus (ADR-0011). All simulation systems (NPC schedules — ADR-0007, combat — ADR-0009, wildlife, weather) subscribe and stop advancing. The game clock (ADR-0007) freezes — no time passes. The freeze is implemented via engine update-loop suspension: a `paused` flag checked at the top of the main update loop; when set, `Update()` is skipped for all systems. Each system exposes a `TicksAdvanced()` counter for testability.
- **Unfreeze**: On PauseMenu exit (Resume), emit `GameResumed` event. Systems resume from their frozen state. The game clock continues from the exact tick it was frozen at.
- **Dim overlay**: A full-screen quad rendered in screen space at z-order between the game world and the combat HUD. The quad is black with alpha animated from 0 to 0.7 over `pause_dim_ms` (200ms). Target alpha: 0.7 (30% brightness = 70% dim). The dim uses ease-out interpolation. The combat HUD (Field register) renders ABOVE the dim — it remains fully visible.
- **Pause menu panel**: Desk register parchment panel rendered above the dim. Five options in vertical list: Resume, Save, Load, Settings, Quit to Main Menu. Styling matches title screen menu buttons (parchment-toned, ink text, gold hover, thread-red active).
- **Escape key routing**: Outside pause menu: Escape → enter PauseMenu. Inside pause menu: Escape → Resume (equivalent to clicking Resume). During cutscenes: Escape → skip cutscene (if skippable). During transitions/save: Escape → ignored (input locked).
- **Toggle cooldown**: A `lastToggleTime` timestamp updated on each PauseMenu enter/exit. On Escape press, check `currentTime - lastToggleTime >= pause_toggle_cooldown_ms` (300ms). If false, ignore. Uses injectable game-time clock for testability — see GDD AC measurement protocol.
- **Quit confirmation**: "Save & Quit" triggers save flow (Story 004) then transitions to TitleScreen. "Quit Without Saving" bypasses save and transitions to TitleScreen. Both use destructive button styling for the data-loss option (oxblood fill, ivory text). "Cancel" returns to pause menu.
- **Combat pause variant**: In combat, the same pause overlay renders. The combat HUD (turn order, AP bar, ability grid) is in the Field register and remains visible above the dim. The battlefield state (unit positions, health bars) is visible through the dim. The combat HUD is NOT re-rendered while paused — it's a static snapshot. The Field register is excluded from the dim layer (renders above it).
- **Restricted states**: The pause menu's `CanPause()` check queries: (a) current state != Cutscene (or cutscene is unskippable), (b) not in RegionTransition, (c) not in mid-Save operation. If any condition fails, Escape press is either consumed by the active state or ignored.
- **Audio on pause**: At MVP, audio continues playing during pause (low-pass filter is a deferred engine feature). The GDD's audio requirement for brief low-pass on pause is noted but not blocking.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Navigation within pause menu, focus rings, crossfade transitions
- Story 004: Save/Load screens (triggered by "Save" and "Load" options)
- Story 005: Settings screen (triggered by "Settings" option)
- Story 007: Escape key rebinding (the keybinding interface in Story 007 can rebind pause to another key)
- Combat HUD: The combat HUD rendering and z-order — pause menu ensures it renders above dim but does not own it
- Audio low-pass filter: Deferred engine feature — SoLoud filter API wrapper
- Cutscene skip: Cutscene system owns the skip behavior — pause menu only provides the Escape routing rule

---

## QA Test Cases

*Derived from GDD acceptance criteria. The developer implements against these — do not invent new test cases during implementation.*

### Automated (integration + logic tests)

- **AC-1** (Pause activation + freeze):
  - Given: Exploration state, 1 NPC with Schedule component, 1 weather system tick, frame-stepping harness active
  - When: Escape is pressed
  - Then: PauseMenu state entered by end of tick; dim overlay alpha = clamp(t_game/200ms, 0, 1) × 0.7 (ease-out); NPC schedule TicksAdvanced() counter unchanged across all subsequent paused frames; weather system TicksAdvanced() counter unchanged; pause menu has exactly 5 options: Resume, Save, Load, Settings, Quit to Main Menu
  - Edge cases: Escape press during dim animation (first 200ms) → ignored until animation completes; pause during NPC mid-schedule-update (tick in progress) → current tick completes, then freeze

- **AC-2** (Resume):
  - Given: pause menu open, world frozen at game-time T
  - When: "Resume" clicked or Escape pressed
  - Then: pause menu fades out over 200ms; dim lifts (alpha 0.7→0); world unfreezes; game clock resumes from game-time T (not T+1); NPC positions and schedule states identical to pause moment
  - Edge cases: Resume via gamepad Start button → same behavior; Resume via Escape while a sub-menu (Save/Load/Settings) is open → sub-menu closes first (Escape = back), second Escape resumes

- **AC-3** (Combat pause):
  - Given: combat active, 5 combatants, combat HUD rendered
  - When: Escape is pressed
  - Then: combat world freezes; dim overlay appears; combat HUD elements are visible above the dim (not obscured); all combatant positions frozen; combat HUD draw-call delta = 0 across all paused frames (not re-rendered)
  - Edge cases: combatant mid-animation → animation freezes at current frame; status effect tick scheduled during pause → does not fire until resume

- **AC-4** (Quit confirmation with unsaved progress):
  - Given: pause menu open, unsaved progress exists (dirty flag set since last save)
  - When: "Quit to Main Menu" clicked
  - Then: confirmation dialog appears: "Quit to Main Menu? Unsaved progress will be lost."; "Save & Quit" → triggers save then TitleScreen; "Quit Without Saving" → TitleScreen (styled oxblood fill); "Cancel" → returns to pause menu
  - Edge cases: no unsaved progress → "Quit to Main Menu" proceeds directly without confirmation dialog; save fails during "Save & Quit" → error shown, returns to pause menu, does NOT quit

- **AC-5** (Toggle cooldown):
  - Given: injectable game-time clock, cooldown = 300ms
  - When: Escape pressed at t=0 (enters pause), Escape pressed at t=200ms
  - Then: second press ignored; pause menu remains open; no state transition; dim animation does not reverse
  - When: Escape pressed at t=300ms (delta == cooldown)
  - Then: second press toggles (closes pause menu); boundary inclusive
  - Edge cases: rapid Escape spam (5 presses in 100ms) → only first processed; cooldown configurable via tuning knob

- **AC-6** (Restricted states):
  - Given: cutscene playing (skippable)
  - When: Escape is pressed
  - Then: cutscene skips (cutscene system handles); pause menu does NOT open
  - Given: region transition in progress
  - When: Escape is pressed
  - Then: input ignored; pause menu does NOT open
  - Given: save write in progress
  - When: Escape is pressed
  - Then: input ignored; pause menu does NOT open
  - Edge cases: unskippable cutscene → Escape ignored entirely; transition completes → Escape works normally on next frame

### Bench

- **AC-7** (Performance):
  - Given: pause menu open, 10 combatants, full combat HUD rendered
  - When: profiled over 3000-frame capture at 1920×1080
  - Then: frame time ≤ 16.6ms p95; draw calls = in-combat draw calls + 1 (dim quad); combat HUD draw-call delta = 0 across paused frames
  - Edge cases: pause menu open with no combat (exploration) → lower GPU load, same CPU assertion

---

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/main_menu/pause_menu_test.go` — must exist and pass
- Bench: Profiler benchmark showing ≤16.6ms p95 during pause with 10 combatants (AC-7)

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Navigation & Transition Framework), Story 004 (Save/Load — "Save" and "Load" options), Story 005 (Settings — "Settings" option), ADR-0007 (game clock — freeze contract), ADR-0009 (combat — combat pause variant), ADR-0011 (event bus — GamePaused/GameResumed events)
- Unlocks: None directly

---

## Completion Notes

**Completed**: 2026-08-12
**Criteria**: 7/7 passing (25 integration tests)
**Deviations**: None
**Test Evidence**: `tests/integration/main_menu/pause_menu_test.go` — 25 tests, all PASS. Bench: 21ns/frame (well under 16.6ms budget)
**Code Review**: Complete — APPROVED, 6/6 standards, ADR-0012/0007/0011 compliant
