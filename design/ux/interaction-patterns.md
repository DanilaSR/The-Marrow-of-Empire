# Interaction Pattern Library

> **Status**: In Design
> **Author**: Claude + ux-designer
> **Last Updated**: 2026-08-09
> **Template**: Interaction Pattern Library

## Overview

This library catalogs every interaction pattern used across The Marrow of Empire's
UI. Patterns are the reusable building blocks of the interface — when two screens
use the same pattern, they should feel like the same interaction. The library
exists to prevent divergence and to give new screens a menu of proven patterns
to compose from.

All patterns follow the art bible's desk register (Section 6): flat parchment
panels, umber frames, pen-and-ink icons, paper-slide motion. The palette is
the UI palette (Section 2.4): parchment `#E3D5B0`, ink `#1C1813`, panel umber
`#2E2820`, thread-red `#C4483A`, amber `#D9A441`, oxblood `#7A2E28`.

## Pattern Catalog

| # | Pattern | Category | Used In |
|---|---------|----------|---------|
| P01 | Desk Overlay | Navigation | Title Screen, Pause Menu, Settings, Journal, Character Sheet |
| P02 | Sliding Panel | Navigation | Dialogue UI |
| P03 | Field Instrument | Navigation | Combat HUD |
| P04 | Typewriter Reveal | Feedback | Dialogue UI |
| P05 | Thinking Indicator | Feedback | Dialogue UI |
| P06 | Topic Chips | Input | Dialogue UI |
| P07 | Pen-and-Ink Icon | Data Display | Combat UI, Journal, Map, Character Sheet |
| P08 | Paper-Slide Transition | Animation | All screens |
| P09 | Amber Pulse | Feedback | Journal, Dialogue UI, Combat UI, Character Sheet |
| P10 | Oxblood Pulse | Feedback | Combat UI, Journal |
| P11 | Thread-Red Connector | Data Display | Journal, Combat UI |
| P12 | Save Slot Card | Data Display | Main Menu (Load Game) |
| P13 | Attribute Bar | Input/Display | Character Creation, Character Sheet |
| P14 | Drag-to-Equip | Input | Character & Inventory UI |
| P15 | Item Grid | Data Display | Character & Inventory UI |
| P16 | Turn Order Bar | Data Display | Combat UI |
| P17 | World-Anchored Bar | Data Display | Combat (health bars) |
| P18 | AP Meter | Data Display | Combat UI |
| P19 | Floating Number | Feedback | Combat UI |
| P20 | Confirmation Dialog | Modal | All destructive actions |
| P21 | Fade Transition | Animation | Living Environment (wait/rest), Region Transitions |
| P22 | Free-Text Input | Input | Dialogue UI |
| P23 | Rest Point Dialog | Input | Living Environment (wait mechanic) |
| P24 | Tabbed Panel | Navigation | Character & Inventory UI |

---

## Patterns

### P01: Desk Overlay

**Category**: Navigation
**Used In**: Title Screen, Pause Menu, Settings, Journal, Character Sheet, Main Menu

**Description**: A full-screen parchment overlay that dims the world to 30% behind it. The desk register — the investigator's workspace. Used when the player needs to think, configure, or review. The world is still visible but subordinate to the task.

**Specification**:
- Full-screen parchment `#E3D5B0` panel
- World dims to 30% opacity over 200ms
- Content scrolls if needed; panel is fixed
- Close: tap outside (if modal), press Esc, or click close button
- Keyboard: Esc closes, Tab navigates, Enter confirms

**When to Use**: Configuration, review, or thinking tasks — settings, journal, character sheet, save/load. When the player needs the desk, not the world.

**When NOT to Use**: Dialogue (use P02), combat (use P03), or any context where the world must remain visibly live. Do not nest desk overlays — only one at a time.

---

### P02: Sliding Panel

**Category**: Navigation
**Used In**: Dialogue UI

**Description**: A bottom-third parchment panel that slides up when conversation begins. The world stays live above it — the interrogation room is a crime scene you can read while they talk (State 2).

**Specification**:
- Occupies lower third of screen (height: 33% viewport)
- Slides up from bottom: 300ms ease-out
- Slides down on conversation end: 200ms
- World above panel: fully visible, no dim
- Contains: NPC portrait, conversation log, text input, topic chips
- Keyboard: Enter submits, Esc ends conversation

**When to Use**: Conversation with NPCs. Any context where the player needs to interact with both the UI and the world simultaneously.

**When NOT to Use**: Menus, journal, settings (use P01). Combat HUD (use P03).

---

### P03: Field Instrument

**Category**: Navigation
**Used In**: Combat HUD

**Description**: Panel umber combat chrome that sits on the un-dimmed living world. Must hold contrast against golden day and cold night. Slides in from edges at combat start; dissolves at combat end.

**Specification**:
- Base: panel umber `#2E2820` at 85-90% opacity
- Elements slide in from their respective edges: initiative bar from top (200ms), AP bar from left, ability grid from bottom
- Dissolves when combat ends (200ms)
- Must read against golden noon AND cold night (Pillar 3)
- Keyboard: number keys 1-6 for abilities, Tab for next target, Enter for End Turn

**When to Use**: Combat only. No other game state uses this register.

**When NOT to Use**: Any non-combat context. The field register exists only during the fight.

---

### P04: Typewriter Reveal

**Category**: Feedback
**Used In**: Dialogue UI

**Description**: NPC response text appears character-by-character at 45-60 characters per second, simulating the act of the NPC speaking. Word-chunk reveal for natural pacing.

**Specification**:
- Speed: 45-60 cps (Normal), configurable: Instant / Normal / Slow
- Skip to complete: click, Enter, Space, or gamepad confirm
- Auto-completes when player starts typing a new question
- Accessibility: screen reader announces completed response once; typewriter animation is visual-only

**When to Use**: NPC dialogue responses. Any text that represents a character speaking in real time.

**When NOT to Use**: System messages, UI labels, journal entries, menus. Only for diegetic NPC speech.

---

### P05: Thinking Indicator

**Category**: Feedback
**Used In**: Dialogue UI

**Description**: Three dots pulse sequentially (0.5s per dot) next to the NPC portrait during LLM generation. Communicates "the NPC is considering their response" — not a loading spinner, but a person thinking.

**Specification**:
- Three ink dots, pulsing in sequence: ● ○ ○ → ○ ● ○ → ○ ○ ● → repeat
- Position: adjacent to NPC portrait in dialogue panel
- Escalates at ~5s: dots → "The witness considers…" with elapsed time
- Timeout at ~10-15s: transitions to error state with Retry and Cancel
- Cancel available during generation

**When to Use**: LLM response generation wait. Any AI-driven delay where the player needs to know the system is working.

**When NOT to Use**: Loading screens, asset loads, save/load — those use different indicators (progress bars, spinners).

---

### P06: Topic Chips

**Category**: Input
**Used In**: Dialogue UI

**Description**: Clickable suggestion chips below the text input. Generated from NPC domains and active investigation threads. Clicking fills the input with the suggestion text.

**Specification**:
- Display: 3-6 chips in a horizontal row, scrollable if overflow
- Style: parchment chip with ink text, amber border on hover
- Click: fills text input with chip text, player can edit before submitting
- Keyboard: arrow keys navigate chips, Enter selects, fills input
- Refresh on each NPC response with updated context

**When to Use**: Conversation input where the player benefits from suggested topics.

**When NOT to Use**: Any context where topics are fixed/limited (menus, settings). Chips suggest; they don't constrain.

---


### P07: Pen-and-Ink Icon

**Category**: Data Display
**Used In**: Combat UI, Journal, Map, Character Sheet

**Description**: Single-stroke ink line icons, one weight (2px at 24px base), flat fills only where the fill carries palette meaning. No gradients, bevels, or drop shadows. Shape carries meaning; color carries accent.

**Specification**:
- Line weight: 2px at 24px icon size
- Flat fills: amber (beneficial/player), oxblood (offensive/damage), ink (utility/neutral)
- Legibility floor: 16px minimum — icons that fail at 16px are redesigned, never scaled
- File naming: `ui_icon_[purpose]_[name]_[size].png`

**When to Use**: Ability icons, status effects, map markers, UI chrome. Anywhere a glyph communicates faster than text.

**When NOT to Use**: Portraits (those are painted, Section 6.1), photographic elements, 3D renders.

---

### P08: Paper-Slide Transition

**Category**: Animation
**Used In**: All screen enter/exit

**Description**: UI panels slide into position as if pulled across a desk. No bounces, springs, scales, or 3D transforms. The desk is flat by law.

**Specification**:
- Enter: slide from edge, 200-300ms ease-out
- Exit: reverse path, 200ms
- Screen-to-screen (menus): crossfade 250ms
- Title screen: fade in 500ms
- The desk never trembles — immune to camera shake

**When to Use**: All UI transitions. Every panel that enters or exits.

**When NOT to Use**: World-space elements (health bars, damage numbers). Those follow world physics, not desk physics.

---

### P09: Amber Pulse

**Category**: Feedback
**Used In**: Journal, Dialogue UI, Combat UI, Character Sheet

**Description**: A gentle amber glow pulse (500ms-1s cycle) signaling attention, reward, or the living present. The lamp light finds the element.

**Specification**:
- Color: amber `#D9A441`
- Duration: 500ms-1s cycle, 1-3 cycles
- Triggers: new thread created, trust increased, turn started, AP refilled, revelation ready
- Never pulses longer than 1s without purpose
- Never more than one dominant pulse at a time

**When to Use**: Positive attention — rewards, new information, the player's turn.

**When NOT to Use**: Warnings, errors, danger (use P10). Neutral information (no pulse).

---

### P10: Oxblood Pulse

**Category**: Feedback
**Used In**: Combat UI, Journal

**Description**: An oxblood-red pulse signaling tension, danger, or a contradiction. The wound made visible.

**Specification**:
- Color: oxblood `#7A2E28`
- Duration: 500ms-1s cycle
- Triggers: contradiction detected, critical health (<25%), encumbrance >90%
- Always paired with a shape cue (pulse outline, icon, numeric readout) — never hue alone (2.5)

**When to Use**: Tension, danger, contradictions. States that need the player's attention.

**When NOT to Use**: Positive feedback (use P09). Routine information.

---

### P11: Thread-Red Connector

**Category**: Data Display
**Used In**: Journal, Combat UI

**Description**: A thread-red line that draws itself between two connected elements. In the journal, it connects contradictory facts. In combat, it connects the player to their target. Connection is the same act in both places.

**Specification**:
- Color: thread-red `#C4483A`
- Animation: line draws itself along the connection path (stitching effect)
- Journal: connects contradictory fact entries
- Combat: targeting reticle and range line from attacker to target
- Weight: 2px in journal, 3px in combat (must read at Far zoom)

**When to Use**: Visualizing connections — investigation contradictions, combat targeting.

**When NOT to Use**: Decorative lines, dividers, borders. Thread-red is the color of connection everywhere.

---

### P12: Save Slot Card

**Category**: Data Display
**Used In**: Main Menu (Load Game)

**Description**: A parchment card displaying save slot metadata: thumbnail screenshot, character name, playtime, region, timestamp. Click to load, right-click/long-press for delete.

**Specification**:
- Card: parchment `#E3D5B0` with umber border
- Thumbnail: 320×180px JPEG, left-aligned
- Metadata: character name (ink, bold), playtime + region (ink, regular), timestamp (small, ink)
- Hover: amber border highlight
- Delete: confirmation dialog (P20) required

**When to Use**: Save slot display in Load Game and Save Game screens.

**When NOT to Use**: Any non-save list. This pattern is specific to save data.

---

### P13: Attribute Bar

**Category**: Input / Data Display
**Used In**: Character Creation, Character Sheet

**Description**: A horizontal bar displaying an attribute name, numeric value (3-18), and modifier (+/-). In creation mode: click +/- to allocate points. In display mode: read-only.

**Specification**:
- Layout: [Attribute Name] [Value bar] [Numeric] [Modifier]
- Value bar: horizontal fill proportional to 3-18 range
- Modifier: displayed as signed integer (+2, -1, +0)
- Creation mode: +/- buttons, point counter updates in real time
- Display mode: read-only, shows current values

**When to Use**: Character creation (interactive) and character sheet (display).

**When NOT to Use**: Any non-attribute value display. Use specific patterns for health, AP, etc.

---

### P14: Drag-to-Equip

**Category**: Input
**Used In**: Character & Inventory UI

**Description**: Drag an item from the inventory grid (P15) to an equipment slot. Compatible slots highlight; incompatible slots dim. Dropping equips the item and updates derived stats immediately.

**Specification**:
- Drag source: inventory grid cell
- Drop target: equipment slot (head, chest, hands, legs, main hand, off hand, accessories)
- Compatible slot: amber highlight on drag hover
- Incompatible slot: dimmed, item snaps back on drop
- Keyboard alternative: click item → press E to equip to first compatible slot

**When to Use**: Equipment management. Any drag-and-drop inventory-to-character interaction.

**When NOT to Use**: Non-equipment items (consumables, quest items). Those use right-click or double-click to use.

---

### P15: Item Grid

**Category**: Data Display
**Used In**: Character & Inventory UI

**Description**: A scrollable grid of inventory items. Each cell shows item icon + stack count. Sorted by category. Click for detail panel.

**Specification**:
- Grid: 6 columns, scrollable vertically
- Cell: 64×64px, item icon centered, stack count bottom-right
- Empty cell: subtle outline
- Sort: weapons → armor → consumables → quest items → materials
- Click: opens item detail panel with stats, description, weight, value

**When to Use**: Inventory display. Any grid of collectible items.

**When NOT to Use**: Equipment slots (use P14), vendor screens (different layout).

---

### P16: Turn Order Bar

**Category**: Data Display
**Used In**: Combat UI

**Description**: Horizontal row of combatant portraits in initiative order at the top of the screen. Current turn highlighted with amber border. Upcoming turns dimmed.

**Specification**:
- Base: panel umber `#2E2820`
- Portraits: circular, 48px diameter
- Current turn: amber border, full saturation
- Upcoming turns: dimmed to ~40% opacity (must retain ≥3:1 contrast on umber)
- Faction coding: amber frame (allies), oxblood frame (enemies) — plus shape cue (rounded vs. angular)
- Hover: name + HP tooltip
- Static order numerals beneath portraits

**When to Use**: Turn-based combat. Any context where turn order must be visible at a glance.

**When NOT to Use**: Real-time combat, non-combat contexts.

---

### P17: World-Anchored Bar

**Category**: Data Display
**Used In**: Combat (health bars)

**Description**: A health bar rendered in world space above each combatant. Faction-fixed hue: amber fill for allies, oxblood fill for enemies. Bar length + numeric readout carry level; low-HP pulses outline.

**Specification**:
- Allied fill: amber `#D9A441`
- Enemy fill: oxblood `#7A2E28`
- Track: soot/umber — never parchment
- Low HP (<25%): oxblood pulse outline (motion cue)
- Numeric readout: combat face, adjacent to bar
- Scales with camera zoom (LOD: numeric detail at Close/Default, fill-ratio at Far)
- Never hue-only encoding (2.5)

**When to Use**: Health, shield, or any resource bar anchored to a world-space entity.

**When NOT to Use**: Screen-space HUD elements. Those use specific patterns (P18 for AP).

---

### P18: AP Meter

**Category**: Data Display
**Used In**: Combat UI

**Description**: Amber fill bar showing current / max Action Points. Depletes in real-time as AP is spent. Refills with amber pulse on turn change.

**Specification**:
- Fill: amber `#D9A441`
- Track: panel umber
- Numeric: combat face, adjacent (e.g., "5/7")
- Deplete animation: real-time fill reduction
- Refill animation: amber pulse + fill to max on turn start

**When to Use**: Action Points in combat. Any numeric resource that depletes and refills per turn.

**When NOT to Use**: Health (use P17), experience (use progress bar pattern).

---

### P19: Floating Number

**Category**: Feedback
**Used In**: Combat UI

**Description**: Numbers that float up from world-space positions on damage, healing, or miss. Combat face, color-coded, with soot outline for contrast against any background.

**Specification**:
- Damage: oxblood `#7A2E28`, float up, fade after 1.5s
- Critical: 1.5× size, heavier weight, amber flash
- Heal: amber `#D9A441` (future)
- Miss: faded ink grey, no float
- All have soot outline or parchment backplate for contrast
- World-space, scales with camera zoom

**When to Use**: Damage, healing, and miss feedback in combat.

**When NOT to Use**: XP gains (use dedicated XP summary), system messages.

---

### P20: Confirmation Dialog

**Category**: Modal
**Used In**: All destructive actions

**Description**: A small parchment modal overlay requesting confirmation before an irreversible action. Destructive button in oxblood.

**Specification**:
- Modal: small parchment card, centered on screen
- World dims behind modal (desk register)
- Message: "Are you sure? [Action description]"
- Buttons: [Destructive action — oxblood face] [Cancel — ink on parchment]
- Keyboard: Enter = confirm, Esc = cancel
- Focus lands on Cancel by default (prevents accidental confirmation)

**When to Use**: Delete save, quit without saving, dismiss party member, any irreversible action.

**When NOT to Use**: Routine actions (equip item, open door). Only for irreversible decisions.

---

### P21: Fade Transition

**Category**: Animation
**Used In**: Living Environment (wait/rest), Region Transitions

**Description**: Full-screen fade to black and back. Used for time passage (wait/rest) and spatial transitions (region boundaries).

**Specification**:
- Fade to black: 300-500ms
- Hold at black: variable (time passage duration)
- Fade in: 300-500ms
- During wait: time-lapse sky visible as faint overlay behind the fade
- Affects world only — UI overlays are not dimmed by this transition

**When to Use**: Time passage, region transitions, loading screens.

**When NOT to Use**: Screen-to-screen navigation (use P08). In-screen state changes.

---

### P22: Free-Text Input

**Category**: Input
**Used In**: Dialogue UI

**Description**: A text input field for natural-language questions. Player types their question and presses Enter to submit. Supported by topic chips (P06) for suggestions.

**Specification**:
- Field: ink text on parchment background
- Submit: Enter key
- During processing: field disabled, shows "thinking" state
- Empty submission: field shakes (3 oscillations, 150ms), low thud
- Keyboard: Tab from chips to field, Enter to submit
- Focus must land in input when conversation opens

**When to Use**: NPC conversation. Any context where free-form text is the primary input.

**When NOT to Use**: Named character naming (use shorter field), search (future).

---

### P23: Rest Point Dialog

**Category**: Input
**Used In**: Living Environment (wait mechanic)

**Description**: A dialog presented when interacting with a rest point (camp, inn). Player selects a wake time from preset options or a custom slider.

**Specification**:
- Trigger: interact with rest_point-tagged entity
- Options: Dawn (6am), Noon (12pm), Dusk (6pm), Midnight (12am)
- Custom: "Wait [N] hours" slider (1-24)
- Health info: "Resting here will restore [full/partial/no] health"
- Confirm: fade to black, time advances at 100×, fade in
- Blocked: during combat, conversation, cutscene

**When to Use**: Wait/pass-time mechanic at rest points.

**When NOT to Use**: Any non-rest interaction.

---

### P24: Tabbed Panel

**Category**: Navigation
**Used In**: Character & Inventory UI

**Description**: A full-screen desk overlay with left sidebar (party member selector) and tabbed center panel (Character / Inventory).

**Specification**:
- Left sidebar: vertical list of party member portraits, amber highlight on selected
- Center: tabbed panel — Character tab and Inventory tab
- Tab switch: content crossfade 200ms, soft page-turn sound
- Tabs: ink labels on parchment, selected tab underlined in thread-red
- Keyboard: Ctrl+Tab to switch tabs, arrow keys to navigate party members

**When to Use**: Multi-tab character/inventory management.

**When NOT to Use**: Single-purpose screens. Use P01 for simple overlays.

---

## Gaps & Patterns Needed

- **Tooltip**: Hover detail for stats, items, abilities — not yet formalized
- **Progress Bar**: XP progression, loading indicators — not yet defined
- **Dropdown/Select**: Settings options (resolution, quality presets) — deferred
- **Slider**: Volume, wait-hours, accessibility settings — deferred
- **Notification Toast**: Brief non-blocking messages ("Game Saved", "Thread Created") — needs pattern
- **Keybind Display**: Showing hotkey labels on UI elements — deferred

## Open Questions

- Should the notification toast pattern auto-dismiss or require player dismissal?
- Should tooltips appear on hover or on click (for gamepad compatibility)?
- Pattern library should be updated whenever a new UX spec introduces a new pattern
