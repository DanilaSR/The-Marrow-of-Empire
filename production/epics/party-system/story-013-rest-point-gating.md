# Story 013: Rest-Point Gating

> **Epic**: Party System
> **Status**: Blocked
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: S (2–4 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

> ⛔ **BLOCKED**: `TR-party-007` has no governing ADR — registry GAP: "needs Party ADR". Run `/architecture-decision` to create the Party System ADR before implementation. Once the ADR is Accepted, flip this story to Ready.

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-007` — Rest-point gating: party management only at rest_point entities; unavailable during combat/conversation/region transitions; party management pauses the live world

**ADR Governing Implementation**: N/A — GAP (registry: "needs Party ADR")
**ADR Decision Summary**: Gating combines three existing decisions — ADR-0007's rest-point taxonomy and clock-pause rules (never advance time during combat, conversation, menus), ADR-0011's event-driven state awareness, and ADR-0002's save restriction matrix (which blocks saves during transitions/cutscenes). The Party ADR must decide how the Party System tracks "current context" (combat/conversation/transition) without duplicating the Camera System's state machine (ADR-0003, TR-cam-005).

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Pure Go state logic. Context tracking should subscribe to existing events (combat.started/ended, conversation start/end, region transitions) rather than polling — verify event types against ADR-0011's catalog and the implemented epics.

**Control Manifest Rules (Feature Layer)**:
- Required: Synchronous event dispatch via EventBus with system priority ordering (ADR-0011)
- Forbidden: Never advance time during combat, conversation, menus, or journal (ADR-0007)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-3 (Party Management), CR-4, scoped to this story:*

- [ ] **AC-1** (AC-PS-12, context blocking): GIVEN the player is in combat, conversation, or a region transition, WHEN the player attempts to open party management, THEN the option is unavailable (greyed out or hidden).

- [ ] **AC-2** (rest-point requirement): GIVEN the player is NOT at a `rest_point` entity (camp, inn, or safe location per Living Environment CR-3/TR-env-009 taxonomy), WHEN the player attempts to open party management, THEN the option is unavailable.

- [ ] **AC-3** (availability): GIVEN the player is at a rest point and not in combat/conversation/transition, WHEN the player attempts to open party management, THEN the option is available (via the wait/rest dialog "Manage Party" option or the dedicated keybinding).

- [ ] **AC-4** (world pause): GIVEN party management is open, THEN the live world is paused — the game clock does not advance and exploration is suspended while the screen is open.

- [ ] **AC-5** (availability state exposure): GIVEN any context state change (combat starts/ends, conversation starts/ends, transition starts/ends, rest-point arrival/departure), THEN the availability state updates within the same frame via event subscription.

- [ ] **Edge case — mid-management context change**: GIVEN party management is open, WHEN combat would otherwise start, THEN combat cannot begin while management is open (world paused); management must close first.

- [ ] **Edge case — wait/rest dialog integration**: GIVEN the wait/rest dialog at a rest point, THEN the "Manage Party" option appears only when the full availability predicate holds.

---

## Implementation Notes

*Derived from GDD CR-3 — to be confirmed against the Party ADR once it exists:*

### Availability Predicate

```go
// PartyManagementAvailable reports whether the management entry points
// (dialog option, keybinding) should be offered. Pure read-only evaluation.
func (p *PartySystem) PartyManagementAvailable() bool
// true ⇔ atRestPoint && !inCombat && !inConversation && !inTransition
```

- `atRestPoint`: party position is within a `rest_point` entity's radius (Living Environment rest-point taxonomy, TR-env-009).
- Context flags update by subscribing to existing events — `combat.started` / `combat.ended`, conversation start/end, region-transition events (ADR-0011 synchronous dispatch; same-frame availability per AC-5). Subscription priorities follow ADR-0011's table.
- The game clock must not advance while management is open — reuse the Living Environment clock-pause mechanism (ADR-0007: never advance time during menus); exploration suspension follows the Entity Framework suspension model (TR-ef-009 guards).

### Exposure

- The wait/rest dialog and the keybinding entry point both consult `PartyManagementAvailable()` (AC-3, edge case). Story 014 renders the greyed/hidden states; this story owns the predicate.
- Emit `party.management_availability_changed` for UI consumption (Story 001's catalog note applies).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 004: Swap/dismissal operations (they call this story's predicate defensively)
- Story 014: The party management screen UI, greyed/hidden rendering
- Living Environment epic: Rest-point entities and wait/rest dialog mechanics

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1**: Each blocked context (combat, conversation, transition) → unavailable
- **AC-2**: Away from rest point → unavailable
- **AC-3**: Rest point + clear context → available via both entry points
- **AC-4**: Open management → clock frozen, world paused; closes → resumes
- **AC-5**: Availability flips same-frame on context events
- **Edge cases**: combat cannot start while management open; dialog option follows the predicate

---

## Test Evidence

**Story Type**: Logic
**Required evidence**:
- Logic: `tests/unit/party/rest_point_gating_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (roster storage), Living Environment epic (complete — rest points, clock pause), Tactical Combat Engine epic (complete — combat context events)
- Unlocks: Story 014
