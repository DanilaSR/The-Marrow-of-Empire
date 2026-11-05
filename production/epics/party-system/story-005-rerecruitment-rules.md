# Story 005: Re-recruitment Rules

> **Epic**: Party System
> **Status**: Ready
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: S (2–4 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-001` — Party roster (max 6), recruitment via trust/dialogue, dismissal with confirmation, companion combat participation

**ADR Governing Implementation (primary)**: ADR-0006: Character Stats & Component Schema
**ADR (secondary)**: ADR-0011: Event Bus Architecture — re-recruitment state changes notify downstream systems via events
**ADR Decision Summary**: Re-recruitment is a pure rules layer over companion state and trust — the formula pattern mirrors ADR-0006's data-driven formulas (all knobs external config, clamp rules explicit). State transitions are notified through the EventBus per ADR-0011; trust is read from Trust & Relationship, never mutated here.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Pure Go logic — no Kaiju-specific APIs. The 72-game-hour trust freeze referenced in betrayal re-recruitment is owned by Trust & Relationship (implemented); Party only reads its state.

**Control Manifest Rules (Feature Layer)**:
- Required: All state mutations go through the owning system's interface, with events for notification (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Forbidden: Never hardcode gameplay values — all knobs external config (coding standards)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-7 (Re-recruitment) and Formulas, scoped to this story:*

- [ ] **AC-1** (AC-PS-10a, betrayal re-recruitment): GIVEN a companion departed via trust betrayal (recruitment threshold 2, penalty 1), WHEN trust is restored to 3 and a party slot is free, THEN the companion's dialogue offers the join option. Accepting transitions them to `InParty` (or `InPool` if full).

- [ ] **AC-2** (AC-PS-10b, dismissal re-recruitment): GIVEN a companion dismissed voluntarily (recruitment threshold 2), WHEN trust is at or above 2 and a slot is free, THEN the companion can be re-recruited at the original threshold with no penalty.

- [ ] **AC-3** (AC-PS-10c, death blocks): GIVEN a companion who died in combat (`cause: death`), WHEN the player travels to the companion's home region, THEN the companion is not present and cannot be re-recruited.

- [ ] **AC-4** (formula): GIVEN `recruitment_threshold = 3` and `departure_repentance_penalty = 2`, WHEN the re-recruitment threshold is calculated, THEN the result is `min(3 + 2, 3) = 3` (clamped at `max_trust_level`).

- [ ] **AC-5** (data-driven): GIVEN any re-recruitment evaluation, THEN all values (`departure_repentance_penalty`, `max_trust_level`, thresholds) come from external config / companion NPC data — none hardcoded in Go.

- [ ] **Edge case — trust freeze interaction**: GIVEN a companion departed via betrayal during Trust & Relationship's 72-game-hour freeze window, THEN re-recruitment is not offered while the freeze is active (Party reads the freeze state; T&R owns it).

- [ ] **Edge case — story departure**: GIVEN a companion departed with `cause: story`, THEN re-recruitment conditions are story-specific, defined in the companion's narrative data — the generic formula does not apply.

---

## Implementation Notes

*Derived from GDD CR-7, Formulas (Re-Recruitment Requirement), and ADR-0006:*

### Re-Recruitment Threshold

```
re_recruitment_threshold = min(recruitment_threshold + departure_repentance_penalty, max_trust_level)
```

- `max_trust_level = 3` (Trust & Relationship CR-1)
- `departure_repentance_penalty = 1` (tuning knob, safe range 0–2)
- All values read from config / companion NPC data at evaluation time (AC-5).

### Per-Cause Rules

| `cause` | Re-recruitment rule |
|---------|---------------------|
| `trust_betrayal` | Trust restored to `re_recruitment_threshold` (penalty applied, clamped). T&R trust freeze (72 game hours) must not be active. |
| `story` | Story-specific conditions from the companion's narrative data — not the generic formula. |
| `dismissed` | Original `recruitment_threshold`, no penalty, no freeze requirement. |
| `death` | Blocked permanently — no re-recruitment path exists (Story 002 enforces the transition block; this story exposes it to dialogue gating). |

### Interface

```go
// CanReRecruit evaluates whether the companion's dialogue may offer the
// join option again. Pure read-only evaluation — no state mutation.
func (p *PartySystem) CanReRecruit(companionID GameEntityID) (bool, []UnmetCondition)
```

- The join itself reuses Story 003's `Recruit` path — this story only defines the *eligibility* evaluation the dialogue uses to show/hide the join option.
- Trust reads are read-only queries against Trust & Relationship (allowed by ADR-0011).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: `Departed` state storage, `cause` persistence, death-block transition enforcement
- Story 003: The join operation itself (`Recruit`)
- Story 006: What triggers a departure in the first place

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1**: betrayal cause, threshold 2 + penalty 1 → eligible only at trust 3
- **AC-2**: dismissed cause → eligible at original threshold, no penalty
- **AC-3**: death cause → never eligible
- **AC-4**: formula clamp — 3 + 1 → 3; 3 + 2 → 3 (clamped)
- **AC-5**: all knobs config-driven
- **Edge cases**: freeze-active blocks betrayal re-recruit; story cause defers to narrative data; pool-full eligibility still returns joinable-but-pooled

---

## Test Evidence

**Story Type**: Logic
**Required evidence**:
- Logic: `tests/unit/party/rerecruitment_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (roster storage), Story 002 (state machine — BLOCKED on Party ADR), Trust & Relationship epic (complete)
- Unlocks: Story 006
