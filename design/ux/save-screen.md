# UX Spec: Save Screen

> **Status**: In Design
> **Author**: Awgyrog + ux-designer
> **Last Updated**: 2026-08-09
> **Journey Phase(s)**: In-game / session end
> **Template**: UX Spec

---

## Purpose & Player Need

The Save Screen serves one purpose: to give the player confidence that their
progress is safe, so they can stop playing without anxiety.

The player arrives here wanting to **preserve their story**. They've just been
exploring the borderland, questioning NPCs, fighting tactical encounters — and
now they need to step away. The Save Screen says: "Your progress is real.
Your investigation is recorded. You can leave now, and everything will be
exactly as you left it."

If this screen didn't exist or was hard to use, the player would either keep
playing past their comfort zone (anxiety) or stop playing and lose progress
(frustration). Both outcomes erode trust in Pillar 2 — The World Remembers.
The save system is the tangible proof of that promise.

The screen must make two things effortless: creating a new save (empty slot →
name → confirm) and updating an existing one (occupied slot → overwrite
confirm). It must also make the cost of overwriting legible — "this is what
you're about to replace" — so the player never overwrites by accident.

---

## Player Context on Arrival

The player arrives at the Save Screen from the Pause Menu, which they opened
during active gameplay. The game world is frozen behind them — exactly as they
left it, dimmed to 30%.

The player's emotional state depends on why they're saving:
- **Ending a session**: Calm, reflective. They've played for an hour or two,
  followed threads, fought encounters. They want to bookmark their investigation
  and step away. The save is the last act of the session — it should feel like
  closing a book you're enjoying, with a ribbon to mark your place.
- **Precautionary save**: Tense, tactical. They're about to enter a dangerous
  area or make a consequential choice. They want insurance. The save should be
  fast — a confirmation, not a ceremony.
- **Achievement lock-in**: Satisfied, proud. They just won a tough fight or
  uncovered a major revelation. They want to make it permanent before anything
  can undo it. The save is a small celebration.

The screen must serve all three moods without friction. The returning player
with 15 saves browsing for the right slot needs clarity; the player making
their first-ever save needs guidance. Both arrive at the same screen, but they
see it differently.

---

## Navigation Position

The Save Screen is context-dependent — only accessible during gameplay via the
Pause Menu. It cannot be reached from the Title Screen (saving requires active
game state).

**[root] → Title Screen → InGame → Pause Menu → Save Screen**

There is no alternate entry path. The Save Screen is never a top-level destination.

---

## Entry & Exit Points

### Entry Points

| Entry Source | Trigger | Player carries this context |
|---|---|---|
| Pause Menu | Player clicks "Save" | Game world frozen behind 30% dim. Current character name, region, playtime, and save slot metadata for all 22 slots. |

### Exit Points

| Exit Destination | Trigger | Notes |
|---|---|---|
| Pause Menu | Back button, Escape, or save completed successfully | If save was completed, brief confirmation was shown before returning. |
| Pause Menu | Player clicks "Cancel" on overwrite confirmation dialog | No save performed. Slot list unchanged. |

---

## Layout Specification

### Information Hierarchy

Elements ranked by visual priority:

1. **Save slot list** (primary): The full scrollable list of existing saves fills
   the main content area. The player's eye lands on their most recent saves first
   (sorted by last-modified descending). Each occupied slot is a card — the
   thumbnail is the fastest visual identifier, then the slot name and character
   name. The player should be able to scan their save history at a glance.

2. **Screen header** (secondary): "Save Game" — context confirming where they are.
   Small, top of the content area, out of the way.

3. **Empty slots** (discoverable): Below the last occupied slot, or interspersed
   if the player has deleted saves. Dimmed, clearly distinct from occupied slots.
   They invite action — "Empty Slot" is an affordance: click here to save.

4. **Back button** (available, not dominant): Bottom or top-left. Returns to
   Pause Menu. Always visible but never competing with the slot list.

5. **Overwrite confirmation / Name-entry prompt** (conditional overlays): Appear
   only when the player interacts with a slot. Modal or slide-in panel over the
   slot list.

### Layout Zones

**Arrangement: Vertical card list (Option A).** Full-screen desk-register overlay:
parchment on the dimmed game world (art bible 6.1 desk register). Single-column
scrollable slot list — thumbnail left, metadata right.

```
┌─────────────────────────────────┐
│ ← Back          Save Game       │  Top bar
├─────────────────────────────────┤
│ ┌──────┐ Slot Name              │
│ │      │ Character · 12h 34m    │  Occupied slot card
│ │ thumb│ Region · 2026-08-10    │
│ └──────┘                        │
│ ┌──────┐ Slot Name              │
│ │      │ Character · 8h 12m     │
│ │ thumb│ Region · 2026-08-09    │
│ └──────┘                        │
│ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ │
│        Empty Slot               │  Empty slot
│ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ │
│        Empty Slot               │
├─────────────────────────────────┤
│  Auto-save · 2026-08-10 08:14  │  System slot (distinct styling)
│  Quicksave  · 2026-08-10 08:12 │
└─────────────────────────────────┘
```

**Zone breakdown:**

| Zone | Position | Purpose |
|---|---|---|
| **Top bar** | Full width, top of screen | Back button (left, returns to Pause Menu) and "Save Game" header (center). Fixed — does not scroll. |
| **Slot list area** | Full remaining height below top bar, scrollable | All 22 save slots in a single vertical column, sorted by last-modified descending. System slots (auto-save, quicksave) pinned at the bottom with distinct styling. |
| **Overlay zone** | Full-screen, conditional | Overwrite confirmation dialog and name-entry prompt. Darken the slot list behind a parchment-centered modal. |

### Component Inventory

| # | Component | Zone | Type | Content | Interactive | Pattern |
|---|---|---|---|---|---|---|
| 1 | Back button | Top bar | Button | "← Back" — ink on parchment, returns to Pause Menu | Yes | Menu Button |
| 2 | Screen header | Top bar | Label | "Save Game" — ink, desk serif | No | — |
| 3 | Slot card (occupied) | Slot list | Card | Thumbnail (320×180), slot name, character name, playtime (h:min), region, timestamp | Yes — click to overwrite | Save Slot Card (new) |
| 4 | Thumbnail | Slot card | Image | 320×180 PNG, the player's last view when they saved | No | — |
| 5 | Slot name | Slot card | Text | Player-given name (manual) or "Auto-save" / "Quicksave" (system) | No | — |
| 6 | Metadata row | Slot card | Text | Character name, playtime, region, timestamp — ink on parchment | No | — |
| 7 | Empty slot row | Slot list | Card | "Empty Slot" — dimmed, non-interactive beyond click affordance | Yes — click to create new save | Empty Slot (new) |
| 8 | Overwrite confirmation | Overlay | Modal | "Overwrite [slot name]?" with slot details (thumbnail, metadata), "Overwrite" / "Cancel" buttons | Yes — Overwrite (amber) / Cancel (ink) | Confirmation Dialog |
| 9 | Name-entry prompt | Overlay | Modal | Text input field (default: character name + timestamp), "Save" / "Cancel" buttons | Yes — text input + Save (amber) / Cancel (ink) | Name Entry Prompt (new) |
| 10 | Scrollbar | Slot list | Scroll control | Vertical scroll for slot list | Yes | — |
| 11 | System slot row | Slot list | Card | Auto-save / Quicksave — same metadata as manual slots, distinct styling (pinned, lighter treatment) | Yes — click to overwrite | Save Slot Card |

**New patterns introduced**: Save Slot Card, Empty Slot, Name Entry Prompt. The
Confirmation Dialog pattern reuses the Menu Button pattern defined in the Title
Screen spec.

### ASCII Wireframe

```
┌──────────────────────────────────────────────────────────┐
│  ← Back                        Save Game                │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐  Borderland Crossing                   │
│  │              │  Alaric · 12h 34m                      │
│  │  [thumbnail  │  The River Valley · 2026-08-10 14:22  │
│  │   320×180]   │                                        │
│  └──────────────┘                                        │
│                                                          │
│  ┌──────────────┐  Before the Old Mine                   │
│  │              │  Alaric · 8h 12m                       │
│  │  [thumbnail  │  The Borderland · 2026-08-09 23:47    │
│  │   320×180]   │                                        │
│  └──────────────┘                                        │
│                                                          │
│  ┌──────────────┐  Hermit's Clearing                     │
│  │              │  Alaric · 5h 03m                       │
│  │  [thumbnail  │  The Wildwood · 2026-08-09 18:15      │
│  │   320×180]   │                                        │
│  └──────────────┘                                        │
│                                                          │
│  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ │
│                     Empty Slot                           │
│  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ │
│                     Empty Slot                           │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Auto-save · Alaric · 0h 47m · 2026-08-10 08:14   │ │
│  └────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Quicksave  · Alaric · 0h 32m · 2026-08-10 08:12  │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
└──────────────────────────────────────────────────────────┘

 TOP BAR                              SLOT LIST
 ────────                              ─────────
 · Back button (left)                 · Cards: 320×180 thumbnail + metadata
 · "Save Game" header (center)        · Occupied: full detail, clickable
 · Fixed, does not scroll             · Empty: dashed border, dimmed, clickable
                                      · System slots: lighter, pinned at bottom
                                      · Sorted: last-modified descending
                                      · ~3.5 visible slots at 1080p, scroll for more
```

---

## States & Variants

| State | Trigger | What Changes |
|---|---|---|
| **Default (with saves)** | Normal entry from Pause Menu, one or more save files exist | Slot list shows occupied slots (sorted by last-modified descending) followed by remaining empty slots. Back button active. All interactions available. |
| **Empty (no saves yet)** | First-ever entry, no save files on disk | All 22 slots show dimmed — 20 "Empty Slot" rows, plus auto-save and quicksave rows also empty. Only available action: click an empty slot to create the first save. Overwrite is not possible. |
| **Overwrite confirmation** | Player clicks an occupied slot | Modal over dimmed slot list: "Overwrite [slot name]?" showing the existing save's thumbnail, character name, playtime, region, and timestamp. "Overwrite" (amber accent) / "Cancel" (ink). |
| **Name-entry prompt** | Player clicks an empty slot | Modal with text input field pre-filled: `[character name] · [timestamp]` (e.g. "Alaric · 2026-08-10 14:22"). "Save" (amber accent) / "Cancel" (ink). |
| **Saving** | Player confirms overwrite or new save | Slot card shows brief in-place "Saving…" indicator. All input locked during the write operation (prevents double-saves). The slot list remains visible — the player can see which slot is being written to. |
| **Save complete** | Save written successfully | Slot card updates in place — new thumbnail captured, metadata refreshed, slot name shown. No modal or toast — the change is self-evident. Player remains on the Save Screen. Back button returns to Pause Menu. |
| **Error (save failed)** | Disk full, permission error, or write failure | Toast over the slot list: oxblood knot glyph + "Save failed — unable to write to [path]. Check folder permissions." Auto-dismisses after 5s or on click/Escape. Player remains on Save Screen, can retry or cancel. |

---

## Interaction Map

Input methods: Keyboard/Mouse (primary) + Gamepad (partial). No touch support.

### Navigation

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Back button | Click / select | Mouse click, Enter/Space (focused), Gamepad A; or Escape key | Amber edge on hover, thread-red underline on press | Returns to Pause Menu. No confirmation — leaving without saving is fine. |
| Slot list | Scroll | Mouse wheel, click-drag scrollbar, D-pad Up/Down at list edges, Page Up/Page Down | Standard scroll | Scrolls the slot list vertically. |
| Slot cards | Navigate between slots | Tab/Shift+Tab (vertical), D-pad Up/Down | Oxblood focus ring on focused card | Focus moves through slots in list order. |

### Slot Interactions

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Occupied slot card | Click / select | Mouse click, Enter/Space, Gamepad A | Amber edge on hover, card highlight on press | Opens overwrite confirmation modal for that slot. |
| Empty slot row | Click / select | Mouse click, Enter/Space, Gamepad A | Amber edge on hover/focus | Opens name-entry prompt modal for a new save. |
| System slot row (auto-save, quicksave) | Click / select | Same as occupied slot card | Same feedback | Opens overwrite confirmation. These slots are system-managed but the player can trigger them by clicking. |

### Overwrite Confirmation Modal

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Overwrite | Click / select | Mouse click, Enter (default), Gamepad A | Amber edge on hover, thread-red underline on press | Begins save operation — transitions to Saving state. |
| Cancel | Click / select | Mouse click, Escape, Gamepad B | Ink edge on hover | Dismisses modal, returns to slot list. No save performed. |

### Name-Entry Prompt Modal

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Text input field | Type / edit name | Keyboard (any text characters); Gamepad: accepts pre-filled default | Characters appear in the field | Player edits the save name. Default: `[Character Name] · [YYYY-MM-DD HH:MM]`. |
| Save | Click / select | Mouse click, Enter (in field or button), Gamepad A | Amber edge on hover, thread-red underline on press | Begins save operation with the entered name. |
| Cancel | Click / select | Mouse click, Escape, Gamepad B | Ink edge on hover | Dismisses modal, returns to slot list. No save performed. |

### Error Toast

| Component | Action | Input(s) | Feedback | Outcome |
|---|---|---|---|---|
| Toast | Dismiss | Mouse click, Enter, Escape, Gamepad A/B, or auto-dismiss after 5s | Toast slides away over 200ms (paper-roll away, per art bible 6.4) | Toast dismissed. Slot list remains fully interactive throughout. |

---

## Events Fired

| Player Action | Event Fired | Payload |
|---|---|---|
| Save Screen opens | `SaveScreen_Opened` | `save_count: N` |
| Click occupied slot | `SaveSlot_Selected` | `slot_id`, `slot_name` |
| Overwrite confirmed | `Save_Overwrite_Confirmed` | `slot_id`, `slot_name` |
| Overwrite cancelled | `Save_Overwrite_Cancelled` | `slot_id` |
| Click empty slot | `Save_New_Clicked` | — |
| New save confirmed | `Save_New_Confirmed` | `slot_name` |
| New save cancelled | `Save_New_Cancelled` | — |
| Save completed | `Save_Completed` | `slot_id`, `file_size_bytes` |
| Save failed | `Save_Failed` | `error_type` ("disk_full" / "permission" / "write_error"), `slot_id` |
| Back to Pause Menu | `SaveScreen_Closed` | — |
| Scroll list | none | — |
| Toast dismissed | none | — |

**State-modifying actions**: `Save_Overwrite_Confirmed` and `Save_New_Confirmed` both
trigger disk write operations. Flagged for architecture — save writes must be atomic
(corrupt partial writes must never replace a valid save file; write-to-temp →
rename is the minimum contract).

---

## Transitions & Animations

All motion follows the art bible 6.4 desk register: paper slides, ink appears, no
bounce, no 3D transforms. The Save Screen is a full-screen desk overlay — the
dimmed game world is visible at the edges.

### Screen Enter / Exit

| Transition | Visual | Duration |
|---|---|---|
| Enter from Pause Menu | Crossfade: Pause Menu fades out, Save Screen fades in | 250ms |
| Exit to Pause Menu (back or save complete) | Crossfade: Save Screen fades out, Pause Menu fades in | 250ms |

### In-Screen State Changes

| State Change | Visual | Duration |
|---|---|---|
| Overwrite confirmation appears | Slot list dims to 30%; modal slides up from bottom (paper onto the desk) | Dim: 200ms; slide: 300ms |
| Overwrite confirmation dismissed | Modal slides down; dim lifts | Slide: 200ms; dim lift: 200ms |
| Name-entry prompt appears | Same as overwrite — dim + slide up | Dim: 200ms; slide: 300ms |
| Name-entry prompt dismissed | Same — slide down + dim lift | Slide: 200ms; dim lift: 200ms |
| Saving | Slot card label changes to "Saving..." in place — no animation, instant text swap | 0ms |
| Save complete | Slot card metadata refreshes in place — thumbnail, playtime, timestamp update instantly | 0ms |
| Error toast appears | Slides up from bottom edge | 200ms |
| Error toast dismissed (click or 5s timeout) | Slides down (paper rolls away) | 200ms |

### Reduced-Motion Alternative

When Reduce Camera Motion is set to "Disabled," all motion durations collapse to
0ms — instant cuts. The crossfade is retained (250ms — navigating between screens
is a spatial transition, not a cosmetic animation).

---

## Data Requirements

| Data | Source System | Read / Write | Notes |
|---|---|---|---|
| Save slot metadata (all 22 slots) | Save/Load System | Read | Slot ID, slot name, character name, playtime, region name, thumbnail (320×180 PNG bytes), timestamp. Loaded on screen entry, refreshed on save complete. |
| Current character name | Game State / Entity Framework | Read | Used as default for new save name-entry prompt. |
| Current playtime | Game State | Read | Displayed in new save name-entry prompt default. |
| Current region name | Game State / Region System | Read | Used in save metadata — captured at save time by the Save/Load System, not by this screen. |
| Thumbnail screenshot | Rendering System | Write (via Save/Load) | 320×180 PNG captured at save time. The Save Screen triggers the save; the Save/Load System owns the capture. |
| Save file data | Save/Load System | Write | The Save Screen triggers the write via Save/Load System. This screen does not write files directly — it delegates to the save orchestration layer. |

The Save Screen does not own game state. It triggers save operations through the
Save/Load System and displays the metadata that system provides. All write
operations are delegated — the screen is a display and trigger surface, not a
data owner.

---

## Accessibility

No `design/accessibility-requirements.md` exists yet — accessibility tier not yet
defined for this project. WCAG-AA is assumed as a baseline throughout this spec.

### Keyboard-Only Navigation

| Action | Key(s) |
|---|---|
| Move focus between slot cards | Tab (next), Shift+Tab (previous). Focus order follows list order — top to bottom. |
| Select focused slot card | Enter or Space |
| Dismiss modal / cancel | Escape |
| Scroll slot list | Page Up / Page Down |
| Navigate within modal (Overwrite/Cancel or Save/Cancel) | Tab / Shift+Tab |
| Confirm modal action | Enter |

Focus ring: oxblood `#7A2E28` 2px on parchment, ≥3:1 contrast (art bible 6.5).
Focus never disappears while the screen is active.

### Gamepad Navigation

| Action | Input |
|---|---|
| Navigate slot list | D-pad Up / Down |
| Select focused slot | A |
| Back / cancel / dismiss | B |

### Text Contrast

Ink (`#1C1813`) on parchment (`#E3D5B0`) ≥ 12:1 — exceeds WCAG AA (≥ 4.5:1).
Dimmed "Empty Slot" text: ink at ~40% opacity (~6:1 on parchment) — still passes AA.

### Color-Independent Communication

- Empty vs. occupied slots: distinguished by opacity dimming + "Empty Slot" label
  text + dashed border treatment — three redundant cues, none hue-dependent.
- Selected/focused slot: oxblood focus ring (shape) + amber edge (color) — the
  ring alone identifies the focused card.
- Overwrite (destructive action): oxblood accent (per art bible 6.5), not
  thread-red. The label text ("Overwrite") is the primary communication; the
  accent is a redundant warning.

### Thumbnail

Slot thumbnails (320×180) are not a primary information carrier — slot name and
timestamp identify a save even if the thumbnail is unrecognizable. This satisfies
the art bible 2.5 requirement that no critical information is communicated by
image alone.

### Reduced Motion

Per art bible 6.7: when Reduce Camera Motion is set to "Disabled," modal
slide/dim durations collapse to 0ms (instant cuts). The screen crossfade is
retained (250ms — navigating between screens is a spatial transition, not a
cosmetic animation).

### Minimum Resolution

1280×720 (art bible 6.7 floor). Thumbnails scale proportionally from the
320×180 source. Metadata text must remain legible without clipping at the
minimum supported width. Reference layout designed at 1920×1080.

### Screen Reader Support

Open question — Kaiju Engine's `engine/ui` system may not expose an
accessibility tree. Assess during implementation. Slot names and metadata are
text-based and should be readable if the engine provides any screen reader API.

### Accessibility Tier

Not yet defined for this project. WCAG-AA is assumed as a baseline throughout
this spec. Consider establishing a tier and creating
`design/accessibility-requirements.md` before Pre-Production gate.

---

## Localization Considerations

### Elements with Translation Risk

| Element | English | ~40% expansion (German) | Risk |
|---|---|---|---|
| "Save Game" header | "Save Game" | "Spiel speichern" (15 chars vs 9) | Medium — header width |
| "Empty Slot" label | "Empty Slot" | "Leerer Slot" / "Leerer Speicherplatz" (11–21 chars vs 10) | Medium — card label width |
| Overwrite dialog header | "Overwrite [slot name]?" | "Überschreiben: [name]?" | **High** — modal text + dynamic slot name must fit in 520px modal |
| Overwrite button | "Overwrite" | "Überschreiben" (13 chars vs 9) | Medium — button width |
| Cancel button | "Cancel" | "Abbrechen" (9 chars vs 6) | Medium |
| Save button | "Save" | "Speichern" (8 chars vs 4) | Low |
| "Saving…" label | "Saving…" | "Wird gespeichert…" (17 chars vs 8) | Low — inline label, can expand |
| Error message body | "Save failed — unable to write to [path]" | ~40% longer | Medium — 320px toast must accommodate |
| "Auto-save" label | "Auto-save" | "Automatisch speichern" (20 chars vs 9) | Medium |
| "Quicksave" label | "Quicksave" | "Schnellspeichern" (15 chars vs 9) | Medium |

### Locale-Specific Formatting

| Data | Risk | Notes |
|---|---|---|
| Timestamp | **High** | Must respect locale: DD/MM/YYYY vs MM/DD/YYYY, 12h vs 24h clock. Date/time format is a system-level setting, not per-save. |
| Playtime (hours:minutes) | Low | Numeric only — colon separator is universal. Verify RTL layouts preserve left-to-right number rendering. |
| Region name | Medium | Region names come from game data and must be translated. Confirm the Save/Load System stores locale-independent region IDs, not display names. |
| Character name | Low | Player-entered — no translation needed. May contain non-ASCII characters (verify UTF-8 support in save file paths). |

### Primary Risk

The overwrite confirmation modal text ("Overwrite [slot name]?") is a concat of a
translated prefix, a player-entered slot name of unknown length, and a question
mark. The slot name may be up to ~40 characters of player text. The modal layout
must handle long slot names by wrapping or truncating with ellipsis. Flag for
localization engineer. **HIGH PRIORITY**.

---

## Acceptance Criteria

- [ ] **Performance**: Save Screen opens within 250ms crossfade from Pause Menu.
  Slot metadata loads and displays before the crossfade completes. Measured from
  "Save" button click in Pause Menu to fully visible slot list.

- [ ] **Navigation**: Clicking "Back" or pressing Escape returns to the Pause
  Menu. No confirmation is required — leaving the Save Screen without saving is
  always allowed.

- [ ] **Empty state**: On first-ever entry (no save files), all slots display as
  "Empty Slot." Clicking an empty slot opens the name-entry prompt with a
  pre-filled default name (`[Character Name] · [Timestamp]`).

- [ ] **Core purpose — new save**: Clicking "Save" in the name-entry prompt writes
  a new save file at the selected slot. The slot card updates in place to show
  the new slot name, thumbnail, playtime, region, and timestamp. The player
  stays on the Save Screen.

- [ ] **Core purpose — overwrite**: Clicking an occupied slot opens the overwrite
  confirmation modal showing the existing save's thumbnail, slot name, character
  name, playtime, region, and timestamp. Clicking "Overwrite" initiates the save;
  "Cancel" dismisses the modal and returns to the slot list unchanged.

- [ ] **Error state**: When a save write fails (disk full, permission error),
  an error toast appears with an oxblood knot glyph and a description of the
  failure. The toast auto-dismisses after 5 seconds or on click/Escape. All
  slot cards remain interactive — the player can retry on a different slot.

- [ ] **Accessibility**: Tab navigates through all slot cards in list order (top
  to bottom), including empty slots. The focus ring (oxblood 2px) is visible
  on the focused card at all times. Enter/Space activates the focused card
  (opens overwrite or name-entry prompt). Gamepad D-pad navigates slots; A
  activates; B cancels.

- [ ] **Resolution floor**: At 1280×720, at least 2 full slot cards are visible
  in the list without scrolling. Thumbnails scale proportionally. Slot names
  and metadata text remain legible without clipping.

---

## Open Questions

- **Player journey map not created**: No `design/player-journey.md` exists. The
  save screen's player context (three saving moods: session-end, precautionary,
  achievement lock-in) is defined in this spec, but hasn't been validated against
  a full journey map. Template at `.claude/docs/templates/player-journey.md`.

- **Accessibility tier not defined**: No `design/accessibility-requirements.md`
  exists. WCAG-AA is assumed as baseline throughout this spec.

- **Kaiju Engine screen reader support**: Unknown whether `engine/ui` exposes an
  accessibility tree. Assess during implementation.

- **Pause Menu UX spec not yet created**: The Save Screen's back-destination
  (Pause Menu) is a provisional contract. The Pause Menu must list "Save" as
  a menu option and must handle the return transition when the Save Screen
  dismisses. Create `design/ux/pause-menu.md` before implementing this screen.

- **Save/Load System data contract**: The slot metadata schema (slot ID format,
  which fields are nullable, whether thumbnails are PNG bytes or file paths)
  must be confirmed with the Save/Load System GDD before implementation.

- **Slot name constraints**: Maximum length? Valid character set? Are newlines
  or special characters allowed? Default: 40 characters max, printable ASCII
  + extended Latin-1.

- **Thumbnail capture ownership**: Does the Save/Load System capture the
  screenshot during the save operation, or does the Save Screen trigger a
  screenshot before delegating the write? The GDD says thumbnails are
  "320×180 PNG" — the capture mechanism is an architecture question.
