# Story 002: Title Screen & New Game Flow

> **Epic**: Main Menu & Settings
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: M (4-6 hours) — title screen background composition, fade-in sequencing, 3 menu buttons with hover/active states, New Game flow (difficulty selection + character creation handoff), confirmation dialogs, greyed-out state logic. HIGH engine risk (Kaiju UI rendering).
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/main-menu-and-settings.md`
**Requirement**: `TR-ui-001`, `TR-ui-004`, `TR-ui-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: Desk register — full-screen parchment overlay with world dimmed to 30%. Title screen is a dark fantasy composed scene (art bible State 6: storybook dusk vista). Paper-slide motion for menu element entry. Kaiju HTML/CSS-like UI for all rendering.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Title screen background rendering (static image + particle overlay per GDD open question). Kaiju UI element fade-in timing. Menu button rendering with hover/active states. Screen-space UI unaffected by camera. Font loading for title typography.

**Control Manifest Rules (Presentation layer)**:
- Required: Desk register — full-screen parchment overlay, world dims to 30%
- Required: Paper-slide motion: enter 200-300ms ease-out, no bounce/scale/spin/flip
- Required: Desk serif (Garamond class) for menus
- Required: Minimum 1280×720, reference 1920×1080 with height-based scaling
- Forbidden: Never use 3D UI or diegetic HUD
- Forbidden: Never use skeuomorphic buttons — pen-and-ink icons only
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/main-menu-and-settings.md` CR-1 and CR-5:*

- [ ] **AC-1**: On first launch (no `settings.cfg`), the title screen background fades in over `title_fade_in_ms` (500ms ±10%). Menu elements (New Game, Load Game, Settings) appear 300ms after the background is visible. The first-launch accessibility prompt (Story 003) appears as a modal; title screen menu buttons are visible but non-interactive until the prompt is dismissed.

- [ ] **AC-2**: The title screen displays: a dark fantasy background composition, three primary menu options (New Game, Load Game, Settings), version number (bottom-right), and "Quit to Desktop" (secondary position). Menu buttons use parchment-toned background with ink-dark text. Hover state: warm gold highlight edge. Active state: thread-red underline.

- [ ] **AC-3**: When the player clicks "New Game" with no existing saves, the difficulty selection screen appears (Narrative / Normal / Tactical). After difficulty is chosen and character is confirmed, the screen crossfades to black (`new_game_fade_ms` = 500ms) and the game begins in the starting region.

- [ ] **AC-4**: When save files exist and the player clicks "New Game", a confirmation dialog appears: "Starting a new game will not overwrite your existing saves." Options: "Begin New Game" / "Cancel". The new game flow proceeds only after "Begin New Game" is clicked.

- [ ] **AC-5**: The "New Game" button is non-interactive (greyed out) while a save write operation is in progress.

- [ ] **AC-6**: When all saves are deleted, "Load Game" on the title screen is greyed out with a tooltip: "No saved games found." "New Game" remains available.

---

## Implementation Notes

*Derived from ADR-0012 Implementation Guidelines:*

- **Background composition**: At MVP, a static image with shader-based particle overlay (drifting embers, pollen, dust motes) is the lowest-risk path per GDD open question. The art director owns the composition; the menu system owns fade-in timing and UI overlay. Reference: art bible State 6 storybook dusk vista — ruined archway, golden forest at sunset, borderland.
- **Menu button styling**: Parchment-toned background (`background-color` referencing desk palette), ink-dark text (`color` from palette), hover via CSS `:hover` with gold edge, active via CSS `:active` with thread-red underline. No skeuomorphic bevels/gradients — pen-and-ink flat icons only.
- **Fade-in sequencing**: Background fades in over `title_fade_in_ms` (500ms) via CSS opacity transition. Menu buttons fade in after `menu_element_delay_ms` (300ms) delay — implemented as a CSS animation-delay or Go-triggered class toggle after a timer.
- **New Game flow**: Screen-flow state machine transitions: TitleScreen → difficulty selection (inline UI state, not a separate screen state) → Character Creation handoff (provisional contract — at MVP, pre-made character template with name entry) → Loading → InGame.
- **Difficulty selection**: Renders three options from the Tactical Combat Engine GDD's difficulty definitions. Stores chosen difficulty in the new game session. The menu is the UI layer only — combat owns the definitions.
- **Confirmation dialogs**: Darkened overlay (80% black) with parchment-modal foreground. Non-destructive buttons: parchment with ink text. See CR-2 and CR-4 for destructive variant styling.
- **Greyed-out state**: "Load Game" when no saves exist, "New Game" during save-in-progress — reduced opacity (40%), no hover effect, cursor: not-allowed, tooltip on hover.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Navigation (Tab/arrows/gamepad), focus rings, crossfade transitions, input blocking
- Story 003: First-launch accessibility prompt modal content and logic
- Story 004: Save/Load screen — "Load Game" button triggers Story 004's screen
- Story 005: Settings screen — "Settings" button triggers Story 005's screen
- Story 006: Pause menu — "Quit to Main Menu" from pause flow
- Character Creation system: provisional handoff contract — at MVP, pre-made template with name entry

---

## QA Test Cases

*Derived from GDD acceptance criteria. The developer implements against these — do not invent new test cases during implementation.*

### Automated (UI state assertions)

- **AC-1** (First-launch fade-in sequence):
  - Given: no `settings.cfg` exists, engine finishes boot
  - When: title screen initializes
  - Then: background opacity transitions from 0 to 1 over 500ms (±10%); menu buttons appear at opacity 1 after 300ms delay; menu buttons have `interactive: false` while accessibility prompt is active
  - Edge cases: `settings.cfg` exists → prompt skipped, buttons interactive immediately; 500ms and 300ms measured via injectable game-time clock

- **AC-3** (New Game, no saves):
  - Given: title screen displayed, no save files on disk
  - When: "New Game" clicked, difficulty selected, character confirmed
  - Then: screen crossfades to black over 500ms, Loading state entered, game begins in starting region
  - Edge cases: character confirmation cancelled → return to title screen; loading fails → Title Screen with error (Story 001 AC-9)

- **AC-4** (New Game with existing saves):
  - Given: title screen displayed, ≥1 save file exists
  - When: "New Game" clicked
  - Then: confirmation dialog appears with text "Starting a new game will not overwrite your existing saves."
  - Edge cases: "Cancel" returns to title screen; "Begin New Game" proceeds to difficulty selection

- **AC-6** (No saves — Load Game greyed out):
  - Given: title screen displayed, zero save files exist
  - When: the screen renders
  - Then: "Load Game" button has reduced opacity (40%), no hover effect, tooltip "No saved games found." on hover; "New Game" is fully interactive
  - Edge cases: saves created after screen load → button state updates reactively (poll or event-driven)

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `tests/unit/main_menu/title_screen_test.go` AND manual walkthrough doc at `production/qa/evidence/story-002-title-screen-evidence.md`

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Navigation & Transition Framework)
- Unlocks: Story 003 (First-Launch Prompt — needs title screen background)

---

## Completion Notes

**Completed**: 2026-08-11
**Criteria**: 6/6 passing (18 unit tests)
**Deviations**: None
**Test Evidence**: `tests/unit/main_menu/title_screen_test.go` — 18 tests, all PASS (57 total across Stories 001-002)
**Code Review**: Complete — APPROVED, no required changes, 6/6 standards, ADR-0012 compliant
**Advisory**: Manual evidence file deferred until Kaiju rendering pass (same as Story 001)
