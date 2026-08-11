# Story 008: Party Sidebar & Companion Inspect

> **Epic**: Character & Inventory UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: T-shirt L
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/character-inventory-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework; ADR-0001: Entity Architecture (reads companion entity data)
**ADR Decision Summary**: The left sidebar shows party member portraits. Clicking a portrait switches the character/inventory view to that party member. Companion inspect shows stats, trust level, and knowledge domains.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju UI rendering — verify portrait image support, tooltip/hover, and dynamic panel content against `kaiju/src/`.

**Control Manifest Rules (this layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space, Desk register (parchment overlay, world dims 30%), Desk serif (Garamond class) for text, Pen-and-ink icons for behavior mode glyphs
- Forbidden: Never use 3D UI or diegetic HUD, Never use amber-on-parchment text
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/character-inventory-ui.md`, scoped to this story:*

- [ ] **AC-20 — Party sidebar displays active members**: GIVEN a party of 4 (player + 3 companions), WHEN the C&I UI is opened, THEN the left sidebar shows all 4 members with the player character at top.
- [ ] **AC-21 — Companion inspect**: GIVEN a companion in the sidebar, WHEN the player clicks "Inspect" on the tooltip, THEN the character sheet opens for that companion showing their stats, trust level, and knowledge domains.
- [ ] **AC-22 — Party member switch preserves tab**: GIVEN the Inventory tab is open for the player character, WHEN the player clicks a companion's portrait in the sidebar, THEN the Inventory tab remains open showing that companion's inventory.
- [ ] **AC-19 — Companion departure switches view**: GIVEN the C&I UI is displaying a companion's inventory, WHEN that companion departs the party (trust betrayal), THEN the UI immediately switches to the player character's sheet.

---

## Implementation Notes

*Derived from ADR-0012, ADR-0001, and ADR-0005:*

- **Left sidebar** (20% width): Persistent across both Character and Inventory tabs.
  - Vertical portrait strip: player character at top, then active companions in party order, then pool companions (greyed out, unselectable).
  - Current selection: amber border (3px) with oxblood focus ring when keyboard-focused.
  - Scrollable if party exceeds sidebar height.
  - Portrait size: 72×72px (configurable via `sidebar_portrait_size` tuning knob, range 48–128).
  - Companion behavior mode glyph: small icon on portrait — Aggressive (sword), Defensive (shield), Passive (eye). Always visible.
- **Party member selection**:
  - Click portrait to select → character/inventory view switches to that member.
  - Keyboard: Tab to focus sidebar, ↑/↓ to cycle members, Enter to select.
  - Gamepad: D-pad up/down to cycle, A to select.
  - Default selection on open: last-viewed party member (session-only, resets to player character on new session).
- **Companion tooltip** (300ms hover delay):
  - Name, level, role/archetype, HP bar (current/max), trust indicator (amber glow at trust 1+, full glow at trust 3).
  - "Inspect" button in tooltip.
  - Dismisses on mouse-out.
- **Companion inspect mode**:
  - Opens the full character sheet for that companion.
  - Same layout as player character PLUS additional sections:
    - Trust & Relationship panel: trust level (0–3), relationship history.
    - Knowledge Domains list: domain name, proficiency indicator.
  - Trust data from `Faction` component or trust system; knowledge domains from NPC Knowledge Graph (ADR-0005).
- **Companion departure** (AC-19):
  - Subscribe to party member departure event via EventBus (ADR-0011).
  - If the departed companion is currently being viewed: immediately switch to player character's sheet.
  - Departed companion's equipment moves to "legacy inventory" — a dedicated tab in player's inventory accessible for 7 in-game days (configurable via `legacy_inventory_days`).
  - After legacy period: equipment transfers to companion's home region stash.
  - Departure journal entry notes transferred items.
- Read party roster from Party System; read companion entity data (Stats, Inventory, Faction, Knowledge) via Entity Framework (ADR-0001).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- [Story 001]: Character sheet layout and stat display (sidebar sits beside it)
- [Story 005]: Inventory grid display (sidebar is persistent; companion inventory uses the same grid)
- [Story 007]: Screen state machine (preserving tab on member switch — this story implements the switch behavior; Story 007 provides the state machine)
- Legacy inventory system (the 7-day timer and home region stash transfer) — this is a separate system; this story only switches the view on departure
- Dragging items to sidebar portrait for character-to-character transfer — out of scope for VS, covered by open question

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-20**: GIVEN a party of 4, WHEN the C&I UI is opened, THEN the left sidebar shows all 4 members with the player character at top
  - Given: Active party contains player character + 3 companions. C&I UI is closed.
  - When: Player presses C or I.
  - Then: Left sidebar displays 4 portrait slots. Player character is at the top position. Three companions below in party order. All 4 are selectable. Player character is selected by default (amber border).
  - Edge cases: Party of 1 (player only — no companions) → sidebar shows only player portrait. Party of 6 (max) → all 6 shown, scroll if needed. Pool companions (recruited but not in active party) → shown below active members, greyed out, unselectable.

- **AC-21**: GIVEN a companion in the sidebar, WHEN the player clicks "Inspect" on the tooltip, THEN the character sheet opens for that companion
  - Given: C&I UI open. Party has at least one companion.
  - When: Hover over companion portrait for 300ms → tooltip appears. Click "Inspect" in the tooltip.
  - Then: Character sheet displays for that companion. Stats shown (from companion's Stats component). Trust & Relationship panel visible with trust level and history. Knowledge Domains list visible with domain names and proficiency indicators. Companion's portrait has amber border (selected state).
  - Edge cases: Companion with trust level 0 → trust indicator shows no glow. Companion with trust 3 → full glow. Companion with no knowledge domains → section shows "No known domains" or is hidden.

- **AC-22**: GIVEN the Inventory tab is open for the player character, WHEN the player clicks a companion's portrait, THEN the Inventory tab remains open showing that companion's inventory
  - Given: C&I UI open on Inventory tab, viewing player character's inventory.
  - When: Click companion X's portrait in the sidebar.
  - Then: Inventory tab stays open. Grid now shows companion X's inventory items. Weight bar updates to companion X's carry weight. Gold display updates to companion X's gold. Companion X's portrait has amber border.
  - Edge cases: Switch from Character tab (should stay on Character tab). Switch to pool companion (unselectable — click ignored). Switch to already-selected member (no-op).

- **AC-19**: GIVEN the C&I UI is displaying a companion's inventory, WHEN that companion departs, THEN the UI switches to the player character's sheet
  - Given: C&I UI open on Inventory tab, viewing companion X's inventory.
  - When: Companion X departs the party (trust betrayal event fires).
  - Then: UI immediately switches to player character's view. Tab preserved (still Inventory). Companion X removed from sidebar. Player character portrait highlighted.
  - Edge cases: Companion death → same switch behavior. Departure while viewing Character tab → same switch. Departure while Level-Up modal is open for that companion → modal closes, switch to player. Multiple companions departing simultaneously → switch to player once.
