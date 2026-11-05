# Character & Inventory UI

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 3 (Tactical Depth — stats and equipment drive build choices)

## Summary

The Character & Inventory UI displays the player's stats, party member stats,
equipment, and inventory. Character sheet with six attributes and derived stats,
equipment slots, inventory grid, and party overview. Parchment-themed per the
art bible's UI palette. Accessible via C key (character) and I key (inventory),
or as tabbed panels in a single screen.

> **Quick reference** — Layer: `Presentation` · Priority: `VS` · Key deps: `Character Stats`, `Party System`

## Overview

The Character & Inventory UI is where the player examines their mechanical
identity. It displays the six attributes with modifiers, derived stats
(health, AP, initiative, evasion, carry capacity), current equipment with
slot-based drag-and-drop, and the party inventory grid. It also surfaces
investigation-relevant stats (persuasion, insight, comprehension) that
aren't shown on the combat HUD.

The UI is a full-screen parchment overlay (desk register, art bible Section
6.1). The world dims behind it. Two tabs: Character and Inventory, with a
party member selector on the left sidebar.

## Player Fantasy

The Character & Inventory UI is where the player meets their mechanical self.
After combat, after investigation, after the numbers have done their work in
the background — this screen is where the player *sees* what their choices
have built.

**"This is who I am."** The character sheet is the mirror. When the player
opens it, they should feel the same satisfaction as opening a well-worn
character journal in a tabletop RPG — their attributes laid out in ink on
parchment, every +1 the record of a choice they made. The six-attribute
spread tells a story at a glance: the 16 STR, 8 CHA warrior reads differently
than the 10 STR, 16 INT, 14 CHA investigator. The UI's job is to make that
story legible without clicking through tabs or hovering for tooltips. The
attribute bars, the derived stat panel, the level and XP — together they
answer the question "who have I become?" without the player needing to ask it.

**"This item matters."** Equipment is the most tangible form of progression
in a CRPG. Finding a sword in a ruin, identifying its properties, slotting it
into your main hand, and watching your damage numbers shift — that loop is the
dopamine engine that drives exploration. The inventory and equipment UI must
make every item feel worth the weight it costs to carry. Comparison tooltips,
slot silhouettes that fill when equipped, the weight bar that slowly shifts
from amber to oxblood — these are not cosmetic. They are the difference
between "I have 14 items" and "I am carrying a story about the ruin I cleared."

**"I know my party."** The party sidebar is the CRPG's answer to "who's got
my back?" A single click shows each companion's core stats, trust level, and
role — not as abstractions, but as the same parchment-and-ink presentation the
player character gets. When a companion's trust indicator glows amber at level
2, the player should feel the weight of that relationship. When they inspect
the companion's knowledge domains, they should think "that's why I brought you."

**Reference**: Baldur's Gate 1-2 (the character record screen as a tactile,
ink-on-parchment artifact — every number is a story), Divinity Original Sin 2
(equipment as the primary progression vector — new gear feels transformative),
Disco Elysium (the thought cabinet as a mirror of the character's mind — stats
tell a story, not just a number).

## Detailed Design

**Character tab:**
- **Left sidebar**: Party member portraits (player + companions). Click to
  select. Current member highlighted with amber border.
- **Center**: Six attributes displayed as named bars with numeric values
  (3–18) and modifier (+/−). Attribute name, value, and modifier.
  Investigation stats highlighted separately: Persuasion, Insight,
  Comprehension (with their current bonuses).
- **Right**: Derived stats panel — Max Health, Action Points, Initiative,
  Evasion, Carry Capacity. All computed from current attributes and equipment.
- **Bottom**: Equipment slots — Head, Chest, Hands, Legs, Main Hand, Off
  Hand, Accessory 1, Accessory 2. Drag item from inventory to equip.
  Empty slots show silhouette outline.

**Inventory tab:**
- **Grid**: Scrollable grid of inventory items. Each cell shows item icon +
  stack count. Sorted by category (weapons, armor, consumables, quest items,
  crafting materials).
- **Item detail**: Click an item to see its stats, description, weight,
  value. Compare with equipped item in same slot.
- **Weight bar**: Current carry weight / max carry capacity. Color: amber
  when under 50%, shifts to oxblood as encumbrance approaches limit.
- **Gold display**: Current gold, top-right of inventory panel.

**Party overview (character tab sidebar hover):** Compact tooltip showing
party member's name, level, class/role, HP bar, and trust level indicator
(amber glow at 1+, full glow at 3).

### States and Transitions

The Character & Inventory UI is a modal full-screen overlay with internal
tab state.

**Screen States:**

| State | Trigger | Behavior |
|-------|---------|----------|
| `Closed` | Default exploration state | UI not visible; C/I keys open to last-used tab |
| `CharacterTab` | Press C, or click Character tab | Character sheet displayed for selected party member |
| `InventoryTab` | Press I, or click Inventory tab | Inventory grid displayed for selected party member |
| `LevelUp` | XP threshold reached while UI open | Modal overlay on character tab; attribute point allocation; cannot be dismissed without allocating point |

**Transitions:**
- `Closed → CharacterTab`: C key or menu command. Opens to last-viewed party member.
- `Closed → InventoryTab`: I key or menu command. Opens to last-viewed party member's inventory.
- `CharacterTab → InventoryTab`: Click Inventory tab or press I.
- `InventoryTab → CharacterTab`: Click Character tab or press C.
- `CharacterTab → LevelUp`: Automatic when XP threshold reached during UI session.
- `Any open state → Closed`: Press Esc, click close button, or press the open-state key again (C toggles character sheet, I toggles inventory).
- `Any open state → Closed (forced)`: Combat starts, conversation starts, or region transition begins. UI closes immediately with no save prompt — all equipment changes are already committed.

**Party Member Selection:**
- Click portrait in left sidebar to select. Current selection highlighted with amber border.
- Keyboard: Tab to focus sidebar, ↑/↓ to cycle members, Enter to select.
- Gamepad: D-pad up/down to cycle, A to select.
- Default selection on open: last-viewed party member (persisted per-session, not per-save — resets to player character on new session).

**Equipment Slot States:**

| State | Visual | Meaning |
|-------|--------|---------|
| `Empty` | Silhouette outline, no item | Slot available, no item equipped |
| `Filled` | Item icon in slot | Item equipped and active — stats applied |
| `Incompatible` | Slot greyed out with X | Item type does not match slot (shown during drag) |
| `CompareMode` | Equipped item left, hovered item right with stat diffs (+/−) | Comparing inventory item to equipped item of same slot category |

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Character Stats & Progression | Reads all stats and derived values | Character sheet displays attributes, modifiers, derived stats, level, and XP. Level-up screen sends attribute point allocation to Stats component. |
| Party System | Reads party composition and companion data | Party sidebar populated from active party members + recruitment pool. Companion inspect shows stats, trust level, knowledge domains (Party System provides trust/knowledge sections). |
| Entity Framework | Reads entity components | `Stats`, `Inventory`, `PlayerKnowledge`, `Faction` components read via EF for display. Equipment changes write to `Inventory` component on the selected entity. |
| Save/Load System | No direct interaction | All state persists via component data in save files. UI state (selected tab, selected member) is session-only, not saved. |
| Main Menu & Settings | Coordinates on keybindings | C (character sheet) and I (inventory) keybindings configurable in Settings. |
| Tactical Combat Engine | Reads equipment stats for combat | TCE reads equipped item bonuses from the `Inventory` component — the UI writes the equipment; TCE reads it independently. |

## Formulas

### Weight Percentage

The weight bar fill percentage and color threshold:

`weight_pct = (current_carry_weight / max_carry_capacity) × 100`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| current_carry_weight | Wc | float | 0–max_carry_capacity | Sum of all inventory item weights in lbs |
| max_carry_capacity | Wmax | int | varies by STR | (STR × 10) + 30 lbs (from Character Stats CR-3) |

**Output Range:** 0% (empty) to 100% (at capacity). Over 100% triggers encumbrance.
**Color thresholds:**
- 0–49%: amber (normal)
- 50–89%: amber shifting toward oxblood (warning)
- 90–100%: oxblood (near limit)
- >100%: oxblood pulsing (encumbered — movement penalty applies)

**Example:** STR 14 character: max = 170 lbs. Carrying 119 lbs → 70%. Bar shows amber-oxblood transition.

### XP Progress

The level-up progress bar fill:

`xp_pct = ((total_xp − xp_for_current_level) / xp_to_next_level) × 100`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| total_xp | X | int | 0–57,000 | Total accumulated XP |
| xp_for_current_level | Lxp | int | varies by level | `150 × L × (L − 1)` (from Character Stats CR-5) |
| xp_to_next_level | Xnext | int | 300–6000 | `Level × 300` (from Character Stats CR-5) |

**Output Range:** 0% (just leveled) to 100% (ready to level up)
**Example:** Level 4 character (Lxp = 150 × 4 × 3 = 1800), total_xp = 2400, Xnext = 1500.
Progress = (2400 − 1800) / 1500 × 100 = 40%.

### Equipment Stat Comparison

When comparing two items for the same slot:

`stat_diff = equipped_item_stat − compared_item_stat`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| equipped_item_stat | Se | int/float | varies | Stat value of currently equipped item in that slot |
| compared_item_stat | Sc | int/float | varies | Stat value of hovered/selected inventory item |

**Display:** Positive diff = equipped is better (shown in amber). Negative diff = compared is
better (shown in green). Zero = equal (shown in ink). Diff formatted with sign: "+2" or "−3".

**Example:** Equipped sword gives +3 damage. Compared axe gives +5 damage.
Diff = 3 − 5 = −2 → displayed as "−2" in green (axe is better).

### Attribute Modifier Display

The attribute modifier is calculated by Character Stats (CR-2); the UI displays it:

`display_format = "{value} ({sign}{modifier})"`

**Example:** STR 16 → "16 (+3)". STR 7 → "7 (−2)". STR 10 → "10 (+0)".

## Edge Cases

- **If an item is dragged to an incompatible equipment slot**: The slot
  shows a greyed-out X indicator. Dropping the item returns it to its
  original inventory position with no effect. No error sound — the visual
  rejection is sufficient.

- **If the player's inventory is full and they attempt to pick up an item**:
  The item remains in the world. A brief toast appears: "Inventory full"
  (2s, amber text). The pickup action is cancelled.

- **If a buff or debuff changes STR and max carry capacity drops below
  current carried weight**: The weight bar immediately shows >100% and
  pulses oxblood. The encumbrance penalty applies per the Character Stats
  rules. No items are lost or dropped — the player must manually address
  the overload.

- **If the player equips an item that changes an attribute (e.g., +2 STR
  ring)**: Derived stats recalculate immediately. The affected attribute
  displays the modified value with a "(+2)" note in green next to the base
  value. All dependent derived stats (carry capacity, melee damage, etc.)
  update on-screen within one frame.

- **If the player opens the Character & Inventory UI during combat**:
  Blocked. The C and I keys are disabled during combat. If the UI is open
  when combat starts, it closes immediately (forced transition, see States
  & Transitions). Equipment cannot be changed during combat.

- **If the player opens the Character & Inventory UI during conversation**:
  Allowed. The UI opens as an overlay; the conversation is not paused but
  is visually dimmed behind the parchment. Equipment changes are committed
  immediately but do not affect the active conversation's stat checks
  (checks use snapshot at conversation start).

- **If a companion leaves the party (trust betrayal, death, dismissal)
  while their inventory is displayed**: The UI immediately switches to
  the player character's sheet. The departed companion's equipment is
  moved to a "legacy inventory" accessible via a dedicated tab in the
  player's inventory for 7 in-game days, then transferred to the companion's
  home region stash. After 7 days, equipment is irretrievable without
  travelling to the companion's home region. The departure journal entry
  notes any transferred items.

- **If the player inspects a party member who is not the active selection
  when a level-up occurs**: The level-up modal appears for the character
  who earned the level, regardless of who is currently being viewed. If
  multiple characters level up simultaneously (shared XP event), modals
  queue in party order (player character first, then companions by join
  order).

- **If gold reaches the maximum displayable value** (999,999): The display
  shows "999,999+". Gold continues to accumulate beyond the display cap.
  The cap is purely cosmetic — no gameplay limit on gold.

- **If the inventory is empty**: The grid shows a single centered line in
  subdued ink: "Your pack is empty." No weight bar (0 lbs). Gold display
  remains visible.

- **If an item is unidentified**: The item shows a "?" icon and its name
  displays as "Unidentified [Item Type]" in muted text. Stats are hidden.
  Equipping is allowed (the player can wear unidentified gear) but stats
  are not applied until the item is identified. The equipped slot shows
  the "?" icon. The item detail panel shows "Identify this item to reveal
  its properties."

- **If the player resizes the game window below 1280×720** (minimum
  supported resolution): The UI scales down maintaining the parchment
  proportions. Below the minimum resolution, the UI displays a message:
  "Minimum resolution: 1280×720" and prevents interaction until the
  window is resized above the threshold.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Character Stats & Progression | C&I UI depends on Char Stats | Reads all six attributes, modifiers, derived stats, level, XP for display. Level-up screen sends attribute point allocation. **Hard** — character sheet is empty without stats. |
| Party System | C&I UI depends on Party System | Reads active party roster, companion metadata, trust levels, knowledge domains for the party sidebar and companion inspect. **Hard** — party sidebar needs party data. |
| Entity Framework | C&I UI depends on EF | Reads `Stats`, `Inventory`, `PlayerKnowledge`, `Faction` components. Writes equipment changes to `Inventory` component. **Hard** — no data access without EF. |
| Save/Load System | No direct dependency | UI state (selected tab, selected member, scroll position) is session-only and not persisted in saves. All character and inventory data persists via component serialization. **Soft** — UI functions identically with or without save/load awareness. |
| Main Menu & Settings | C&I UI depends on Settings | C (character sheet) and I (inventory) keybindings read from Settings. **Soft** — defaults work without Settings integration. |
| Tactical Combat Engine | TCE depends on C&I UI (indirect) | TCE reads equipped item stats from `Inventory` component. The UI writes equipment; TCE reads it independently — no direct dependency. **Soft**. |

## Tuning Knobs

| Parameter | Default | Safe Range | Effect |
|-----------|---------|------------|--------|
| `inventory_grid_columns` | 6 | 4–10 | More columns = denser grid; fewer = larger icons, more scrolling |
| `inventory_grid_cell_size` | 64px | 48–96 | Larger cells = fewer visible items, better readability |
| `weight_bar_warning_pct` | 50 | 30–70 | Earlier warning = more conservative UI feedback |
| `weight_bar_critical_pct` | 90 | 75–95 | Earlier critical = more urgency for encumbrance |
| `equipment_slots` | 8 (Head, Chest, Hands, Legs, Main Hand, Off Hand, Accessory 1, Accessory 2) | 6–12 | More slots = more equipment granularity, more items to manage |
| `toast_duration_ms` | 2000 | 1000–4000 | How long "Inventory full" and similar toasts remain visible |
| `legacy_inventory_days` | 7 | 1–30 | How many in-game days departed companion equipment stays in legacy inventory |
| `gold_display_cap` | 999999 | 99999–9999999 | Cosmetic cap only — gold still accumulates beyond display limit |
| `min_supported_width` | 1280 | 1024–1920 | Minimum horizontal resolution before UI locks |
| `min_supported_height` | 720 | 576–1080 | Minimum vertical resolution before UI locks |
| `sidebar_portrait_size` | 72px | 48–128 | Larger portraits = more visual presence, less space for content |

## Visual/Audio Requirements

| Event | Visual | Audio |
|-------|--------|-------|
| Screen open | Full-screen parchment slides in (300ms); world dims | Parchment unfurl |
| Tab switch | Content crossfades (200ms) | Soft page-turn |
| Item equipped | Item icon slides from inventory to equipment slot | Soft click + subtle metallic ring |
| Stat hover | Tooltip appears with formula breakdown | None |
| Encumbrance warning | Weight bar pulses oxblood at >90% | Low thud |

## UI Requirements

📌 **UX Flag — Character & Inventory UI**: This system IS a UI system. The
spec below defines the screens it owns. Run `/ux-design` for each screen during
Pre-Production.

### Character Sheet Screen

Accessible via C key or menu command. Full-screen parchment overlay.

**Layout:**
- **Left sidebar** (20% width): Vertical portrait strip — player character at
  top, followed by active companions in party order, then pool companions
  (greyed out, unselectable). Current selection highlighted with amber border
  (3px, oxblood focus ring). Scrollable if party exceeds sidebar height.
- **Center panel** (50% width): Six attribute bars, each showing name (left),
  numeric value with modifier (right), and a horizontal bar proportional to
  the value (3–18). Investigation-relevant stats (Persuasion, Insight,
  Comprehension) displayed below attributes in a distinct "Investigation"
  sub-panel with their current bonuses and source breakdown.
- **Right panel** (30% width): Derived stats block — Max Health, Action Points,
  Initiative, Evasion, Carry Capacity. Each shows the computed value and the
  formula breakdown on hover/tooltip.
- **Bottom strip**: Equipment slots arranged horizontally — Head, Chest, Hands,
  Legs | Main Hand, Off Hand, Accessory 1, Accessory 2. Each slot is a square
  cell (72×72px) with item icon or silhouette outline.
- **XP bar**: Full-width bar across bottom of center panel, showing current
  level, XP progress (fill bar), and XP to next level.
- **Tabs**: Two tabs at top — "Character" (active by default) and "Inventory".
  Amber highlight on active tab.

**Interaction:**
- Click attribute for detailed breakdown modal (base value + equipment bonuses +
  temporary effects = total, with each source listed).
- Hover derived stat for formula tooltip.
- Drag item from inventory to equipment slot (equip). Drag from equipment slot
  to inventory (unequip). Right-click equipped item to unequip to first empty
  inventory slot.
- Click equipment slot with empty cursor → highlight compatible inventory items.
- Tab/Shift+Tab: cycle through focusable elements (sidebar → attributes →
  equipment slots → tabs). Enter/Space: activate focused element.
- Gamepad: left stick/D-pad to navigate, A to select, X to unequip, Y to
  compare.

### Inventory Screen

Accessible via I key or Inventory tab.

**Layout:**
- Same left sidebar as Character Sheet (persistent across tabs).
- **Grid** (center): Scrollable grid of inventory items. Default 6 columns ×
  N rows. Each cell 64×64px showing item icon + stack count (bottom-right
  corner, amber text). Empty cells show subtle grid lines.
- **Category filter** (top of grid): Row of filter buttons — All, Weapons,
  Armor, Consumables, Quest Items, Crafting. Amber underline on active filter.
- **Item detail panel** (right): Shows selected item's name, icon (large),
  description text, stats list, weight, value. If item is equipment and same
  slot is occupied, shows comparison (equipped left, selected right, stat
  diffs with +/− coloring).
- **Weight bar** (below grid): Horizontal bar showing current weight / max
  capacity with percentage. Amber → amber-oxblood → oxblood → oxblood pulsing
  based on thresholds (see Formulas).
- **Gold display** (top-right corner): Coin icon + gold amount.

**Interaction:**
- Click item to select (detail panel updates). Double-click to equip/use.
- Drag item to equipment slot or to sidebar party member portrait to transfer.
- Right-click: context menu (Equip / Use / Drop / Examine / Split Stack).
- Shift+click: split stack (modal with quantity slider).
- Sort: click category filter to filter; default sort: category → name.
- Search bar (top of grid): type to filter items by name.
- Gamepad: D-pad grid navigation, A to select/use, Y for context menu, X to
  drop, LB/RB to change category filter.

### Level-Up Modal

Overlay on Character Sheet. Cannot be dismissed without action.

- **Header**: "Level Up — Level [N] → [N+1]"
- **Body**: Health increase shown (+N HP). Every 4th level: attribute point
  to allocate — six attribute options with + button. Current modifier shown.
  Attribute bars pulse amber on hover for allocatable attributes.
- **Footer**: "Confirm" button (disabled until point allocated if 4th level).
  No cancel button — the modal is mandatory.

### Party Overview

Accessible as tooltip on sidebar portrait hover, or as compact panel.

- **Tooltip mode**: Name, level, role/archetype, HP bar (current/max), trust
  indicator. 300ms hover delay. Dismisses on mouse-out.
- **Inspect mode**: Click "Inspect" in tooltip to open full character sheet
  for that companion. Same layout as player character but with additional
  sections from Party System: Trust & Relationship panel (trust level, history)
  and Knowledge Domains list (domain name, proficiency indicator).
- **Companion behavior mode indicator**: Small glyph on companion portrait
  (Aggressive = sword, Defensive = shield, Passive = eye). Shown on party
  sidebar at all times.

### Accessibility

- All stat values conveyed by both position (bar fill) and numeric text —
  never color alone.
- Weight bar color changes accompanied by percentage text.
- Tab-navigable: every interactive element in sidebar → attributes →
  equipment → grid → tabs order.
- Screen reader: parchment screen announced as "Character Sheet — [character
  name]" or "Inventory — [character name]".
- Reduce-motion: disables parchment slide animation; instant open/close.
- Text scaling: all text respects a `ui_text_scale` setting (0.75× to 1.5×).

## Acceptance Criteria

**Display Correctness:**

- **AC-1**: GIVEN a character with STR 16 (+3), WHEN the character
  sheet is opened, THEN STR displays as "16 (+3)" with the modifier correctly
  calculated.

- **AC-2 — Derived stat display**: GIVEN a character with CON 14 (+2) at
  level 3, WHEN the character sheet is opened, THEN Max Health displays as
  "54 HP" (20 + 28 + 6).

- **AC-3 — Attribute bar proportional fill**: GIVEN a character with STR 16
  and CHA 8, WHEN the character sheet is displayed, THEN the STR bar is
  approximately 2× the width of the CHA bar (±5% tolerance).

- **AC-4 — XP progress bar**: GIVEN a level 4 character (Lxp = 1800) with
  2400 total XP (Xnext = 1500), WHEN the character sheet is opened, THEN
  the XP bar fill is at 40% (±2% tolerance).

- **AC-5 — Modifier sign display**: GIVEN characters with attribute values
  7, 10, and 16, WHEN their sheets are displayed, THEN modifiers appear as
  "−2", "+0", and "+3" respectively.

**Equipment & Inventory:**

- **AC-6**: GIVEN an item dragged from inventory to an equipment
  slot, WHEN the slot is compatible, THEN the item equips and derived stats
  update immediately.

- **AC-7 — Incompatible slot rejection**: GIVEN a helmet dragged to the
  Hands equipment slot, WHEN the item is dropped, THEN the slot shows a
  greyed-out X indicator and the item returns to its original inventory
  position with no equipment change.

- **AC-8 — Equipment stat comparison**: GIVEN an equipped sword (+3 damage)
  and a compared axe (+5 damage) in inventory, WHEN the player selects the
  axe, THEN the comparison panel shows "−2" in green next to Damage.

- **AC-9 — Attribute-changing equipment**: GIVEN a character with STR 14
  equipping a ring that grants +2 STR, WHEN the ring is equipped, THEN STR
  displays as "14 (+2)" and derived stats (carry capacity, melee damage)
  recalculate within one frame.

- **AC-10**: GIVEN inventory weight at 95% of max, WHEN the
  inventory tab is open, THEN the weight bar pulses oxblood.

- **AC-11 — Weight bar color at normal**: GIVEN inventory weight at 30%
  of max, WHEN the inventory tab is open, THEN the weight bar is solid amber
  with no pulse.

- **AC-12 — Weight bar color at warning**: GIVEN inventory weight at 70%
  of max, WHEN the inventory tab is open, THEN the weight bar shows the
  amber-oxblood transition color.

- **AC-13 — Inventory full toast**: GIVEN a full inventory (weight at max
  capacity), WHEN the player attempts to pick up a world item, THEN a toast
  appears: "Inventory full" for 2 seconds (±200ms) in amber text and the
  item remains in the world.

- **AC-14 — Empty inventory**: GIVEN a character with no items in inventory,
  WHEN the inventory tab is opened, THEN the grid shows "Your pack is empty."
  in subdued ink and the gold display remains visible.

**State Transitions:**

- **AC-15 — Combat blocks UI**: GIVEN the C&I UI is closed and combat is
  active, WHEN the player presses C or I, THEN the UI does not open.

- **AC-16 — Combat forces close**: GIVEN the C&I UI is open on the Character
  tab, WHEN combat starts, THEN the UI closes immediately with no prompt.

- **AC-17 — Tab switching**: GIVEN the UI is open on the Character tab, WHEN
  the player clicks the Inventory tab or presses I, THEN the UI switches to
  the Inventory tab with a 200ms crossfade and the same party member remains
  selected.

- **AC-18 — Level-up modal mandatory**: GIVEN a character at level 4 earning
  enough XP to reach level 5 while the UI is open, WHEN the XP is awarded,
  THEN the Level-Up modal appears showing the HP increase and the modal
  cannot be dismissed without pressing Confirm.

- **AC-19 — Companion departure switches view**: GIVEN the C&I UI is
  displaying a companion's inventory, WHEN that companion departs the party
  (trust betrayal), THEN the UI immediately switches to the player
  character's sheet.

**Party Interaction:**

- **AC-20 — Party sidebar displays active members**: GIVEN a party of 4
  (player + 3 companions), WHEN the C&I UI is opened, THEN the left sidebar
  shows all 4 members with the player character at top.

- **AC-21 — Companion inspect**: GIVEN a companion in the sidebar, WHEN the
  player clicks "Inspect" on the tooltip, THEN the character sheet opens for
  that companion showing their stats, trust level, and knowledge domains.

- **AC-22 — Party member switch preserves tab**: GIVEN the Inventory tab is
  open for the player character, WHEN the player clicks a companion's portrait
  in the sidebar, THEN the Inventory tab remains open showing that companion's
  inventory.

**Accessibility:**

- **AC-23 — Keyboard navigation**: GIVEN the C&I UI is open and focused,
  WHEN the player presses Tab repeatedly, THEN focus moves through sidebar →
  attributes → equipment slots → tabs and wraps back to the first element.

- **AC-24 — Reduce-motion**: GIVEN the `reduce_motion` setting is enabled,
  WHEN the C&I UI is opened, THEN the parchment appears instantly with no
  slide animation and tab switches have no crossfade.

- **AC-25 — Resolution floor**: GIVEN the game window is resized below
  1280×720, WHEN the C&I UI is open, THEN the UI displays "Minimum
  resolution: 1280×720" and prevents interaction.

## Open Questions

| Question | Owner | Deadline | Resolution |
|----------|-------|----------|-----------|
| Should the inventory be purely weight-based or add a slot cap (e.g., max 50 unique items) to prevent degenerate hoarding? | Game Designer | Before VS implementation | — |
| How should item stacking work? What items stack (consumables, crafting materials)? What's the max stack size? | Systems Designer | Before VS implementation | — |
| Should the UI support equipment set presets (save/load a full equipment configuration for quick swapping)? | UX Designer | Before Full Vision | — |
| How are quest items handled visually — distinct category, protected from dropping/selling? | Game Designer | Before VS implementation | — |
| Should item tooltips show lore/flavor text in addition to mechanical stats? If so, what's the priority order? | Narrative Director | Before VS implementation | — |
| Should the level-up screen be skippable/deferrable (bank the level-up for later), or is it mandatory immediate allocation? | Game Designer | Before VS implementation | Currently designed as mandatory — reconsider if playtest shows it disrupts flow. |
| What information does the companion tooltip show at a glance vs. requiring Inspect? Currently: name, level, role, HP bar, trust. Is this sufficient? | UX Designer | Before VS implementation | — |
| Should inventory sorting remember the player's last choice (per-session, per-save)? | UX Designer | Before VS implementation | — |
| How does the UI handle item identification — is there an "Identify" action in the context menu, or is it automatic on pickup? | Systems Designer | Before VS implementation | — |
