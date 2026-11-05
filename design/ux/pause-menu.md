# UX Spec: Pause Menu

> **Status**: In Design
> **Author**: Awgyrog + ux-designer
> **Last Updated**: 2026-08-10
> **Journey Phase(s)**: In-game
> **Template**: UX Spec

---

## Purpose & Player Need

The Pause Menu is the player's in-game escape hatch. Pressing Escape freezes
the world and presents five options: Resume, Save, Load, Settings, and Quit to
Main Menu. It is the bridge between playing and managing — a moment of stillness
where the player can step away, save their progress, adjust a setting, or leave
the game entirely.

The player arrives here wanting to **pause**. The world stops instantly — NPC
schedules freeze, combat halts, weather pauses, wildlife holds. The Pause Menu
must appear immediately with no perceptible delay, because the player's trust
that "pause means pause" is absolute.

## Player Context on Arrival

The player was just in the game world — exploring, fighting, or talking. They
pressed Escape (or an equivalent gamepad button). The game world dims behind a
30% overlay. The pause is always voluntary, always instant, and always complete.
Nothing advances while the menu is open.

From combat, the battlefield remains visible through the dim — the player can
see positions, health bars, and turn order. No tactical information is obscured.

## Navigation Position

`[root] → Title Screen → InGame → Pause Menu`

The Pause Menu is always accessible during gameplay. It is blocked during
cutscenes (Escape skips the cutscene instead), region transitions, and
mid-save operations.

## Entry & Exit Points

### Entry

| Source | Trigger | Context |
|---|---|---|
| InGame | Escape key (rebindable), Gamepad Start/Menu button | World freezes instantly. Game state at moment of pause. |

### Exit

| Destination | Trigger | Notes |
|---|---|---|
| InGame | Resume, Escape again, Gamepad B | World unfreezes. No time has passed. |
| Save Screen | Save | World remains frozen behind Save Screen. |
| Load Screen | Load | World remains frozen behind Load Screen. Unsaved progress warning if applicable. |
| Settings Screen | Settings | World remains frozen behind Settings Screen. Graphics category limited. |
| Title Screen | Quit to Main Menu → confirm | If unsaved progress: "Quit to Main Menu? Unsaved progress will be lost." Options: Save & Quit / Quit Without Saving / Cancel. |

## Layout Specification

### Information Hierarchy

1. **Game world (dimmed)** — the frozen battlefield/exploration behind 30% dim. Visible but not interactive. The pause menu is an overlay, not a replacement.
2. **Menu buttons** — Resume, Save, Load, Settings, Quit to Main Menu. Centered, vertical stack. Resume is the default focus.
3. **Quit to Main Menu** — separated from the four primary options by a gap, secondary styling.

### Layout Zones

```
┌─────────────────────────────────────┐
│                                     │
│         [dimmed game world]         │
│         at 30% brightness           │
│                                     │
│              Resume                 │
│              Save                   │
│              Load                   │
│              Settings               │
│                                     │
│         Quit to Main Menu           │
│                                     │
└─────────────────────────────────────┘
```

| Zone | Position | Purpose |
|---|---|---|
| **Dim overlay** | Full viewport, z=0 | Soot `#141210` at 72% opacity over frozen game world. The world is visible but muted. Combat UI renders above the dim layer. |
| **Menu panel** | Centered, z=1 | Five buttons in a vertical stack on parchment-toned panel, or transparent buttons over the dim. |

### Component Inventory

| # | Component | Type | Interactive | Pattern |
|---|---|---|---|---|
| 1 | Resume | Button | Yes — closes menu, unfreezes world | Menu Button |
| 2 | Save | Button | Yes — opens Save Screen | Menu Button |
| 3 | Load | Button | Yes — opens Load Screen | Menu Button |
| 4 | Settings | Button | Yes — opens Settings Screen | Menu Button |
| 5 | Quit to Main Menu | Button | Yes — confirmation required if unsaved progress | Menu Button (secondary) |
| 6 | Dim overlay | Panel | No | — |

## States & Variants

| State | Trigger | What Changes |
|---|---|---|
| **Default** | Escape pressed during exploration or dialogue | Five buttons, Resume focused by default. World frozen behind 30% dim. |
| **Combat pause** | Escape pressed during tactical combat | Same layout. Combat UI (health bars, AP, turn order, ability bar) renders above the dim layer — the player can see the battlefield. World freeze guarantees no combat logic advances. |
| **Quit confirmation** | Player clicks "Quit to Main Menu" with unsaved progress | Modal: "Quit to Main Menu? Unsaved progress will be lost." Save & Quit (amber) / Quit Without Saving (oxblood) / Cancel (ink). |
| **Restricted** | Escape pressed during cutscene, region transition, or mid-save | Menu does not open. Cutscene: Escape skips the cutscene instead. Transition/save: input is locked. |

## Interaction Map

Input: Keyboard/Mouse (primary) + Gamepad (partial).

| Component | Action | Input | Feedback | Outcome |
|---|---|---|---|---|
| Resume | Click | Mouse, Enter (default focus), Gamepad A, Escape | Amber edge on hover | Menu closes, world unfreezes. 200ms fade-out. |
| Save | Click | Mouse, Enter/Space, Gamepad A | Amber edge on hover | Opens Save Screen. 250ms crossfade. |
| Load | Click | Mouse, Enter/Space, Gamepad A | Amber edge on hover | Opens Load Screen. 250ms crossfade. |
| Settings | Click | Mouse, Enter/Space, Gamepad A | Amber edge on hover | Opens Settings Screen. 250ms crossfade. |
| Quit to Main Menu | Click | Mouse, Enter/Space, Gamepad A | Amber edge on hover | If unsaved: quit confirmation. If clean: crossfade to Title Screen. |
| Navigate buttons | Up/Down | D-pad Up/Down, Arrow keys | Oxblood focus ring | Wrap from Quit back to Resume. |

## Events Fired

| Action | Event | Payload |
|---|---|---|
| Pause Menu opened | `Pause_Opened` | `context: "exploration"/"combat"/"dialogue"` |
| Resume | `Pause_Resumed` | — |
| Navigate to Save/Load/Settings | None (each screen fires its own open event) | — |
| Quit confirmed | `Game_QuitToMenu` | `saved_before_quit: true/false` |
| Pause cooldown ignored | None | Rapid Escape presses within 300ms are ignored — cooldown prevents flicker. |

## Transitions & Animations

| Transition | Visual | Duration |
|---|---|---|
| Menu opens | World dims to 30% brightness; menu buttons fade in | Dim: 200ms; buttons: 200ms |
| Menu closes (Resume) | Buttons fade out; dim lifts | 200ms |
| Menu closes (to another screen) | Crossfade to Save/Load/Settings | 250ms |
| Quit confirmation appears | Dim + modal slide up | Dim: 200ms; slide: 300ms |
| Reduced motion | All collapse to 0ms | — |

## Data Requirements

| Data | Source | R/W | Notes |
|---|---|---|---|
| Unsaved progress flag | Game State | Read | True if changes since last save. Drives quit confirmation. |
| Game context (exploration/combat/dialogue) | Game State | Read | Determines whether combat UI renders above dim. |
| Pause cooldown timer | Engine | Read | 300ms minimum between pause/unpause transitions. |

The Pause Menu owns the pause freeze. All other systems must respect it — no
game logic advances while the pause state is active.

## Accessibility

Same baseline as all specs. Keyboard: Tab through buttons, Enter/Space to select,
Escape to Resume. Gamepad: D-pad Up/Down, A to select, B to Resume. Focus ring
always visible. Escape has one job in this context: Resume (not pause-toggle flicker
— the cooldown prevents double-trigger).

## Localization

Button labels are the primary risk. "Quit to Main Menu" (19 chars) → German
"Zurück zum Hauptmenü" (21 chars) — moderate. Buttons are centered and flex to
content width, so expansion is accommodated naturally.

## Acceptance Criteria

- [ ] **Performance**: Pause Menu opens within one frame of Escape press. World
  freeze is immediate — no game logic advances between key press and freeze.
- [ ] **Resume**: Clicking Resume, pressing Escape, or pressing Gamepad B closes
  the menu and unfreezes the world. No game time has passed.
- [ ] **Navigation**: Save, Load, and Settings buttons correctly open their
  respective screens with the Pause Menu as the back-destination.
- [ ] **Combat pause**: During combat, the combat UI renders above the dim layer
  and remains fully legible. No combat logic advances while paused.
- [ ] **Quit with unsaved progress**: Clicking "Quit to Main Menu" with unsaved
  progress shows the confirmation dialog. "Save & Quit" saves then quits.
  "Quit Without Saving" quits immediately. "Cancel" returns to the menu.
- [ ] **Cooldown**: Pressing Escape twice within 300ms does not toggle the
  pause state. The second press within the cooldown window is ignored.

## Open Questions

- Player journey, accessibility tier, Kaiju screen reader: same gaps as all specs.
- **Pause freeze mechanism**: How does the engine suspend the update loop?
  Does Kaiju expose a global pause flag, or must each system check it?
- **Cutscene skip**: Escape during cutscenes skips instead of pausing. Does
  the cutscene system expose a "skippable" flag, or is this a universal behavior?