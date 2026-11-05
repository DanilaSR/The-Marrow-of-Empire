# Story 005: Settings Screen & Persistence

> **Epic**: Main Menu & Settings
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: XL (7-9 hours) — 6-category tabbed settings, instant-apply + debounced JSON persistence, resolution preview/revert, per-category + all reset, corrupted recovery, deferred-feature greyed-out controls, in-game Graphics restrictions, ≤2ms p95 bench. Largest story in epic.
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/main-menu-and-settings.md`
**Requirement**: `TR-ui-001`, `TR-ui-005`, `TR-ui-006`, `TR-ui-007`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework; also references ADR-0003 (Camera settings), ADR-0006 (gameplay settings)

**ADR Decision Summary**: Six-category settings organized as tabs. All settings apply instantly on change — no Apply/Cancel button. JSON persistence to `settings.cfg` (debounced: dirty-flag + flush within 500ms). Resolution/display mode changes use a 3-second preview with auto-revert at 15 seconds. Per-category and all-settings reset with confirmation. Missing/corrupted `settings.cfg` → safe defaults.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju UI sliders, dropdowns, checkboxes, toggle controls. JSON encode/decode via `encoding/json` (matches Kaiju editor precedent `editor_settings.go`). Runtime renderer reconfiguration for resolution/display mode changes. FPS cap via `host.SetFrameRateLimit(targetFPS)` — already available.

**Control Manifest Rules (Presentation layer)**:
- Required: Desk register — parchment panel for settings
- Required: Desk serif for labels and descriptions
- Required: Minimum 1280×720, reference 1920×1080 with height-based scaling
- Required: Body text 16px at reference, minimum 12px effective
- Forbidden: Never use amber-on-parchment text
- Forbidden: Never encode game-critical information by hue alone
- Guardrail: Settings screen render ≤2ms p95 CPU (profiler benchmark)

---

## Acceptance Criteria

*From GDD `design/gdd/main-menu-and-settings.md` CR-3:*

- [x] **AC-1**: The Settings screen renders six categories as tabs or a vertical category list: Gameplay, Camera, Audio, Controls, Accessibility, Graphics. Each category contains the settings defined in the GDD's CR-3 table.

- [x] **AC-2**: All settings apply instantly on change — no "Apply" or "Cancel" button. The new value is applied to the target system immediately and written to `settings.cfg` (JSON) via a debounced write: dirty-flag set on change, flush within 500ms of last change or on clean shutdown. Never synchronous disk I/O in the render loop.

- [x] **AC-3**: Resolution and display mode changes show a 3-second preview countdown: "Keep changes? (Y) / Revert (N)." If no response within 15 seconds, changes auto-revert to the previous working settings. The Y/N confirmation is a keyboard-prompt overlay; the Settings screen remains accessible behind it.

- [x] **AC-4**: "Reset to Defaults" button at the bottom of each category page resets that category's settings only. "Reset All Settings" button in the footer resets every setting in all categories. Both require a confirmation dialog. Other categories are unaffected by a per-category reset.

- [x] **AC-5**: On launch, if `settings.cfg` is missing or corrupted, the system creates a new `settings.cfg` with all default values. The first-launch accessibility prompt is shown. No error is displayed to the player.

- [x] **AC-6**: If `settings.cfg` write fails mid-session (disk full, permission change): current in-memory settings remain active, error is logged, retry on next change. If retry also fails, show a non-blocking toast: "Settings could not be saved. Check available disk space." (3-second auto-dismiss).

- [x] **AC-7**: Deferred engine features (VSync toggle, texture quality, anti-aliasing, audio output device selection) render as greyed-out controls with a "Coming Soon" tooltip. FPS cap is NOT greyed out — `host.SetFrameRateLimit` already exists.

- [x] **AC-8**: When accessed from the Pause Menu (in-game), the Graphics category hides resolution, fullscreen, and display mode controls — these cannot be changed during gameplay. All other categories are fully available.

- [x] **AC-9**: Settings screen UI update + render-command generation ≤ 2ms p95 CPU (profiled over 3000-frame capture at 1920×1080, Go benchmark instrumentation, excludes background scene rendering).

---

## Implementation Notes

*Derived from ADR-0012 and GDD CR-3:*

- **Category organization**: Vertical tab list on the left, settings panel on the right. Active tab: amber gold indicator + bold weight (two cues — never color alone). Each category is a separate UI panel; tab switching shows/hides panels (no page reload).
- **Settings controls by type**:
  - Sliders (volume, sensitivity, zoom speed): track in deep stone, fill in amber gold, thumb in pale ivory. Range + step defined per setting. Value label shows current number.
  - Dropdowns (difficulty, resolution, display mode, FPS cap, texture quality, AA, colorblind mode, text size): parchment background, ink text, amber gold selection highlight + checkmark icon.
  - Toggles/checkboxes (auto-pause, tutorial hints, edge scroll, invert zoom, mute when unfocused, VSync, subtitles, speaker name): ink-dark label with toggle switch (amber gold when on).
  - Keybinding rows: per Story 007. The Controls category embeds the keybinding interface.
- **Persistence (`settings.cfg`)**:
  - Format: JSON (matches Kaiju editor precedent `editor_settings.go` using `settings.json` with `encoding/json`).
  - Location: platform-standard user data directory (`$XDG_DATA_HOME`/Linux, `%APPDATA%`/Windows, `~/Library/Application Support`/macOS per ADR-0002).
  - Schema: Flat JSON object with nested groups matching the 6 categories. Each category is a JSON sub-object. Example: `{"camera": {"edge_scroll_speed": 15, "zoom_transition_speed": 1.0}, "audio": {"master_volume": 0.8}}`.
  - Write strategy: In-memory struct mirroring the JSON schema. On any setting change: update struct field → apply to target system → set dirty flag. A goroutine wakes on dirty flag, waits 500ms (debounce), then `json.MarshalIndent` + `os.WriteFile`. On clean shutdown, flush immediately. Never `fsync` in the render loop.
  - Read strategy: On boot, `os.ReadFile` → `json.Unmarshal` into struct. Parse errors → log warning, create defaults struct, write fresh `settings.cfg`, trigger first-launch accessibility prompt.
- **Instant apply contract**: Each settings category has an `Apply(settings)` function. On change: update in-memory settings → call `Apply` → set dirty. The target system reads the new value on next frame/tick. No polling — the menu pushes changes.
  - Camera settings → Camera System's config struct
  - Audio settings → Audio System's mixer (volume levels)
  - Gameplay settings → relevant gameplay system configs
  - Graphics settings → engine renderer (resolution/mode/V-sync/FPS via `host.SetFrameRateLimit`)
  - Controls → Input Binding System registry (per Story 007)
  - Accessibility → Camera System + UI text scale
- **Resolution preview flow**: On resolution/display mode change: apply the change → start 3-second timer → show overlay: "Keep changes? (Y) / Revert (N)" with countdown. Y → make permanent (write to `settings.cfg`). N or timeout → restore previous settings. The overlay is non-blocking for the settings screen behind it (player can navigate away).
- **Reset to defaults**: Each category has a `Defaults()` function returning the default settings struct. "Reset to Defaults" → confirmation dialog → replace category settings with defaults → apply → mark dirty. "Reset All" → confirmation → all categories to defaults → apply all → mark dirty.
- **Deferred feature controls**: Greyed out with `opacity: 0.4`, `cursor: not-allowed`, tooltip "Coming Soon" on hover. The control is rendered and the settings schema reserves the key, but the Apply function is a no-op for these settings.
- **In-game variant**: When `entryPoint == PauseMenu`, Graphics category excludes resolution/display mode/fullscreen controls. Implemented as conditional rendering — the controls are not just greyed out, they are absent from the panel.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Navigation within settings screen, focus rings, crossfade transitions
- Story 003: First-launch accessibility prompt (triggered when `settings.cfg` is missing/corrupted — this story provides the detection and defaults creation)
- Story 006: Pause menu "Settings" option (triggers this screen)
- Story 007: Keybinding capture interface — the Controls category embeds Story 007's UI
- Camera System: Camera setting definitions and the actual camera behavior — menu provides UI only
- Audio System: Audio mixer, device enumeration, actual volume control — menu provides UI only
- Input Binding System: Keybinding registry, action definitions, conflict resolution logic — menu provides UI only
- Engine: Renderer reconfiguration (resolution, display mode, VSync), texture quality pipeline, AA shaders — menu sends config values only

---

## QA Test Cases

*Derived from GDD acceptance criteria. The developer implements against these — do not invent new test cases during implementation.*

### Automated (integration tests)

- **AC-2** (Instant apply + persistence):
  - Given: Settings screen open to Camera category, edge_scroll_speed = 15
  - When: the player changes Edge Scroll Speed to 20
  - Then: the Camera System receives the new value (20) within one frame; `settings.cfg` is written with `camera.edge_scroll_speed: 20` within 500ms
  - Edge cases: rapid changes (5 changes in 200ms) → only the final value is written (debounce); shutdown mid-debounce → flush on clean shutdown writes latest value; parse error reading back → defaults used

- **AC-3** (Resolution preview/revert):
  - Given: Settings screen, Graphics category, current resolution 1920×1080
  - When: the player selects 1280×720
  - Then: resolution changes immediately; 3-second countdown overlay appears: "Keep changes? (Y) / Revert (N)"; Y → resolution persists, written to `settings.cfg`; N → reverts to 1920×1080; no response within 15s → auto-reverts
  - Edge cases: Y pressed after display becomes unusable → Settings screen still accessible, player can change back; auto-revert timer uses game-time clock, not wall-clock

- **AC-4** (Reset to defaults):
  - Given: Camera category settings modified (edge_scroll_speed = 20, default = 15)
  - When: "Reset to Defaults" clicked and confirmed
  - Then: all Camera settings return to defaults; `settings.cfg` updated with defaults; Audio category settings are unchanged
  - Edge cases: "Reset All Settings" → all 6 categories to defaults, confirmation required; cancel → no changes

- **AC-5** (Missing/corrupted settings recovery):
  - Given: `settings.cfg` is deleted
  - When: the game launches
  - Then: a new `settings.cfg` is created with all defaults; first-launch accessibility prompt shown; no error displayed to player
  - Edge cases: corrupted JSON → same behavior; unwritable directory → `settings.cfg` creation fails → defaults in memory for session

- **AC-6** (Write failure toast):
  - Given: settings changed, `settings.cfg` write fails (simulated full disk)
  - When: the debounce flush triggers
  - Then: error is logged; in-memory settings remain active; on next change, retry occurs; if retry also fails, toast appears: "Settings could not be saved. Check available disk space."; toast auto-dismisses after 3 seconds
  - Edge cases: 3rd change after 2 failures → retry again (not permanently failed); successful write clears error state silently

- **AC-8** (In-game Graphics restrictions):
  - Given: Settings screen entered from Pause Menu
  - When: Graphics category is selected
  - Then: resolution, display mode, and fullscreen controls are absent; all other Graphics controls (FPS cap, VSync, texture quality, AA) are present
  - Edge cases: entered from Title Screen → all Graphics controls present; switching entry point mid-session not possible (entry point is fixed at screen open time)

- **AC-9** (Performance benchmark):
  - Given: Settings screen with all 6 categories rendered and populated with default values
  - When: profiled over 3000-frame capture at 1920×1080
  - Then: menu UI update + render-command generation ≤ 2ms p95 CPU
  - Edge cases: category tab switch does not spike frame time (panel show/hide, no reload)

### Manual verification

- **AC-1** (Six-category rendering):
  - Setup: Open Settings from Title Screen
  - Verify: All 6 categories visible as tabs/list; each category shows correct controls per GDD CR-3 table; all controls are interactive (except deferred features)
  - Pass condition: Each category matches the GDD's CR-3 table contents

- **AC-7** (Deferred feature controls):
  - Setup: Open Settings → Graphics category
  - Verify: VSync toggle, texture quality dropdown, anti-aliasing dropdown are greyed out with "Coming Soon" tooltip; FPS cap slider is interactive; audio output device dropdown (Audio category) is greyed out
  - Pass condition: Greyed-out controls cannot be interacted with; tooltip appears on hover

---

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/main_menu/settings_screen_test.go` — must exist and pass
- Bench: Profiler benchmark showing ≤2ms p95 for settings screen render (AC-9)

**Status**: [x] Created and passing — `tests/integration/main_menu/settings_screen_test.go` (25 integration tests + 1 benchmark; AC-1..AC-9 mapped 1:1 in the file header)

---

## Completion Notes

**Completed**: 2026-08-12
**Criteria**: 9/9 passing (25 integration tests + 1 benchmark in `tests/integration/main_menu/settings_screen_test.go`)
**Benchmark**: Settings screen p95 = 0.053ms / mean = 0.034ms over 3000 frames (CI measurement; threshold 2ms, TR-menu-014)
**Code Review**: Pending — ready for review
**External dependencies**: All stubbed by design — settings.cfg file I/O, camera/audio/gameplay/renderer application, and first-launch prompt triggering are injected seams the game layer wires:
- `WithSettingsClock` — injectable game-time clock (ms), shared with `NewTransitionSystem`
- `WithApplySetting(key, value)` — instant-apply to the target system
- `WithSettingsWriter(key, value) error` — debounced flush seam (game layer owns JSON marshaling + file I/O; contract is dotted keys + string values, e.g. `camera.edge_scroll_speed` → `"20"`)
- `WithHasSettingsFile` / `WithSettingsReader` — AC-5 boot probe and pre-population
- `WithReRunAccessibility` — hooks the Story 003 re-run path
- `CheckFirstLaunch()` — flags missing/corrupted settings for the Story 003 prompt owner; the prompt itself is out of scope

**Deviations** (approved by lead-programmer before implementation):
- CR-3 lists `reduce_camera_motion` under both Camera and Accessibility; canonical placement is Accessibility only, persisting the flat key `camera_motion` (Story 003's `SettingCameraMotion`), not a dotted key.
- AC-3 Y/N keyboard prompt: the flow's `Dispatch` translates no letter keys (Story 001 out of scope), so the screen exposes `HandleKey("y"/"n")` for the game-layer key router plus focusable Keep/Revert overlay buttons for mouse/gamepad.
- Debounce is clock-polled in `Update()` (goroutine-free core, matching `save_load_screen.go`); `Flush()` provides the clean-shutdown immediate write. No goroutine and no timer.
- AC-6 toast retry state machine: first failure logs + keeps dirty keys; a second consecutive failure shows the toast; every subsequent flush retries (never permanently stuck); success clears the error state silently.
- AC-8: resolution and display mode are absent (not greyed); "fullscreen" is the default display-mode option value, so the story's "fullscreen control" is covered by hiding `graphics.display_mode`.
- AC-7: FPS cap is interactive (not deferred); VSync, texture quality, AA, and audio output device are deferred with the "Coming Soon" tooltip and reject interaction (not focusable, programmatic setters refuse).

**Notes for reviewers**: All exported string constants are LOCALIZATION-marked; the settings registry is data-driven (`defaultSettingsTable`, camera knobs taken from the Camera System GDD tuning table); render commands use the allocation-free caller-scratch pattern (`RenderCommands(dst []RenderCommand)`); value strings are 2-decimal trimmed ("0.8", never "0.30000000000000004").

---

## Dependencies

- Depends on: Story 001 (Navigation & Transition Framework), JSON settings schema definition (this story owns it), Camera System (for camera settings contract), Audio System (for audio settings contract), Input Binding System (for keybinding settings contract — Story 007)
- Unlocks: None directly (other systems consume `settings.cfg`)
