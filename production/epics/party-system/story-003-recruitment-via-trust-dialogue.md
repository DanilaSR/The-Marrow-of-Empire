# Story 003: Recruitment via Trust & Dialogue

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
**ADR (secondary)**: ADR-0006: Character Stats & Component Schema — recruitment validation reads companion components (Stats, Knowledge) through the typed component stores
**ADR Decision Summary**: Recruitment is a cross-system flow — the AI Dialogue Engine initiates the conversation, the Party System validates conditions and processes the join. Per ADR-0011, mutations go through the Party System's interface with events for notification; direct cross-system calls are permitted only for read-only queries (Party reads trust from Trust & Relationship's read interface).

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Cross-system integration at the dialogue boundary. ADR-0011 verification item: `events.Event` dispatch model (sync vs async) — confirm against `kaiju/src/engine/` before wiring recruitment events. No other Kaiju-specific APIs.

**Control Manifest Rules (Feature Layer)**:
- Required: Synchronous event dispatch via EventBus with system priority ordering (ADR-0011)
- Required: Event subscribers register by event type + priority (1 = highest) (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations — all mutations go through the owning system's interface (ADR-0011)
- Forbidden: Never embed NPC knowledge in Go code — recruitment thresholds come from companion NPC data (ADR-0005)
- Guardrail: Event dispatch (5 subscribers): <1µs (ADR-0011)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-2 (Companion Recruitment), scoped to this story:*

- [ ] **AC-1** (AC-PS-01, join into active party): GIVEN a companion NPC with trust level 2 and `recruitment_threshold = 2`, WHEN the player asks them to join via dialogue and a party slot is available, THEN the companion appears in the party screen's active slots, follows in formation during exploration, and is present in the next combat encounter's roster. The companion is tagged `pc_controlled` (verifiable via debug overlay).

- [ ] **AC-2** (AC-PS-02, join into pool when full): GIVEN a full party (4 members at VS), WHEN a companion with sufficient trust agrees to join, THEN the companion appears in the recruitment pool (party management screen) and is NOT added to the active formation. The companion is visible at the next rest point.

- [ ] **AC-3** (trust threshold insufficient): GIVEN a companion whose trust is below their `recruitment_threshold`, WHEN the player asks them to join, THEN the request is rejected and the companion does not transition state.

- [ ] **AC-4** (story condition unmet): GIVEN a companion whose trust is sufficient but whose required story condition (`discovery_trigger`, same format as NPC Knowledge Graph discovery triggers) is unmet, WHEN the player asks them to join, THEN recruitment is deferred — the companion indicates via dialogue what needs to happen first, and the condition is tracked in the companion's recruitment data.

- [ ] **AC-5** (trust retention): GIVEN a companion recruited at trust level 2, THEN the companion retains trust level 2 after joining (recruitment must not reset trust).

- [ ] **Edge case — VS vs Full Vision cap**: GIVEN the same recruitment request, THEN the capacity check uses the configured tier value (`vs_party_size` 4 / `full_vision_party_size` 6) — never a hardcoded number.

---

## Implementation Notes

*Derived from GDD CR-2 and ADR-0011/ADR-0006:*

### Validation Flow

Recruitment is initiated through dialogue. The AI Dialogue Engine (ADE, already implemented) handles the conversation; the Party System exposes:

```go
// Recruit validates conditions and processes the join.
// Reads trust (read-only query via Trust & Relationship interface).
// Returns: joined (InParty or InPool), unmet conditions for dialogue to surface.
func (p *PartySystem) Recruit(companionID GameEntityID) (RecruitResult, error)

type RecruitResult struct {
    Joined    bool
    Slot      SlotKind // SlotActive | SlotPool
    Unmet     []UnmetCondition // trust_too_low, story_condition, roster_full(handled), ...
}
```

- **Trust check**: `trust_level >= recruitment_threshold`, both read from the companion's NPC/recruitment data (data-driven, per-companion configurable — never in Go code). Trust read is a read-only query — permitted by ADR-0011.
- **Story condition**: `discovery_trigger` on the companion entity, same format as NPC Knowledge Graph discovery triggers (ADR-0005). Unmet → return the condition so ADE can phrase the companion's deferral response (AC-4).
- **Slot assignment**: free active slot → `InParty` + `pc_controlled` tag (Story 001 owns the tag); roster full → `InPool` (AC-2).

### Event Contract

- Emit `party.recruited` (companion ID, slot) after a successful join, synchronous per ADR-0011, so formation (Story 011), TCE roster (Story 007), and UI update in the same frame.
- ADE triggers recruitment via the Party System's interface, not by mutating party state directly.

### State Machine Note

The `Unrecruited → InParty` / `Unrecruited → InPool` transitions are owned by the state machine (Story 002, currently blocked). This story implements the validation + assignment logic and calls the state transition; with Story 002 blocked, coordinate the storage of state with the Party ADR before implementation.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Roster storage and capacity enforcement internals
- Story 002: State transition definitions and persisted `cause`
- Story 005: Re-recruitment rules after departure (different thresholds)
- Story 012: What the recruited companion's knowledge contributes while active

---

## QA Test Cases

> Generated by /qa-plan sprint-6 (2026-08-13). Story remains Blocked on PARTY-002 (which is blocked on ADR-0016 acceptance).

**Automated (integration) — `tests/integration/party/recruitment_test.go`:**
- **AC-1**: trust 2 / threshold 2 / slot free → active slots + `pc_controlled` + next-encounter roster
- **AC-2**: sufficient trust / roster full → pool only, not in formation
- **AC-3**: trust below threshold → rejected, no state change
- **AC-4**: story condition unmet → deferred, condition tracked, surfaced via dialogue
- **AC-5**: recruitment does not reset trust (stays 2)
- **Edge cases**: tier-configured capacity; double-recruit idempotent; recruitment during combat/conversation rejected (Story 013 gating)

**Estimated test count**: ~10
## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/party/recruitment_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (roster storage), Story 002 (state machine — BLOCKED on Party ADR), Trust & Relationship epic (complete)
- Unlocks: Story 012
