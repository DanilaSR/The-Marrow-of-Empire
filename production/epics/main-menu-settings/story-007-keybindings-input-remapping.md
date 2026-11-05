# Story 007: Keybindings & Input Remapping

> **Epic**: Main Menu & Settings
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: M (4-6 hours) — per-action binding interface with capture mode, conflict auto-unbind, modifier key support, default restoration, reserved key warnings, gamepad presets, JSON persistence. Builds on existing BindingRegistry from Story 001.
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-13

## Context

**GDD**: `design/gdd/main-menu-and-settings.md`
**Requirement**: `TR-ui-007`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework; depends on Input Binding System (`design/gdd/input-binding-system.md`)

**ADR Decision Summary**: Accessibility: tab order + focus ring for all UI including keybinding interface. Keybinding persistence via `settings.cfg` JSON. The Input Binding System owns the runtime binding registry; Main Menu provides the keybinding UI that reads and writes it.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju UI key-capture mode (raw key events during rebind). Modifier key detection (Shift, Ctrl, Alt). Input routing — during rebind, all keys captured by the binding UI, not dispatched to the game. AccessKit screen reader support for binding status announcements.

**Control Manifest Rules (Presentation layer)**:
- Required: Accessibility: tab order + focus ring (≥3:1), arrow-key navigation
- Forbidden: Never encode game-critical information by hue alone — "Unbound" warning must include icon + text + color
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/main-menu-and-settings.md` CR-6:*

- [ ] **AC-1**: The Controls → Keybindings screen displays all game actions with their current primary and secondary bindings. Clicking an action row enters capture mode: the row highlights, and the next key (or key + modifier combination) pressed is bound as the primary binding for that action. Pressing Escape during capture cancels without changing the binding.

- [ ] **AC-2**: If the captured key is already bound to another action, the old action is unbound (cleared) and the new binding takes its place. The old action shows "Unbound" with a warning highlight (icon + amber text — two cues). No key maps to more than one action at any time.

- [ ] **AC-3**: Modifier keys (Shift, Ctrl, Alt) are valid as modifiers. The binding system stores `[modifier] + [key]` pairs. A modifier alone (pressed and released without another key) does not complete the binding — capture mode remains active waiting for the base key.

- [ ] **AC-4**: Each action row has a "Restore Defaults" button that restores the default binding for that action only. A "Restore All Defaults" button restores the default binding for every action. Both clear any conflict warnings on affected rows.

- [ ] **AC-5**: When the player attempts to rebind Escape (reserved for Pause Menu) or F9 (reserved for Quick-Load), a warning appears: "This key is reserved for [function]. Rebinding may conflict with system behaviour." The player can override after acknowledging the warning.

- [ ] **AC-6**: At least one binding must remain for each reserved function (Escape=pause, F9=quick-load). The system blocks unbinding the last remaining binding for any reserved function with a warning: "This is the only binding for [function]. It cannot be removed."

- [ ] **AC-7**: Gamepad bindings use preset profiles at MVP (not per-button remappable). The player selects from available preset layouts via a dropdown.

- [ ] **AC-8**: All keybindings persist to `settings.cfg` under the Controls category. They are read on boot and applied to the Input Binding System. Bindings are not per-save — they persist across all sessions and save files.

---

## Implementation Notes

*Derived from ADR-0012 and GDD CR-6:*

- **Binding data model**: Each action has a unique action ID (string), a display name, a category, a primary binding (`KeyChord` = modifier flags + key code), a secondary binding (same type), and a default primary + secondary. The Input Binding System (`design/gdd/input-binding-system.md`) owns the `KeyChord` struct, action registry, capture semantics, conflict resolution, defaults table, and JSON serialization contract. This story implements the UI that reads and writes that registry.
- **Action list rendering**: Actions displayed as rows in a scrollable table: Action Name | Primary Binding | Secondary Binding | Restore Defaults button. Rows are grouped by category (Movement, Camera, Combat, UI, System). The current binding is displayed as a text label showing the key name (e.g., "W", "Ctrl+S", "Space").
- **Capture mode**: Clicking a binding slot (primary or secondary) enters capture mode:
  1. The slot highlights (amber gold border, pulsing gently)
  2. A "Press a key..." prompt appears in the slot
  3. The UI listens for raw key-down events. All key events are consumed by the capture UI — not dispatched to the game.
  4. On key-down: if the key is a modifier (Shift/Ctrl/Alt), store it and wait for the base key. If it's a non-modifier, complete the binding.
  5. On Escape during capture: cancel, restore previous binding, exit capture mode.
  6. On completion: run conflict check, update binding, exit capture mode.
- **Conflict resolution**: After capture, check the Input Binding System registry: does any other action have this key as primary or secondary? If yes, unbind the old action's slot (set to `KeyChord{}` zero value). The old action's row updates to show "Unbound" with a warning icon (⚠) and amber text. The conflict is resolved atomically — the cascade stops at one hop (the directly conflicting action). The final state is always consistent: every key maps to at most one action.
- **Default restoration**: Per-row: read the action's default `KeyChord` from the Input Binding System's defaults table, set it as the current binding, clear any warning. All-defaults: iterate all actions, restore defaults, clear all warnings. No confirmation for per-row restore; confirmation required for all-defaults ("Restore all keybindings to defaults?").
- **Reserved key handling**: Reserved functions (Escape=pause menu, F9=quick-load) are defined in a reserved-keys registry. On capture of a reserved key: show warning dialog ("This key is reserved for Pause Menu...") with "Bind Anyway" / "Cancel". On unbind: check if this is the last binding for a reserved function; if yes, block with warning ("This is the only binding for Pause Menu. It cannot be removed.").
- **Gamepad presets**: A dropdown listing preset layout names (e.g., "Default", "Southpaw", "Left-Handed"). Selecting a preset applies all gamepad bindings from that preset. Presets are defined in a static data structure (not user-editable at MVP). No per-button gamepad remapping.
- **Persistence**: On any binding change, the Controls category of `settings.cfg` is updated (via Story 005's persistence path). The JSON schema for bindings: `{"controls": {"bindings": {"move_forward": {"primary": {"key": "W", "modifiers": 0}, "secondary": {"key": "Up", "modifiers": 0}}, ...}, "gamepad_preset": "Default"}}`. On boot, `settings.cfg` → Input Binding System → UI displays current state.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Navigation within keybinding screen, focus rings, Escape = back
- Story 005: Settings → Controls category embedding, `settings.cfg` read/write infrastructure, JSON schema for non-binding Controls settings (mouse sensitivity, gamepad sensitivity, invert mouse wheel zoom)
- Input Binding System (`design/gdd/input-binding-system.md`): `KeyChord` struct, action registry, capture state machine, conflict resolution logic, defaults table, JSON serialization, runtime key dispatch — the UI reads and writes this system's API, it does not implement the registry
- Gamepad input system: Gamepad button mapping, D-pad/analog stick interpretation, preset profile data — the UI provides the preset selector; the Input Binding System applies the selected preset

---

## QA Test Cases

> Generated by /qa-plan sprint-6 (2026-08-13). Existing test file `tests/integration/main_menu/keybindings_test.go` has 24 tests — bring to green; add cases only for uncovered ACs.

**Automated (integration):**
- **AC-1 capture**: click slot → next key binds to primary; Escape during capture cancels without change
- **AC-2 conflict auto-unbind**: stolen key → old action "Unbound" with icon + amber text (two cues); no key maps to two actions; cascade stops at one hop
- **AC-3 modifiers**: Shift/Ctrl/Alt stored as `{key, modifiers}`; modifier-only press keeps capture active; left/right modifier = same flag
- **AC-4 restore**: per-row (no confirmation) and Restore All (confirmation); warnings cleared
- **AC-5/AC-6 reserved keys**: Escape/F9 warning with override; last remaining binding for reserved function cannot be removed
- **AC-8 persistence**: round-trip through `settings.cfg`; missing/corrupt file → defaults for Controls, other categories unaffected
- **Edge cases**: non-printable key (F5) bound as key code; rebind to same-action secondary slot → no conflict; multiple modifiers stored in bitmask

**Manual:**
- AC-7 gamepad presets: dropdown shows Default/Southpaw/Left-Handed; applies immediately; persists across restart
- Capture mode UI: amber-gold pulsing highlight + "Press a key..." prompt; screen reader announces capture state
- Evidence: screenshots to `production/qa/evidence/`; sign-off: ux-designer + accessibility-specialist

**Estimated test count**: ~24 (existing) + AC gap fill
## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/main_menu/keybindings_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Navigation & Transition Framework), Story 005 (Settings screen — Controls category embeds this UI, `settings.cfg` persistence), Input Binding System (`design/gdd/input-binding-system.md` — owns KeyChord struct, action registry, capture, conflict resolution, defaults, serialization)
- Unlocks: None directly

---

## Completion Notes

**Completed**: 2026-08-13
**Criteria**: 8/8 passing (automated). Manual-visual checklist items (capture highlight rendering, screen-reader announcements, in-game gamepad application) remain for the QA manual pass — see `production/qa/qa-plan-sprint-6-2026-08-13.md`.
**Deviations**: ADVISORY — (1) keybinding requirement text lives in TR-menu-005; story cites TR-ui-007; (2) AC-2 "amber text" vs control-manifest amber-on-parchment rule — reconcile before markup pass; (3) Input Binding System scope overlap (GDD file missing; screen implements registry/conflict/persistence).
**Test Evidence**: `tests/integration/main_menu/keybindings_test.go` — 34 tests pass (`-count=1` green); `make test-all` green (pre-existing unrelated `tests/unit/entity` perf flake surfaced separately).
**Code Review**: Complete — `/code-review` run 2026-08-13: 3 specialists, 2 fix rounds (restore-path conflict resolution, "Unbound" persistence token, dialog keyboard routing), final verdict APPROVED WITH SUGGESTIONS.

