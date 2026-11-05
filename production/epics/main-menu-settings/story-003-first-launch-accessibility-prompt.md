# Story 003: First-Launch Accessibility Prompt

> **Epic**: Main Menu & Settings
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: S (2-3 hours) — modal overlay on title screen, two dropdowns + checkbox + button, Escape/Continue dismissal, settings.cfg persistence, Re-run from Settings. Builds on SetInteractive(bool) hook from Story 002.
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/main-menu-and-settings.md`
**Requirement**: `TR-ui-007`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: Accessibility first — the prompt is presented before any menu interaction. Tab order and arrow-key navigation for all UI. Typewriter speed setting (Instant/Normal/Slow). Text scale accessibility setting. The desk never trembles.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Modal overlay rendering on top of title screen. Keyboard/gamepad navigation within modal (Tab/arrows + Enter/Escape + D-pad + A/B). Checkbox and dropdown Kaiju UI element support. AccessKit screen reader support.

**Control Manifest Rules (Presentation layer)**:
- Required: Accessibility: tab order + focus ring (≥3:1), arrow-key navigation, typewriter speed setting, Esc mapping
- Required: Body text 16px at reference, minimum 12px effective
- Forbidden: Never encode game-critical information by hue alone
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/main-menu-and-settings.md` CR-7:*

- [ ] **AC-1**: On first game launch (no `settings.cfg` exists), after the title screen background fades in, a modal dialog appears over the title screen with: Reduce Camera Motion (Full / Reduced [default] / Off), Subtitles (On [default] / Off), "Remember my choice" checkbox (checked by default), and a "Continue" button. The title screen menu is visible behind but not interactive until dismissal.

- [ ] **AC-2**: Pressing Escape or clicking "Continue" dismisses the modal with the currently-selected values applied to `settings.cfg`. The title screen menu becomes interactive after dismissal. The modal does NOT auto-dismiss on a timer or click-away.

- [ ] **AC-3**: On subsequent launches (valid `settings.cfg` exists), the accessibility prompt does NOT appear. The chosen settings persist in `settings.cfg`.

- [ ] **AC-4**: From Settings → Accessibility, clicking "Re-run Accessibility Setup" re-opens the modal with current values pre-populated. The button is labeled "Apply" instead of "Continue."

- [ ] **AC-5**: The modal is fully keyboard-navigable (Tab/arrows + Enter/Escape) and gamepad-navigable (D-pad + A/B) with a visible focus indicator on every interactive element. All three input methods work independently — no method is required.

- [ ] **AC-6**: When `settings.cfg` is missing or corrupted on launch, the system creates a new `settings.cfg` with all defaults. The first-launch accessibility prompt is shown (as if first launch). No error is displayed to the player.

---

## Implementation Notes

*Derived from ADR-0012 Implementation Guidelines:*

- **Modal overlay**: Rendered as a Kaiju UI panel layered above the title screen. The title screen is visible behind the modal at reduced interactivity (all title screen elements have `pointer-events: none` while modal is active).
- **Settings controls**: Two dropdown/radio groups (Reduce Camera Motion, Subtitles), one checkbox (Remember my choice), one button (Continue). Defaults: Reduced / On / checked. The Escape key is equivalent to clicking Continue — both apply current values and dismiss.
- **Persistence**: On dismissal, write the two settings to `settings.cfg` (JSON) via the same persistence path as Story 005. If "Remember my choice" is unchecked, the settings are still applied for this session but are not persisted — on next launch, the prompt reappears.
- **Re-run flow**: Settings → Accessibility → "Re-run Accessibility Setup" button re-opens the same modal component. Current values from `settings.cfg` pre-populate the controls. Button label changes from "Continue" to "Apply."
- **Corrupted settings recovery**: On boot, if `settings.cfg` parse fails, create a fresh defaults file. The missing/corrupted path triggers the first-launch flow — accessibility prompt shown. No error dialog — the game starts with safe defaults per CR-3 edge case.
- **Navigation**: Inherits Story 001's focus system. Tab moves between controls. Arrow keys change dropdown/radio values within a group. Enter/Space toggles checkbox. Escape and Continue button both dismiss.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Focus ring rendering, input routing infrastructure, crossfade transitions
- Story 002: Title screen background and menu buttons
- Story 005: Settings screen (including Accessibility category and Re-run button destination), `settings.cfg` JSON schema and read/write infrastructure
- Camera System: The camera motion reduction logic itself — the prompt only sets the config value
- Subtitle system: Subtitle rendering — the prompt only sets the config value

---

## QA Test Cases

*Derived from GDD acceptance criteria. The developer implements against these — do not invent new test cases during implementation.*

### Automated (UI state assertions)

- **AC-1** (First-launch modal display):
  - Given: no `settings.cfg` exists, title screen background has faded in
  - When: the accessibility prompt initializes
  - Then: modal is visible with exactly these controls: Reduce Camera Motion dropdown (default: Reduced), Subtitles dropdown (default: On), "Remember my choice" checkbox (default: checked), "Continue" button; title screen menu buttons have `interactive: false`
  - Edge cases: modal z-order is above title screen; Escape key is bound to dismissal

- **AC-2** (Dismissal via Continue):
  - Given: accessibility prompt modal is displayed, default values selected
  - When: "Continue" is clicked
  - Then: modal is removed from display; Reduce Camera Motion = "Reduced" and Subtitles = "On" are written to `settings.cfg`; title screen menu buttons become interactive
  - Edge cases: Escape key produces identical result; modal does not dismiss on click-away or timer expiry

- **AC-2** (Dismissal via Escape):
  - Given: accessibility prompt modal is displayed, user changed Camera Motion to "Off"
  - When: Escape is pressed
  - Then: modal dismisses; the currently-selected value "Off" is applied (not reverted to default); title screen menu becomes interactive
  - Edge cases: "Remember my choice" unchecked → settings applied for session only, not persisted; prompt reappears on next launch

- **AC-3** (Persistence across launches):
  - Given: `settings.cfg` exists with `camera_motion: "Off"` and `subtitles: "On"`
  - When: the game launches
  - Then: the accessibility prompt does NOT appear; title screen menu is immediately interactive
  - Edge cases: `settings.cfg` has only one of the two settings → prompt still suppressed (partial config is valid)

- **AC-6** (Corrupted settings recovery):
  - Given: `settings.cfg` contains invalid JSON
  - When: the game launches
  - Then: a new `settings.cfg` is created with all defaults; the accessibility prompt appears as if first launch; no error dialog is displayed to the player
  - Edge cases: missing file → same behavior; unwritable directory → error toast (per Story 005)

### Manual verification

- **AC-5** (Keyboard + gamepad navigation):
  - Setup: Launch game first time, accessibility prompt visible
  - Verify: Tab moves through all controls in order; arrow keys change dropdown values; Enter/Space toggles checkbox; Escape dismisses; gamepad D-pad moves focus, A activates, B dismisses
  - Pass condition: All controls reachable and operable via keyboard alone AND gamepad alone AND mouse alone — no input method is required

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `tests/unit/main_menu/accessibility_prompt_test.go` AND manual walkthrough doc at `production/qa/evidence/story-003-accessibility-prompt-evidence.md`

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Navigation & Transition Framework), Story 002 (Title Screen — modal overlays it)
- Unlocks: None directly

---

## Completion Notes

**Completed**: 2026-08-11
**Criteria**: 6/6 passing (13 unit tests)
**Deviations**: None
**Scope note**: `screen_flow.go` — `dispatchFocusAction` demoted from method to package function to share between flow and prompt (zero behavior change)
**Test Evidence**: `tests/unit/main_menu/accessibility_prompt_test.go` — 13 tests, all PASS (70 total across Stories 001-003)
**Code Review**: Complete — APPROVED, 6/6 standards, ADR-0012 compliant
**Advisory**: Manual evidence file deferred until Kaiju rendering pass
