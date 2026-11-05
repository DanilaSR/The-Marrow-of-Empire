# ADR-0012: UI Framework

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | UI |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | All Presentation-layer GDDs, Art Bible Section 6, `kaiju/AGENTS.md` |
| **Post-Cutoff APIs Used** | Kaiju HTML/CSS-like UI system |
| **Verification Required** | Kaiju UI rendering API, font loading, accessibility hooks (AccessKit), input routing |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — UI reads entity data) |
| **Enables** | All 7 Presentation-layer systems |
| **Blocks** | Dialogue UI, Combat UI, Investigation Journal, Character UI, Map, Main Menu |
| **Ordering Note** | Must be Accepted before any UI implementation begins |

## Context

The Marrow of Empire has 7 UI systems (Dialogue, Combat, Investigation Journal, Character & Inventory, Map, Audio, Main Menu) rendered through Kaiju's HTML/CSS-like UI system. The art bible defines a "desk register" — flat parchment panels, umber frames, pen-and-ink icons, paper-slide motion. No 3D UI, no diegetic HUD. The UI palette diverges from the world palette but shares its vocabulary (Section 2.4).

## Decision

**Kaiju HTML/CSS-like UI with the desk register.** All UI is rendered in screen space (not world space). Three registers: the desk (parchment overlays), the field (umber combat chrome), and the world (anchored health bars, damage numbers). UI never trembles with camera shake. Minimum supported resolution: 1280×720; reference: 1920×1080 with height-based scaling.

### Register Assignment

| Register | Elements | Treatment |
|----------|----------|-----------|
| **Desk** | Journal, menus, settings, save/load, character sheet, pause | Parchment panels, world dims to 30% |
| **Desk edge** | Dialogue panel | Parchment, world stays live (State 2) |
| **Field** | Combat HUD, turn order, AP bar, ability grid | Panel umber, no dim |
| **World** | Health bars, damage numbers, targeting, interaction prompts | Anchored in world space, subject to Pillar 3 readability |

### Typography
- Desk serif (Garamond class): journal, dialogue, menus — humanist old-style
- Combat face (grotesque sans, tabular figures): AP, HP, damage numbers, initiative
- World lettering: painted/etched, authored as texture — never typeset in the world

### Motion Language
Paper slides (300ms ease-out), ink appears (typewriter 45-60 cps), thread draws (contradiction connectors), lamp dims the world. Nothing bounces, scales, spins, or flips. The desk never trembles (immune to camera shake).

### Accessibility
- Tab order and arrow-key navigation for all UI
- Oxblood/umber focus ring (≥3:1 contrast)
- Esc = end conversation (in dialogue), Esc = pause (outside dialogue)
- Typewriter speed setting: Instant / Normal (45-60 cps) / Slow
- Text scale accessibility setting — floor: 16px body at reference resolution

### Font Loading
Fonts ship with the game as OFL-licensed files. No runtime network fonts. Final font selection (specific Garamond-class and grotesque) deferred to technical-artist.

## GDD Requirements Addressed
All 7 Presentation-layer GDDs — Dialogue UI, Combat UI, Investigation Journal, Character & Inventory UI, Map System, Audio System, Main Menu & Settings. Art Bible Section 6 (all subsections).

## Performance Implications
UI render: <1ms per frame for static UI. Journal open/close transition: 200-300ms. Typewriter text: DOM-safe chunk updates at 45-60 cps. No frame drops during UI animations.
