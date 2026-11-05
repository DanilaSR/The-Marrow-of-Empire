# UX Spec: Load Screen

> **Status**: In Design
> **Author**: Awgyrog + ux-designer
> **Last Updated**: 2026-08-10
> **Journey Phase(s)**: Session start / in-game
> **Template**: UX Spec

---

## Purpose & Player Need

The Load Screen serves one purpose: to let the player return to exactly the
story they left.

The player arrives here wanting to **continue**. They have saves — each one a
moment in their investigation frozen in time — and they're choosing which thread
to resume. From the Title Screen, this is the start of a session: anticipation,
the breath before stepping back into the borderland. From the Pause Menu, it's
a tactical choice: reloading to try a different approach, or recovering from
a wrong turn.

The screen's job is to make every save slot legible as a moment in time —
"this is where you were, this is what you were doing, this is who you were
with" — so the player can pick the right one with confidence. The thumbnail,
the region name, the playtime all answer one question: "What will I return
to if I load this?"

If this screen was hard to use or untrustworthy, the player would second-guess
their choice, load the wrong save, lose progress, or worse: encounter a
corrupted save with no clear path to recovery. The Load Screen must handle
failure gracefully — a corrupted save is a problem with a path to resolution
(restore from backup, delete, or cancel), not a dead end. The Steam Cloud
conflict dialog, when it appears, is the same promise in a different form:
your saves are safe, and you are in control of which version to keep.

---

## Player Context on Arrival

The Load Screen has two distinct arrival contexts, each with a different
emotional register:

**From the Title Screen**: The player has just launched the game, passed the
riverside camp title screen, and clicked "Load Game." They are starting a
session — calm, expectant, choosing their point of re-entry. The emotional
register is anticipation: "Which story am I returning to?"

**From the Pause Menu**: The player is already in the game world. They pressed
Escape, opened the pause overlay, and clicked "Load." The world is frozen
behind them at the moment they paused. The load is a deliberate break from the
current timeline — they're about to leave this version of events behind:
- **Tactical reload**: A fight went badly, a choice had unintended consequences.
  They want to try again. Tense, decisive.
- **Session switch**: Switching to a different character or thread. Curious,
  exploratory.
- **Corrupted state recovery**: Something went wrong and they're reverting to
  a known-good save. Frustrated, remedial — the screen must feel reliable.

In all Pause Menu cases, the player is about to leave the current game state
behind. If they have unsaved progress, the Load Screen must warn them — "you
will lose unsaved progress" — before proceeding. This is not a barrier; it's
information the player deserves.

---

## Navigation Position

The Load Screen is reached from two different parents, and its back-destination
depends on which one the player came from:

**Path A**: `[root] → Title Screen → Load Screen`
**Path B**: `[root] → Title Screen → InGame → Pause Menu → Load Screen`

It is never a top-level destination. Both paths share the same screen — the Load
Screen remembers its entry point to determine the back-destination.

---

## Entry & Exit Points

### Entry Points

| Entry Source | Trigger | Player carries this context |
|---|---|---|
| Title Screen | Player clicks "Load Game" | No active game state. Save slot metadata for all 22 slots. Steam Cloud conflict resolved before this screen appears (if applicable). |
| Pause Menu | Player clicks "Load" | Active game state frozen behind 30% dim. Save slot metadata for all 22 slots. Unsaved progress warning context if applicable. |

### Exit Points

| Exit Destination | Trigger | Notes |
|---|---|---|
| InGame | Player clicks an occupied slot → load begins | One-way — loading replaces the current game state. The load operation begins: crossfade to warm soot (300ms), load the save, crossfade in to the game world. |
| Title Screen | Back button or Escape (when entered from Title Screen) | Returns to Title Screen. No confirmation. |
| Pause Menu | Back button or Escape (when entered from Pause Menu) | Returns to Pause Menu. No confirmation. |
| Title Screen | Load failed (corrupted save, version mismatch) | Error toast, then player returns to Title Screen or the Load Screen remains open with the corrupted slot flagged. |
| Title Screen | Steam Cloud conflict unresolved | On game launch, the conflict dialog appears before the Title Screen. If "Decide Later" is chosen, the Load Screen opens but saves may be out of sync. |

---

## Layout Specification

The Load Screen shares the Save Screen's vertical card list layout (GDD CR-2:
"same slot-list layout"). The arrangement, zones, and card structure are
identical. Only the interactions and conditional states differ.

### Information Hierarchy

Same ranking as Save Screen:

1. **Save slot list** (primary): Scrollable vertical list, sorted by last-modified
   descending. Each occupied slot is a card — thumbnail left, metadata right.
   The player scans their save history to choose where to resume.
2. **Screen header** (secondary): "Load Game" — context label.
3. **Empty slots** (discoverable): Dimmed, clearly distinct, but **non-interactive**
   (unlike Save Screen — you can't create a new save here).
4. **Back button** (available, not dominant): Returns to Title Screen or Pause
   Menu depending on entry point.
5. **Corrupted save dialog / Steam Cloud dialog** (conditional overlays): Appear
   only on corrupted slot click or on-launch cloud conflict.

### Layout Zones

Same as Save Screen — top bar + scrollable slot list + overlay zone. See
`design/ux/save-screen.md` Layout Zones for the full zone breakdown. The only
structural addition is the Steam Cloud conflict dialog, which appears as a modal
at the same overlay level as the corrupted save dialog.

### Component Inventory

Components shared with Save Screen use the same patterns. New or changed
components are noted.

| # | Component | Zone | Type | Content | Interactive | Pattern |
|---|---|---|---|---|---|---|
| 1 | Back button | Top bar | Button | "← Back" — returns to entry point (Title Screen or Pause Menu) | Yes | Menu Button |
| 2 | Screen header | Top bar | Label | "Load Game" | No | — |
| 3 | Slot card (occupied) | Slot list | Card | Thumbnail, slot name, character, playtime, region, timestamp | Yes — click to **load** | Save Slot Card |
| 4 | Thumbnail | Slot card | Image | 320×180 PNG | No | — |
| 5 | Slot name | Slot card | Text | Player-given or system name | No | — |
| 6 | Metadata row | Slot card | Text | Character · playtime · region · timestamp | No | — |
| 7 | Empty slot row | Slot list | Card | "Empty Slot" — dimmed, **non-interactive** | No | Empty Slot |
| 8 | Corrupted save dialog | Overlay | Modal | "This save file is corrupted." + Restore from backup / Delete save / Cancel | Yes | Corrupted Save Dialog (new) |
| 9 | Steam Cloud conflict | Overlay | Modal | "Your local and cloud saves differ." with both versions' details + Keep Local / Keep Cloud / Decide Later | Yes | Cloud Conflict Dialog (new) |
| 10 | Unsaved progress warning | Overlay | Modal | "Load game? Unsaved progress will be lost." + Load Anyway / Cancel | Yes | Confirmation Dialog |
| 11 | Scrollbar | Slot list | Scroll | Vertical scroll | Yes | — |
| 12 | System slot row | Slot list | Card | Auto-save / Quicksave — click to load | Yes | Save Slot Card |

**New patterns**: Corrupted Save Dialog, Cloud Conflict Dialog.
**Reused patterns**: Menu Button (Title Screen), Save Slot Card + Empty Slot (Save Screen), Confirmation Dialog (Title Screen + Save Screen).

---

## States & Variants

| State | Trigger | What Changes |
|---|---|---|
| **Default (with saves)** | Normal entry, one or more save files exist | Slot list shows occupied slots (sorted by last-modified descending) followed by empty slots. Occupied slots clickable; empty slots dimmed + non-interactive. Back button active. |
| **Empty (no saves)** | Entry when no save files exist on disk | All 22 slots show dimmed "Empty Slot." All slots non-interactive. Back button is the only available action. |
| **Unsaved progress warning** | Player clicks an occupied slot from the Pause Menu entry, and unsaved progress exists | Modal: "Load game? Unsaved progress will be lost." with the selected save's slot name, thumbnail, and metadata. "Load Anyway" (amber) / "Cancel" (ink). Skipped entirely when entered from Title Screen (no active game state to lose). |
| **Corrupted save** | Player clicks a slot flagged as corrupted (checksum mismatch) | Modal: "This save file is corrupted." with oxblood knot glyph. Options: "Restore from backup" (if `.conflict_backup` exists), "Delete save" (oxblood, destructive), "Cancel" (ink). |
| **Steam Cloud conflict** | Game launch detects local vs. cloud save mismatch | Modal appears before the Load Screen: "Your local and cloud saves differ." Shows both versions: timestamp, playtime, region for each. "Keep Local" / "Keep Cloud" / "Decide Later" (proceeds to Load Screen with conflict unresolved; auto-saves disabled until resolution). |
| **Loading** | Player confirms load (after unsaved progress warning if applicable) | Crossfade to warm soot `#141210` over 300ms. Load Screen not visible. Input locked. On success: crossfade into game world at the loaded save point. On failure: return to Load Screen error state. |
| **Load failed** | Corrupted save detected during load, version mismatch, or file read error | Returns to Load Screen with error toast: oxblood knot glyph + failure description. The affected slot is flagged with a visual corruption marker (oxblood icon on the card). Toast auto-dismisses after 5s or on click/Escape. All other slot cards remain interactive — the player can load a different save. |

---

## Interaction Map

Input methods: Keyboard/Mouse (primary) + Gamepad (partial). No touch support.

### Navigation

Same as Save Screen: Tab/Shift+Tab through slot cards in list order, D-pad
Up/Down for gamepad, Escape for back, Page Up/Down for scroll.

### Slot Interactions

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Occupied slot card | Click / select | Mouse click, Enter/Space, Gamepad A | Amber edge on hover, card highlight on press | If entered from Pause Menu with unsaved progress: opens unsaved progress warning. Otherwise: begins load immediately (Loading state). |
| Empty slot row | — | — | No hover response, no click, non-interactive cursor | — |
| System slot row | Click / select | Same as occupied | Same feedback | Loads the auto-save or quicksave. Unsaved progress warning applies if from Pause Menu. |

### Unsaved Progress Warning

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Load Anyway | Click / select | Mouse click, Enter (default), Gamepad A | Amber edge on hover, thread-red underline on press | Begins load operation — transitions to Loading state. |
| Cancel | Click / select | Mouse click, Escape, Gamepad B | Ink edge on hover | Dismisses modal, returns to slot list. |

### Corrupted Save Dialog

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Restore from backup | Click / select | Mouse click, Enter (default if backup exists), Gamepad A | Amber edge on hover | Restores the `.conflict_backup` file. If successful: slot card updates to show restored save. If no backup exists: this option is greyed out. |
| Delete save | Click / select | Mouse click, Gamepad A | Oxblood edge on hover (destructive) | Deletes the corrupted save file. Confirmation: "Delete this save permanently?" Slot becomes empty. |
| Cancel | Click / select | Mouse click, Escape, Gamepad B | Ink edge on hover | Dismisses modal, returns to slot list. |

### Steam Cloud Conflict Dialog

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Keep Local | Click / select | Mouse click, Enter, Gamepad A | Amber edge on hover | Local saves overwrite cloud saves. Conflict resolved. Proceed to Load Screen. |
| Keep Cloud | Click / select | Mouse click, Gamepad A | Amber edge on hover | Cloud saves overwrite local saves. Conflict resolved. Proceed to Load Screen. |
| Decide Later | Click / select | Mouse click, Gamepad B | Ink edge on hover | Conflict unresolved. Proceed to Load Screen. Auto-saves disabled until resolved. Unchosen version kept as `.conflict_backup` for 30 days. |

### Error Toast (Load Failed)

Same as Save Screen error toast: oxblood knot glyph, 5s auto-dismiss, click/Escape dismiss, slot list remains fully interactive.

---

## Events Fired

| Player Action | Event Fired | Payload |
|---|---|---|
| Load Screen opens | `LoadScreen_Opened` | `entry_point: "title_screen"/"pause_menu"`, `save_count: N` |
| Click occupied slot | `LoadSlot_Selected` | `slot_id`, `slot_name` |
| Unsaved progress warning shown | `Load_UnsavedWarning_Shown` | `slot_id` |
| Load confirmed (after warning or direct) | `Load_Confirmed` | `slot_id` |
| Load cancelled | `Load_Cancelled` | `slot_id` |
| Corrupted save detected on click | `Save_Corrupted_Detected` | `slot_id`, `error_type` ("checksum" / "version_mismatch") |
| Restore from backup | `Save_Restore_Backup` | `slot_id` |
| Delete corrupted save | `Save_Deleted` | `slot_id` |
| Steam Cloud: Keep Local | `CloudConflict_Resolved` | `resolution: "local"` |
| Steam Cloud: Keep Cloud | `CloudConflict_Resolved` | `resolution: "cloud"` |
| Steam Cloud: Decide Later | `CloudConflict_Deferred` | — |
| Load completed | `Load_Completed` | `slot_id`, `load_time_ms` |
| Load failed | `Load_Failed` | `slot_id`, `error_type` |
| Back to previous screen | `LoadScreen_Closed` | `entry_point` |
| Scroll list | none | — |

**State-modifying actions**: `Save_Deleted` permanently removes a save file.
`Load_Confirmed` triggers a full game-state replacement (load operation).
`CloudConflict_Resolved` triggers a cloud sync operation. All three must be
atomic — flagged for architecture.

---

## Transitions & Animations

All motion follows the art bible 6.4 desk register. Same as Save Screen for
shared behaviors; loading-specific transitions noted below.

### Screen Enter / Exit

| Transition | Visual | Duration |
|---|---|---|
| Enter from Title Screen | Crossfade: Title Screen fades out, Load Screen fades in | 250ms |
| Enter from Pause Menu | Crossfade: Pause Menu fades out, Load Screen fades in | 250ms |
| Exit to Title Screen (back) | Crossfade to Title Screen | 250ms |
| Exit to Pause Menu (back) | Crossfade to Pause Menu | 250ms |

### In-Screen State Changes

| State Change | Visual | Duration |
|---|---|---|
| Unsaved progress warning appears | Slot list dims to 30%; modal slides up | Dim: 200ms; slide: 300ms |
| Unsaved progress warning dismissed | Modal slides down; dim lifts | Slide: 200ms; dim lift: 200ms |
| Corrupted save dialog appears | Same dim + slide up | Dim: 200ms; slide: 300ms |
| Corrupted save dialog dismissed | Slide down + dim lift | Slide: 200ms; dim lift: 200ms |
| Steam Cloud conflict appears | Modal over dimmed title screen background | Dim: 200ms; slide: 300ms |
| Loading begins | Crossfade to warm soot `#141210` | 300ms |
| Loading succeeds | Crossfade from soot into game world (at loaded save point) | 500ms |
| Loading fails | Return to Load Screen with error toast | Instant cut back + toast slide-up 200ms |
| Error toast appears | Slides up from bottom edge | 200ms |
| Error toast dismissed | Slides down | 200ms |

### Reduced-Motion Alternative

When Reduce Camera Motion is "Disabled": all motion collapses to 0ms. Screen
crossfades are retained (250ms/300ms — spatial transitions, not cosmetic
animations). The loading crossfade to soot is retained (it masks the load
operation, not for aesthetic effect).

---

## Data Requirements

| Data | Source System | Read / Write | Notes |
|---|---|---|---|
| Save slot metadata (all 22 slots) | Save/Load System | Read | Same schema as Save Screen. Loaded on screen entry. |
| Save file data | Save/Load System | Read | Loaded on player confirmation. This screen does not read save files directly — it delegates to the load orchestration layer. |
| Corrupted save flag | Save/Load System | Read | Checksum validation result. Determines whether the corrupted save dialog appears. |
| Steam Cloud conflict data | Save/Load System / Steam API | Read | Local vs. cloud save metadata comparison. Only relevant at game launch. |
| Unsaved progress flag | Game State | Read | True if the player has made changes since the last save. Only relevant when entered from Pause Menu. |
| Backup file existence | Save/Load System | Read | `.conflict_backup` file check for the "Restore from backup" option. |
| Save deletion | Save/Load System | Write (via event) | Triggered by "Delete save" in corrupted save dialog. |
| Cloud sync | Save/Load System / Steam API | Write (via event) | Triggered by "Keep Local" or "Keep Cloud" resolution. |

The Load Screen is read-only for game state. All write operations (delete,
restore, cloud sync) are event-triggered and delegated to the Save/Load System.
The screen is a display and trigger surface, not a data owner.

---

## Accessibility

No `design/accessibility-requirements.md` exists yet — accessibility tier not yet
defined for this project. WCAG-AA is assumed as a baseline.

### Keyboard-Only Navigation

Same as Save Screen: Tab/Shift+Tab through slot cards in list order. Enter/Space
to select. Escape to dismiss dialogs or return to previous screen. Focus ring
oxblood 2px on parchment, ≥3:1 contrast. Focus never disappears.

**Additional**: The Steam Cloud conflict dialog must be fully keyboard-navigable
before the Load Screen is interactable. Tab order: local version info → cloud
version info → Keep Local → Keep Cloud → Decide Later.

### Gamepad Navigation

Same as Save Screen: D-pad Up/Down through slots, A to select, B to cancel/back.

### Color-Independent Communication

- Corrupted save slots: oxblood corruption icon + "Corrupted" label text +
  non-interactive state — three redundant cues.
- Unsaved progress warning: modal text is the primary communication; amber
  "Load Anyway" button is a redundant emphasis.

### Reduced Motion

Modal slide/dim collapse to 0ms. Screen crossfades and loading crossfade
retained (functional transitions, not cosmetic).

### Minimum Resolution

1280×720. Same slot card scaling as Save Screen.

### Screen Reader Support

Open question — Kaiju Engine limitation, same as all UX specs.

### Accessibility Tier

Not yet defined. WCAG-AA assumed.

---

## Localization Considerations

Same risks as Save Screen with these additions:

| Element | English | Risk | Notes |
|---|---|---|---|
| "Load Game" header | "Load Game" | Medium | "Spiel laden" (11 chars) |
| "Load Anyway" | "Load Anyway" | Medium | Button width |
| "Restore from backup" | "Restore from backup" | **High** | "Aus Sicherung wiederherstellen" (30 chars) — long button |
| "Delete save" | "Delete save" | Medium | Destructive button, oxblood accent |
| Steam Cloud dialog body | Multi-line text | **High** | "Your local and cloud saves differ. Which would you like to keep?" + two save summaries — modal must handle ~40% text expansion |
| "Keep Local" / "Keep Cloud" | Buttons | Medium | Button pair, must be clearly distinct |
| "Decide Later" | "Decide Later" | Medium | "Später entscheiden" (18 chars) |
| Corrupted save dialog | "This save file is corrupted." | Medium | Modal body text |

**Primary risk**: The Steam Cloud conflict dialog body is the longest text on any
menu screen. It must fit in a 520px modal with ~40% expansion in German while
still displaying both save summaries legibly.

---

## Acceptance Criteria

- [ ] **Performance**: Load Screen opens within 250ms crossfade. Save slot
  metadata loads and displays before the crossfade completes.

- [ ] **Navigation — dual entry**: Entered from Title Screen, clicking Back
  returns to Title Screen. Entered from Pause Menu, clicking Back returns to
  Pause Menu. The screen correctly remembers its entry point.

- [ ] **Core purpose — load**: Clicking an occupied slot from the Title Screen
  begins the load immediately: crossfade to warm soot (300ms), load the save,
  crossfade into the game world. The player arrives at the saved game state.

- [ ] **Unsaved progress warning**: When entered from Pause Menu with unsaved
  progress, clicking an occupied slot shows the warning modal: "Load game?
  Unsaved progress will be lost." with the selected save's details. "Load
  Anyway" proceeds; "Cancel" returns to the slot list. When entered from
  Title Screen (no active game state), this warning is skipped.

- [ ] **Corrupted save**: Clicking a corrupted save slot shows the dialog:
  "This save file is corrupted." with Restore from backup, Delete save, and
  Cancel options. Restore recovers the backup; Delete removes the corrupted
  file and empties the slot; Cancel returns to the list. If no backup exists,
  Restore is greyed out.

- [ ] **Steam Cloud conflict**: On game launch with a cloud conflict, the
  conflict dialog appears before the Load Screen. Both versions are shown with
  timestamp, playtime, and region. Choosing "Keep Local" or "Keep Cloud"
  resolves the conflict. Choosing "Decide Later" proceeds to the Load Screen
  with auto-saves disabled.

- [ ] **Empty state**: When no save files exist, all 22 slots display as
  "Empty Slot" and are non-interactive. Only the Back button is available.

- [ ] **Accessibility**: Tab navigates through all slot cards. Enter/Space
  activates the focused slot. Steam Cloud conflict dialog is fully
  keyboard-navigable (Tab order: version info → Keep Local → Keep Cloud →
  Decide Later). Focus ring always visible.

- [ ] **Resolution floor**: At 1280×720, at least 2 full slot cards are visible
  without scrolling. Thumbnails scale proportionally. All text remains legible.

---

## Open Questions

- **Player journey map not created**: Same gap as all UX specs.
- **Accessibility tier not defined**: Same gap as all UX specs.
- **Kaiju Engine screen reader support**: Unknown.
- **Pause Menu UX spec not yet created**: The Load Screen's back-destination
  from Pause Menu entry is a provisional contract. Create `design/ux/pause-menu.md`
  before implementing.
- **Save/Load System data contract**: Slot metadata schema needs confirmation.
- **Steam Cloud API integration**: Does Kaiju Engine have Steamworks bindings,
  or is this a game-layer concern? Resolve before implementing the conflict
  dialog (GDD open question).
- **Backup retention**: `.conflict_backup` files are kept for 30 days per GDD,
  but the cleanup mechanism (on launch? on save? cron?) is undefined.
- **Conflict detection timing**: Does the Steam Cloud conflict check run at
  game launch (before the Title Screen) or when the Load Screen opens? The
  spec assumes launch-time; confirm with architecture.
- **Title Screen UX spec cross-reference**: The Title Screen's "Load Game"
  button states that it is greyed out when no saves exist. The Load Screen's
  empty state should match — the player should never reach an empty Load
  Screen via the Title Screen path (the button blocks it).
