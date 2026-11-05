# Story 001: Combat HUD Framework & Turn Order

> **Epic**: Combat UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: 4h
> **Manifest Version**: 2026-08-09
> **Last Updated**: [set by /dev-story when implementation begins]

## Context

**GDD**: `design/gdd/combat-ui.md`
**Requirement**: `TR-ui-002`, `TR-ui-004`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: All UI rendered in screen space through Kaiju's HTML/CSS-like UI system. Three registers: the Field register (umber combat chrome, no world dim) hosts the combat HUD. Paper-slide motion only: enter 200-300ms ease-out, exit 200ms — no bounce, scale, spin, or flip. Combat face font (grotesque sans, tabular figures) for all combat numbers.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI system — verify rendering API, font loading, dynamic element updates (turn order reordering), input routing. Post-cutoff APIs, no LLM training data exists.

**Control Manifest Rules (this layer)**:
- Required: Field register (panel umber, no dim) for combat HUD; paper-slide motion 200-300ms ease-out; combat face (grotesque sans, tabular figures) for AP/HP/damage numbers; minimum 1280×720, reference 1920×1080 with height-based scaling
- Forbidden: Never use 3D UI or diegetic HUD; never allow amber-on-parchment text (amber is accent/fill only — umber panel background avoids this); never encode game-critical information by hue alone
- Guardrail: UI render (static) <1ms/frame; no frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/combat-ui.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN combat with 4 player characters and 3 enemies, WHEN initiative is rolled, THEN the turn order bar shows all 7 portraits in correct initiative order (descending) with the first combatant highlighted by an amber border, and upcoming turns dimmed.
- [ ] **AC-2**: GIVEN the combat HUD, WHEN combat starts, THEN the turn order bar slides in from the top (200ms ease-out), the bottom bar (AP meter + ability menu + End Turn) slides in from the bottom (200ms ease-out), and the Field register panel umber background is established with the world undimmed.
- [ ] **AC-3**: GIVEN the current turn ends, WHEN initiative advances to the next combatant, THEN the amber highlight slides from the current portrait to the next portrait (paper-slide motion, no bounce/scale/spin), and the AP bar refills with an amber pulse.
- [ ] **AC-4**: GIVEN combat ends, WHEN the Resolution state triggers, THEN all combat HUD elements slide out (200ms ease-out — top bar exits upward, bottom bar exits downward).
- [ ] **AC-5**: GIVEN the turn order bar, WHEN the player hovers over a combatant portrait, THEN a tooltip displays the combatant's name and current/max HP in combat face font.

---

## Implementation Notes

*Derived from ADR-0012:*

- **Field register**: The combat HUD uses the Field register — panel umber background, no world dimming. This is distinct from Desk (parchment, dims world) and Desk edge (parchment, world live). The umber panel must read against both golden day and cold night lighting (test at Dawn, Day, Dusk, and Night light states).
- **Layout**: Top bar (turn order — horizontal flex row of portrait chips, full width). Bottom bar (flex row: AP meter left, ability grid center, End Turn button right). World overlays for health bars, damage numbers, targeting (stories 003–005). Use Kaiju HTML/CSS-like layout.
- **Turn order bar**: Horizontal row of combatant portrait thumbnails in initiative order (descending, left to right). Each portrait is a small circular or rounded-square frame. Current turn: amber border highlight (#D4A040 or art-bible-defined amber), full opacity. Upcoming turns: dimmed (opacity 0.5 or desaturated). Acted turns: marked with a subtle check or slightly further dimmed. Hover tooltip: `position: absolute` or Kaiju-equivalent popup showing name + "HP: 42/56" in combat face (grotesque sans, tabular figures).
- **Slide animations**: Combat start — top bar slides in via `transform: translateY(-100%)` → `translateY(0)` over 200ms ease-out. Bottom bar: `translateY(100%)` → `translateY(0)` over 200ms ease-out. Combat end — reverse (top exits upward, bottom exits downward). No bounce, scale, spin, or flip. Audio triggers: combat start/end sounds via EventBus → Audio System.
- **Turn change animation**: When initiative advances: remove amber highlight class from current portrait, add to next portrait. The highlight transition uses a CSS transition on `border-color` and `box-shadow` over 150ms ease-out — this is the "slide" of the highlight, not a positional slide. AP bar refill: animate AP bar fill width from current to max over 200ms with an amber pulse (brief brighter glow at completion).
- **Combat state integration**: Subscribe to `CombatStarted`, `TurnChanged`, `CombatEnded` events via EventBus. On `CombatStarted`: populate turn order from initiative data, trigger slide-in. On `TurnChanged`: update highlight + AP bar. On `CombatEnded`: trigger slide-out.
- **Fonts**: Combat face (grotesque sans, tabular figures) for all numbers in the HUD — AP, HP in tooltips, initiative numbers. Tabular figures mean all digits occupy the same width — critical for AP counting down without layout shift. OFL-licensed font files shipped with the game. Final font selection deferred to technical-artist.
- **Resolution**: Reference 1920×1080 with height-based scaling. Minimum 1280×720 — at min resolution, portraits may shrink but must remain identifiable. Tooltip text minimum 12px effective.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: AP meter fill/depletion logic and display, ability menu grid, End Turn button
- Story 003: World-anchored health bars above combatants
- Story 004: Floating damage numbers, crit effects, combat audio feedback
- Story 005: Targeting reticle, range line, flanking/cover indicators, hit chance display
- Story 006: Victory/defeat resolution screens

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1** (turn order displays correct initiative order):
  - Setup: Start combat with 4 player characters (initiative values: 8, 5, 3, 1) and 3 enemies (initiative: 6, 4, 2)
  - Verify: Portraits appear in order: PC1(8), Enemy1(6), PC2(5), Enemy2(4), PC3(3), Enemy3(2), PC4(1). First portrait (PC1) has amber border highlight. Remaining portraits are dimmed.
  - Pass condition: All 7 portraits visible in correct descending initiative order, first highlighted

- **AC-2** (combat start slide-in animations):
  - Setup: Trigger combat encounter start from exploration
  - Verify: Top bar slides down from top edge over 200ms (ease-out, no bounce). Bottom bar slides up from bottom edge over 200ms. Panel umber background visible. World behind is undimmed (playable area still visible).
  - Pass condition: Both bars fully visible within 200ms, ease-out curve, no bounce, world undimmed

- **AC-3** (turn change highlight transition):
  - Setup: In active combat, end the current combatant's turn
  - Verify: Amber highlight leaves current portrait and appears on next portrait over ~150ms (border-color transition). AP bar fills to max with amber pulse (brief brightness spike at completion).
  - Pass condition: Highlight moves cleanly, AP bar refills, no layout shift, no bounce/scale/spin

- **AC-4** (combat end slide-out):
  - Setup: Defeat all enemies to trigger combat end
  - Verify: Top bar slides up (exits upward) over 200ms. Bottom bar slides down (exits downward) over 200ms. World overlays (health bars, damage numbers) also hide.
  - Pass condition: All combat UI hidden within 200ms, exploration UI restored

- **AC-5** (hover tooltip on portrait):
  - Setup: In active combat with visible turn order bar
  - Verify: Hover cursor over a combatant portrait → tooltip appears showing name + "HP: current/max" in grotesque sans tabular figures. Move cursor away → tooltip dismisses.
  - Pass condition: Tooltip content correct, font correct, appears/disappears on hover

---

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/combat_ui/hud_framework_test.go` OR playtest doc

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: None (first story in epic — can start immediately, but requires combat engine to emit `CombatStarted`/`TurnChanged`/`CombatEnded` events)
- Unlocks: Story 002 (AP Meter & Ability Menu), Story 003 (World Health Bars), Story 004 (Damage Numbers), Story 005 (Targeting), Story 006 (Resolution Screens)
