# HUD Design

> **Status**: In Design
> **Author**: Awgyrog + ux-designer
> **Last Updated**: 2026-08-10
> **Template**: HUD Design

---

## HUD Philosophy

The Marrow of Empire uses an information-dense HUD where all decision-relevant
data is always visible. The player should never need to open a menu to answer a
tactical question — health, action points, turn order, and ability availability
are present on screen at all times during combat. During exploration, the HUD
thins to party status, interaction prompts, and the clock — the borderland
deserves to be seen — but it never disappears entirely.

The HUD follows the art bible's three-register system (6.1):

- **World-anchored**: Health bars, damage numbers, status icons, and interaction
  tags live in world space where the information lives, so the eye never leaves
  the crime scene. Subject to Pillar 3 lighting discipline and the 4u/10u/20u
  LOD contract.
- **Field register**: Combat instruments — AP bar, turn order, ability grid,
  End Turn — use panel umber surfaces. The desk, carried into the dark.
- **Desk register**: Menu overlays — journal, pause, save/load — use parchment
  on the dimmed world. The investigator's thinking.

The thread-red line crosses all three registers: the combat targeting reticle
is the same thread that connects facts in the journal. Connection is the same
act in both places.

Density adapts to context. Exploration shows party portraits, health bars,
interaction prompts, and the game clock — enough to navigate and stay oriented.
Combat fills the screen edges with the full tactical instrument. The transition
between densities is immediate — the HUD snaps to context when combat begins
or ends, no animation. The player's eye should find the new elements where it
expects them; nothing should move between states.

---

## Information Architecture

### Full Information Inventory

| # | Element | Source GDD | Register | Category |
|---|---|---|---|---|
| 1 | Health bars (party + enemies) | combat-ui | World-anchored | Must Show (combat) |
| 2 | AP bar | combat-ui | Field | Must Show (combat) |
| 3 | Turn order / initiative bar | combat-ui | Field | Must Show (combat) |
| 4 | Ability grid | combat-ui | Field | Must Show (combat) |
| 5 | End Turn | combat-ui | Field | Must Show (combat) |
| 6 | Damage numbers (floating) | combat-ui | World-anchored | Must Show (combat) |
| 7 | Targeting reticle + range line | combat-ui | World-anchored | Must Show (combat) |
| 8 | Flanking arcs | combat-ui | World-anchored | Must Show (combat) |
| 9 | Status effects | combat-ui | World-anchored | Must Show (combat) |
| 10 | Cover indicator | combat-ui | World-anchored | Must Show (combat) |
| 11 | Hit chance display | combat-ui | Field | Must Show (combat) |
| 12 | Crit indicators | combat-ui | World-anchored | Must Show (combat) |
| 13 | Miss/evade indicators | combat-ui | World-anchored | Must Show (combat) |
| 14 | Party portrait sidebar | character-stats-progression | Field | Must Show (exploration) |
| 15 | Companion behavior mode | character-stats-progression | Field | Must Show (exploration) |
| 16 | Game clock | living-environment | Field | Must Show (exploration) |
| 17 | Interaction prompts | entity-framework | World-anchored | Must Show (exploration) |
| 18 | Conversation panel | dialogue-ui | Desk (world-edge) | Contextual (dialogue) |
| 19 | NPC portrait | dialogue-ui | Desk | Contextual (dialogue) |
| 20 | Topic suggestion chips | dialogue-ui | Desk | Contextual (dialogue) |
| 21 | Trust indicator | trust-relationship | Desk | Contextual (dialogue) |
| 22 | Text input field | question-system | Desk | Contextual (dialogue) |
| 23 | Thread discovery notification | investigation-journal | Desk | Contextual (investigation) |
| 24 | Entity quick-ref chips | investigation-journal | Desk | Contextual (exploration) |
| 25 | Chain completion / revelation prompt | investigation-journal | Desk | Contextual (investigation) |
| 26 | Item comparison tooltips | character-inventory-ui | Desk | Contextual (inventory) |

### Categorization

| Category | Count | Elements |
|---|---|---|
| **Must Show (combat)** | 13 | #1–13: All combat tactical information. Visible whenever combat is active. |
| **Must Show (exploration)** | 4 | #14–17: Party sidebar, game clock, interaction prompts. Always visible during exploration. |
| **Contextual (dialogue)** | 5 | #18–22: Conversation panel. Slides in during NPC interaction, slides out on conversation end. |
| **Contextual (investigation)** | 4 | #23–26: Thread notifications, entity chips, revelation prompts, tooltips. Appear when relevant. |

**Conflict check**: The HUD Philosophy says "information-dense" and the Must Show
lists total 17 items during combat. This is consistent with classic CRPGs
(Baldur's Gate, Pillars of Eternity) where the combat HUD fills screen edges.
Exploration thins to 4 items — the borderland gets its breathing room.

---

## Layout Zones

```
┌─────────────────────────────────────────────────────────┐
│ ┌─────────────────────────────────────────────────────┐ │
│ │              Initiative Bar (turn order)            │ │ ← top edge
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌──┐                                                   │
│ │P │                GAME WORLD                         │
│ │a │              (exploration / combat)               │
│ │r │                                                   │
│ │t │   · health bars over combatants                  │
│ │y │   · damage numbers floating                      │
│ │  │   · targeting reticle + range line               │
│ │s │   · flanking arcs   · cover icons               │
│ │i │   · status effect glyphs                        │
│ │d │   · interaction prompts (exploration)            │
│ │e │                                                   │
│ │b │                                                   │
│ │a │                                                   │
│ │r │                                                   │
│ └──┘                                                   │
│                                                         │
│ ┌──────────────────────┐  ┌──────────────────────────┐ │
│ │  AP Bar (bottom-left)│  │   Ability Grid (bottom)  │ │ ← bottom edge
│ │  amber fill, number  │  │   umber chips, hotkeys   │ │
│ └──────────────────────┘  └──────────────────────────┘ │
│                                        ┌──────────────┐│
│                                        │  End Turn    ││ ← bottom-right
│                                        │  amber fill  ││
│                                        └──────────────┘│
│ ┌─────────────────────────────────────────────────────┐ │
│ │         Entity Quick-Ref Bar (exploration)          │ │ ← bottom bar
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

| Zone | Position | Register | Elements | Visible When |
|---|---|---|---|---|
| **Party sidebar** | Left edge, top-to-bottom | Field (panel umber) | Party portraits (#14), behavior mode glyphs (#15) | Exploration + combat |
| **Initiative bar** | Top edge, full-width | Field (panel umber) | Turn order portraits (#3), current turn amber rim | Combat only |
| **Game world** | Center, full remaining space | World | Health bars (#1), damage numbers (#6), targeting (#7), flanking (#8), status effects (#9), cover (#10), crits (#12), miss (#13), interaction prompts (#17) | Always |
| **AP bar** | Bottom-left | Field (amber on umber) | Current AP / Max AP bar + number (#2) | Combat only |
| **Ability grid** | Bottom-center | Field (umber chips) | Ability icons with hotkeys (#4) | Combat only |
| **End Turn** | Bottom-right | Field (amber fill) | "End Turn" button (#5) | Combat only |
| **Clock** | Top-right corner | Field (umber chip) | 24-hour time, day/night visual state (#16) | Exploration |
| **Entity quick-ref** | Bottom bar, full-width | Desk (parchment chips) | Recently referenced entity chips (#24) | Exploration |
| **Conversation panel** | Bottom-third | Desk (parchment, world stays live) | NPC portrait (#19), topic chips (#20), trust (#21), text input (#22) | Dialogue only |
| **Thread notification** | Top-right (below clock) | Desk (amber pulse) | "New thread discovered" (#23) | On discovery |
| **Revelation prompt** | Center-overlay | Desk (parchment modal) | "Threads converge — revelation" (#25) | On chain completion |

### Visual Budget

| Context | Max simultaneous HUD elements | Max screen coverage |
|---|---|---|
| Exploration | ~8 (sidebar + clock + entity bar + prompts) | ~15% (sidebar + bottom bar + top-right chip) |
| Combat | ~20 (sidebar + initiative + AP + abilities + End Turn + all world-anchored) | ~25% (edges only — center always clear) |
| Dialogue | ~7 (conversation panel + sidebar) | ~35% (bottom-third panel + sidebar) |

---

## HUD Elements

Combat elements follow the art bible 6.6 specification exactly. See art bible
Section 6.6 for full visual treatment of health bars, AP bar, turn order,
ability grid, End Turn, damage numbers, targeting, flanking arcs, cover, status
effects, and hit/crit/miss indicators. This section specifies the elements NOT
covered in the art bible.

### Exploration Elements

**Party Portrait Sidebar (#14)**
- **Zone**: Left edge, vertical stack
- **Visual**: Panel umber column. Each companion: painted portrait (small, framed), name label (parchment-on-umber), health bar (amber fill, compact, no numeric readout), behavior mode glyph (Aggressive=sword, Defensive=shield, Passive=eye, ink line-work)
- **Interaction**: Click to select. Selected: amber border on portrait. Hover: tooltip with full stats summary. Scrollable if party exceeds sidebar height. Tab to cycle.
- **Update**: Real-time — health bar reflects current HP, behavior mode changes on command

**Game Clock (#16)**
- **Zone**: Top-right corner
- **Visual**: Panel umber chip. 24-hour time ("14:32"). Below: day/night state glyph — sun (Dawn/Day/Dusk) or moon (Night/Deep Night). Five light states at a glance.
- **Update**: Continuous at 10:1 compression (1 game minute = 6 real seconds)
- **Interaction**: Hover shows full date. Freezes during combat, greyed out.

**Interaction Prompts (#17)**
- **Zone**: World-anchored, above/near interactable entity
- **Visual**: Parchment chip with ink label ("Talk," "Examine," "Open"). Soot outline for readability over any world value. Appears on proximity.
- **Input**: Click prompt or press interact key (default E, rebindable)

### Dialogue Elements

**Conversation Panel (#18)**
- **Zone**: Bottom third, slides up from bottom edge
- **Visual**: Parchment panel. World stays live above — no dim (art bible 6.1 dialogue exception). Contains NPC portrait, dialogue text, topic chips, trust indicator, text input.
- **Animation**: Slides up 300ms on conversation start, slides down 200ms on end
- **Typewriter**: Word-chunk reveal 45–60 chars/sec (art bible 6.4). Skip: click/Enter/Space. Instant: accessibility option.

**Topic Suggestion Chips (#20)**
- Parchment chips with ink labels. Click to ask that topic or type custom question. Chips update as conversation context changes.

**Trust Indicator (#21)**
- Small amber bar on conversation panel. Reflects NPC trust level. Increases/decreases based on choices.

### Investigation Elements

**Thread Discovery Notification (#23)**
- **Zone**: Top-right, below clock
- **Visual**: Amber pulse. Parchment chip: "New thread: [name]" with amber thread glyph. Fades in, holds 3s, fades out. Queue: subsequent notifications stack below; oldest auto-dismisses after 5s. During combat: queued, batch-delivered on combat end.

**Entity Quick-Ref Chips (#24)**
- **Zone**: Bottom bar, full-width (hidden during combat)
- **Visual**: Parchment chips with ink labels ("Captain Veren," "Old Mine"). Click to open full journal entry. Appear as entities are mentioned/discovered.

**Chain Completion / Revelation (#25)**
- **Zone**: Center-overlay
- **Visual**: Amber glow parchment modal: "The threads converge." Shows connected facts. "Open Journal" / "Dismiss." Triggers when 3+ facts point to same target.

---

## Dynamic Behaviors

### Context Transitions

| Transition | HUD Change | Duration |
|---|---|---|
| Exploration → Combat | Party sidebar persists. Initiative slides in from top. AP bar from left. Ability grid from bottom. End Turn appears. Entity quick-ref hides. Clock freezes + dims. | 200ms per edge |
| Combat → Exploration | Combat elements slide out. Entity quick-ref reappears. Clock unfreezes. | 200ms |
| Exploration → Dialogue | Conversation panel slides up. Entity quick-ref hides. | 300ms |
| Dialogue → Exploration | Conversation panel slides down. Entity quick-ref reappears. | 200ms |
| Any → Pause | Dim 30%. All HUD remains visible behind dim. Combat UI renders above dim layer. | 200ms |

### Combat Lifecycle

- **Round start**: AP refills, amber pulse on AP bar. Turn order updates — current turn gets amber rim.
- **Action selected**: Targeting mode — valid targets get thread-red rim, invalid targets desaturate. Range line draws.
- **Action executed**: Damage numbers float, health bars deplete, AP decreases, status glyphs appear.
- **Enemy eliminated**: Health bar shatters. Portrait greys in initiative bar, then removes.
- **Combat end**: Victory/defeat desk overlay. Combat HUD dissolves.

---

## Platform & Input Variants

### Keyboard/Mouse (Primary)
- All HUD elements clickable
- Hotkeys for abilities (1–9, rebindable)
- Tab: cycle selected party member
- E: interact | C: character sheet | J: journal | Escape: pause
- Mouse wheel: zoom camera

### Gamepad (Partial)
- D-pad L/R: cycle party member | D-pad U/D: cycle abilities
- A: confirm | B: cancel/back | X: interact | Y: End Turn
- Start: pause | LB/RB: cycle enemy targets

### No Touch
PC-only. No touch layouts.

---

## Accessibility

WCAG-AA committed per `design/accessibility-requirements.md`. Art bible 6.7
defines the full contract. HUD-specific:

- **Health bars**: Faction-fixed hue + numeric readout + bar length — three cues (art bible 2.5, 6.6)
- **Damage numbers**: Combat face, ≥14px at Far zoom, soot outline over any world value
- **Turn order**: Portrait + name + amber rim (current) / dimmed (upcoming) — color not the sole cue
- **Colorblind modes**: Protanopia/Deuteranopia/Tritanopia remap the combat palette (Settings → Accessibility)
- **Reduced motion**: Combat HUD slide-in → 0ms. Health deplete animation disabled. Typewriter instant.
- **Text scale**: HUD text must scale with accessibility setting. Floor at 13px combat face (art bible 6.2).
- **Keyboard-only combat**: All abilities reachable via hotkeys. Tab for character selection. D-pad/arrows for grid.

---

## Open Questions

- **Minimap**: Classic CRPGs often include one. Not in current GDD inventory. Add if needed.
- **Quick-save indicator**: Brief "Quick-saved" confirmation on F5? Not currently specced.
- **Party member death**: Downed state in sidebar — portrait greyed? Oxblood marker on health bar?
- **Entity quick-ref persistence**: Across sessions? Per-character or per-save?
- **Dialogue log scrollback**: Is there a scrollable history in the conversation panel?
- **Combat log**: Scrollable damage log below AP bar? Many CRPGs include this. Not currently specced.
