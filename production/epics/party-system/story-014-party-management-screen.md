# Story 014: Party Management Screen

> **Epic**: Party System
> **Status**: Blocked
> **Layer**: Feature
> **Type**: UI
> **Estimate**: L (8–12 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

> ⛔ **BLOCKED**: `TR-party-007` has no governing ADR — registry GAP: "needs Party ADR". Run `/architecture-decision` to create the Party System ADR before implementation. Presentation rules (ADR-0012) are already Accepted — the blocker is the party logic underneath this screen. Once the Party ADR is Accepted, flip this story to Ready.

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-007` — Rest-point gating: party management only at rest_point entities; unavailable during combat/conversation/region transitions; party management pauses the live world

**ADR Governing Implementation (presentation)**: ADR-0012: UI Framework — this screen is Desk-register parchment UI with paper-slide motion, tab order + focus ring, arrow-key navigation
**ADR Governing Implementation (logic)**: N/A — GAP (registry: "needs Party ADR") — the screen drives Story 004's swap/dismiss operations and Story 013's availability predicate
**ADR Decision Summary**: The screen renders party state (Story 001), drives management operations (Story 004), and obeys the availability gate (Story 013). ADR-0012's Presentation rules are binding for all rendering; the missing Party ADR governs the data contract the screen reads.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI, no JS runtime (ADR-0012). Verify Kaiju UI capabilities (list scrolling, focus management, dialog overlays) against `kaiju/src/engine/` per the manifest's Engine Verification list. Pause-world-while-open interacts with the engine's update loop — coordinate with Story 013.

**Control Manifest Rules (Presentation Layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space; Desk register (parchment overlay, world dims 30%) (ADR-0012)
- Required: Paper-slide motion: enter 200–300ms ease-out, exit 200ms — no bounce/scale/spin/flip; Desk never trembles (ADR-0012)
- Required: Accessibility: tab order + focus ring (≥3:1), arrow-key navigation, Esc mapping (ADR-0012)
- Forbidden: Never use 3D UI or diegetic HUD — the desk is flat by law (ADR-0012)
- Forbidden: Never encode game-critical information by hue alone (ADR-0012)
- Guardrail: UI render (static): <1ms/frame; no frame drops during UI animations (ADR-0012)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` UI Requirements, CR-3, scoped to this story:*

- [ ] **AC-1** (screen contents): GIVEN party management opens at a rest point, THEN the screen shows the active party (4 slots, filled or empty), the recruitment pool (scrollable list sorted by trust level), and companion stats on hover/select (name, role, HP, trust level, knowledge domains).

- [ ] **AC-2** (swap interaction): GIVEN the screen is open, THEN click-to-swap works between active and pool; keyboard: Tab-focus + Enter; gamepad: d-pad + A. Empty slots show a subdued "recruit to fill" affordance.

- [ ] **AC-3** (rank toggle): GIVEN the screen is open, THEN a front/back rank toggle is available per active companion (writes Story 011's rank config).

- [ ] **AC-4** (scout toggle): GIVEN the screen is open, THEN a scout designation toggle is available, enforcing single-scout rules (Story 010).

- [ ] **AC-5** (dismissal confirm): GIVEN a dismissal action on a pool companion, THEN a confirmation dialog appears warning that the companion's knowledge domains will no longer be available; confirming calls Story 004's `Dismiss(..., confirmed=true)` (AC-PS-11 UI part); declining changes nothing.

- [ ] **AC-6** (companion inspect): GIVEN a companion selected, THEN a full character sheet opens — stats, equipment, knowledge domains, trust level (numeric + visual indicator, not color-only), recruitment story. The sheet layout is shared with the Character & Inventory UI; Party supplies the trust/knowledge/recruitment sections.

- [ ] **AC-7** (behavior mode control): GIVEN an active companion, THEN a mode control (dropdown or cycle button: Aggressive/Defensive/Passive) is available, writing Story 008's mode config; the mode glyph persists on the companion portrait for at-a-glance legibility.

- [ ] **AC-8** (gating rendering): GIVEN the availability predicate (Story 013) is false, THEN the entry points render greyed out or hidden (AC-PS-12 UI part).

- [ ] **AC-9** (world pause + reduce motion): GIVEN the screen is open, THEN the live world is paused (Story 013) and all party animations (swap, departure fades, alert animations) respect the `reduce_motion` accessibility setting.

- [ ] **AC-10** (accessibility): GIVEN the screen, THEN keyboard navigation covers all controls (tab order, arrow keys, Enter to activate), focus rings meet ≥3:1 contrast, and Esc closes the screen.

---

## Implementation Notes

*Derived from GDD UI Requirements and ADR-0012:*

### Screen Structure

- Desk register (ADR-0012): full-screen parchment overlay, world dims to 30%, paper-slide enter 200–300ms ease-out / exit 200ms.
- Layout: active slots (4) on the left/center, pool list (scrollable, sorted by trust level) on the right, detail panel on select, confirm dialog as a modal overlay.
- Shared components: the companion inspect sheet layout is owned by the Character & Inventory UI epic — this story consumes it and supplies the trust/knowledge/recruitment sections per the GDD's shared-component note.
- Trust display must include a numeric value + a non-color indicator (ADR-0012: never hue alone).

### Data Contract (Party System side)

- Reads: `ActiveRoster()`, pool list, per-companion rank (Story 011), scout designation (Story 010), behavior mode (Story 008), trust level (Trust & Relationship, read-only), knowledge domains (NPC Knowledge Graph, read-only).
- Writes: swap via Story 004's interface, dismiss with explicit `confirmed=true`, rank/scout/mode via their owning stories' interfaces.
- Entry points (wait/rest dialog option, keybinding) consult Story 013's `PartyManagementAvailable()`; greyed/hidden per AC-8.

### Verification

UI story — evidence is a manual walkthrough doc or interaction test per testing standards. Screenshots required for the art/UX sign-off (verification-driven development).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 004: Swap/dismiss operation semantics behind the buttons
- Story 013: The availability predicate itself
- Story 010/008/011: Scout, mode, and rank data models (this story renders and drives them)
- Full Vision: free-form formation editor
- Behavior-mode HUD glyph during exploration: `design/ux/hud.md` shared exploration HUD spec

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

Manual check: open party management at a rest point; verify active slots, trust-sorted pool, hover stats; swap via mouse/keyboard/gamepad; toggle ranks and scout; dismiss with confirm/decline; inspect sheet; mode cycle; blocked entry points in combat/conversation/transition; Esc close; keyboard-only pass; reduce-motion pass.
- Setup: reach a rest point with ≥1 active companion + ≥1 pool companion
- Verify: all AC-1..AC-10 behaviors
- Pass condition: full manual walkthrough documented in `production/qa/evidence/party-management-evidence.md` with screenshots, including the keyboard-only and reduce-motion passes

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `production/qa/evidence/party-management-evidence.md` (manual walkthrough doc or interaction test)

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 004 (swap/dismiss operations), Story 013 (availability predicate — BLOCKED on Party ADR), Character & Inventory UI epic (complete — shared inspect sheet)
- Unlocks: None
