# Story 006: Departure & Death Handling

> **Epic**: Party System
> **Status**: Blocked
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: M (5–7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

> ⛔ **BLOCKED**: `TR-party-002` has no governing ADR — registry GAP: "needs Party ADR". Run `/architecture-decision` to create the Party System ADR before implementation. Once the ADR is Accepted, flip this story to Ready.

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-002` — Companion state machine: Unrecruited/InParty/InPool/Departed; Departed carries persisted cause; PC slot always locked; death blocks re-recruitment; departures deferred during combat/conversation

**ADR Governing Implementation**: N/A — GAP (registry: "needs Party ADR")
**ADR Decision Summary**: Departure spans four systems — Trust & Relationship (trust-drop trigger), Tactical Combat Engine (death, combat-end deferral flush), Persistent World State / Investigation Journal (journal entries, world events), and Presentation (scenes, banners, reactions). The Party ADR must fix the event flow and priority ordering; ADR-0011's synchronous dispatch and ADR-0007's clock-pause rules constrain the deferral design.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Deferral executes at combat/conversation end — confirm `events.Event` dispatch ordering across systems per ADR-0011 Verification Required. Presentation pieces (fades, banners) follow ADR-0012; VS knockout uses existing TCE combat-resolution hooks.

**Control Manifest Rules (Feature Layer)**:
- Required: Synchronous event dispatch via EventBus with system priority ordering (ADR-0011)
- Required: Death must be recorded BEFORE trust events fire — priority ordering is correctness, not style (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Guardrail: All event handlers combined <2ms/frame (ADR-0011)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-7 (Companion Departure), Edge Cases, scoped to this story:*

- [ ] **AC-1** (AC-PS-04a, trust departure — immediate): GIVEN a companion with `departure_threshold = 1` at trust level 2, WHEN a trust-loss event reduces trust to 1 while outside combat and conversation, THEN the departure scene plays (goodbye dialogue + companion reactions), a journal entry is created, and the companion is removed from the active formation.

- [ ] **AC-2** (AC-PS-04b, deferred — combat): GIVEN the same trigger during an active combat encounter, WHEN the encounter ends, THEN a notification banner appears ("[Name] is leaving"), the departure scene plays, and the companion is removed from the party.

- [ ] **AC-3** (AC-PS-04c, deferred — conversation): GIVEN the same trigger during an active conversation, WHEN the conversation ends, THEN the departure scene plays and the companion is removed.

- [ ] **AC-4** (AC-PS-15, story departure): GIVEN a companion with a story departure trigger satisfied, WHEN the trigger fires, THEN a departure scene plays (final dialogue + companion reactions), a journal entry is created, the companion transitions to `Departed` with `cause: story`, and the companion returns to their home region.

- [ ] **AC-5** (presentation requirements): GIVEN any departure type, THEN the presentation includes: goodbye dialogue (authored, context-specific), reaction lines from at least one other active companion per companion with trust level ≥ 1 toward the departing companion, and a journal entry recording the departure event.

- [ ] **AC-6** (VS knockout, no death): GIVEN a companion reaches 0 HP at VS (`companion_death_enabled = false`), THEN the companion is unconscious for the remainder of combat and revives with 1 HP after combat — no `Departed` transition, no death countdown.

- [ ] **AC-7** (Full Vision death): GIVEN `companion_death_enabled = true`, WHEN a companion reaches 0 HP and fails to stabilize within 3 rounds (TCE death rules), THEN the companion dies permanently — `Departed` with `cause: death`, death animation, body remains, notification banner, reactions from other companions.

- [ ] **Edge case — solo continuation**: GIVEN the only non-PC active member departs mid-expedition, THEN the player continues solo (party size 1) and can return to camp to bring companions from the pool.

- [ ] **Edge case — death precedence**: GIVEN a companion dies and trust drops below the departure threshold in the same combat, THEN death takes precedence — the trust-betrayal departure path is not triggered.

- [ ] **Edge case — pool member killed**: GIVEN a companion in the recruitment pool is killed by a world event while waiting at camp, THEN the companion transitions to `Departed` with `cause: death` and the player is notified on next rest.

---

## Implementation Notes

*Derived from GDD CR-7, Edge Cases — to be confirmed against the Party ADR once it exists:*

### Trigger Sources

- **Trust betrayal**: subscribe to Trust & Relationship's trust-change event (ADR-0011, priority per ADR table — Trust at priority 3). When `trust_level <= departure_threshold` for an active or pooled companion → departure with `cause: trust_betrayal`. Note the T&R contract: trust clamps at 0, and reaching the clamp IS the departure event for threshold-0 companions.
- **Story**: scripted triggers fire a departure request with `cause: story` + companion-specific scene data from the companion's narrative data.
- **Death**: subscribe to TCE's entity-died event. At VS (`companion_death_enabled = false`) the companion enters knockout per existing TCE combat-resolution hooks (AC-6) — no party state change. At Full Vision, death is permanent (AC-7). The Party ADR must decide who owns the `companion_death_enabled` option — Party config or TCE config.

### Deferral

- Pending departure records (companion ID + cause + presentation data) are held while combat or conversation is active; flushed at the first end event. The flush ordering relative to other end-of-combat processing (XP, loot, clock resume) is a Party ADR decision point — propose: flush after post-combat stabilization but before UI refresh.
- During deferral the companion fights/talks normally (state machine Story 002 owns the transition; this story owns the trigger + flush).

### Presentation Contract

- Departure presentation (scene, banner, reactions, journal entry) is driven by data from this system: companion ID, `cause`, authored dialogue references, reacting companion list (trust ≥ 1 toward the departing companion), journal entry payload. Presentation rendering is out of scope (UI epics consume this contract).
- Respect `reduce_motion` for all departure animations (GDD UI Requirements).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: `Departed` transition storage and `cause` persistence
- Story 005: Whether a departed companion can come back (re-recruitment rules)
- Story 014: Rendering the departure banner/scene UI
- Investigation Journal epic (complete): Journal entry mechanics — this story emits the payload

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1**: Trust drop outside combat/conversation → immediate scene + journal + formation removal
- **AC-2/AC-3**: Same trigger inside combat/conversation → deferred to end, banner then scene
- **AC-4**: Story trigger → scene + journal + `Departed` `cause: story` + home-region return
- **AC-5**: Every departure type produces goodbye dialogue, ≥1 reaction per trust≥1 companion, journal entry
- **AC-6**: VS 0 HP → knockout, 1 HP revive, no state change
- **AC-7**: FV death → `cause: death`, permanent, presentation fires
- **Edge cases**: solo continuation; death precedence over betrayal; pool-member death notification on next rest

---

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/party/departure_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 002 (state machine — BLOCKED on Party ADR), Story 005 (re-recruitment eligibility), Trust & Relationship epic (complete), Tactical Combat Engine epic (complete)
- Unlocks: None
