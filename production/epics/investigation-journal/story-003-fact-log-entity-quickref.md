# Story 003: Fact Log & Entity Quick-Ref Bar

> **Epic**: Investigation Journal
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: 3-4 hours
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/investigation-journal.md`
**Requirement**: `TR-invj-003`
*(Fact log tab: all discovered facts for a chain organized by source NPC with trust level, timestamps, contradiction flags, and entity reference chips. Entity quick-ref bar: recently referenced entities as clickable chips. EntitySelected state. TR-ID pending formal registration via /architecture-review.)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: All UI rendered in screen space via Kaiju HTML/CSS-like UI. Desk register: parchment panels, desk serif typography. Entity references rendered as clickable chips with ink text on parchment. Tab navigation within detail view.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Dynamic content rendering (fact lists, entity chips), tab switching within detail panel, data binding from multiple systems (Thread Investigation + Entity Framework). Post-cutoff APIs require verification against `kaiju/src/`.

**Control Manifest Rules (Presentation layer)**:
- Required: Kaiju HTML/CSS-like UI, screen space only; Desk serif (Garamond class) for journal text; Never encode game-critical info by hue alone (contradiction flagged by both thread-red marker AND text label); Body text 16px ref, min 12px effective
- Forbidden: Never amber-on-parchment text (entity chips: amber border OK, amber fill/text NOT OK); Never skeuomorphic buttons
- Guardrail: UI render <1ms/frame static; No frame drops during UI updates

---

## Acceptance Criteria

*From GDD `design/gdd/investigation-journal.md`, scoped to this story:*

- [ ] **AC-1**: Detail view has tab navigation — "Threads" tab and "Facts" tab. Switching tabs changes center panel content while maintaining selected chain context
- [ ] **AC-2**: Fact log shows all discovered facts for the selected chain, organized by source NPC. Each fact displays: content text, source NPC name, trust level at time of discovery (e.g., "Trust: Acquaintance"), game-time timestamp, and contradiction flag (thread-red marker + "CONTRADICTS" label if applicable)
- [ ] **AC-3**: Entity references within facts render as clickable chips. Clicking an entity chip opens EntitySelected state — detail panel shows that entity's facts and connections
- [ ] **AC-4**: Entity quick-ref bar at bottom of journal shows recently referenced entities as clickable chips (max 8, most recent first). Entity names use ink (#1C1813) on parchment (#E3D5B0) chip background
- [ ] **AC-5**: EntitySelected state shows entity name as header, list of all facts referencing this entity across all chains, and list of connected entities (other entities referenced alongside this one)
- [ ] **AC-6**: Empty state: when no chain is selected, fact log area shows "Select a chain to view discovered facts"; when selected chain has no facts, show "No facts discovered for this chain yet"

---

## Implementation Notes

*Derived from ADR-0012 Implementation Guidelines:*

**Tab navigation**: Two tabs in the detail panel header: "Threads" (default) and "Facts". Tabs render as ink text on parchment with an active-state underline in umber (not amber — amber-on-parchment fails contrast). Tab switch is instant (no animation) — the fact list replaces the thread list in the center panel.

**Fact log data source**: Read facts from Thread Investigation's `PlayerKnowledge` component (`discovered_facts[]`). Each fact has: content text, source NPC's GameEntityID, trust level at discovery time (look up from Trust & Relationship system if available, otherwise show "Unknown"), game-time timestamp (from GameClock at discovery), contradiction references (list of contradicting fact IDs), and entity references (list of GameEntityIDs mentioned in the fact).

**Organization by source NPC**: Group facts by source NPC. Section header = NPC display name (resolved from Entity Framework via GameEntityID). Facts within each NPC group sorted by discovery time (oldest first). Trust level shown as label: "Stranger" (0), "Acquaintance" (1), "Trusted" (2), "Confidant" (3).

**Contradiction flag**: Facts that have `contradicts` references display a thread-red (#C4483A) marker — a small "↯" icon to the left of the fact, plus a "CONTRADICTS" label in thread-red (12px, all-caps). The marker is both a shape (zigzag arrow) AND a color (thread-red), satisfying the "never hue alone" rule. The full contradiction side-by-side view is Story 004 — this story only flags which facts are contradictory.

**Entity chips**: Entity references within fact text render as inline clickable chips. Chip style: ink (#1C1813) text on parchment (#E3D5B0) background, 1px umber border, 4px border-radius, 14px desk serif. Hover: border brightens to amber (the accent, not the fill). Click: transitions to EntitySelected state.

**Entity quick-ref bar**: Bottom bar (full width, ~60px, slightly darker parchment or umber-tinted background to distinguish from main panels). Shows up to 8 most recently referenced entities (tracked from user interactions: clicked entities, entities in viewed facts). Left-to-right order: most recent first. Each chip same style as inline entity chips. Overflow: if more than 8, show first 7 + "..." chip that's not clickable (future: search).

**EntitySelected state**: When an entity is selected (via chip click or quick-ref bar), the center panel header changes to the entity name (desk serif, 22px, ink). Below: section "Referenced In" listing all facts across ALL chains that reference this entity (grouped by chain). Section "Connected Entities" listing other entities that co-appear in facts with this one. Clicking a connected entity transitions EntitySelected to that entity. A "Back to investigation" link/button returns to the previously selected chain (ChainSelected state).

**Back navigation**: Breadcrumb or back button. ChainSelected → EntitySelected (clicking entity chip) → back button returns to ChainSelected. EntitySelected → EntitySelected (clicking connected entity) → back button returns to previous entity. This is a simple stack: push on entity select, pop on back.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Journal shell, three-panel layout, open/close lifecycle
- Story 002: Thread list display in detail view, chain list sidebar
- Story 004: Contradiction side-by-side comparison view with thread-red connector line, revelation prompts with amber glow
- Story 005: Map centering on entity location
- Full entity detail (stats, appearance, full history): entity inspection is a separate system — this story shows only facts and connections

---

## QA Test Cases

**Manual checks (UI story — visual verification required):**

- **AC-1**: Tab navigation
  - Setup: Journal open, chain selected that has both threads and facts
  - Verify: Click "Facts" tab — center panel switches to fact log. Click "Threads" tab — center panel returns to thread list. Active tab has visible indicator (umber underline). Chain context preserved across tab switches
  - Pass condition: Tab switching is instant, context preserved, correct content per tab

- **AC-2**: Fact log content
  - Setup: Chain with facts from 2 different NPCs at different trust levels, including one fact with a contradiction flag
  - Verify: Facts grouped by NPC with section headers. Each fact shows content, source name, trust level label, game-time timestamp. Contradicted fact has thread-red "↯ CONTRADICTS" marker
  - Pass condition: All fact fields rendered correctly; NPC grouping accurate; contradiction marker visible

- **AC-3**: Entity chip click → EntitySelected
  - Setup: Fact containing entity reference (e.g., "Captain Veren")
  - Verify: Click entity chip — center panel switches to entity detail showing entity name header, "Referenced In" facts, and "Connected Entities". Back button visible
  - Pass condition: EntitySelected state active; facts and connections populated

- **AC-4**: Entity quick-ref bar
  - Setup: Interact with 5+ different entities (click entity chips across multiple facts/chains)
  - Verify: Bottom bar shows clicked entities as chips in most-recent-first order. Click a chip in the bar → EntitySelected for that entity
  - Pass condition: Up to 8 chips visible; order correct; chips clickable

- **AC-5**: EntitySelected content
  - Setup: Entity referenced in facts across 2 different chains
  - Verify: EntitySelected shows entity name header, facts grouped by chain under "Referenced In", connected entities listed under "Connected Entities". Click connected entity → navigates to that entity
  - Pass condition: Cross-chain fact aggregation works; connected entity navigation works

- **AC-6**: Empty states
  - Setup: Open journal with no chain selected
  - Verify: Detail panel shows "Select a chain to view discovered facts" message (muted umber, centered)
  - Setup: Select chain with zero facts
  - Verify: Facts tab shows "No facts discovered for this chain yet"
  - Pass condition: Both empty states render appropriate messages

---

## Test Evidence

**Story Type**: UI
**Required evidence**: Manual walkthrough doc at `production/qa/evidence/story-003-fact-log-evidence.md`
**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Journal Shell & Layout), Thread Investigation Story 001 (Complete ✅), Thread Investigation Story 002 (Complete ✅), Entity Framework (Complete ✅), Trust & Relationship Story 001 (Complete ✅)
- Unlocks: Story 004 (Contradiction Display & Revelation Prompts)
