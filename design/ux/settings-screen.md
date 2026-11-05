# UX Spec: Settings Screen

> **Status**: In Design
> **Author**: Awgyrog + ux-designer
> **Last Updated**: 2026-08-10
> **Journey Phase(s)**: First launch / session start / in-game
> **Template**: UX Spec

---

## Purpose & Player Need

The Settings Screen serves one purpose: to let the player shape the game to
their body, their hardware, and their preferences — once, so they never have
to think about it again.

The player arrives here wanting to **configure**. They might be setting up
for the first time (first launch, straight from the accessibility prompt),
fixing a comfort issue mid-session (audio too loud, camera too fast), or
establishing muscle memory (rebinding keys to match twenty years of CRPG
habit). The screen's job is to make every option findable, every change
immediately obvious, and every setting persistent.

Settings apply instantly on change — no "Apply" button, no "Cancel" that
silently discards. The player sees what they're changing as they change it.
The one exception is display mode and resolution, which use a 3-second
preview with "Keep changes? / Revert" to prevent the player from locking
themselves into an unusable display.

The screen is organized into six categories. From the Title Screen, all six
are available. From the Pause Menu (in-game), the Graphics category is limited
— resolution, fullscreen, and display mode changes are blocked while the game
world is loaded. The screen remembers which path the player came from and
adjusts accordingly.

---

## Player Context on Arrival

**From the Title Screen**: The player hasn't entered the game world yet. They're
setting up — exploring options, establishing preferences. Calm, exploratory,
no time pressure. All six categories are available.

**From the Pause Menu**: The player is in-game. Something is wrong — the camera
is too fast, the music is too loud, a keybinding is uncomfortable. They want to
fix it and get back to playing. Focused, task-oriented. They already know which
category they need. The Graphics category is limited (resolution/fullscreen
changes blocked while the game world is active).

## Navigation Position

Same dual-entry pattern as the Load Screen. The Settings Screen remembers its
entry point to determine the back-destination.

**Path A**: `[root] → Title Screen → Settings Screen`
**Path B**: `[root] → Title Screen → InGame → Pause Menu → Settings Screen`

## Entry & Exit Points

### Entry Points

| Entry Source | Trigger | Player carries this context |
|---|---|---|
| Title Screen | Player clicks "Settings" | All categories available. No active game state. `settings.cfg` loaded. |
| Pause Menu | Player clicks "Settings" | Graphics category limited (no resolution/fullscreen/display mode changes). Active game state behind dim. `settings.cfg` loaded. |

### Exit Points

| Exit Destination | Trigger | Notes |
|---|---|---|
| Title Screen | Back button or Escape (entered from Title Screen) | All settings already persisted — no "save settings" step. |
| Pause Menu | Back button or Escape (entered from Pause Menu) | All settings already persisted. |
| Title Screen | Graphics preview: "Revert" or 15s timeout | Previous display settings restored. Screen returns to Settings at the Graphics category. |
| — (stays on Settings) | Graphics preview: "Keep changes" | New display settings applied. Player stays on Settings Screen. |

---

## Layout Specification

### Information Hierarchy

1. **Category tabs** (primary): The six category labels in a vertical sidebar.
   The player must find their category before they can change anything. Active tab
   is visually distinct — amber edge + ink label weight shift.
2. **Settings content area** (secondary): The controls for the selected category —
   sliders, dropdowns, toggles, keybinding rows. This is where the player acts.
3. **Reset to Defaults** (discoverable): Per-category button at the bottom of the
   content area. Present but not dominant.
4. **Reset All Settings** (discoverable): Footer button. Requires confirmation.
5. **Back button** (available, not dominant): Top-left, returns to entry point.

### Layout Zones

**Arrangement: Left sidebar tabs (Option A).**

```
┌──────┬──────────────────────────────────┐
│ ←    │          Settings                │  Header
├──────┼──────────────────────────────────┤
│ Game │                                  │
│ Cam  │     Category content area        │
│ Audio│     (settings controls for       │
│ Ctrl │      the active category)        │
│ Acc  │                                  │
│ Grph │                                  │
│      │          [Reset to Defaults]     │
├──────┴──────────────────────────────────┤
│          [Reset All Settings]           │  Footer
└─────────────────────────────────────────┘
```

| Zone | Position | Purpose |
|---|---|---|
| **Header** | Full width, top | Back button (left) + "Settings" header (center). Fixed. |
| **Category sidebar** | Left column, ~25% width | Vertical list of 6 category labels. Active category highlighted. Fixed — does not scroll. |
| **Content area** | Right column, ~75% width | Scrollable area showing the controls for the active category. Content changes instantly when switching tabs — no transition. |
| **Footer** | Full width, bottom | "Reset All Settings" button. Fixed. Requires confirmation. |

### Component Inventory

#### Shared (all categories)

| # | Component | Zone | Type | Interactive | Pattern |
|---|---|---|---|---|---|
| 1 | Back button | Header | Button | Yes | Menu Button |
| 2 | Screen header | Header | Label | No | — |
| 3 | Category tab ×6 | Sidebar | Button/tab | Yes — click to switch category | Category Tab (new) |
| 4 | Reset to Defaults | Content area | Button | Yes — confirmation required | Destructive Button |
| 5 | Reset All Settings | Footer | Button | Yes — confirmation required | Destructive Button |

#### Gameplay

| # | Component | Type | Content | Pattern |
|---|---|---|---|---|
| G1 | Difficulty | Dropdown | Narrative / Normal / Tactical | Dropdown |
| G2 | Auto-pause on conversation | Toggle | On / Off | Toggle |
| G3 | Tooltip verbosity | Dropdown | Minimal / Standard / Detailed | Dropdown |
| G4 | Tutorial hints | Toggle | On / Off | Toggle |

#### Camera

| # | Component | Type | Content | Pattern |
|---|---|---|---|---|
| C1 | Edge scroll | Dropdown | Off / Pan Only / Scroll Only / Both | Dropdown |
| C2 | Edge scroll dead zone | Slider | 0–50px | Slider (new) |
| C3 | Edge scroll speed | Slider | 1–30 u/s | Slider |
| C4 | Keyboard pan speed | Slider | 1–30 u/s | Slider |
| C5 | Gamepad pan speed | Slider | 1–30 u/s | Slider |
| C6 | Zoom transition speed | Slider | 1–10 | Slider |
| C7 | Reduce camera motion | Dropdown | Off / Reduced / Disabled | Dropdown |
| C8 | Invert zoom direction | Toggle | On / Off | Toggle |
| C9–C17 | Pan/zoom keybinds | Keybinding row ×9 | Primary + secondary key slot | Keybinding Row (new) |

#### Audio

| # | Component | Type | Content | Pattern |
|---|---|---|---|---|
| A1 | Master volume | Slider | 0–100% | Slider |
| A2 | Music volume | Slider | 0–100% | Slider |
| A3 | SFX volume | Slider | 0–100% | Slider |
| A4 | Ambient volume | Slider | 0–100% | Slider |
| A5 | Dialogue volume | Slider | 0–100% | Slider |
| A6 | Mute when unfocused | Toggle | On / Off | Toggle |
| A7 | Audio output device | Dropdown | List of system audio devices | Dropdown |

#### Controls

| # | Component | Type | Content | Pattern |
|---|---|---|---|---|
| K1–Kn | Per-action keybinding rows | Keybinding row | Action label + primary slot + secondary slot + Restore Default button | Keybinding Row |
| M1 | Mouse sensitivity | Slider | 0.1–2.0 | Slider |
| M2 | Gamepad sensitivity | Slider | 0.1–2.0 | Slider |
| M3 | Invert mouse wheel zoom | Toggle | On / Off | Toggle |

#### Accessibility

| # | Component | Type | Content | Pattern |
|---|---|---|---|---|
| AC1 | Reduce camera motion | Dropdown | Off / Reduced / Disabled | Dropdown |
| AC2 | Subtitles | Toggle | On / Off | Toggle |
| AC3 | Subtitle background opacity | Slider | 0–100% | Slider |
| AC4 | Colorblind mode | Dropdown | Off / Protanopia / Deuteranopia / Tritanopia | Dropdown |
| AC5 | Re-run Accessibility Setup | Button | Opens the first-launch prompt modal with current values | Button |

#### Graphics

| # | Component | Type | Content | Available in-game? | Pattern |
|---|---|---|---|---|---|
| GR1 | Resolution | Dropdown | List of supported resolutions | No (blocked) | Dropdown |
| GR2 | Display mode | Dropdown | Fullscreen / Borderless Window / Windowed | No (blocked) | Dropdown |
| GR3 | FPS cap | Dropdown | 30 / 60 / 120 / Unlimited | Yes | Dropdown |
| GR4 | VSync | Toggle | On / Off | Yes | Toggle |
| GR5 | Texture quality | Dropdown | Low / Medium / High | Yes | Dropdown |
| GR6 | Anti-aliasing | Dropdown | Off / FXAA / SMAA | Yes | Dropdown |

**New patterns**: Category Tab, Slider, Keybinding Row, Dropdown (formalized from Title Screen's first-launch modal).

**Graphics in-game limitation**: GR1 (Resolution) and GR2 (Display mode) are
greyed out with a tooltip when entered from the Pause Menu: "Not available
while in-game. Return to the Title Screen to change this setting."

---

## States & Variants

| State | Trigger | What Changes |
|---|---|---|
| **Default** | Normal entry | Active category shown. All controls interactive (except in-game Graphics limits). Settings apply instantly on change — no "Apply" step. |
| **Graphics preview active** | Resolution or display mode changed | 3-second countdown overlay on the content area: "Keep changes? (Y) / Revert (N)". All other controls locked. Auto-reverts after 15s of no response. The rest of the Settings Screen remains visible. |
| **Reset confirmation** | Player clicks "Reset to Defaults" for a category | Modal: "Reset [category name] to defaults?" with a summary of which settings will change. "Reset" (oxblood, destructive) / "Cancel" (ink). |
| **Reset All confirmation** | Player clicks "Reset All Settings" | Modal: "Reset ALL settings to defaults? This affects every category and cannot be undone." "Reset All" (oxblood) / "Cancel" (ink). |
| **In-game variant** | Entered from Pause Menu | Graphics: Resolution (GR1) and Display mode (GR2) greyed out with tooltip: "Not available while in-game. Return to the Title Screen to change this setting." All other categories and controls normal. |
| **Keybinding conflict** | Player binds a key already assigned to another action | The old action's row shows "Unbound" with an oxblood warning icon. Conflict resolved atomically — each key maps to at most one action. The player must explicitly rebind or restore the unbound action. |
| **First-launch re-run** | Accessibility → "Re-run Accessibility Setup" clicked | The first-launch accessibility prompt modal appears over the Settings Screen with current values pre-populated (not defaults). Same modal as Title Screen's first-launch state. "Start Game" returns to Settings. |

---

## Interaction Map

Input methods: Keyboard/Mouse (primary) + Gamepad (partial). No touch support.

### Shared Controls

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Category tab | Switch category | Mouse click, Tab to sidebar then Up/Down arrows, Gamepad D-pad Up/Down | Amber edge on active tab, ink label weight shift | Content area updates instantly to show that category's controls. No animation — instant swap. |
| Back button | Click | Mouse, Enter/Space, Gamepad A, Escape | Amber edge on hover | Returns to entry point (Title Screen or Pause Menu). All settings already persisted. |
| Reset to Defaults | Click | Mouse, Enter/Space, Gamepad A | Amber edge on hover | Opens reset confirmation modal for the active category. |
| Reset All Settings | Click | Mouse, Enter/Space, Gamepad A | Amber edge on hover | Opens Reset All confirmation modal. |

### Sliders

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Slider track | Drag / nudge | Mouse drag, Left/Right arrow keys (when focused), Gamepad D-pad Left/Right | Amber fill expands/contracts in real time. Numeric value updates next to slider. | Setting value changes continuously during drag. Persisted on release/defocus. |
| Slider (keyboard) | Focus + nudge | Tab to slider, then arrow keys | Oxblood focus ring on slider thumb. Each arrow press nudges by one increment. | Value changes per press. |

### Dropdowns

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Dropdown | Open | Mouse click, Enter/Space (focused), Gamepad A | List expands below/above the field. Selected item has amber tint + ink check glyph. | — |
| Dropdown option | Select | Mouse click, Enter, Gamepad A; Escape or click-away to close without changing | Selected option highlights. List closes. | Value applies instantly. |

### Toggles

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Toggle | Toggle | Mouse click, Enter/Space (focused), Gamepad A | Umber track, parchment knob slides to On/Off position. Ink check glyph in the On state. | Value applies instantly. |

### Keybinding Rows

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Primary/Secondary slot | Click to bind | Mouse click on the slot, Enter/Space (focused), Gamepad A | Slot enters "Listening..." state — amber pulse on the slot label | — |
| Slot (listening) | Press key | Any keyboard key or key + modifier (Shift/Ctrl/Alt) | Key name appears in slot. If conflict: old action's slot shows "Unbound" with oxblood warning. | Binding updated. Slot exits listening state. |
| Slot (listening) | Cancel | Escape while listening | Slot returns to previous binding. | — |
| Restore Default (per action) | Click | Mouse click, Enter/Space, Gamepad A | Binding resets to default. Conflict on the old key, if any, is resolved. | Binding updated. |

### Graphics Preview

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Keep changes | Press Y or click | Y key, mouse click, Gamepad A | Preview overlay dismisses. | New display settings applied permanently. |
| Revert | Press N or click | N key, mouse click, Escape, Gamepad B | Preview overlay dismisses. | Previous display settings restored. |
| Auto-revert | No response for 15s | — | Preview overlay dismisses. | Previous display settings restored. |

### Reset Confirmation Modals

Same interaction pattern as all other confirmation dialogs: amber/oxblood buttons, Enter to confirm, Escape to cancel.

---

## Events Fired

| Player Action | Event Fired | Payload |
|---|---|---|
| Settings Screen opens | `Settings_Opened` | `entry_point: "title_screen"/"pause_menu"` |
| Category switched | `Settings_Category_Switched` | `category: "gameplay"/"camera"/...` |
| Any setting changed | `Setting_Changed` | `setting_id`, `old_value`, `new_value` |
| Setting change applied | `Setting_Applied` | `setting_id`, `value` |
| Reset to Defaults (category) | `Settings_Reset_Category` | `category` |
| Reset All Settings | `Settings_Reset_All` | — |
| Keybinding conflict | `Keybinding_Conflict` | `action_id`, `key`, `conflict_action_id` |
| Keybinding changed | `Keybinding_Changed` | `action_id`, `slot: "primary"/"secondary"`, `key` |
| Graphics preview: Keep | `Display_Change_Confirmed` | `resolution`, `display_mode` |
| Graphics preview: Revert | `Display_Change_Reverted` | `previous_resolution`, `previous_display_mode` |
| Graphics preview: Auto-revert | `Display_Change_AutoReverted` | `previous_resolution`, `previous_display_mode` |
| Re-run accessibility setup | `Accessibility_Setup_ReRun` | — |
| Back to previous screen | `Settings_Closed` | `entry_point` |

**State-modifying actions**: Every `Setting_Changed` event writes to `settings.cfg`.
The Settings Screen is the primary writer of persistent configuration. Writes are
immediate (on change, not on exit). Flagged for architecture — settings persistence
must handle rapid successive writes (e.g., slider drag) without disk thrashing
(debounce to 100ms after last change).

---

## Transitions & Animations

All motion follows art bible 6.4 desk register.

### Screen Enter / Exit

| Transition | Visual | Duration |
|---|---|---|
| Enter from Title Screen or Pause Menu | Crossfade | 250ms |
| Exit to Title Screen or Pause Menu | Crossfade | 250ms |

### In-Screen State Changes

| State Change | Visual | Duration |
|---|---|---|
| Category switch | Content area updates instantly — no animation, no crossfade | 0ms |
| Reset confirmation appears | Dim + modal slide up | Dim: 200ms; slide: 300ms |
| Reset confirmation dismissed | Modal slide down + dim lift | Slide: 200ms; dim lift: 200ms |
| Graphics preview overlay appears | Overlay fades in over content area | 200ms |
| Graphics preview dismissed | Overlay fades out | 200ms |
| Keybinding slot enters "Listening..." | Slot label pulses amber | Continuous until key pressed or Escape |
| Setting value change (slider) | Amber fill moves in real time | Continuous during drag |

### Reduced-Motion Alternative

Crossfades retained (250ms). Modal slide/dim collapse to 0ms. "Listening..." pulse disabled.

---

## Data Requirements

| Data | Source System | Read / Write | Notes |
|---|---|---|---|
| All current settings values | `settings.cfg` | Read (on open), Write (on every change) | Debounced writes — 100ms after last change to avoid disk thrashing during slider drag. |
| Supported resolutions | Rendering System / OS | Read | Enumerated at boot. |
| Audio output devices | Audio System / OS | Read | Enumerated on Audio category open. |
| Keybinding action list | Input System | Read / Write | Action-to-key map. Read at boot and on Controls category open. Written on binding change. |
| First-launch accessibility values | `settings.cfg` | Read / Write | Read for pre-populating the re-run modal. |
| Default values (all settings) | Game constants / config | Read | Used by Reset to Defaults and Reset All. |

The Settings Screen is the primary writer of `settings.cfg`. It does not own the
default values — those are defined by each system (Camera, Audio, Input, etc.)
and exposed as constants. The screen reads defaults for the Reset flow but never
hardcodes them.

---

## Accessibility

No `design/accessibility-requirements.md` exists. WCAG-AA assumed.

### Keyboard-Only Navigation

| Action | Key(s) |
|---|---|
| Switch category | Tab to sidebar, then Up/Down arrows |
| Navigate between controls in category | Tab / Shift+Tab |
| Adjust slider | Left/Right arrows (when slider focused) |
| Open dropdown | Enter or Space (when dropdown focused) |
| Navigate dropdown options | Up/Down arrows |
| Select dropdown option | Enter |
| Toggle a toggle | Enter or Space |
| Bind a key | Click slot with Enter, press desired key, or Escape to cancel |
| Graphics preview: Keep | Y |
| Graphics preview: Revert | N or Escape |
| Dismiss modal / back | Escape |

### Gamepad Navigation

| Action | Input |
|---|---|
| Switch category | D-pad Left to sidebar, then Up/Down |
| Navigate controls | D-pad Up/Down |
| Adjust slider | D-pad Left/Right |
| Activate / select | A |
| Back / cancel | B |

### Slider Accessibility

Sliders display their current numeric value next to the track. The value is
readable without interacting with the slider. Slider increments are coarse
enough to be adjustable via keyboard/gamepad (not free-floating precision).
Minimum increment: 1 for integer sliders, 0.1 for float sliders.

### Color-Independent Communication

- Active category tab: amber edge + ink weight shift (two cues)
- Unbound keybinding: oxblood icon + "Unbound" label text (two cues)
- Greyed-out Graphics options (in-game): desaturation + tooltip text + label (three cues)
- Keybinding conflict: oxblood warning icon + label change (two cues)

### Minimum Resolution

1280×720. At this resolution, the sidebar width must accommodate the longest
category label (see Localization). The content area must fit at least one full
slider row + label without horizontal scrolling.

---

## Localization Considerations

### Category Labels

The sidebar width is driven by the longest category label. In German:

| English | German | Chars |
|---|---|---|
| Gameplay | Spielablauf | 11 |
| Camera | Kamera | 6 |
| Audio | Audio | 5 |
| Controls | Steuerung | 9 |
| Accessibility | Barrierefreiheit | 16 |
| Graphics | Grafik | 6 |

"Barrierefreiheit" (16 chars) is the stress case. The sidebar must accommodate
this label without truncation at minimum resolution.

**HIGH PRIORITY** — sidebar width must be set to fit the longest localized
category label. Do not abbreviate.

### Additional Risks

- Setting labels: many are compound terms ("Edge scroll dead zone," "Subtitle
  background opacity"). All must accommodate ~40% expansion.
- Dropdown options: "Borderless Window" → "Randloses Fenster" (17 vs 18 chars —
  similar length, but some languages will differ more).
- Keybinding action names: must be translated. "Center on Party" → action label
  width drives the keybinding row layout.
- Graphics preview text: "Keep changes? (Y) / Revert (N)" — the Y/N key prompts
  must match the keyboard layout (QWERTZ vs QWERTY — Y and Z are swapped on
  German keyboards).

---

## Acceptance Criteria

- [ ] **Performance**: Settings Screen opens within 250ms crossfade. `settings.cfg`
  is loaded and all control values are populated before the crossfade completes.

- [ ] **Navigation — dual entry**: Entered from Title Screen, Back returns to
  Title Screen. Entered from Pause Menu, Back returns to Pause Menu. All six
  categories available from Title Screen; Graphics resolution/display mode
  greyed out from Pause Menu with explanatory tooltip.

- [ ] **Instant apply**: Changing any setting (slider, dropdown, toggle) applies
  the change immediately and persists to `settings.cfg`. There is no "Apply" or
  "Save" button. The player can change settings across multiple categories,
  press Back, and all changes are retained.

- [ ] **Graphics preview**: Changing resolution or display mode triggers a
  3-second preview overlay with "Keep changes? (Y) / Revert (N)." Pressing Y or
  clicking "Keep" applies the change permanently. Pressing N, Escape, or
  clicking "Revert" restores the previous setting. After 15 seconds of no
  response, the change auto-reverts.

- [ ] **Reset to Defaults**: Clicking "Reset to Defaults" in a category opens a
  confirmation modal listing the settings that will change. Confirming resets
  only that category's settings. Other categories are unaffected.

- [ ] **Keybinding**: Clicking a keybinding slot enters "Listening..." state.
  Pressing a key or key+modifier binds it. If the key is already assigned to
  another action, the old action shows "Unbound" with a warning. Pressing
  Escape while listening cancels. "Restore Default" per action restores the
  default binding.

- [ ] **Accessibility**: Tab navigates through all controls in the active
  category. Arrow keys adjust sliders and navigate dropdowns. Focus ring is
  always visible. All slider values are displayed as text alongside the track.

- [ ] **Resolution floor**: At 1280×720, the sidebar accommodates the longest
  localized category label. The content area shows at least one full control
  row without horizontal scrolling.

---

## Open Questions

- **Player journey map not created**: Same gap as all UX specs.
- **Accessibility tier not defined**: Same gap.
- **Kaiju screen reader support**: Unknown.
- **Kaiju input binding API**: The Controls category requires runtime access to
  the action-to-key map. Does Kaiju expose a binding registry, or must the game
  layer maintain its own? (GDD open question — unblocks the keybinding UI.)
- **Debounce strategy for slider writes**: 100ms after last change is proposed.
  Confirm with architecture — is there an existing debounce utility in the engine?
- **Audio output device enumeration**: Does Kaiju's Audio System expose a
  device list? If not, this dropdown may need an OS-specific integration.
- **Default values source**: Each system (Camera, Audio, Input) owns its
  defaults. The Settings Screen reads them but doesn't define them. Confirm
  the contract — constants file per system? Config struct with `Defaults()`?
- **Pause Menu UX spec**: Not yet created. The Settings Screen's in-game
  back-destination is a provisional contract.
