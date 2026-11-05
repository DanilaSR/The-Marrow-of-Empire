# UX Spec: Title Screen

> **Status**: In Design
> **Author**: Awgyrog + ux-designer
> **Last Updated**: 2026-08-09
> **Journey Phase(s)**: First launch / session start
> **Template**: UX Spec

---

## Purpose & Player Need

The Title Screen serves one purpose: to make the player feel the game's world before
they enter it, and to offer them a clear, calm choice about what to do next.

The player arrives here wanting to **begin** — either a new story, a continued one,
or the act of configuring the experience to their needs. The screen must say: "This
world exists. It is beautiful and dangerous. You are about to enter it — when you're
ready."

If this screen didn't exist, the player would be dropped into the game world with no
tonal preparation, no chance to load a save, and no opportunity to configure settings
before the experience begins. The title screen is the breath before the dive.

On first launch specifically, the screen also serves as the gate for the
accessibility prompt — ensuring the player is comfortable before they take their
first step into the borderland.

---

## Player Context on Arrival

The player arrives at the Title Screen after launching the game and passing the
engine splash. They come voluntarily — they've chosen to spend time in this world.

**The scene**: A small riverside camp at golden hour. The river moves slowly; a
merchant ship or two are moored at the bank. A campfire burns, its smoke rising
gently. The forest watches from the far bank. The atmosphere is calm, warm,
lived-in — this is a place people pass through, a crossing point, not a ruin.
The world is alive and inviting, and it is waiting for you.

**First launch arrival**: The player is new to the borderland. The scene says:
"Welcome. This world is real, and it has been here long before you arrived.
There is much to discover — come explore." The accessibility prompt appears as a
natural part of this welcome: before the journey begins, let's make sure you're
comfortable.

**Returning arrival**: The player has been here before. The camp is the same, but
the feeling is different — reflective, a moment of pause. The scene says: "You've
walked these roads, you've pieced together fragments of the truth, you carry the
weight of what you know. Rest here a moment. The fire is still warm." The menu
appears without the accessibility prompt; the player's story is waiting for them
in the Load Game screen.

The player's emotional state on arrival is **calm expectancy** — not tension, not
urgency. The title screen is a place you can linger. Some players will sit with
the campfire ambience before clicking anything. That is the design intent.

---

## Navigation Position

The Title Screen is the root of the menu hierarchy. It has no parent — it is the
top-level destination and the first screen the player sees at launch.

**[root] → Title Screen**

**Always reachable via**: "Quit to Main Menu" from the Pause Menu (with confirmation
if unsaved progress exists).

**Bypassed by**: Development command-line argument `--load <save_id>` — transitions
directly to the Loading state, skipping the Title Screen entirely. Not a
player-facing path.

---

## Entry & Exit Points

### Entry Points

| Entry Source | Trigger | Player carries this context |
|---|---|---|
| Game launch | Engine boot complete, splash screens finished | No context (first launch) or existing `settings.cfg` + save files (returning) |
| Pause Menu → Quit to Main Menu | Player confirms quit dialog | Unsaved progress warning was shown; player chose to leave |
| Load failed | Save file corrupted or version mismatch during Loading state | Error context describing the failure |

### Exit Points

| Exit Destination | Trigger | Notes |
|---|---|---|
| New Game flow | Player clicks "New Game" | Leads to difficulty selection → character creation → Loading → InGame. If saves exist, confirmation dialog appears first ("Starting a new game will not overwrite your existing saves"). |
| Load Game screen | Player clicks "Load Game" | Opens the save slot browser. Back button returns to Title Screen. Greyed out with tooltip "No saved games found" if no saves exist. |
| Settings screen | Player clicks "Settings" | Settings remembers Title Screen as entry point for back-navigation. |
| Quit to Desktop | Player clicks "Quit to Desktop" | Closes the application. No confirmation unless a save/load operation is in progress. |

---

## Layout Specification

### Information Hierarchy

Elements ranked by visual priority — what the player's eye lands on, in order:

1. **Background scene** (primary): The riverside camp fills the full viewport. Every
   other element sits on top of it. The scene is the first impression — the world
   exists before the menu does. The player's eye lands on the campfire, then the
   river, then the ships, the far forest. The menu hasn't appeared yet.

2. **Game title** (secondary, appears after background): "The Marrow of Empire"
   fades in 300ms after the background. Positioned to read against the scene without
   obscuring its focal points. The title anchors the identity — this is the name of
   the world you're looking at.

3. **Menu options** (tertiary, appears with title): New Game, Load Game, Settings,
   Quit to Desktop. The player's eye reaches these last in the visual sequence —
   the world and its name have already spoken. The options are the invitation to act.

4. **Version number** (discoverable): Small, low-contrast, bottom-right corner.
   Present when the player looks for it, invisible when they don't.

### Layout Zones

**Arrangement: Offset left (Option B).** The UI occupies the left third of the
screen; the scene occupies the full viewport with its focal area in the right
two-thirds, unobstructed. The menu reads as a traveler's journal laid beside the
camp — present but not dominant.

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  ┌──────────────┐                                        │
│  │              │         SCENE (right ⅔)                │
│  │    TITLE     │                                        │
│  │              │     · campfire (central focus)         │
│  │  ──────────  │     · river (midground)               │
│  │              │     · merchant ships (moored)          │
│  │  New Game    │     · far forest (background)          │
│  │  Load Game   │     · golden hour sky                  │
│  │  Settings    │                                        │
│  │  Quit        │                                        │
│  │              │                                        │
│  └──────────────┘                                        │
│                                                    ver   │
└──────────────────────────────────────────────────────────┘
```

**Zone breakdown:**

| Zone | Position | Purpose |
|---|---|---|
| **Scene zone** | Full viewport; right ⅔ is the unobstructed focal area | The riverside camp with subtle ambient motion — drifting smoke from the campfire, slow river current, gentle ship sway, leaves rustling in the far forest. The visual first impression. |
| **Menu zone** | Left ⅓, vertically centered | Game title at top, four menu options stacked below. Rendered as a parchment-toned panel — a traveler's journal or map laid beside the camp. |
| **Version zone** | Bottom-right corner | Small, low-contrast build identifier. Present when looked for, invisible when not. |
| **Overlay zone** | Full-screen, conditional | First-launch accessibility modal only. Darkens the scene behind a parchment-centered modal. |

### Component Inventory

| # | Component | Zone | Type | Content | Interactive | Pattern |
|---|---|---|---|---|---|---|
| 1 | Background scene | Scene | Animated backdrop | Riverside camp at golden hour: campfire smoke, river current, ship sway, forest foliage | No | — |
| 2 | Game title | Menu | Static text | "The Marrow of Empire" in ink on parchment | No | — |
| 3 | New Game button | Menu | Button | "New Game" — ink text on parchment, amber highlight on hover, thread-red underline on focus | Yes | Menu Button (new) |
| 4 | Load Game button | Menu | Button | "Load Game" — same styling. Greyed out if no saves exist, with tooltip "No saved games found" | Yes | Menu Button |
| 5 | Settings button | Menu | Button | "Settings" — same styling | Yes | Menu Button |
| 6 | Quit to Desktop button | Menu | Button | "Quit to Desktop" — secondary styling, separated from primary options by a small gap or horizontal rule | Yes | Menu Button |
| 7 | Version number | Version | Static text | Build version string, small, low-contrast ink on dark stone | No | — |
| 8 | First-launch modal | Overlay | Modal (conditional) | Accessibility prompt: Reduce Camera Motion dropdown, Subtitles toggle, "Remember my choice" checkbox, "Start Game" button | Yes | Accessibility Prompt Modal (new) |

**New patterns introduced**: Menu Button, Accessibility Prompt Modal. Flag for
addition to the interaction pattern library.

### ASCII Wireframe

```
┌─────────────────────────────────────────────────────────────────────┐
│  ╔══════════════════════════════════════════════════════════════╗  │
│  ║                        SCENE AREA                          ║  │
│  ║              ┌──────────────────────────┐                   ║  │
│  ║              │  ┌──────────────────┐    │   ┌───────────┐  ║  │
│  ║              │  │ THE MARROW       │    │   │  ☁  ☁    │  ║  │
│  ║              │  │ OF EMPIRE        │    │   │   golden  │  ║  │
│  ║              │  └──────────────────┘    │   │   hour    │  ║  │
│  ║              │  ─────────────────────  │   │   sky     │  ║  │
│  ║              │                          │   └───────────┘  ║  │
│  ║              │  ▶ New Game              │                   ║  │
│  ║              │    Load Game             │   ≈≈≈≈≈≈≈≈≈≈    ║  │
│  ║              │    Settings              │   ≈≈ RIVER ≈    ║  │
│  ║              │                          │   ≈≈≈≈≈≈≈≈≈≈    ║  │
│  ║              │    Quit to Desktop       │                   ║  │
│  ║              │                          │   ⛵ ──┐         ║  │
│  ║              │  ┌──────────────────┐    │   ship │moored  ║  │
│  ║              │  │ parchment panel  │    │        │        ║  │
│  ║              │  │ (traveler's      │    │   🔥              ║  │
│  ║              │  │  journal)        │    │  /campfire\     ║  │
│  ║              │  │                  │    │  ♨ smoke  ♨     ║  │
│  ║              │  └──────────────────┘    │                 ║  │
│  ║              │                          │  ≈ shore ≈     ║  │
│  ║              └──────────────────────────┘                 ║  │
│  ║                                                          ║  │
│  ║              🌳🌳🌳  far forest, far bank  🌳🌳🌳          ║  │
│  ╚══════════════════════════════════════════════════════════╝  │
│                                                       v0.1.0   │
└─────────────────────────────────────────────────────────────────────┘

 MENU ZONE (left ⅓)                  SCENE (right ⅔)
 ─────────────────                    ──────────────
 · Parchment-toned panel             · Campfire: central visual anchor
 · Title in ink at top               · Smoke drifts upward (animated)
 · 3 primary options stacked         · River: midground, slow current
 · Quit separated below              · 2 ships moored at near bank (gentle sway)
 · ▶ = hover/selected state          · Far forest: background, leaves rustle
                                     · Golden hour sky: warm amber tones
                                     · v0.1.0: version, bottom-right
```

---

## States & Variants

| State | Trigger | What Changes |
|---|---|---|
| **Default (returning player)** | Normal launch with existing `settings.cfg` | Background scene visible with subtle ambient motion. Game title and all 4 menu buttons interactive. Version number visible bottom-right. |
| **First launch** | No `settings.cfg` exists | Background scene dims behind the accessibility prompt modal. Menu buttons visible but non-interactive until modal is dismissed. Modal: Reduce Camera Motion dropdown (default: Reduced), Subtitles toggle (default: On), "Remember my choice" checkbox (default: checked), "Start Game" button. Modal cannot be dismissed without clicking "Start Game." |
| **No saves exist** | Save directory empty or no save files found | "Load Game" button desaturated to ~40% opacity (per art bible 6.5: saturation is life — a dead control is a bleached control). Tooltip on hover: "No saved games found." All other buttons normal. |
| **New Game confirmation** | Player clicks "New Game" while one or more saves exist | Confirmation dialog overlays the scene (desk register: parchment modal, world dimmed behind it): "Starting a new game will not overwrite your existing saves." Options: "Begin New Game" (amber accent) / "Cancel" (ink). |
| **Loading** | Player confirms new game (after difficulty + character) or selects a save to load | Hard cut to warm soot `#141210`. Title screen not visible. Input locked. On success: crossfade into game world. On failure: return to Title Screen error state. |
| **Load failed** | Corrupted save or version mismatch during Loading state | Returns to Title Screen with error toast: oxblood knot glyph + failure description. Toast auto-dismisses after 5s or on click. Menu fully interactive (the player can load a different save, start a new game, or open settings). |

**Not a title screen state**: The accessibility prompt modal can also appear via Settings → "Re-run Accessibility Setup" — same modal, current values pre-populated, "Start Game" returns to Settings. That modal state is owned by the Settings screen, not the Title Screen.

---

## Interaction Map

Input methods: Keyboard/Mouse (primary) + Gamepad (partial). No touch support.

### Menu Buttons

All four menu buttons share the same interaction pattern. Navigation between them:
Tab (forward), Shift+Tab (backward), Gamepad D-pad Up/Down. Wrap-around: Quit →
New Game and vice versa. Focus ring: oxblood 2px on parchment, ≥3:1 contrast
(art bible 6.5).

| Component | Action | Input(s) | Immediate Feedback | Outcome |
|---|---|---|---|---|
| New Game | Click / select | Mouse click, Enter/Space (focused), Gamepad A | Amber edge on hover (the lamp finds it — mouse only), oxblood ring on focus (keyboard/gamepad, per art bible 6.5), thread-red underline on press | If saves exist: confirmation dialog appears. If no saves: straight to difficulty selection → character creation → Loading → InGame. |
| Load Game | Click / select | Mouse click, Enter/Space, Gamepad A | Same hover/press feedback | Opens Load Game screen. |
| Load Game (no saves) | — | — | Button desaturated (~40% opacity), no hover response | — |
| Settings | Click / select | Mouse click, Enter/Space, Gamepad A | Same hover/press feedback | Opens Settings screen (remembers Title Screen as back-destination). |
| Quit to Desktop | Click / select | Mouse click, Enter/Space, Gamepad A | Same hover/focus feedback. Press: oxblood accent (destructive — art bible 6.5 + resolution record). | Closes application. No confirmation unless save/load operation is in progress. |

### Load Game — Tooltip (no saves)

| Component | Action | Input(s) | Immediate Feedback | Outcome |
|---|---|---|---|---|
| Load Game (disabled) | Hover / focus | Mouse hover, Tab to focus | Tooltip appears on hover/focus: "No saved games found." Ink text on parchment chip. | Informational only — no action available. |

### First-Launch Accessibility Prompt Modal

| Component | Action | Input(s) | Immediate Feedback | Outcome |
|---|---|---|---|---|
| Reduce Camera Motion dropdown | Open / select | Mouse click, Enter/Space (focused), Gamepad A | Oxblood ring on focus; dropdown expands with options | Options: Off / Reduced / Disabled. Selection persists instantly. |
| Subtitles toggle | Toggle | Mouse click, Enter/Space, Gamepad A | Oxblood ring on focus; toggle flips On/Off with ink check mark | Value persists instantly. |
| "Remember my choice" checkbox | Toggle | Mouse click, Enter/Space, Gamepad A | Oxblood ring on focus; checkbox fills with ink check when checked | Persists immediately. Checked by default. |
| Start Game button | Click / select | Mouse click, Enter/Space, Gamepad A | Amber edge on hover, thread-red underline on press | Modal dismisses. Settings written to `settings.cfg`. Menu buttons become interactive. Modal cannot be dismissed without clicking this button. |

### New Game Confirmation Dialog

| Component | Action | Input(s) | Immediate Feedback | Outcome |
|---|---|---|---|---|
| Begin New Game | Click / select | Mouse click, Enter (default), Gamepad A | Amber edge on hover (positive action) | Dialog dismisses → difficulty selection → character creation → Loading → InGame. |
| Cancel | Click / select | Mouse click, Escape, Gamepad B | Ink edge on hover (neutral) | Dialog dismisses, returns to Title Screen default state. |

### Error Toast (Load Failed)

| Component | Action | Input(s) | Immediate Feedback | Outcome |
|---|---|---|---|---|
| Toast | Dismiss | Mouse click, Enter, Escape, Gamepad A/B, or auto-dismiss after 5s | Toast slides away over 200ms (paper-roll away, art bible 6.4) | Toast dismissed. Menu remains fully interactive throughout. |

### Non-Interactive

| Component | Action | Feedback | Outcome |
|---|---|---|---|
| Background scene | None | No hover, no click response, no cursor change. The scene is pure atmosphere. | — |
| Game title | None | Static text — no interaction. | — |
| Version number | None | Static text — no interaction. | — |

---

## Events Fired

| Player Action | Event Fired | Payload / Data |
|---|---|---|
| Title screen appears (first launch) | `TitleScreen_FirstLaunch` | `settings_exist: false`, `save_count: 0` |
| Title screen appears (returning) | `TitleScreen_Return` | `settings_exist: true`, `save_count: N`, `last_playtime: timestamp` |
| Click "New Game" | `NewGame_Clicked` | `has_saves: true/false` |
| New Game confirmation accepted | `NewGame_Confirmed` | — |
| New Game confirmation cancelled | `NewGame_Cancelled` | — |
| Click "Load Game" | `LoadGame_Opened` | navigates to Load screen |
| Click "Settings" | `Settings_Opened` | `entry_point: "title_screen"` |
| Click "Quit to Desktop" | `Application_Quit` | — |
| First-launch prompt: "Start Game" | `Accessibility_FirstLaunch_Completed` | `reduce_camera_motion: "Off"/"Reduced"/"Disabled"`, `subtitles: true/false`, `remember_choice: true/false` |
| Load failed — return to title screen | `Load_Failed` | `error_type: "corrupted"/"version_mismatch"`, `save_slot_id` |
| Accessibility modal appears | none | — |
| Background scene loads | none | — |
| Version number display | none | — |

**State-modifying action**: The first-launch accessibility prompt writes Reduce Camera
Motion and Subtitles values to `settings.cfg` on "Start Game." This is the first
write to persistent configuration — it must succeed, or the game must start with safe
defaults per GDD CR-7. Flagged for architecture: `settings.cfg` bootstrap path needs
to handle the case where the file cannot be written on first launch.

---

## Transitions & Animations

All motion follows the art bible 6.4 desk register: paper slides, ink appears, no
bounce, no 3D transforms, no spring. The title screen is screen-space UI — immune to
camera shake by definition (the desk never trembles).

### Screen Enter

| Step | Transition | Duration |
|---|---|---|
| 1. Background scene | Fade in from warm soot `#141210` | 500ms |
| 2. Menu elements (title + buttons) | Fade in, after background is visible | 300ms delay, then fade |
| 3a. First launch — modal appears | Background dims to 30%; modal slides up from bottom (paper onto the desk) | Dim: 200ms; slide: 300ms |
| 3b. First launch — modal dismissed | Modal slides down; dim lifts | Slide: 200ms; dim lift: 200ms |

### Screen Exit

| Exit | Transition | Duration |
|---|---|---|
| To Load Game / Settings | Crossfade (screen A fades out while screen B fades in) | 250ms |
| To New Game (after confirmation + character creation) | Hard cut to warm soot `#141210` (Loading state) | 0ms |
| Quit to Desktop | Application closes — no in-game transition | — |

### In-Screen State Changes

| State Change | Transition | Duration |
|---|---|---|
| New Game confirmation dialog appears | Dim to 30% + dialog slides up from bottom | Dim: 200ms; slide: 300ms |
| New Game confirmation dialog dismissed | Dialog slides down + dim lifts | Slide: 200ms; dim lift: 200ms |
| Error toast appears (load failed) | Slides up from bottom edge | 200ms |
| Error toast dismissed (click or timeout) | Slides down (paper rolls away) | 200ms |
| Load Game button greys out (no saves detected) | Desaturate in place — no motion, just the color leaches out (saturation is life) | ~150ms crossfade to desaturated state |

### Reduced-Motion Alternative

Per art bible 6.7 (accessibility contract): when Reduce Camera Motion is set to
"Disabled," all motion durations collapse to 0ms — instant cuts. The background
scene still fades in (atmospheric, not functional — the fade is the tone, not the
information). UI elements, modals, and toasts appear instantly. No pulses run.
No critical information is gated behind animation — the screen is fully readable
with motion off.

---

## Data Requirements

| Data | Source System | Read / Write | Notes |
|---|---|---|---|
| Save file count (and existence) | Save/Load System | Read | Determines whether "Load Game" is greyed out or interactive. Only count is needed at title screen level — full slot metadata loads when the player opens the Load Game screen. |
| `settings.cfg` existence | Settings persistence layer | Read | Determines first-launch vs. returning state. File path: user data directory. |
| Build version string | Engine / build system | Read | Displayed bottom-right. Read at boot, never changes during session. |
| Reduce Camera Motion setting | `settings.cfg` | Write (first launch only) | Written when player clicks "Start Game" on the accessibility prompt. Default: "Reduced." |
| Subtitles setting | `settings.cfg` | Write (first launch only) | Written when player clicks "Start Game" on the accessibility prompt. Default: On. |
| Remember my choice flag | `settings.cfg` | Write (first launch only) | Written when player clicks "Start Game." Default: true. |

The title screen does not own or manage game state. Its only write path is the
first-launch accessibility prompt → `settings.cfg`. All other data is read-only or
belongs to the screens the title screen navigates to (Load Game screen owns slot
browsing; Settings screen owns configuration persistence).

---

## Accessibility

No `design/accessibility-requirements.md` exists yet — accessibility tier not yet
defined for this project. The following covers what can be specified from the art
bible (6.7), the GDD (CR-7), and standard UX accessibility baseline.

### Keyboard-Only Navigation

| Action | Key(s) |
|---|---|
| Move focus forward | Tab |
| Move focus backward | Shift+Tab |
| Activate focused button | Enter or Space |
| Dismiss dialog / toast | Escape |
| First-launch modal: full navigation | Tab through Reduce Camera Motion dropdown → Subtitles toggle → "Remember my choice" checkbox → Start Game button; Shift+Tab reverses |

**Focus order**: New Game → Load Game → Settings → Quit to Desktop → wraps to
New Game. Focus ring: oxblood `#7A2E28` 2px on parchment, ≥3:1 contrast (art bible
6.5 — amber fails AA on parchment). Focus never disappears: if an element is
reachable, its ring is visible.

### Gamepad Navigation

| Action | Input |
|---|---|
| Navigate button list | D-pad Up / Down |
| Select / activate | A |
| Cancel / back | B (no-op at title screen root; dismisses confirmation dialog or toast) |

### Text Contrast

Ink (`#1C1813`) on parchment (`#E3D5B0`) ≥ 4.5:1 — passes WCAG AA (art bible 2.4).
Version number uses lower contrast intentionally (discoverable, not primary
information).

### Color-Independent Communication

Per art bible 2.5, no game-critical information is encoded by hue alone:
- Button hover uses amber edge (color) + value shift (lighter edge against parchment)
- Button focus uses oxblood ring (color) + 2px outline (shape) — redundant cues
- "Load Game" disabled state uses desaturation + tooltip text + non-interactive cursor, not color alone

### Reduced Motion

Per art bible 6.7: when Reduce Camera Motion is set to "Disabled," all UI motion
durations collapse to 0ms (instant cuts). Background scene fade is atmospheric, not
functional, and retains its fade. No critical information is gated behind animation.

### Minimum Resolution

1280×720 (art bible 6.7 floor). The left-third menu zone must remain wide enough
for readable button text and the full game title at this resolution. Reference
layout designed at 1920×1080.

### Screen Reader Support

Open question — Kaiju Engine's `engine/ui` system (HTML/CSS-like) may not expose an
accessibility tree. Assess during implementation. If unavailable at engine level, a
self-voicing menu narration option (toggle in accessibility settings) may be needed
as a stopgap.

### Accessibility Tier

Not yet defined for this project. WCAG-AA is assumed as a baseline throughout this
spec. Consider establishing an accessibility tier and creating
`design/accessibility-requirements.md` before Pre-Production gate.

---

## Localization Considerations

### Elements with Translation Risk

| Element | English | ~40% expansion (German) | Risk |
|---|---|---|---|
| Game title | "The Marrow of Empire" | May not be translated (proper name — confirm with narrative-director) | Low |
| New Game | "New Game" | "Neues Spiel" (11 chars vs 8) | Medium — button width |
| Load Game | "Load Game" | "Spiel laden" (11 chars vs 9) | Medium — button width |
| Settings | "Settings" | "Einstellungen" (13 chars vs 8) | **High** — nearly doubles in length |
| Quit to Desktop | "Quit to Desktop" | "Zum Desktop" / "Beenden" (11–12 chars vs 16) | Medium — button width |
| Tooltip | "No saved games found" | "Keine Spielstände gefunden" (25 chars vs 20) | Low — tooltip can expand to fit |
| Confirmation dialog body | "Starting a new game will not overwrite your existing saves." | ~40% longer | Low — dialog body text wraps |
| Accessibility modal options | "Off / Reduced / Disabled" | "Aus / Reduziert / Deaktiviert" | Low — dropdown widths accommodate |

### Primary Risk

The left-third menu zone's button width must accommodate ~40% text expansion without
clipping or overflow. **"Einstellungen" (Settings, German) is the stress case** —
the shortest English label produces the longest German one. The parchment panel
design must either:

- Use generous horizontal padding with buttons that flex to content width, or
- Set button width to the longest localized string and accept shorter labels having
  extra space.

Flag for localization engineer. **HIGH PRIORITY**.

### Non-Issues

- **Version number**: Machine-generated, no translation required.
- **Game title**: Proper name. Confirm with narrative-director whether it should
  remain untranslated (standard for CRPG titles). If translated, the title zone
  must accommodate the longest localized rendering.
- **Numbers and dates**: None displayed on this screen. Save slot timestamps appear
  on the Load Game screen, not here.

---

## Acceptance Criteria

- [ ] **Performance**: The title screen background fades in within 500ms (±10%) of
  engine boot completing, and menu elements appear within 300ms (±10%) after the
  background is visible. Measured from the last splash screen frame to fully visible
  menu.

- [ ] **Navigation**: Clicking "Settings" opens the Settings screen. Pressing
  Back/Escape from Settings returns to the Title Screen. The settings screen's
  back-destination is correctly `TitleScreen`, not `PauseMenu`.

- [ ] **Empty state**: When no save files exist, "Load Game" is greyed out
  (desaturated to ~40% opacity), non-interactive, and displays the tooltip
  "No saved games found" on hover/focus.

- [ ] **Core purpose**: On first launch (no `settings.cfg`), the accessibility prompt
  modal appears over the title screen background. The menu buttons are visible but
  non-interactive. Clicking "Start Game" dismisses the prompt, makes the buttons
  interactive, and persists the chosen settings to `settings.cfg`. On subsequent
  launches, the prompt does not appear.

- [ ] **Accessibility**: Tab navigates through all four menu buttons in order
  (New Game → Load Game → Settings → Quit to Desktop → wraps to New Game). The
  focus ring (oxblood 2px) is visible on the focused button at all times, and the
  focused button can be activated with Enter or Space.

- [ ] **Error state**: When a save fails to load (corrupted file or version mismatch),
  the title screen displays an error toast with an oxblood knot glyph and a
  description of the failure. The toast auto-dismisses after 5 seconds or on
  click/Escape. All menu buttons remain interactive during and after the toast.

- [ ] **Resolution floor**: At 1280×720, all menu buttons and the game title are
  fully visible without clipping, and all text meets the minimum 13px legibility
  floor.

---

## Open Questions

- **Player journey map not created**: No `design/player-journey.md` exists. The title
  screen's player context (first launch vs. returning arrival) is defined in this
  spec, but it hasn't been validated against a full journey map. Template available
  at `.claude/docs/templates/player-journey.md`. Create one before Pre-Production
  gate.

- **Accessibility tier not defined**: No `design/accessibility-requirements.md`
  exists. WCAG-AA is assumed as baseline throughout this spec. Define a tier and
  create the requirements doc before Pre-Production gate.

- **Kaiju Engine screen reader support**: Unknown whether `engine/ui` exposes an
  accessibility tree. Assess during implementation. If unavailable, a self-voicing
  menu narration option may be needed as a stopgap.

- **Game title translation**: "The Marrow of Empire" — should this remain
  untranslated (standard for CRPG titles), or be localized? Confirm with
  narrative-director.

- **Title screen background implementation**: Pre-rendered video loop, real-time
  rendered scene, or static image with shader-based motion (clouds, particles)?
  Affects implementation complexity and memory budget. At MVP, a static image with
  particle overlay is the lowest-risk path (per GDD CR-1 open question).

- **Art bible ivory usage ruling pending**: Save slot metadata ink-on-parchment;
  slider thumb amber or ink — ivory stays rationed to empire items. Destructive
  buttons use oxblood, not thread-red. Pending ux-designer confirmation (resolution
  record, 2026-08-09).

- **Character Creation handoff undefined**: The GDD assumes a pre-made character
  template with name entry at MVP. The title screen → difficulty selection →
  character creation → InGame flow depends on a system that doesn't have a GDD yet.
  Contract: menu triggers character creation → creation returns a confirmed
  character → menu begins the game.

- **Kaiju input binding API**: The keybinding screen (Settings, not Title Screen)
  requires runtime access to the action-to-key map. Not a title screen dependency,
  but adjacent — resolving it unblocks the Settings UX spec.
