# Main Menu & Settings

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: All four — the menu is the entry point and configuration surface for the entire experience

## Summary

The Main Menu & Settings system is the player's gateway to The Marrow of Empire —
the title screen that sets the dark fantasy tone, the save/load interface that
makes "continue playing" possible, and the settings hub where every configurable
aspect of the game lives. It provides four screen contexts (Title Screen,
Save/Load, Settings, Pause Menu) with a 7-state screen-flow machine, full
keybinding remapping, and a first-launch accessibility prompt. Settings persist
per-user in `settings.cfg` (JSON), not per-save. The menu is built on Kaiju
Engine's HTML/CSS-like UI system and is the connective tissue between the
player's intent and the game's response.

> **Quick reference** — Layer: `Presentation` · Priority: `MVP` · Key deps: `Save/Load System, Camera System, Kaiju UI System`

## Overview

The Main Menu & Settings system is the player's first and last interaction with
The Marrow of Empire. It is the threshold: the title screen that sets the tone
before a single line of dialogue is spoken, the save/load interface that makes
"continue playing" possible, and the settings hub where the player shapes the
game to their needs — keybindings, audio levels, accessibility options, and
camera preferences.

The system provides four distinct screen contexts, each serving a different
player moment:

- **Title Screen**: The first thing the player sees. A dark fantasy composition —
  the game's visual identity rendered as a still frame — with the game title and
  three choices: New Game, Load Game, and Settings. This is the emotional on-ramp.
- **Save/Load Screens**: The bridge between sessions. The player browses save slots
  (each showing character name, playtime, region, and a thumbnail screenshot),
  creates new saves, loads existing ones, or deletes old ones. Steam Cloud conflict
  resolution lives here. These screens inherit from and extend the Save/Load
  System's data contracts.
- **Settings Screen**: The player's control surface. Every configurable aspect of
  the game lives here, organized by category: Gameplay, Camera, Audio, Controls
  (keybindings), Accessibility, and Graphics. Settings persist per-user in
  `settings.cfg`, not per-save — the player sets things once.
- **Pause Menu**: The in-game overlay. Accessible during gameplay (except during
  cutscenes and region transitions), it offers: Resume, Save, Load, Settings, and
  Quit to Main Menu. The pause menu is a lightweight overlay that dims the game
  world behind it.

The system also owns the **first-launch accessibility prompt** — a one-time modal
over the title screen offering reduced camera motion and subtitles with "remember
my choice," ensuring the game is comfortable before the player enters the world.

Without Main Menu & Settings: the player cannot start, save, load, configure, or
quit the game. It is the connective tissue between the player's intent and the
game's response.

## Player Fantasy

The Main Menu & Settings system serves three distinct player feelings, all before
the first NPC is questioned:

**The threshold is real.** The title screen is not a generic "press start" prompt —
it is the game's visual identity rendered as a still composition. The dark fantasy
borderland, the Gothic ruins, the warm-gold light of a dying sun — all present
before the player clicks anything. The menu says: "This world exists. You are
about to enter it." A player should feel the game's tone before they make their
first choice. The title screen is the first frame of the painting.

**The game respects you.** Every setting the game exposes is an acknowledgement
that players are different. One player needs reduced camera motion to play
comfortably. Another wants edge-scrolling disabled and drag-pan only. A third
rebinds every key to match twenty years of CRPG muscle memory. The settings
screen says: "Play your way." No setting is buried, no preference is silently
discarded, and the accessibility options are presented up front — not hidden in
an "Advanced" tab. The game wants you to be comfortable.

**Your story is safe.** The save/load screens are the tangible proof of Pillar 2
(The World Remembers). Each save slot is a bookmark in the player's personal
investigation — character name, playtime, region, a thumbnail of where they left
off. Loading a save feels like returning to a story in progress, not restoring a
file. And when the player saves, the brief confirmation (a subtle flash, nothing
intrusive) says: "It's safe now. You can leave."

**Reference touchstones**: *Disco Elysium*'s title screen — the world is present
before the menu is; *Divinity: Original Sin 2*'s save/load UI — each slot tells a
story with thumbnail and metadata; *Pillars of Eternity*'s settings
comprehensiveness — the game trusts the player to configure their experience.

The Main Menu & Settings success metric is not "the player enjoyed the settings
screen." It is: the player starts the game immersed, configures it once and never
thinks about settings again, saves with confidence, and returns to exactly the
story they left.

> Note: `creative-director` not consulted — Lean mode. Review manually before production.

## Detailed Design

### Core Rules

**CR-1: Title Screen**

The title screen is the player's first interaction with The Marrow of Empire. It
presents the game's visual identity before any UI element appears:

- **Background**: An ambient scene with subtle motion — a composed dark fantasy
  landscape (ruined archway, golden forest at sunset, the borderland) with slow
  camera drift or gentle environmental animation (drifting particles, swaying
  foliage). No characters, no combat — the world breathing, waiting.
- **Transition in**: After the engine splash and studio logo, the title screen
  fades in over 500ms. The menu elements appear 300ms after the background is
  visible.
- **Menu options**: Three primary choices:
  - **New Game** — begins the new game flow (CR-5)
  - **Load Game** — opens the Load Game screen (CR-2)
  - **Settings** — opens the Settings screen (CR-3)
- **Additional elements**: Version number (bottom-right), "Quit to Desktop"
  (secondary position)
- **First-launch behavior**: On first launch, the accessibility prompt (CR-7)
  appears as a modal over the title screen before any menu interaction. The title
  screen is visible behind it.

**CR-2: Save/Load Screens**

The Save and Load screens share a common slot-list layout. They are distinct
screens (not one combined screen) to clarify the player's intent.

**Save Screen** (accessed from Pause Menu):
- Displays all 22 save slots (20 manual + 1 auto-save + 1 quicksave), sorted by
  last-modified timestamp descending
- Each occupied slot shows:
  - Slot name (player-given for manual saves; "Auto-save" / "Quicksave" for
    system slots)
  - Character name
  - Playtime (hours:minutes)
  - Region name
  - Thumbnail screenshot (320×180, PNG)
  - Timestamp (date + time)
- Empty slots show "Empty Slot" with dimmed styling
- Clicking an occupied slot: confirmation dialog — "Overwrite [slot name]?" with
  details of what will be overwritten. Options: "Overwrite" / "Cancel"
- Clicking an empty slot: opens the name-entry prompt (default: character name +
  timestamp), then saves
- Back button returns to the Pause Menu

**Load Screen** (accessed from Title Screen or Pause Menu):
- Same slot-list layout as Save Screen
- Clicking an occupied slot: begins the load operation — crossfade to black
  (300ms), load the save, crossfade in to the game world
- Empty slots are non-interactive
- Clicking a corrupted save slot: error dialog — "This save file is corrupted."
  Options: "Restore from backup" / "Delete save" / "Cancel"
- Back button returns to the previous screen (Title Screen or Pause Menu)

**Steam Cloud conflict** (on game launch, if local and cloud saves differ):
- A modal dialog appears before the title screen: "Your local and cloud saves
  differ. Which would you like to keep?"
- Shows both versions: timestamp, playtime, region for each
- Options: "Keep Local" / "Keep Cloud" / "Decide Later" (proceeds to title
  screen; conflict unresolved)
- **If "Decide Later" is chosen**: A persistent warning banner is displayed on the
  title screen and in the pause menu: "Auto-save disabled — resolve Steam Cloud
  conflict in Settings → Save/Load." Auto-saves are blocked until the conflict is
  resolved. The player can open Settings → Save/Load at any time to resolve it.
- The unchosen version is kept as `.conflict_backup` for 30 days

**CR-3: Settings Screen**

The Settings screen is organized into six categories, presented as tabs or a
vertical category list:

| Category | Contents |
|----------|----------|
| **Gameplay** | Difficulty, auto-pause on conversation toggle, tooltip verbosity, tutorial hints toggle |
| **Camera** | All settings defined in the Camera System GDD: edge scroll toggle/mode, edge scroll dead zone, edge scroll speed, keyboard pan speed, gamepad pan speed, zoom transition speed, reduce camera motion dropdown, invert zoom direction toggle, and keybinds for Center on Party, Center on Selected, pan directions, zoom in/out, drag pan |
| **Audio** | Master volume, Music volume, SFX volume, Ambient volume, Dialogue volume, mute when unfocused toggle, audio output device selection |
| **Controls** | Keybindings for all actions (CR-6), mouse sensitivity, gamepad sensitivity, invert mouse wheel zoom |
| **Accessibility** | Reduce Camera Motion (Full / Reduced / Off), Extra Close Zoom toggle, Follow Active Unit toggle, Outline Thickness (slider), Marker Size (slider), Marker Opacity (slider), Marker Distance Labels toggle, Occluder Transparency (slider), Subtitles (Off / On), Subtitle Size (Small / Medium / Large), Subtitle Background Opacity (slider), Speaker Name toggle, Colorblind Mode (Off / Protanopia / Deuteranopia / Tritanopia), Text Size (Default / Large / Extra Large), "Re-run Accessibility Setup" button |
| **Graphics** | Resolution dropdown, display mode (Fullscreen / Borderless Window / Windowed), FPS cap (30 / 60 / 120 / Unlimited), VSync toggle, texture quality (Low / Medium / High), anti-aliasing (Off / FXAA / SMAA) |

**Persistence**: All settings apply instantly on change and are saved to
`settings.cfg` in the user data directory. The file format is **JSON**, matching
Kaiju Engine's editor precedent (`editor_settings.go` uses `settings.json` with
`encoding/json`). A write is triggered on change and debounced (dirty-flag + flush
within 500ms of last change or on clean shutdown — never synchronous disk I/O in
the render loop). On boot, `settings.cfg` is parsed via `json.Decode`; parse
errors or missing files trigger the default-creation path. No "Apply" or "Cancel"
button — the player sees changes as they make them. The one exception: resolution
and display mode changes show a 3-second preview with "Keep changes? (Y) /
Revert (N)" to prevent the player from locking themselves into an unusable display
mode. If no response within 15 seconds, changes auto-revert.

**Reset to defaults**: A "Reset to Defaults" button at the bottom of each category
page resets that category's settings. A "Reset All Settings" button in the footer
resets every setting in all categories. Both require confirmation.

**CR-4: Pause Menu**

The Pause Menu is an overlay accessible during gameplay via the Escape key
(rebindable):

- **Activation**: Pressing Escape pauses the game world (full freeze — NPC
  schedules, combat, wildlife, weather progression all stop) and opens the pause
  overlay. The game world behind the overlay dims to 30% brightness over 200ms.
- **Options**:
  - **Resume** — closes the pause menu, unfreezes the world
  - **Save** — opens the Save Screen
  - **Load** — opens the Load Screen
  - **Settings** — opens the Settings Screen (in-game variant: Graphics category
    is limited — no resolution/fullscreen/display mode changes while in-game)
  - **Quit to Main Menu** — confirmation dialog: "Quit to Main Menu? Unsaved
    progress will be lost." Options: "Save & Quit" / "Quit Without Saving" /
    "Cancel"
- **Restricted states**: The pause menu is blocked during:
  - Cutscenes (Escape skips the cutscene instead, if skippable)
  - Region transitions (input is already locked by the Camera System)
  - Mid-save operations (Save/Load System locks input during save write)
- **Combat pause**: Pausing during combat shows the same overlay. The combat UI
  remains visible behind the dim — the player can see the battlefield state while
  paused. No tactical information is obscured.
- **Dismissal**: Pressing Escape again or clicking Resume closes the menu. The
  world unfreezes. Fade-in over 200ms.

**CR-5: New Game Flow**

When the player clicks "New Game":

1. If save files exist: informational dialog — "Starting a new game will not
   overwrite your existing saves." Options: "Begin New Game" / "Cancel"
2. **Difficulty selection**: Present difficulty options (Narrative / Normal /
   Tactical). Exact difficulty parameters are defined in the Tactical Combat
   Engine GDD; the menu provides the selection UI and stores the chosen difficulty
   in the new game session.
3. **Character creation trigger**: The menu hands off to the Character Creation
   system (future GDD — at MVP, a pre-made character template with name entry is
   sufficient). The menu's contract: when the player confirms their character,
   begin the game in the starting region at the defined entry point.
4. Transition: crossfade to black (500ms), load the starting region, crossfade in
   to the game world. The Camera System starts in Exploration state at Default zoom
   on the player character.

**CR-6: Keybindings & Input Remapping**

The Controls category includes a full keybinding interface:

- **Binding model**: Per-action binding — each game action has a primary and
  secondary key slot. The player clicks the action row, presses the desired key
  or key combination, and the binding is updated.
- **Conflict resolution**: If the player binds a key already assigned to another
  action, the old binding is cleared (unassigned) and the new binding takes its
  place. The old action shows "Unbound" with a warning highlight until the player
  assigns a new key or restores the default. No silent double-bindings — every key
  maps to at most one action.
- **Default restoration**: A "Restore Defaults" button per action row and a
  "Restore All Defaults" button for the entire keybinding set.
- **Modifier keys**: Shift, Ctrl, and Alt are valid as modifiers (e.g., Ctrl+S).
  The binding system stores `[modifier] + [key]` pairs.
- **Reserved keys**: Keys with critical system functions (Escape = pause menu,
  F9 = quick-load) are rebindable but display a warning: "This key is reserved
  for [function]. Rebinding may conflict with system behaviour." The player can
  override after acknowledging the warning.
- **Gamepad**: Gamepad bindings use preset profiles at MVP (not per-button
  remappable). The player selects from available preset layouts.
- **Persistence**: Keybindings are saved to `settings.cfg` as part of the Controls
  category. They persist across sessions and are not per-save.

**CR-7: First-Launch Accessibility Prompt**

On the very first game launch (no `settings.cfg` exists):

- A modal dialog appears over the title screen after the initial fade-in
- The modal offers two settings with independent controls:
  - **Reduce Camera Motion**: Full / Reduced (default) / Off
  - **Subtitles**: On (default) / Off
- A "Remember my choice" checkbox is checked by default — both settings are
  persisted to `settings.cfg` on confirmation
- The modal has two paths: click "Continue" (or press Enter) to confirm and
  proceed to the title screen; press Escape to dismiss with the currently-selected
  values (equivalent to "Continue")
- The modal does NOT auto-dismiss on a timer or on click-away
- **Dismissal**: Pressing Escape or clicking "Continue" dismisses the modal with
  the currently-selected values (defaults if unchanged). This matches the Camera
  System GDD's CR-5: Escape/Skip = accessible defaults applied. The player is
  not forced to interact with every control — the defaults are safe.
- The "Continue" button confirms settings and proceeds to the title screen menu
- The modal is fully keyboard-navigable (Tab/arrows + Enter/Escape) and
  gamepad-navigable (D-pad + A/B) with a visible focus indicator on every
  interactive element
- The title screen menu is visible behind the modal but not interactive until
  the modal is dismissed
- In Settings → Accessibility, a "Re-run Accessibility Setup" button re-opens
  this modal (with current values pre-populated; the button is labeled "Apply"
  instead of "Continue")

**CR-8: Navigation Accessibility**

All screens and dialogs in the Main Menu & Settings system are fully navigable by
three input methods independently — no screen requires a specific input device:

- **Keyboard**: Tab/Shift+Tab moves focus between interactive elements. Arrow keys
  navigate within groups (category list, slot grid, setting rows). Enter/Space
  activates. Escape = back/cancel at every level.
- **Gamepad**: D-pad/left stick moves focus. A = activate, B = back/cancel.
  Start/Options = pause toggle (CR-4).
- **Mouse**: Click to focus and activate. Scroll wheel for sliders and lists.

**Focus indicator**: Every focusable element renders a visible focus ring (ink
outline, ≥3:1 contrast against all menu backgrounds, 2px width, DPI-scaled).
Color changes alone (amber highlight, thread-red underline) are secondary cues —
never the sole indicator of focus or selection state. The focus ring is always
present during keyboard/gamepad navigation; it may be hidden during mouse
navigation and shown on first keyboard/gamepad input.

**Reserved key protection**: The Escape key and F9 (quick-load) cannot be
completely unbound. At least one binding must remain for each reserved function.
The system blocks unbinding the last remaining binding for any reserved function
with a warning: "This is the only binding for [function]. It cannot be removed."

### States and Transitions

The Main Menu & Settings system is a screen-flow state machine:

| State | Description | Valid Transitions |
|-------|-------------|-------------------|
| `TitleScreen` | Initial state after engine boot and splash screens. Title screen background visible with menu options. | → `LoadGame`, → `Settings`, → `InGame` (via New Game flow complete) |
| `LoadGame` | Save slot browser for loading a saved game. | → `TitleScreen` (back), → `InGame` (load completes successfully), → `TitleScreen` (load failed) |
| `InGame` | Normal gameplay. The pause menu is NOT visible. | → `PauseMenu` (Escape key), → `TitleScreen` (quit to menu confirmed) |
| `PauseMenu` | In-game overlay. World is frozen. Pause menu options visible. | → `InGame` (resume), → `SaveGame`, → `LoadGame`, → `Settings`, → `TitleScreen` (quit confirmed) |
| `SaveGame` | Save slot browser for saving, accessed from PauseMenu. | → `PauseMenu` (back or save complete) |
| `Settings` | Settings screen. Reachable from TitleScreen or PauseMenu. | → `TitleScreen` (back, if entered from title), → `PauseMenu` (back, if entered from pause) |
| `Loading` | Transitional state. Black screen, save or new-game data being loaded. Input locked. | → `InGame` (load complete), → `TitleScreen` (load failed) |

**Transition rules**:
- All screen-to-screen transitions use a 250ms crossfade (screen A fades out while
  screen B fades in). The `Loading` state uses a hard cut to black — no crossfade;
  the load operation begins immediately.
- The back button (or Escape in sub-screens) always returns to the previous screen.
- `Settings` remembers its entry point (TitleScreen vs PauseMenu) to determine the
  back-destination.
- `InGame` → `TitleScreen` always requires confirmation if unsaved progress exists.
- The accessibility prompt (first launch) is an overlay state that precedes
  `TitleScreen` — it does not replace it; the title screen is visible behind it.

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Save/Load System | Main Menu depends on Save/Load | Receives save slot metadata (character name, playtime, region, thumbnail, timestamp) for display. Sends save/load/delete commands. Receives Steam Cloud conflict data. |
| Camera System | Camera settings read by Main Menu | Main Menu renders the camera settings defined in the Camera System GDD (15 settings). On change, writes the new value to `settings.cfg`; the Camera System reads settings on next frame. The Camera System's first-launch accessibility prompt is owned by Main Menu (CR-7). |
| Input System | Keybindings managed by Main Menu | The Controls → Keybindings screen reads the current binding map from the Input System, displays it, and writes changes back. The Input System is the authoritative owner of the binding map at runtime; `settings.cfg` is the persistence layer. |
| Audio System | Audio settings managed by Main Menu | Settings screen exposes master/music/SFX/ambient/dialogue volume sliders. Changes are sent to the Audio System immediately (instant apply). The Audio System owns the actual mixer; the menu owns the UI. |
| Entity Framework | None direct | The menu does not query entities. Character name in save slots comes from Save/Load metadata, not a live EF query. |
| Rendering System (Kaiju) | Graphics settings applied by Main Menu | Resolution, display mode, VSync, FPS cap, texture quality, anti-aliasing are read from `settings.cfg` at boot and applied via engine renderer configuration. Changes trigger the preview-countdown flow (CR-3). |

## Formulas

The Main Menu & Settings system has no gameplay formulas. It is a screen-flow and
configuration system. Its "calculations" are timing constants and structural rules:

**Transition Timing Constants:**

| Constant | Value | Notes |
|----------|-------|-------|
| `title_fade_in_ms` | 500ms | Title screen background fade-in after splash |
| `menu_element_delay_ms` | 300ms | Delay before menu buttons appear after background |
| `screen_crossfade_ms` | 250ms | Crossfade duration for all screen-to-screen transitions |
| `pause_dim_ms` | 200ms | Time to dim game world behind pause overlay |
| `new_game_fade_ms` | 500ms | Crossfade to black when starting a new game |
| `save_load_fade_ms` | 300ms | Crossfade when loading a save |

**Display Mode Preview:**

```
preview_countdown = 3 seconds
auto_revert_timeout = 15 seconds
```

If no player response within `auto_revert_timeout`, the previous display settings
are restored.

**Save Slot Count:**

```
total_slots = manual_slots + auto_save_slots + quicksave_slots
           = 20 + 1 + 1
           = 22
```

All timing values are game-time (delta-time based), frame-rate independent, and
implemented as tuning knobs (see Tuning Knobs).

> Note: `systems-designer` not consulted — Lean mode. Review manually before production.

## Edge Cases

- **If `settings.cfg` is missing or corrupted on launch**: The system creates a new
  `settings.cfg` with all defaults. The first-launch accessibility prompt is shown
  (as if it were the first launch). No error is shown to the player — the game
  starts with safe defaults.

- **If `settings.cfg` write fails mid-session (disk full, permission change)**:
  The current in-memory settings remain active for the session. An error is logged.
  On next setting change, the system retries the write. No player-facing error
  unless the retry also fails — then a non-blocking toast: "Settings could not be
  saved. Check available disk space." (3-second auto-dismiss).

- **If the player binds a key already reserved for a system function (e.g.,
  Escape for pause)**: The system shows a warning: "This key is reserved for
  [function]." The player can override after acknowledging. The reserved function
  continues to work using any remaining bindings. The system warns if a reserved
  function has zero bindings remaining after the rebind.

- **If the player changes resolution to an unsupported mode**: The 3-second
  preview countdown begins. If the display is unusable, the player does nothing
  for 15 seconds, and the setting auto-reverts to the previous working resolution.
  If the player accidentally confirms (presses Y), the Settings screen remains
  accessible — they can navigate back and change it. The game never permanently
  locks itself into an unviewable display mode.

- **If the player quits during the Loading state** (Alt+F4 while loading a save):
  The save file is not corrupted (the load operation is read-only). On next
  launch, the game starts at the Title Screen normally. Any temporary state written
  during the partial load is cleaned up on next launch.

- **If Steam Cloud is unavailable on launch (network down, Steam offline)**: The
  game launches normally with local saves. Cloud sync is deferred. No error is
  shown unless a save conflict is detected on next sync. The "Decide Later" option
  in the conflict dialog handles the unresolved case.

- **If the player deletes all saves and returns to the Title Screen**: "Load Game"
  is greyed out with a tooltip: "No saved games found." "New Game" remains
  available. No error — this is a valid and expected state.

- **If the save directory does not exist or is not writable**: On first save
  attempt, the system attempts to create the directory. If creation fails
  (permissions), the player sees: "Save failed — unable to write to [path].
  Check folder permissions." The game continues; the player can resolve the
  permissions issue and retry.

- **If the player triggers New Game while a save operation is in progress**: The
  "New Game" button is non-interactive (greyed out) while a save write is in
  progress. Concurrent save/load operations are not permitted.

- **If the player creates a cascading keybinding conflict (A steals from B, B
  steals from C, C steals from A across multiple operations)**: The conflict
  resolution is atomic per operation — only the directly conflicting action is
  unbound. The cascade stops at one hop. The final state is always consistent:
  every key maps to at most one action.

- **If the player presses Escape rapidly (pause/unpause/pause within 500ms)**:
  The pause menu enforces a 300ms cooldown after each state transition. Rapid
  Escape presses within the cooldown are ignored. This prevents visual flicker
  from the dim animation reversing mid-play and prevents the world from
  freezing/unfreezing in rapid succession.

- **If the player accesses the Pause Menu during combat**: The world freeze
  guarantees no game logic advances while the menu is open. A combatant cannot
  die from a status effect, an enemy cannot take a turn, and a timer cannot
  expire during the pause. The freeze is implemented via engine update-loop
  suspension — verified by design, not by hoping.

- **If the first-launch accessibility prompt appears and the display is at an
  unsupported mode at first boot**: Since no `settings.cfg` exists yet, the
  engine boots at a safe default resolution (1280×720 windowed). The prompt is
  always visible and navigable regardless of display hardware.

- **If the game is launched with a development command-line argument that
  bypasses the menu (e.g., `--load save_01`)**: The menu system respects the
  argument and transitions directly to the Loading state with the specified save.
  The Title Screen is skipped. This is a development convenience, not a
  player-facing feature.

## Dependencies

**Systems Main Menu & Settings depends on:**

| System | Direction | Nature of Dependency |
|--------|-----------|---------------------|
| Save/Load System | Main Menu depends on Save/Load | Receives save slot metadata (character name, playtime, region, thumbnail, timestamp) for display. Sends save/load/delete commands. Receives Steam Cloud conflict data. **Hard** — the save/load screens cannot function without save data. |
| Camera System | Main Menu reads Camera settings spec | Main Menu renders the camera settings defined in the Camera System GDD. Settings changes are written to `settings.cfg`; the Camera System reads them on next frame. **Soft** — the Settings screen can render without the Camera System being active (e.g., at the Title Screen), but the Camera category will be empty if the Camera System GDD's settings table is unavailable. |
| Input Binding System | Main Menu depends on Input Binding System | Keybinding screen reads the action-to-key binding map from the Input Binding System and writes changes back. The Input Binding System owns the runtime binding registry (action registry, KeyChord struct, capture, conflict resolution, defaults, serialization). This is a new Foundation-tier system — see `design/gdd/input-binding-system.md`. **Hard** — the keybinding UI cannot function without a binding registry. |
| Audio System (Kaiju Engine) | Main Menu reads/changes Audio settings | Settings screen exposes volume sliders. Changes are sent to the Audio System immediately. **Soft** — the Audio category renders regardless of whether audio is playing. If the Audio System is not initialized, volume sliders are greyed out. |
| Rendering System (Kaiju) | Main Menu applies Graphics settings | Resolution, display mode, VSync, FPS cap, texture quality, anti-aliasing are applied via engine renderer configuration at boot and on change. **Hard** — the Graphics category cannot function without a renderer to configure. |
| Kaiju UI System (`engine/ui`) | Main Menu is built on Kaiju UI | All screens are rendered using Kaiju's HTML/CSS-like UI system. UI elements (panels, buttons, labels, sliders, dropdowns, checkboxes, scroll areas) are Kaiju `engine.Entity` instances managed by `ui.Manager`. **Hard** — engine-level dependency. |

**Systems that depend on Main Menu & Settings:**

| System | Direction | Nature of Dependency |
|--------|-----------|---------------------|
| Save/Load System | Save/Load depends on Main Menu for UI | Save/Load provides data; Main Menu provides the save/load screens and confirmation dialogs. The Save/Load System GDD declares this dependency: "Main Menu needs save slots to display." |
| Camera System | Camera depends on Main Menu for settings UI | Camera defines settings; Main Menu renders the UI for them and persists values to `settings.cfg`. The first-launch accessibility prompt (Camera CR-5) is owned and rendered by Main Menu. |
| Audio System | Audio depends on Main Menu for settings UI | Audio owns the mixer; Main Menu provides the volume sliders and mute toggle that control it. |
| Input Binding System | Input Binding depends on Main Menu for keybinding UI | Input Binding owns the registry; Main Menu provides the keybinding interface that reads and writes it. |

**Undesigned dependencies (provisional contracts):**

| System | Status | Provisional assumption |
|--------|--------|------------------------|
| Character Creation | No GDD — not in systems index | Main Menu hands off to Character Creation after difficulty selection. At MVP, a pre-made character template with name entry is sufficient. Contract: menu triggers character creation → creation returns a confirmed character → menu begins the game. |
| Tactical Combat Engine | GDD exists — In Design | Difficulty settings (Narrative / Normal / Tactical) defined by Combat Engine GDD. Menu provides the selection UI and stores the chosen difficulty. The Combat GDD (`design/gdd/tactical-combat-engine.md`) owns the difficulty definitions. |

**Deferred Engine Features (MVP — engine extensions not required for launch):**

The following settings controls reference engine capabilities that do not yet
exist in Kaiju Engine. At MVP, these controls are greyed out with a "Coming Soon"
tooltip. The menu exposes them so the settings screen structure is complete; the
engine work to enable them is tracked separately.

| Setting | Engine Prerequisite | Owner |
|---------|-------------------|-------|
| VSync toggle | Runtime `SelectPresentMode` reconfiguration | Engine Programmer |
| FPS cap slider | Already exists — `host.SetFrameRateLimit(targetFPS)` in `kaiju/src/engine/host.go:423` | Already available |
| Texture quality (Low/Med/High) | Renderer texture-quality config surface | Technical Artist |
| Anti-aliasing (Off/FXAA/SMAA) | Post-process AA shader pipeline | Technical Artist |
| Audio output device selection | SoLoud device enumeration wrapper | Engine Programmer |
| Steam Cloud sync | Steam RemoteStorage API wrapper | Engine Programmer |
| Low-pass filter on pause | SoLoud filter API wrapper (C layer has it) | Engine Programmer |

> Note: Input Binding System is being split into its own Foundation-tier GDD
> (`design/gdd/input-binding-system.md`). The keybinding UI described in CR-6
> depends on it. See the Input Binding System GDD for the binding registry,
> KeyChord struct, capture semantics, conflict resolution, defaults table, and
> JSON serialization contract.

## Tuning Knobs

| Parameter | Default | Safe Range | Effect of Increase | Effect of Decrease |
|-----------|---------|------------|-------------------|-------------------|
| `title_fade_in_ms` | 500ms | 200–1000 | Slower, more dramatic title reveal | Faster, snappier menu access |
| `menu_element_delay_ms` | 300ms | 100–600 | Longer pause before menu buttons appear | Buttons appear sooner after background |
| `screen_crossfade_ms` | 250ms | 100–500 | Slower, more deliberate screen transitions | Snappier, may feel abrupt |
| `pause_dim_ms` | 200ms | 100–400 | Slower dim animation on pause | Faster dim, may feel jarring |
| `pause_toggle_cooldown_ms` | 300ms | 150–600 | Longer cooldown between pause/unpause | Shorter cooldown, risk of flicker from rapid toggling |
| `new_game_fade_ms` | 500ms | 200–1000 | Longer black screen before new game loads | Shorter transition, may show loading artifacts |
| `save_load_fade_ms` | 300ms | 150–600 | Longer black screen during load | Shorter, may show partial load state |
| `display_preview_countdown_sec` | 3 | 2–10 | More time to evaluate display change | Less time, higher risk of accidental lock-in |
| `display_auto_revert_timeout_sec` | 15 | 5–30 | More time before auto-reverting display change | Faster auto-revert, safer against lock-in |
| `max_save_slots` | 20 | 5–50 | More manual save slots available | Fewer slots, less disk usage (~3MB per slot) |
| `thumbnail_width_px` | 320 | 160–640 | Larger thumbnails in save slots, larger save files | Smaller thumbnails, smaller save files |
| `toast_duration_ms` | 3000 | 1500–5000 | Toast messages stay longer | Toasts dismiss faster |
| `conflict_backup_retention_days` | 30 | 7–90 | Unchosen cloud conflict saves kept longer | Cleaned up sooner |

## Visual/Audio Requirements

### Visual Requirements

The Main Menu & Settings system is the player's first visual impression of the
game. Its visual design must align with the art bible's UI register: **divergent
register, shared vocabulary** — the menu uses parchment/ink/thread-red with empire
ivory rationed, distinguishing it from the warm-gold game world while sharing the
same color language.

**Title Screen Background**: An ambient composed scene from the dark fantasy
borderland. The art bible's warm-gold light language is the default — golden hour,
the ruined archway, the forest breathing. Slow camera drift or subtle particle
motion (drifting embers, pollen, dust motes in sunbeams). The scene is a painting
in motion — no characters, no combat, no UI until the menu fades in. The art
director owns the composition; the Main Menu system owns the fade-in timing and
the UI overlay.

**Menu Styling**:

| Element | Visual Treatment |
|---------|-----------------|
| Menu buttons (New Game, Load Game, Settings) | Parchment-toned background with ink-dark text. Hover: warm gold highlight edge. Active: thread-red underline. |
| Save slot cards | Dark stone card with thumbnail, metadata text in pale ivory (≥12:1 contrast on dark stone). Selected slot: amber gold border + ink underline (two cues — never color alone). Cards never use parchment background — metadata is illegible on parchment (1.17:1). |
| Settings categories | Tab-style or vertical list with ink-dark labels. Active category: amber gold indicator + bold weight (two cues). |
| Sliders | Track in deep stone, fill in amber gold, thumb in pale ivory. |
| Dropdowns | Parchment background, ink text, amber gold selection highlight + checkmark icon. |
| Confirmation dialogs | Darkened overlay (80% black) with parchment-modal foreground. Destructive action buttons (Delete, Quit Without Saving): oxblood fill (#7A2E28) with ivory text (≥4.5:1). Non-destructive buttons: parchment with ink text. |
| Pause menu overlay | 30% brightness dim over the game world — darkens without hiding. The world remains legible through the overlay. |
| First-launch accessibility prompt | Parchment modal over the title screen. Amber gold accent on the active selection. |

**Transitions**:
- All screen transitions: 250ms crossfade. Screen A fades out while screen B
  fades in.
- Title screen: background fades in from black (500ms), menu elements fade in
  after 300ms delay.
- Pause: game world dims to 30% brightness over 200ms. Overlay menu fades in
  simultaneously.
- Loading: hard cut to black (0ms), then fade in to game world on load complete.

### Audio Requirements

| Event | Audio Feedback | Priority |
|-------|---------------|----------|
| Title screen appears | Ambient nature soundscape begins (low volume, establishing mood) | Medium |
| Button hover | Subtle UI hover sound — soft, non-intrusive | Low |
| Button click / menu navigation | UI click — crisp, short, satisfying | Low |
| Pause menu opens | Brief low-pass filter on game audio (world "drops away") + subtle pause sound | Medium |
| Pause menu closes / resume | Audio filter lifts, world returns | Medium |
| Save complete (Save screen) | Subtle confirmation chime | Low |
| Load complete (fade in to game) | Ambient audio fades in with the visual fade | Medium |
| Confirmation dialog appears | Subtle alert tone | Low |
| First-launch prompt appears | None — the title screen ambience continues | — |

> Note: `art-director` and `audio-director` not consulted — Lean mode. Review
> manually before production.

## UI Requirements

The Main Menu & Settings system IS a UI system. Its entire specification is a UI
specification. This section summarizes the UI contracts that other systems should
reference, and flags what needs dedicated UX specs in Pre-Production.

**Screens requiring dedicated UX specs** (run `/ux-design` per screen before
implementation):

| Screen | UX Spec Path | Owner | Notes |
|--------|-------------|-------|-------|
| Title Screen | `design/ux/title-screen.md` | Main Menu | Layout, button placement, background composition brief for art director |
| Save Screen | `design/ux/save-screen.md` | Main Menu | Slot card layout, thumbnail placement, overwrite confirmation dialog |
| Load Screen | `design/ux/load-screen.md` | Main Menu | Same layout as Save; corrupted-save error state, Steam Cloud conflict dialog |
| Settings Screen | `design/ux/settings-screen.md` | Main Menu | 6-category organization, per-category layout, keybinding interface, display preview flow |
| Pause Menu | `design/ux/pause-menu.md` | Main Menu | Overlay layout, dim behavior, combat-state variant |
| First-Launch Prompt | `design/ux/accessibility-prompt.md` | Main Menu | Modal layout, keyboard/gamepad navigation, "remember my choice" control |

**UI contracts owned by this system and consumed by others:**

| Contract | Consumer | Description |
|----------|----------|-------------|
| Settings persistence (`settings.cfg`) | Camera System, Audio System, Input System | All settings are written to and read from `settings.cfg`. Other systems declare their settings schema; Main Menu provides the UI and persistence layer. |
| First-launch accessibility prompt | Camera System | The prompt is defined in Camera CR-5 but owned and rendered by Main Menu (CR-7). Camera specifies the settings; Main Menu renders the modal. |
| Save/Load screen data contract | Save/Load System | Save/Load provides slot metadata; Main Menu renders it. Contract: slot ID, name, character name, playtime, region name, thumbnail (PNG, 320×180), timestamp. |
| Pause state | Camera System, Tactical Combat Engine, AI Dialogue Engine | Main Menu owns the pause freeze. Other systems must respect it — no game logic advances while `PauseMenu` state is active. |

**Cross-system UI flags:**

- Edge scroll must be disabled when the cursor is over any UI panel (Camera System
  CR-4). Main Menu screens are full-viewport UI — edge scroll is suppressed
  entirely while any menu screen is active.
- UI panels render in screen space (Camera System UI Requirements) — they are
  unaffected by camera shake, zoom, or transitions. The menu system is
  screen-space by definition.
- The pause menu dim must not obscure the combat UI during combat pause (CR-4).
  The combat HUD renders above the dim layer.

> 📌 **UX Flag — Main Menu & Settings**: This system requires 6 dedicated UX specs
> before implementation. Run `/ux-design` for each screen in Pre-Production before
> writing epics. Stories that reference menu screens should cite
> `design/ux/[screen].md`, not the GDD directly.

## Acceptance Criteria

> **Measurement protocol**: All timing ACs use an injectable game-time clock and
> fixed-step 60Hz simulation harness with white-box access to UI element state
> (opacity, position, enabled/disabled flags). Fade assertions verify
> `clamp(t_game / duration, 0, 1)` per frame — exact, tolerance-free at the unit
> level. Freeze assertions require each simulation system (NPC schedules, combat,
> wildlife, weather) to expose a `TicksAdvanced()` counter or state snapshot;
> the AC asserts the counter is unchanged across all paused frames. "Within one
> frame" means "by the end of the same update tick" under the frame-stepping
> harness. Performance ACs follow the Camera System GDD precedent: p95 over a
> 3000-frame capture at 1920×1080 on the reference benchmark configuration
> (to be defined in Technical Preferences), measured with Go benchmark
> instrumentation. Wall-clock ±10% tolerances are demoted to bench smoke checks;
> the unit-level assertion is the exact game-time formula.
>
> ACs are tagged with their evidence tier per project standards:
> **Logic** (unit-testable), **Integration** (multi-system), **UI** (automated
> state assertion, or manual walkthrough), **Visual** (screenshot + lead
> sign-off), **Bench** (profiler benchmark).
>
> **Prerequisites**: ACs referencing `settings.cfg` require the JSON settings
> schema to be defined. ACs referencing keybindings require the Input Binding
> System's registry seam. ACs referencing combat require the Tactical Combat
> Engine's tick-counter contract. Deferred-feature controls (VSync, texture
> quality, AA, audio device, Steam Cloud sync) are excluded from automated ACs
> at MVP — they ship greyed out.

### Title Screen & New Game

- **GIVEN** the game launches for the first time (no `settings.cfg` exists), **WHEN**
  the engine finishes boot, **THEN** the title screen background fades in over
  `title_fade_in_ms` (±10%), the first-launch accessibility prompt appears as a
  modal over the title screen, and the title screen menu buttons are visible but
  non-interactive until the prompt is dismissed.

- **GIVEN** the title screen is displayed, **WHEN** the player clicks "New Game"
  with no existing saves, **THEN** the difficulty selection screen appears. After
  difficulty is chosen and character is confirmed, the screen crossfades to black
  and the game begins in the starting region.

- **GIVEN** save files exist, **WHEN** the player clicks "New Game", **THEN** a
  confirmation dialog appears: "Starting a new game will not overwrite your
  existing saves." The new game flow proceeds only after the player clicks
  "Begin New Game."

### Save/Load Screens

- **GIVEN** the player opens the Load Game screen with 5 existing saves, **WHEN**
  the screen loads, **THEN** all 5 occupied slots display character name, playtime,
  region name, thumbnail, and timestamp. Empty slots show "Empty Slot" with
  dimmed styling. Occupied slots are clickable; empty slots are not.

- **GIVEN** the player clicks an occupied save slot in the Save screen, **WHEN**
  the slot is selected, **THEN** an overwrite confirmation dialog appears showing
  details of the existing save. Clicking "Overwrite" begins the save; clicking
  "Cancel" returns to the slot list.

- **GIVEN** a corrupted save file (checksum mismatch), **WHEN** the player clicks
  it in the Load screen, **THEN** an error dialog appears with options "Restore
  from backup," "Delete save," and "Cancel." The game does not attempt to load
  the corrupted file.

- **GIVEN** a Steam Cloud conflict (local and cloud saves differ), **WHEN** the
  game launches, **THEN** a conflict dialog appears before the title screen
  showing both versions with timestamp, playtime, and region. The player's choice
  is applied; the unchosen version is kept as `.conflict_backup`.

### Settings

- **GIVEN** the Settings screen is open to the Camera category, **WHEN** the player
  changes Edge Scroll Speed from 15 to 20 u/s, **THEN** the new value is applied
  immediately to the Camera System and written to `settings.cfg` within one frame.

- **GIVEN** the Settings screen is open to the Graphics category, **WHEN** the
  player changes the resolution, **THEN** a 3-second preview begins with "Keep
  changes? (Y) / Revert (N)." If no response within 15 seconds, the previous
  resolution is restored.

- **GIVEN** the player clicks "Reset to Defaults" in the Camera category, **WHEN**
  the confirmation is accepted, **THEN** all Camera settings return to their
  default values and are written to `settings.cfg`. Other categories are
  unaffected.

- **GIVEN** `settings.cfg` is deleted or corrupted, **WHEN** the game launches,
  **THEN** a new `settings.cfg` is created with all default values. The
  first-launch accessibility prompt is shown. No error is displayed to the player.

### Pause Menu

- **GIVEN** the player is in Exploration state with one scheduled NPC (Schedule component active) and one weather system tick, **WHEN** Escape is pressed under the frame-stepping harness, **THEN** (a) PauseMenu state is entered by the end of that tick, (b) the dim overlay alpha reaches target 0.7 over `pause_dim_ms` game-time following `clamp(t_game / pause_dim_ms, 0, 1) × 0.7`, (c) each simulation system's tick counter is identical across all subsequent paused frames (freeze observable), and (d) the pause menu contains exactly five options: Resume, Save, Load, Settings, Quit to Main Menu. [Integration — freeze contract + UI state]

- **GIVEN** the pause menu is open, **WHEN** the player clicks "Resume" or
  presses Escape, **THEN** the pause menu fades out, the dim lifts, and the game
  world unfreezes. No game time has passed — NPC schedules, combat timers, and
  wildlife positions are identical to when the pause was triggered.

- **GIVEN** the player is in combat, **WHEN** Escape is pressed, **THEN** the
  combat world freezes with the combat UI still visible behind the dimmed
  overlay. All combatants remain in their paused positions. The player can see
  the battlefield state through the overlay.

- **GIVEN** the pause menu is open, **WHEN** the player clicks "Quit to Main Menu"
  with unsaved progress, **THEN** a confirmation dialog appears: "Quit to Main
  Menu? Unsaved progress will be lost." Options: "Save & Quit," "Quit Without
  Saving," "Cancel."

- **GIVEN** two Escape presses separated by less than `pause_toggle_cooldown_ms`
  game-time (default 300ms) under the frame-stepping harness, **WHEN** the second
  press arrives, **THEN** it is ignored (no state transition, no dim animation
  reversal). The pause menu state does not toggle. Boundary: delta == cooldown
  is permitted (second press toggles). [Logic — unit-testable with injected clock]

### Keybindings

- **GIVEN** the Controls → Keybindings screen is open, **WHEN** the player clicks
  an action row and presses a key, **THEN** the key is bound as the primary
  binding for that action. If the key was bound to another action, the old action
  is unbound and shows "Unbound" with a warning highlight.

- **GIVEN** an action has "Unbound" status due to a conflict, **WHEN** the player
  clicks "Restore Defaults" on that row, **THEN** the default keybinding for
  that action is restored. The conflict warning clears.

- **GIVEN** the player attempts to rebind Escape (pause menu), **WHEN** the key
  is pressed in the binding interface, **THEN** a warning appears: "This key is
  reserved for Pause Menu. Rebinding may conflict with system behaviour." The
  player can proceed after acknowledging.

### First-Launch Accessibility Prompt

- **GIVEN** the game launches for the first time, **WHEN** the title screen
  appears, **THEN** the accessibility prompt is displayed as a modal with Reduce
  Camera Motion (default: Reduced), Subtitles (default: On), a "Remember my
  choice" checkbox (default: checked), and a "Continue" button. Pressing Escape
  or clicking "Continue" dismisses the modal with the currently-selected values
  applied to `settings.cfg`. The title screen menu becomes interactive after
  dismissal. [UI — automated state assertion]

- **GIVEN** the accessibility prompt was completed on first launch, **WHEN** the
  game is relaunched, **THEN** the prompt does not appear. The chosen settings
  persist in `settings.cfg`.

- **GIVEN** the player navigates to Settings → Accessibility, **WHEN** "Re-run
  Accessibility Setup" is clicked, **THEN** the accessibility prompt re-opens
  with current values pre-populated.

### Screen Transitions

- **GIVEN** any screen-to-screen transition (Title→Load, Settings→Title,
  Pause→Save, etc.), **WHEN** the transition triggers, **THEN** a crossfade of
  `screen_crossfade_ms` (±10%) occurs. During the crossfade, input is not
  processed (prevents double-clicks across screens).

- **GIVEN** the Loading state is active, **WHEN** a save fails to load (corrupted,
  version mismatch), **THEN** the screen transitions to the Title Screen with an
  error message describing the failure. The game does not crash or hang.

### Performance

- **GIVEN** the Settings screen (all 6 categories rendered, populated with default
  values), **WHEN** profiled over a 3000-frame capture at 1920×1080 on the
  reference benchmark configuration (to be defined in Technical Preferences),
  **THEN** menu UI update + render-command generation ≤ 2ms p95 CPU (measured
  with Go benchmark instrumentation; excludes background scene rendering).
  [Bench — profiler benchmark]

- **GIVEN** the pause menu open during combat with 10 combatants and the full
  combat HUD rendered, **WHEN** profiled over a 3000-frame capture, **THEN**
  frame time ≤ 16.6ms p95, and the number of draw calls in the pause state
  equals the in-combat draw calls + exactly one additional draw call (the dim
  quad). The combat HUD is not re-rendered while paused (assert HUD draw-call
  delta = 0 across paused frames). [Bench — profiler benchmark + draw-call
  instrumentation]

> Note: `qa-lead` not consulted — Lean mode. Review manually before production.

## Open Questions

| Question | Owner | Deadline | Resolution |
|----------|-------|----------|-----------|
| How does the Main Menu hand off to Character Creation? Character Creation is not in the systems index. At MVP, a pre-made character template with name entry is assumed. | Game Designer | Before implementation | — |
| What is Kaiju Engine's input binding API? The keybinding interface (CR-6) requires runtime access to the action-to-key map. Does Kaiju expose this, or does the game layer maintain its own binding registry? | Engine Programmer | Before implementation (ADR) | — |
| What UI elements does Kaiju's `engine/ui` provide for dropdowns, sliders, and scroll areas? The Settings screen requires these. Are they built-in or custom? | Engine Programmer | Before UX specs | — |
| Does Kaiju Engine have a splash screen / studio logo mechanism, or does the game own the full boot sequence including splashes? | Engine Programmer | Before implementation | — |
| Steam Cloud API — what is the Go integration path? Does Kaiju have Steamworks bindings, or is this a game-layer concern? | Engine Programmer | Before implementation (ADR) | — |
| What settings does the Tactical Combat Engine require? Difficulty levels (Narrative/Normal/Tactical) are placeholder names — the Combat GDD owns the actual definitions. | Game Designer | When Tactical Combat Engine GDD is authored | — |
| Should the Settings screen include a "Gameplay" category at MVP, or defer it until gameplay systems (combat, investigation) define their settings? | Game Designer | Before implementation | Recommended: include the category with the subset of settings that exist at MVP (difficulty) and add others as their GDDs are authored. |
| Should the title screen background be a pre-rendered video loop, a real-time rendered scene, or a static image with shader-based motion (clouds, particles)? | Art Director + Technical Artist | Before vertical slice | Affects implementation complexity and memory budget. At MVP, a static image with particle overlay is the lowest-risk path. |
