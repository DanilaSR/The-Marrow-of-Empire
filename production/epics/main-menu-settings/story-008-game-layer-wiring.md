# Story 008: Game-Layer Wiring — Title, Settings & Keybindings Live

> **Epic**: Main Menu & Settings
> **Status**: In Progress
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: L (6-8 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-13

## Context

**GDD**: `design/gdd/main-menu-and-settings.md`
**Requirement**: TR-menu-* catalog (screen flow, settings persistence) + `TR-ui-007` / `TR-menu-005` (keybindings) — lookup fresh in `docs/architecture/tr-registry.yaml` at dev time
**ADR Governing Implementation**: ADR-0012: UI Framework (primary); ADR-0002: Save/Load Serialization (settings.cfg file adapter)
**ADR Decision Summary**: All UI renders in screen space through Kaiju's HTML/CSS-like UI system; settings persist via `settings.cfg` JSON. The screens (Story 001-007) are complete headless controllers with injected seams — this story implements the host adapters that wire them into the running game.
**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: The vertical slice proved the UI recipe: markup + `position: absolute` + PX top/left/width/height + z-index; show/hide via entity `Activate()/Deactivate()`; NO runtime UI tree mutations (static label pools — `anyChildDirty` panic); markup labels need zero-width-space content + `InnerLabel()` binding. Follow `src/ui/mainmenu/*` render-command contracts; verify input APIs against `kaiju/src/`.

---

## Acceptance Criteria

- [ ] **AC-1** (production title screen): GIVEN the standalone build boots, THEN the production `mainmenu` screen flow renders the title screen (Title → New Game continues to the slice world). The slice's ad-hoc `title_screen.go` is retired or routed through the flow.
- [ ] **AC-2** (settings.cfg file adapter): GIVEN a settings change in-game, THEN the value is written to `settings.cfg` on disk (Story 005 debounce contract); on reboot the store is re-read; a missing/corrupt file falls back to defaults with no error dialog.
- [ ] **AC-3** (keybindings live): GIVEN the Controls → Keybindings screen in-game, THEN a rebind completes via capture mode, persists across restart, and the host input layer applies it (host adapter implements the HandleKey contract: while `CapturesInput()` is true raw keys route to the screen first; a false return dispatches to the flow — the contract documented in `keybindings_screen.go` header).
- [ ] **AC-4** (no key leakage): GIVEN capture mode active, THEN no key is dispatched to the game world (verified live: press W in capture → player does not move).
- [ ] **AC-5** (menu ↔ game transitions): GIVEN the Settings or Keybindings screen open from the game, THEN Back/Escape returns to the previous screen state; the game world remains live behind Desk-register parchment (world dims per ADR-0012 register rules).

---

## Implementation Notes

- Build on the slice game layer: `src/game/game.go` owns the shared `ui.Manager`; screens register their focusables and emit render commands per frame; the host adapter consumes them with the UI recipe above.
- **settings.cfg adapter**: implement real file I/O (path injectable for tests, temp dirs in tests) behind the existing `WithSettingsReader/Writer` and `WithKeybindingsReader/Writer/HasSettingsFile` seams. No goroutines; debounce flush on clean shutdown.
- **Input routing**: the host input layer calls `HandleKey` first while any screen's `CapturesInput()` is true; false → dispatch through the flow/focus manager. Verify the engine's raw key delivery (KeyEvent fields: Key, Shift/Ctrl/Alt booleans) against `kaiju/src/platform/hid/` — modifier side-collapse per keybindings_screen contract.
- **Screen flow ↔ Kaiju**: one shared `ui.Manager` (dual-manager elements don't render — slice finding); static label pools sized per screen; no runtime UI tree mutations.
- Prefer reusing the slice's proven render helpers; retire `title_screen.go` only if the flow replacement is live-verified.

## Out of Scope

- Pause menu freeze contract (Story 006 ACs — follow-up wiring pass, tracked in sprint notes)
- Save/Load screens' real file operations (INT-6 story: save-load-system/story-004-game-layer-wiring.md)
- Gamepad input interpretation (Input Binding System, per Story 007 scope)
- Visual polish of the markup (functional wiring first; ux-designer polish pass later)

---

## QA Test Cases

**Automated (integration) — `tests/integration/main_menu/game_wiring_test.go`:**
- Adapter round-trip: write settings via screen → file exists → fresh adapter reads same values; missing file → defaults; corrupt JSON → defaults, other categories unaffected
- HandleKey routing contract: capture state → key consumed (true); panel state → false passthrough
- Settings→Keybindings navigation with the real adapter injected

**Manual/live (ai_driver, BLOCKING for this story):**
- Standalone build boots to production title; New Game works
- Rebind W→G in-game, restart, G moves the player, W does not
- Capture mode: pressing keys does not move the player (AC-4)
- Evidence: screenshots to `production/qa/evidence/` (title, settings, keybindings capture)

**Estimated test count**: ~8 + 4 live checks

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/main_menu/game_wiring_test.go` — must exist and pass
- Live: ai_driver screenshots in `production/qa/evidence/` — title/settings/capture + restart persistence

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001-007 of this epic (Complete), slice game layer (`src/game/`) — done
- Unlocks: In-game keybinding application (Input Binding System runtime wiring)
