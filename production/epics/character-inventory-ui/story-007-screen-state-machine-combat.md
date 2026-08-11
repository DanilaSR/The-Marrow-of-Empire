# Story 007: Screen State Machine & Combat Integration

> **Epic**: Character & Inventory UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: T-shirt M
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/character-inventory-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-004`, `TR-ui-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework; ADR-0011: Event Bus (combat start/end events)
**ADR Decision Summary**: The C&I UI is a modal full-screen overlay with defined states and transitions. Combat events (via EventBus) force UI state changes. Paper-slide motion (200-300ms ease-out) governs enter/exit transitions.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju UI animation/transition support, EventBus integration — verify against `kaiju/src/`. Post-cutoff APIs: UI animation timing, event subscription from UI layer.

**Control Manifest Rules (this layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space, Paper-slide motion: enter 200-300ms ease-out, exit 200ms — no bounce/scale/spin/flip, Desk never trembles — UI immune to camera shake, Synchronous event dispatch via EventBus with system priority ordering
- Forbidden: Never use 3D UI or diegetic HUD
- Guardrail: Journal open/close transition: 200-300ms, No frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/character-inventory-ui.md`, scoped to this story:*

- [ ] **AC-15 — Combat blocks UI**: GIVEN the C&I UI is closed and combat is active, WHEN the player presses C or I, THEN the UI does not open.
- [ ] **AC-16 — Combat forces close**: GIVEN the C&I UI is open on the Character tab, WHEN combat starts, THEN the UI closes immediately with no prompt.
- [ ] **AC-17 — Tab switching**: GIVEN the UI is open on the Character tab, WHEN the player clicks the Inventory tab or presses I, THEN the UI switches to the Inventory tab with a 200ms crossfade and the same party member remains selected.

---

## Implementation Notes

*Derived from ADR-0012 and ADR-0011:*

- **Screen state machine** with these states and transitions:
  - `Closed` → `CharacterTab`: C key or menu command. Paper-slide enter (300ms ease-out). Opens to last-viewed party member.
  - `Closed` → `InventoryTab`: I key or menu command. Paper-slide enter (300ms ease-out).
  - `CharacterTab` ↔ `InventoryTab`: Tab click or C/I key. Content crossfade (200ms). Same party member stays selected.
  - Any open state → `Closed`: Esc, close button, or toggle key (C when on Character tab, I when on Inventory tab). Paper-slide exit (200ms).
  - Any open state → `Closed (forced)`: Combat starts, conversation starts, or region transition. Immediate close — no animation, no save prompt. Equipment changes are already committed.
  - `CharacterTab` → `LevelUp`: Automatic when XP threshold reached (see Story 002).
- **Combat integration** (via EventBus, ADR-0011):
  - Subscribe to combat start/end events from the EventBus.
  - On combat start: if UI is open, force close immediately. Set a `combat_active` flag that blocks C and I key input.
  - On combat end: clear `combat_active` flag. C and I keys work normally again.
  - Key blocking is checked before the open transition — if `combat_active`, C and I key presses are silently ignored.
- **Conversation integration**: Similar pattern — conversation start does not force close the UI (it's allowed during conversation, per the GDD). Conversation start does NOT block C/I keys.
- **Region transition**: Force close on region transition begin.
- **Tab state persistence**: The last-used tab (Character vs. Inventory) and last-viewed party member are session-only state — not persisted to save files. Reset to player character on new session.
- **Animation**: Paper-slide for open/close (full panel slides in/out). Crossfade for tab switch (content fades, chrome stays). Reduce-motion setting (Story 009) disables both.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- [Story 002]: Level-Up modal (the `LevelUp` state transition triggers it, but the modal itself is Story 002)
- [Story 008]: Party member selection within the sidebar (state machine preserves selection; Story 008 implements the sidebar)
- [Story 009]: Reduce-motion disabling animations (state machine provides the animation hook; Story 009 checks the setting)
- Conversation UI opening/closing — this story only handles whether C&I UI stays open during conversation

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-15**: GIVEN the C&I UI is closed and combat is active, WHEN the player presses C or I, THEN the UI does not open
  - Given: Combat is in progress. C&I UI is closed.
  - When: Player presses the C key. Player presses the I key.
  - Then: Neither keypress opens the UI. No partial render, no flash. The game world remains visible without the parchment overlay.
  - Edge cases: Press C during enemy turn, player turn, any combat phase — all blocked. Press C right as combat ends (event ordering: combat end event clears flag before next input processed). Gamepad menu button also blocked. Menu command (clicking UI button if one exists outside keyboard) also blocked.

- **AC-16**: GIVEN the C&I UI is open on the Character tab, WHEN combat starts, THEN the UI closes immediately with no prompt
  - Given: C&I UI is open and displaying the Character tab. Player is viewing their character sheet.
  - When: Combat starts (enemy spotted, scripted encounter triggers, etc.).
  - Then: The UI closes immediately — no animation, no delay, no "save before closing?" prompt. The parchment overlay is gone. The game world is fully visible. Combat HUD appears (from combat UI system).
  - Edge cases: UI open on Inventory tab when combat starts — same behavior. Level-Up modal open when combat starts — modal closes, UI closes, level-up is deferred (re-appears after combat if still pending). UI was open viewing a companion's sheet — same forced close.

- **AC-17**: GIVEN the UI is open on the Character tab, WHEN the player clicks the Inventory tab or presses I, THEN the UI switches to the Inventory tab with 200ms crossfade and same party member selected
  - Given: C&I UI open on Character tab, viewing party member X.
  - When: Player clicks the "Inventory" tab at the top of the screen, OR presses the I key.
  - Then: Content area crossfades from character sheet to inventory grid over 200ms. The left sidebar remains static (no crossfade). Party member X is still selected (sidebar highlight unchanged). Inventory shown is for party member X.
  - Edge cases: Press I when already on Inventory tab → closes the UI (toggle behavior). Click Character tab or press C to switch back — same crossfade, same member. Rapid tab switching (click Character then Inventory in quick succession) — transitions should not stack/queue; latest target wins.
