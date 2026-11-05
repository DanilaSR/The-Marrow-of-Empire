# Story 001: Navigation Accessibility & Screen Transition Framework

> **Epic**: Main Menu & Settings
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: M (4-6 hours) — foundation UI: 3-input navigation, focus system, 7-state screen machine, crossfade transitions, reserved key enforcement. HIGH engine risk (Kaiju custom UI verification required).
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

---

## Completion Notes

**Completed**: 2026-08-11
**Criteria**: 10/10 passing (all automated — 39 unit tests)
**Deviations**: None
**Test Evidence**: `tests/unit/main_menu/navigation_transitions_test.go` — 39 tests, all PASS
**Code Review**: Complete — 2 bugs fixed (Activate false-positive return, back-stack state loss), Dispatch refactored to ≤10 complexity, re-reviewed and APPROVED
**Advisory**: Manual evidence file `production/qa/evidence/story-001-navigation-transitions-evidence.md` not yet created — deferred until Story 002 ships the first renderable screen

## Context

**GDD**: `design/gdd/main-menu-and-settings.md`
**Requirement**: `TR-ui-004`, `TR-ui-007`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: All UI rendered in screen space via Kaiju HTML/CSS-like UI. Three-input navigation required (keyboard Tab/arrows/Enter/Escape, gamepad D-pad/A/B, mouse click/scroll). Focus ring ≥3:1 contrast on all interactive elements. Paper-slide transitions 200-300ms ease-out. Desk never trembles.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI capabilities must be verified against `kaiju/src/`. Focus ring rendering, input routing (keyboard/gamepad/mouse dispatch), transition animation hooks. AccessKit screen reader support per ADR-0012 verification list.

**Control Manifest Rules (Presentation layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space
- Required: Paper-slide motion: enter 200-300ms ease-out, exit 200ms — no bounce/scale/spin/flip
- Required: Accessibility: tab order + focus ring (≥3:1), arrow-key navigation, Esc mapping
- Forbidden: Never use 3D UI or diegetic HUD
- Forbidden: Never encode game-critical information by hue alone
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/main-menu-and-settings.md` CR-8 and transition rules:*

- [ ] **AC-1**: All screens and dialogs are navigable by keyboard alone (Tab/Shift+Tab moves focus, arrows within groups, Enter/Space activates, Escape = back/cancel at every level).

- [ ] **AC-2**: All screens and dialogs are navigable by gamepad alone (D-pad/left stick moves focus, A = activate, B = back/cancel, Start/Options = pause toggle).

- [ ] **AC-3**: All screens and dialogs are navigable by mouse alone (click to focus and activate, scroll wheel for sliders and lists).

- [ ] **AC-4**: Every focusable element renders a visible focus ring: ink outline, ≥3:1 contrast against all menu backgrounds, 2px width, DPI-scaled. Focus ring is always present during keyboard/gamepad navigation. It may be hidden during mouse navigation and shown on first keyboard/gamepad input.

- [ ] **AC-5**: Color changes alone (amber highlight, thread-red underline) are secondary cues — never the sole indicator of focus or selection state.

- [ ] **AC-6**: The Escape key and F9 (quick-load) cannot be completely unbound. At least one binding must remain for each reserved function. The system blocks unbinding the last remaining binding for any reserved function with a warning.

- [ ] **AC-7**: All screen-to-screen transitions (Title→Load, Settings→Title, Pause→Save, etc.) use a crossfade of `screen_crossfade_ms` (250ms ±10%). During the crossfade, input is not processed (prevents double-clicks across screens).

- [ ] **AC-8**: The Settings screen remembers its entry point (TitleScreen vs PauseMenu) to determine the back-destination. The back button (or Escape in sub-screens) always returns to the previous screen.

- [ ] **AC-9**: The Loading state (save load failure: corrupted, version mismatch) transitions to the Title Screen with an error message describing the failure. The game does not crash or hang.

- [ ] **AC-10**: `InGame` → `TitleScreen` always requires confirmation if unsaved progress exists.

---

## Implementation Notes

*Derived from ADR-0012 Implementation Guidelines:*

- **Screen-space rendering**: All menu UI renders in screen space, not world space. UI elements are Kaiju `engine.Entity` instances managed by `ui.Manager`. UI is immune to camera shake — the desk never trembles.
- **Input routing**: Implement three independent input paths (keyboard, gamepad, mouse) that all produce the same navigation actions. No screen requires a specific input device. Input is blocked during crossfade transitions.
- **Focus system**: Maintain a focusable-element registry per screen. Tab/Shift+Tab moves forward/backward. Arrow keys navigate within groups. Focus ring is rendered as a CSS border/outline on the focused element (ink color, 2px, DPI-scaled).
- **Transition state machine**: Implement the 7-state screen-flow machine (TitleScreen, LoadGame, InGame, PauseMenu, SaveGame, Settings, Loading) with transition rules as defined in the GDD. Each transition triggers a crossfade animation; input is queued/discarded during the animation.
- **Reserved key enforcement**: Maintain a set of reserved functions (Escape=pause, F9=quick-load). The unbind operation checks: if this is the last binding for a reserved function, reject with warning.
- **Font loading**: OFL-licensed fonts ship with the game. No runtime network fonts. Final font selection (Garamond-class serif for desk, grotesque sans for combat) deferred to technical-artist.
- **Minimum resolution**: 1280×720. Reference: 1920×1080 with height-based scaling. Body text 16px at reference.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Title screen background, menu buttons, New Game flow
- Story 003: First-launch accessibility prompt modal
- Story 004: Save/Load slot browser, Steam Cloud conflict dialog
- Story 005: Settings screen UI, per-category layouts, persistence
- Story 006: Pause menu overlay, world freeze, combat pause variant
- Story 007: Keybinding capture interface, conflict resolution UI

---

## QA Test Cases

*Derived from GDD acceptance criteria. The developer implements against these — do not invent new test cases during implementation.*

### Automated (UI state assertions)

- **AC-1** (Keyboard navigation):
  - Given: any screen with 5 focusable elements
  - When: Tab is pressed 5 times
  - Then: focus visits all 5 elements in order; Shift+Tab reverses the order
  - Edge cases: focus wraps from last to first (or stops at boundary per screen design); disabled elements are skipped

- **AC-4** (Focus ring visibility):
  - Given: a screen is displayed and keyboard navigation is active
  - When: focus moves to a button
  - Then: the focused element renders a visible outline with ≥3:1 contrast ratio against its background, 2px width
  - Edge cases: focus ring updates within one frame of focus change; ring is hidden during mouse navigation until first keyboard/gamepad input

- **AC-7** (Crossfade transition):
  - Given: screen A is displayed
  - When: a transition to screen B is triggered
  - Then: a crossfade of `screen_crossfade_ms` (250ms) occurs; input events arriving during the crossfade are discarded
  - Edge cases: rapid double-click on a transition-triggering button does not cause double-transition; transition completes even if frame time spikes

- **AC-6** (Reserved key protection):
  - Given: Escape is bound only to Pause Menu (1 binding)
  - When: the player attempts to unbind Escape
  - Then: the operation is blocked with warning "This is the only binding for Pause Menu. It cannot be removed."
  - Edge cases: unbinding a non-last binding for a reserved function succeeds; F9 follows same rules

- **AC-9** (Loading state error):
  - Given: a save file with checksum mismatch
  - When: the player attempts to load it
  - Then: the screen transitions to Title Screen with an error message; the game does not crash or hang
  - Edge cases: version mismatch produces same behavior; partial-load temp state cleaned up on next launch

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `tests/unit/main_menu/navigation_transitions_test.go` AND manual walkthrough doc OR interaction test at `production/qa/evidence/story-001-navigation-transitions-evidence.md`

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: None (foundation for all other menu stories)
- Unlocks: Story 002 (Title Screen), Story 004 (Save/Load), Story 005 (Settings), Story 006 (Pause Menu), Story 007 (Keybindings)
