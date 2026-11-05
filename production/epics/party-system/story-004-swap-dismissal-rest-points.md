# Story 004: Swap & Dismissal at Rest Points

> **Epic**: Party System
> **Status**: Ready
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: M (5–7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-001` — Party roster (max 6), recruitment via trust/dialogue, dismissal with confirmation, companion combat participation

**ADR Governing Implementation (primary)**: ADR-0011: Event Bus Architecture
**ADR (secondary)**: ADR-0006: Character Stats & Component Schema — swap reads companion stats for UI display; roster mutations follow the typed-store pattern
**ADR Decision Summary**: Swap and dismissal mutate roster state — per ADR-0011 these mutations go exclusively through the Party System's interface, with synchronous events notifying downstream systems (formation, TCE roster, UI) in the same frame. Dismissal does not reset trust (trust is owned by Trust & Relationship; Party only reads it).

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Same-frame event fan-out on roster changes — verify `events.Event` dispatch model per ADR-0011's Verification Required. The confirmation dialog itself is UI-owned (Story 014); this story implements the operations behind it.

**Control Manifest Rules (Feature Layer)**:
- Required: Synchronous event dispatch via EventBus with system priority ordering (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Guardrail: Event dispatch (5 subscribers): <1µs; all event handlers combined <2ms/frame (ADR-0011)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-3 (Party Management), CR-7 (Dismissal), scoped to this story:*

- [ ] **AC-1** (AC-PS-03a, swap-out): GIVEN a companion in `InParty` at a rest point, WHEN the player swaps them to the pool, THEN the companion is removed from the active formation, appears in the pool list on the party management screen, and is absent from the next combat encounter's roster.

- [ ] **AC-2** (AC-PS-03b, swap-in): GIVEN a companion in `InPool` at a rest point, WHEN the player swaps them into the active party, THEN the companion appears in the active formation, follows in exploration, and is present in the next combat encounter's roster.

- [ ] **AC-3** (AC-PS-11, dismissal): GIVEN a companion in `InPool`, WHEN the player dismisses them at a rest point and confirms the dismissal dialog, THEN the companion transitions to `Departed` with `cause: dismissed`, is removed from the pool list, and returns to their home region (per-companion location in the companion's NPC data).

- [ ] **AC-4** (dismissal keeps trust): GIVEN a companion dismissed voluntarily, THEN the companion's trust level is unchanged by dismissal.

- [ ] **AC-5** (confirmation required): GIVEN a dismissal request, WHEN the confirmation dialog is declined, THEN no state changes occur.

- [ ] **Edge case — swap PC**: GIVEN a swap attempt targeting the player character, THEN the operation is rejected (PC slot locked — Story 001 AC-1).

- [ ] **Edge case — departure on dismissal**: GIVEN a dismissed companion, THEN the companion is NOT automatically accessible at future rest points — re-recruitment requires travel to the home region (Story 005 owns the threshold rules).

---

## Implementation Notes

*Derived from GDD CR-3, CR-7, and ADR-0011:*

### Operations API

```go
// SwapIn / SwapOut move a companion between Active and Pool at a rest point.
// Returns an error if not at a rest point (Story 013 owns the availability
// gate — this story calls it defensively) or if the roster is at capacity.
func (p *PartySystem) SwapIn(companionID GameEntityID) error
func (p *PartySystem) SwapOut(companionID GameEntityID) error

// Dismiss requires prior user confirmation (UI-owned). Party records the
// transition and emits the event; trust is untouched.
func (p *PartySystem) Dismiss(companionID GameEntityID, confirmed bool) error
```

- Swap-in capacity check delegates to Story 001's roster rules (`vs_party_size` / `full_vision_party_size` from config).
- `SwapIn`/`SwapOut` update the `pc_controlled` tag (Story 001 ownership) as part of the roster mutation.
- Dismissal confirmation is a UI concern (Story 014): the Party System refuses `Dismiss(..., confirmed=false)` — the dialog's warning text (knowledge domains no longer available) is authored in Story 014.
- Home region for `Departed` companions comes from the companion's NPC data (per-companion, data-driven — ADR-0005 style).

### Event Contract

- `party.member_added` / `party.member_removed` on swap (Story 001 defines the types).
- `party.dismissed` on confirmed dismissal, carrying the companion ID — downstream: formation (Story 011) drops the companion, TCE roster (Story 007) excludes them, PWS may log the event for world-state purposes.
- All synchronous, ADR-0011 priority ordering; handlers must complete within the 2ms/frame combined budget.

### State Machine Note

The `InParty → InPool` / `InPool → Departed(cause: dismissed)` transitions are defined by the state machine (Story 002, blocked). This story calls the transition interface; with Story 002 blocked, coordinate the transition storage with the Party ADR before implementation.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Transition definitions and persisted `cause`
- Story 005: Re-recruitment thresholds after dismissal/betrayal/death
- Story 013: Rest-point availability gating (combat/conversation/transition blocking)
- Story 014: The party management screen and confirmation dialog UI

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1/AC-2**: Swap both directions → formation, pool list, and next encounter roster all reflect the change
- **AC-3**: Confirmed dismissal → `Departed` `cause: dismissed`, removed from pool, home-region return recorded
- **AC-4**: Trust value identical before/after dismissal
- **AC-5**: Declined confirmation → zero state changes, no events emitted
- **Edge cases**: swap targeting PC rejected; swap at capacity rejected; double-dismiss idempotent

---

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/party/party_management_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (roster storage), Story 002 (state machine — BLOCKED on Party ADR)
- Unlocks: Story 014
