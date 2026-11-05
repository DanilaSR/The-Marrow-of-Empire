# Visual Design Brief — Title Screen: "The Marrow of Empire"

**Status**: Draft
**Applies to**: `design/ux/title-screen.md` + art bible Sections 2.4, 3 (State 6), 6.1–6.7
**Consumer**: ui-programmer (Phase 3), scene artist, technical-artist (particles)
**Reference resolution**: 1920×1080 · Floor: 1280×720
**Author**: art-director
**Date**: 2026-08-09

---

## 1. Overview — The Journal Beside the Camp

The title screen is one desk element laid beside a living painting. The scene owns
the frame; the menu is a traveler's journal set on the shore — parchment, ink, and
thread in the flat register (art bible 2.4/6.1), never pretending to live inside the
painting. The register math: the scene is the world (warm, painterly, golden hour);
the menu is the desk (flat parchment, ink, oxblood ring); the two never blend — the
panel is the boundary, and the scene does not dim behind it.

Three laws govern every element below: the register is flat (2.4), no critical state
lives on hue alone (2.5), and the desk is where truth is allowed to be orderly
(State 5).

---

## 2. The Scene — The Painting (artist brief)

### 2.1 Composition

- **Canvas**: 16:9, authored at 2048×1024 (PoT, exact 16:9). Painterly, golden
  register, warm key light from a low sun right-of-center (~70% width). Long shadows.
  The river is a warm mirror catching gold.
- **Read order** (per UX spec hierarchy): campfire first, then river, then ships,
  then far forest.
  - Campfire: ~62% width, ~68% height — the warmest glow in the frame.
  - River: midground band, slow, mirroring gold.
  - Ships: two merchant hulls moored at the near bank, ~78–88% width. Worn but
    working — faded paint, honest wear.
  - Far forest: across the water, warm-dark mass with interior shadow — the forest
    watches.
  - Horizon: ~56% height.
  - Pale mist tease: a static, ambiguous pale river-mist patch in the far-forest
    shadow. Never moves, never breathes, never resolves — atmosphere, not empire
    material. The two-light collision is teased at the absolute minimum.
- **The panel band (left 0–33% width)**: calm content — shoreline, reeds, a leaning
  mooring post, calm water. Mid-dark values — no bright sky hotspot in this band
  (the panel's contrast depends on it).
- **Warmth, not cozy**: the far forest holds real shadow; one worn detail. The world
  has edges even here.
- **No characters. No empire material, no ivory, no ghost glow.** No pure black, no
  pure white (art bible 9.8).
- **Safe areas**: all focal content inside the center ~66% of both axes; top/bottom
  ~10% overscan bands kept simple (crop-tolerant for 21:9/16:10).

### 2.2 Motion — "Painting in motion"

MVP: static painting + particle overlay. Motion vocabulary: drift, float, settle;
nothing pops, nothing screensaver.

| Element | Spec | Budget |
|---|---|---|
| Smoke plume (campfire) | 24–32 soft sprites, alpha blend, slow rise-drift right, irregular | ≤32 |
| Embers | ≤8, small additive cores, 2–3 visible at once | ≤8 |
| River shimmer | ≤24 subtle sparkle sprites, alpha, slow | ≤24 |
| Far-bank leaves | ≤8, slow drift, alpha | ≤8 |
| Ship sway (optional) | 2D transform on separate transparent layer, ±2px, 8–10s sine | — |
| Camera drift | NONE at MVP — static painting | — |

Total ≤ 100 particles, ≤ 4 emitters, long imperceptible loops (≥30s). All sprites
from the shared soft-sprite atlas. No UI particles anywhere on this screen — the
desk does not breathe; the smoke is the scene's, never the menu's.

---

## 3. Visual Style Notes — The Desk Register

### 3.1 Color assignments

All values are the 2.4 palette, flat, unmodified by world lighting.

| Component | Treatment | Hex |
|---|---|---|
| Parchment panel | Parchment fill at 92–95% flat alpha, no drop shadow, non-uniform hand-cut corner radii 3/5/4/6px, optional 1px inner hairline ink at 25% | `#E3D5B0` |
| Game title | Ink, desk serif small caps | `#1C1813` |
| Title rule (below title) | 1px ink hairline at 25%, width = content width | `#1C1813` @25% |
| New Game / Load Game / Settings buttons | Ink label on parchment, transparent face | `#1C1813` |
| Quit to Desktop button | Same, secondary treatment | `#1C1813` |
| Button hover (mouse) | Amber 2px left-edge rule + label Regular→Medium | `#D9A441` |
| Button press | Thread-red 2px underline | `#C4483A` |
| Keyboard/gamepad focus | Oxblood 2px ring, 2px outside button bounds — never amber | `#7A2E28` |
| Disabled (Load Game, no saves) | Ink at 40% opacity — bleached, no hover response | `#1C1813` @40% |
| Separator above Quit | 1px ink hairline at 25%, width = content width | `#1C1813` @25% |
| Version number | Parchment at 70% opacity, 1px soot outline, over scene's darker bottom band | `#E3D5B0` / `#141210` |
| Modal/confirm dim | Soot overlay at 72% flat → scene at ~28–30% | `#141210` @72% |
| Modal panel | Same parchment treatment as main panel | `#E3D5B0` |
| Dropdown | Parchment list, ink label + ink chevron glyph; selected row = thread-red tint at 18% + ink check glyph | per 6.5 |
| Subtitles toggle | Umber track `#2E2820`, parchment knob; On = knob right + ink check in track | `#2E2820`/`#E3D5B0` |
| "Remember my choice" checkbox | 18px parchment box, 1px ink-at-50% border; checked = ink check glyph | `#1C1813` |
| Start Game / Begin New Game | Positive action — hover amber edge, press thread-red underline | `#D9A441`/`#C4483A` |
| Cancel | Neutral — hover ink edge (50%), press ink underline | `#1C1813` |
| Error toast | Parchment chip, 1px ink-25% border, ink text 14px + oxblood knot glyph 24px | `#C4483A` glyph |
| Load Game tooltip | Parchment chip, 1px ink-25% border, ink 13px | `#1C1813` |

**Ivory usage: none on this screen.** No empire items exist here.

### 3.2 Typography

| Element | Face | Size @1080 | Weight | Treatment |
|---|---|---|---|---|
| Game title | Desk serif (Garamond class) | 56px, two lines | Regular | Small caps, letterspacing +3%, line-height 1.15, ink |
| Menu buttons | Desk serif | 18px | Regular → Medium on hover | Ink on parchment |
| Modal body | Desk serif | 16px (body floor) | Regular | Ink |
| Modal headers | Desk serif | 24px | Regular, small caps | Ink |
| Version | Desk serif | 13px (caption floor) | Regular | Parchment @70% + soot outline |
| Tooltip / toast | Desk serif | 13–14px | Regular | Ink |

Font: EB Garamond / Cormorant / Source Serif — final pick with technical-artist;
OFL-licensed, ships with game. Required cuts: Regular, Medium, Italic + OpenType
small caps. No blackletter, no cold modern serif.

### 3.3 Layout and proportions

All values scale by `s = height/1080`.

- **Panel**: left offset 48px from screen edge, width 480px (clamped:
  `min(480·s, 0.9 · screen-width/3)`), vertically centered, padding 48px top/bottom
  / 40px sides.
- **Internal flow**: title block → 1px title rule → primary buttons → 24px gap +
  1px separator rule → Quit.
- **Buttons**: height 44px, horizontal padding 24px, label 18px, gap 12px,
  left-aligned, flex to content width (localization stress case: "Einstellungen").
- **Panel total height** ≈ 460px at 1080 — a floating sheet, not a full-height column.
- **Modal**: width 520px, centered, padding 40px; controls stacked with 20px gaps;
  Start Game right-aligned; no close glyph.
- **Version**: 16px from bottom-right corner, text 13px.
- **Toast**: bottom-center, padding 16px, knot glyph left of text.

### 3.4 Scene light × parchment panel

1. The drawn colors never shift — the panel's palette values are the 2.4 values
   regardless of what is behind them.
2. Uniform alpha is flat — 92–95% flat opacity allowed; gradients are not.
3. Contrast: the painting's left band must be calm and mid-dark so ink-on-panel
   stays ≥ 4.5:1 at the brightest left-third pixel.
4. The scene never dims behind the panel. Only the modal and confirmation dialog dim.
5. No tint, no glow, no drop shadow — translucency is the separator.

### 3.5 Button states

| State | New Game / Load Game / Settings | Quit to Desktop |
|---|---|---|
| Default | Ink label on parchment, no accents | Ink label; 1px ink-25% hairline above |
| Hover (mouse) | Amber 2px left-edge rule, 80ms fade + label Regular→Medium | Oxblood 2px left-edge rule, 80ms fade |
| Focus (Tab/D-pad) | Oxblood 2px ring, 2px outside bounds, instant (0ms) | Same |
| Hover + focus | Both cues coexist: ring outside, lamp edge inside | Same |
| Pressed | Thread-red 2px underline, 80ms | Oxblood 2px underline, 80ms |
| Disabled | Ink at 40% opacity, no hover, tooltip on hover/focus | — |

No scale, no 3D transform, no bounce, no positional shift on press — the desk is
flat by law (6.4).

---

## 4. Animation / Motion Spec

Easing curves:
- **Ease-out** `cubic-bezier(0.22, 0.61, 0.36, 1)` — all entries
- **Ease-in** `cubic-bezier(0.55, 0.06, 0.68, 0.19)` — all exits
- **Ease-in-out** `cubic-bezier(0.65, 0, 0.35, 1)` — crossfades only

All motion = translateY + opacity only. No rotate, skew, scale, or 3D.

| Moment | Visual | Duration | Easing |
|---|---|---|---|
| Scene enter | Fade in from warm soot `#141210` | 500ms | ease-out |
| Menu arrival (t=800ms) | Sheet slides up (translateY 32px→0) + fade | 300ms | ease-out |
| Version number | Fade in only, +100ms after sheet | 200ms | ease-out |
| Modal in | Soot dim @72% + modal slides up (48px→0) + fade | Dim 200ms; slide 300ms | ease-out |
| Modal out | Modal slides down (0→40px) + fade; dim lifts | Slide 200ms; lift 200ms | ease-in |
| Toast in / out | Slides up from bottom (24px→0) / away (0→24px) + fade | 200ms / 200ms | ease-out / ease-in |
| Crossfade to Load/Settings | Pure opacity cross, both screens simultaneously | 250ms | ease-in-out |
| Load Game desaturate | Color leaches: ink → ink@40% over opacity crossfade | 150ms | ease-out |
| Hover edge / press underline | 2px rules fade in/out | 80ms | ease-out |
| Focus ring | Instant (0ms) — focus is never gated behind animation | 0ms | — |
| Reduced motion ("Disabled") | All above → 0ms instant, except scene fade (500ms retained — atmospheric) | — | — |

---

## 5. Asset Manifest

| # | Asset | Type | Dims | Format | Notes |
|---|---|---|---|---|---|
| 1 | `ui_bg_title_camp_goldenhour_1080p.png` | Painted backdrop | 2048×1024 PoT | PNG | Full painting per Section 2.1. No ivory, no pure black/white. Left band calm and mid-dark. |
| 2 | `ui_bg_title_ships_moored_large.png` | Transparency layer (optional) | 2048×1024 | PNG | Ships + near bank only, rest transparent. For ±2px sway. Cut for MVP if needed. |
| 3 | `ui_panel_parchment_512.png` | Tileable grain (optional) | 512×512 PoT | PNG | Paper grain at ≤3% amplitude. If not subtle, omit — flat is the law. |
| 4 | `ui_icon_check_ink_48.png` | Pen-and-ink glyph | 48×48 (2× display) | PNG | Single stroke, 4px at 48px canvas. Dropdown selection, checkbox, toggle-on. |
| 5 | `ui_icon_chevron_ink_48.png` | Pen-and-ink glyph | 48×48 | PNG | Dropdown affordance, same stroke law. |
| 6 | `ui_icon_knot_oxblood_48.png` | Pen-and-ink glyph | 48×48 | PNG | Broken thread — error toast glyph. Line work, no fill. |
| 7 | `ui_font_deskserif_[weight].otf` | Font | — | OTF/TTF | Desk serif, Garamond class, OFL. Cuts: Regular, Medium, Italic + small caps. |
| 8 | `vfx_sprite_smoke_puff_64.png` (+ ember, sparkle, leaf) | Soft sprites | 64×64 each | PNG | Shared soft-sprite atlas. Smoke = alpha blend, embers = small additive cores. |

No assets needed for: focus ring, button faces, hairline rules, dim overlay, version
text, modal — all CSS-composed. Cursor: OS default.

---

## 6. Resolution & Scaling Strategy

**Two-scale system**: layout scales by `height/1080`; text scales but clamps to
legibility floors.

| Element | @1080 | @720 clamped |
|---|---|---|
| Title | 56px | 37px |
| Button label | 18px | 16px (clamped) |
| Version | 13px | 13px (clamped) |
| Panel width | 480px | 320px |

**Stress cases**: "Einstellungen" (German) at 16px clamp in 267px content width
(must verify). Aspect ratios other than 16:9 — painting is cover-cropped.

**Verification**: screenshots at 1920×1080, 1280×720, 1024×768; ink-on-panel ≥4.5:1
over brightest left-third pixel; oxblood ring ≥3:1; focus ring always visible; all
German strings fit; reduced-motion = instant UI + retained scene fade.

---

## 7. Art Bible Conflicts & Resolutions

| # | Item | Resolution |
|---|---|---|
| 1 | State 6 "ruined vista" vs riverside camp | State 6 amended to riverside camp (user approved) |
| 2 | UX spec focus wording: "amber edge on hover/focus" | Focus = oxblood ring only; amber edge = mouse hover only |
| 3 | GDD dropdown selection = amber; art bible = thread-red | Resolved to thread-red 18% tint + ink check |
| 4 | "Fade from black" → soot | All fade targets use warm soot `#141210` |
| 5 | Title screen panel doesn't dim the scene (desk dim rule exception) | Codified in art bible 6.1 (user approved) |
| 6 | Two-light collision tease | Pale river-mist patch added to scene (user approved) |

---

## 8. Implementation Notes (Kaiju HTML/CSS-like UI)

- Flat fills and 2px borders via CSS; `background-color` with RGBA for panel/dim;
  **no box-shadow anywhere**.
- Slides via `transform: translateY` + `transition` with cubic-bezier curves; never
  rotate/skew/scale.
- Focus ring: `outline`-style oxblood 2px, offset 2px, instant (no transition).
- Fonts ship OFL; small caps via OpenType `font-variant: small-caps`.
- Scene: full-viewport image + particle overlay (technical-artist owns emitter system).
- Reduced motion: single setting gate collapses all UI transition durations to 0;
  scene fade is the only retained transition.
- Text-scale accessibility option must not push text below legibility floors.
