# Story 006: Inventory Toast Notifications

> **Epic**: Character & Inventory UI
> **Status**: Complete
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: T-shirt S
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/character-inventory-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: Brief non-blocking toast notifications appear for inventory feedback (e.g., "Inventory full") in the Field or World register, not disrupting the Desk overlay.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju UI toast/timed notification support — verify against `kaiju/src/`. May need a simple timer-based UI element.

**Control Manifest Rules (this layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space
- Forbidden: Never use amber-on-parchment text — toast uses amber text on a darker transient background, not on parchment
- Guardrail: No frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/character-inventory-ui.md`, scoped to this story:*

- [ ] **AC-13 — Inventory full toast**: GIVEN a full inventory (weight at max capacity), WHEN the player attempts to pick up a world item, THEN a toast appears: "Inventory full" for 2 seconds (±200ms) in amber text and the item remains in the world.

---

## Implementation Notes

*Derived from ADR-0012:*

- **Toast system**: A lightweight notification that appears briefly and auto-dismisses.
  - Display duration: 2000ms (configurable via `toast_duration_ms` tuning knob, range 1000–4000ms).
  - Text: "Inventory full" — amber text on a dark transient background (not on parchment — avoids the amber-on-parchment forbidden pattern).
  - Position: Centered near the bottom of the screen or near the inventory grid (exact position TBD during implementation — should be noticeable but not obstruct interaction).
  - Behavior: Appears, holds for duration, fades out. Non-interactive — does not block input. Multiple toasts queue (show one at a time).
- **Trigger**: When the player attempts to pick up a world item (interaction) and the inventory weight is at or above max capacity, the pickup action is cancelled, the item remains in the world, and the toast fires.
- The toast system should be reusable for other notification types beyond inventory (e.g., "Quest item added", "Equipment broken") — design the toast component as a general-purpose notification mechanism.
- Accessibility: toast text must be screen-reader accessible. Toast appearance must not rely on color alone.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- [Story 005]: The weight bar and its color states (toast fires WHEN weight bar shows >100%)
- [Story 007]: Combat forcing the UI closed (toast is independent of screen state)
- Other toast types (quest, equipment, combat) — this story builds the reusable toast mechanism but only implements the "Inventory full" trigger

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-13**: GIVEN a full inventory, WHEN the player attempts to pick up a world item, THEN a toast appears "Inventory full" for 2 seconds and the item remains in the world
  - Setup: Fill a character's inventory so that current weight equals or exceeds max carry capacity. Place a pickable item in the game world nearby.
  - Verify: Attempt to pick up the world item (click interact, press interact key, or whatever the pickup action is). Observe the toast notification.
  - Pass condition: Toast appears with text "Inventory full" in amber. Toast is visible for approximately 2 seconds (1800–2200ms acceptable range). Toast auto-dismisses (no click required). The world item remains in place — it was NOT picked up. No item was added to inventory. Weight did not change.

  - Edge cases:
    - Attempt pickup when inventory is at exactly 100% (at capacity, not over) — toast still fires (cannot exceed capacity).
    - Attempt pickup when inventory is at 99% but item weight would push over 100% — toast fires (the pickup would exceed capacity).
    - Rapid pickup attempts during toast display — each failed attempt should either re-trigger the toast timer or be ignored (no toast spam).
    - Toast behavior when C&I UI is not open (world interaction while exploring) — toast must still appear (it's a World/Field notification, not Desk).
