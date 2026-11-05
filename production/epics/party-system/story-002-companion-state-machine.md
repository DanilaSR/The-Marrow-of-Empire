# Story 002: Companion State Machine

> **Epic**: Party System
> **Status**: Blocked
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (5–7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

> ⛔ **BLOCKED**: `TR-party-002` has no governing ADR — registry GAP: "needs Party ADR". Run `/architecture-decision` to create the Party System ADR before implementation. Once the ADR is Accepted, flip this story to Ready.

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-002` — Companion state machine: Unrecruited/InParty/InPool/Departed; Departed carries persisted cause; PC slot always locked; death blocks re-recruitment; departures deferred during combat/conversation

**ADR Governing Implementation**: N/A — GAP (registry: "needs Party ADR")
**ADR Decision Summary**: The companion lifecycle is the architectural core of the Party System and currently has no ADR. The Party ADR must decide: where companion state lives (component field vs. PartyState store vs. dedicated store), how `cause` is persisted, and how departure deferral interacts with ADR-0011's synchronous event rules and ADR-0007's clock-pause rules.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Pure Go state model. Persistence of the `cause` field follows the Entity Framework component-store pattern once the Party ADR fixes the storage decision.

**Control Manifest Rules (Feature Layer)**:
- Required: All state mutations go through the owning system's interface, with events for notification (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Guardrail: Event dispatch (5 subscribers): <1µs (ADR-0011)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` States and Transitions section, CR-7 (Companion Departure), scoped to this story:*

- [ ] **AC-1** (states + persisted cause): GIVEN any companion, THEN the companion is in exactly one of `Unrecruited` / `InParty` / `InPool` / `Departed`, and a `Departed` companion carries a persisted `cause` of `trust_betrayal`, `story`, `death`, or `dismissed` that survives save/load.

- [ ] **AC-2** (departure deferral — combat): GIVEN a departure trigger fires during an active combat encounter, WHEN the encounter ends, THEN the departure transition executes (AC-PS-04b state part) — the companion fights normally for the remainder of the encounter.

- [ ] **AC-3** (departure deferral — conversation): GIVEN a departure trigger fires during an active conversation, WHEN the conversation ends, THEN the departure transition executes (AC-PS-04c state part) — the companion does not interject for the remainder of the conversation.

- [ ] **AC-4** (death blocks re-recruitment): GIVEN a companion in `Departed` with `cause: death`, WHEN a re-recruitment attempt is made, THEN the `Departed → InParty` / `Departed → InPool` transitions are permanently blocked (AC-PS-10c state part).

- [ ] **AC-5** (dismissal state): GIVEN a companion in `InPool`, WHEN dismissed with confirmation, THEN the companion transitions to `Departed` with `cause: dismissed` (AC-PS-11 state part).

- [ ] **AC-6** (story departure state): GIVEN a story departure trigger satisfied, WHEN the trigger fires, THEN the companion transitions to `Departed` with `cause: story` (AC-PS-15 state part).

- [ ] **Edge case — death precedence**: GIVEN a companion dies AND trust drops below the departure threshold in the same combat, THEN death takes precedence — `cause: death` is recorded and the trust-betrayal path is not triggered.

- [ ] **Edge case — pool member death**: GIVEN a companion in the recruitment pool is killed by a world event while waiting at camp, THEN the companion transitions to `Departed` with `cause: death`; the player is notified on next rest.

---

## Implementation Notes

*Derived from GDD States and Transitions, CR-7, Edge Cases — to be confirmed against the Party ADR once it exists:*

- Implement the four states and every transition listed in the GDD table, including the `Departed → Unrecruited` return-home path and its re-recruitment guards.
- `cause` is a persisted enum field on the `Departed` state — never inferred, never lost across save/load.
- Deferral mechanism must hold a pending-transition record (companion ID + cause) until combat/conversation end, then execute before any other party processing in that frame. The exact ordering rule (before/after other post-combat events such as XP) is a Party ADR decision point.
- Death precedence over trust-betrayal departure must be an explicit, unit-tested rule.
- Subscribe to Trust & Relationship events (trust drop → departure check) and TCE events (death, combat ended → flush deferred transitions) per ADR-0011 priority ordering. Priority band assignment is a Party ADR decision point — propose `party` at priority between Trust (3) and Thread Investigation (4).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Roster storage, capacity, PC slot locking
- Story 005: Re-recruitment threshold formulas (this story only enforces the death-block)
- Story 006: Departure presentation (scenes, banners, journal entries, reactions)
- Story 004: The dismissal operation UI and confirmation flow

---

## QA Test Cases

> Generated by /qa-plan sprint-6 (2026-08-13). Story remains Blocked until ADR-0016 accepted (sprint task PRC-001).

**Automated (unit) — `tests/unit/party/state_machine_test.go`:**
- **AC-1**: exactly one of Unrecruited/InParty/InPool/Departed; Departed.cause ∈ {trust_betrayal, story, death, dismissed} survives save/load
- **AC-2/AC-3**: departure trigger during combat/conversation → deferred until end; companion behaves normally meanwhile
- **AC-4**: cause: death → re-recruitment transitions permanently blocked
- **AC-5/AC-6**: dismissal (with confirmation) and story trigger → Departed with correct cause
- **Edge cases**: death takes precedence over trust betrayal in same combat; pool-member death via world event; illegal transition (InPool → Departed with no cause) rejected

**Estimated test count**: ~12
## Test Evidence

**Story Type**: Logic
**Required evidence**:
- Logic: `tests/unit/party/state_machine_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (roster storage)
- Unlocks: Story 003, Story 004, Story 005, Story 006
