# Accessibility Requirements

> **Status**: In Design
> **Author**: Claude + ux-designer
> **Last Updated**: 2026-08-09
> **Committed Tier**: WCAG-AA equivalent for game UI

## Tier Commitment

The Marrow of Empire commits to **WCAG-AA equivalent** accessibility for all
player-facing UI. This means:

- All text meets minimum contrast ratios (4.5:1 for normal text, 3:1 for large text)
- All interactive elements are keyboard-navigable with visible focus indicators
- No information is communicated by color alone
- All motion and animation can be reduced or disabled
- Text can be resized without loss of content or functionality

## Input Accessibility

### Keyboard Navigation
- Every interactive element must be reachable via Tab/Shift+Tab in logical order
- Enter/Space activates the focused element
- Esc closes the current overlay/panel (dialogue: ends conversation; outside: opens pause)
- Arrow keys navigate within compound elements (topic chips, inventory grid, party sidebar)
- Number keys 1-6 activate combat ability slots
- Focus indicator: oxblood or umber outline, ≥3:1 contrast against all backgrounds
- Focus must land in the text input when conversation opens
- Focus must not be stolen when topic chips refresh after NPC response

### Gamepad Navigation
- D-pad/left stick navigates between elements
- A button = Enter/click
- B button = Esc/back
- Shoulder buttons = tab between panels
- Gamepad supported as secondary input (primary: keyboard/mouse per technical-preferences)

### Input Remapping
- All keyboard hotkeys rebindable via Settings > Controls
- Default bindings: J (journal), C (character), I (inventory), M (map), F5 (quicksave), F9 (quickload), Home (center camera), Esc (pause/back)

## Visual Accessibility

### Contrast Requirements
| Element | Required Ratio | Verified |
|---------|---------------|----------|
| Body text (ink `#1C1813` on parchment `#E3D5B0`) | ≥4.5:1 | ✅ 12.1:1 |
| Large text / headings | ≥3:1 | ✅ |
| UI controls (buttons, inputs) | ≥3:1 | ✅ |
| Focus indicators | ≥3:1 | ✅ oxblood/umber on all surfaces |
| Amber on parchment | — | ❌ 1.5:1 — amber is never body text |
| Oxblood `#7A2E28` on parchment | ≥4.5:1 (text) | ✅ 6.4:1 |
| Thread-red `#C4483A` on parchment | ≥3:1 (large/non-text) | ⚠️ 3.3:1 — use for ≥2px lines and large glyphs only |

### Colorblind Safety
Per art bible Section 2.5 — no game-critical information communicated by hue alone:

| Pair | Backup Cues |
|------|-------------|
| Health (amber/oxblood) | Bar length + numeric readout + faction frame shape (rounded vs. angular) |
| Ivory vs. Amber (revelation pair) | Motion (ivory breathes/shimmers) + value (higher key) + sound (hum) + source anchoring |
| Thread-red vs. Moss | Value separation (oxblood dark, moss mid) + shape (blood pools glossy, foliage textured) |
| Indigo vs. Soot | Warm pools always coexist; silhouettes stay readable; threat uses rim light |

### Text Scaling
- Base body text: 16px at 1920×1080 reference
- Scale factor = screen height / 1080
- Minimum effective text size: 12px (below this, text is redesigned, not shrunk)
- Accessibility setting: Text Scale slider (0.8×–2.0×)
- World-anchored text (damage numbers): scales with camera zoom per LOD contract; minimum on-screen size 14px

## Motion & Animation

### Motion Reduction
- Setting: Reduce Motion (toggle) — default: off
- When enabled:
  - All UI slide/fade transitions replaced with instant cuts
  - Typewriter effect disabled — text appears instantly
  - Amber/oxblood pulses disabled — static indicators instead
  - Camera shake disabled globally
  - Thinking indicator: static "..." instead of animated dots

### Animation Constraints
- No animation exceeds 1 second without purpose
- Only one dominant motion at a time
- No flashing content >3 flashes per second (photosensitive epilepsy safety)
- Combat screen shake: configurable intensity 0.0–2.0; 0.0 = disabled

## Screen Reader Support

Kaiju Engine targets platforms with AccessKit support. Requirements:

- Journal entries: announced as "Thread: [description]. Source: [NPC name]."
- NPC responses: announced once when complete (not character-by-character during typewriter)
- UI state changes: announced ("Conversation started with Greta." "Combat started. 4 enemies.")
- Interactive elements: all have accessible labels
- Error states: announced with error description
- Empty states: announced ("No active threads. Talk to NPCs to discover leads.")

## Subtitles & Captions

- All NPC dialogue is text-based — subtitles are inherent
- Critical audio cues (combat start, revelation hum, danger sound) have optional visual indicators
- Setting: Audio Cues → Visual (toggle) — default: off. Shows brief text for critical audio events.

## Platform-Specific

### Linux (primary target)
- AccessKit support via Kaiju Engine (Godot 4.5+ ships AccessKit; Kaiju compatibility TBD)
- System text scaling respected where possible
- High contrast mode: not required (game provides its own)

### Windows
- Same as Linux requirements
- Xbox gamepad support via XInput

## Open Questions

| Question | Owner | Deadline |
|----------|-------|----------|
| Does Kaiju Engine's HTML/CSS-like UI support AccessKit screen reader integration? | Engine Programmer | Before UI implementation |
| Should there be a high-contrast UI mode that replaces the parchment palette with solid black-on-white? | UX Designer | Before vertical slice |
| Gamepad support: required for MVP or deferred to VS? Technical-preferences says "Partial" | Producer | Before MVP sprint planning |
