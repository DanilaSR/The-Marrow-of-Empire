# Story 008: Companion AI Behavior Modes (Exploration)

> **Epic**: Party System
> **Status**: Blocked
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (5–7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

> ⛔ **BLOCKED**: `TR-party-003` has no governing ADR — registry GAP: "needs Party ADR or ADR-0009 extension". Run `/architecture-decision` (Party ADR, or extend ADR-0009) before implementation. Once the governing ADR is Accepted, flip this story to Ready.

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-003` — Companion AI behavior modes (Aggressive/Defensive/Passive), set during exploration, persist across save/load; TCE translates mode → target selection + ability-use in AllyTurn

**ADR Governing Implementation**: N/A — GAP (registry: "needs Party ADR or ADR-0009 extension")
**ADR Decision Summary**: Behavior modes are party-owned configuration consumed by two systems (exploration AI here, TCE in Story 009). The governing ADR must decide where mode state lives, whether exploration behavior runs through the Entity Framework suspension model, and how modes interact with ADR-0009's combat AI ownership.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Exploration behaviors (movement toward threats, flee paths) eventually need movement/pathfinding — verify the nav-mesh contract (ADR-0004, TR-asset-011) before implementing the movement parts. The mode model itself is pure Go.

**Control Manifest Rules (Feature Layer)**:
- Required: All state mutations go through the owning system's interface, with events for notification (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Forbidden: Never hardcode gameplay values — all knobs external config (coding standards)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-5 (Companion AI Behavior), exploration column, scoped to this story:*

- [ ] **AC-1** (AC-PS-07, aggressive initiation): GIVEN a companion set to Aggressive mode and the player NOT in stealth, WHEN hostile wildlife is detected within `aggressive_detection_range` (15 units) during exploration, THEN the companion begins moving toward the threat within 2 seconds with no player input, and continues until contact or combat begins.

- [ ] **AC-2** (AC-PS-07b, aggressive respects stealth): GIVEN a companion set to Aggressive mode and the player IS in stealth, WHEN hostile wildlife enters detection range, THEN the companion does NOT initiate combat.

- [ ] **AC-3** (defensive exploration): GIVEN a companion set to Defensive mode during exploration, THEN the companion stays within `defensive_proximity_radius` (5 units) of the player character and does not initiate combat.

- [ ] **AC-4** (passive exploration): GIVEN a companion set to Passive mode during exploration, WHEN a threat appears, THEN the companion does not attack and flees toward the nearest party member at 1.5× exploration speed until ≥ `flee_distance` (15 units) from the threat.

- [ ] **AC-5** (passive, no support abilities): GIVEN a Passive-mode companion with zero support abilities, THEN the companion remains at the player's position and does nothing — intentional behavior, not a bug.

- [ ] **AC-6** (mode lifecycle): GIVEN a newly recruited companion, THEN the companion defaults to Defensive mode; modes are set only during exploration (never mid-combat); mode changes persist across save/load.

- [ ] **AC-7** (scout precedence): GIVEN a companion who is both scout and Aggressive, THEN scout behavior takes precedence — the scout stops and alerts on detection rather than attacking (cross-contract with Story 010).

- [ ] **Edge case — mode persistence across regions**: GIVEN a mode set in region A, THEN the mode persists when the party transitions to region B (save/load round-trip).

---

## Implementation Notes

*Derived from GDD CR-5, Tuning Knobs — to be confirmed against the governing ADR once it exists:*

### Mode Model

```go
type BehaviorMode int
const (
    BehaviorAggressive BehaviorMode = iota
    BehaviorDefensive
    BehaviorPassive
)
```

- Mode is a per-companion field owned by the Party System (write-ownership); TCE and exploration AI read it (read-only).
- Default `Defensive` on recruitment; set during exploration only; serialized with the PartyState per ADR-0002 full-state saves.
- Mode change emits a `party.mode_changed` event (Story 001's catalog note applies).

### Exploration Rules

- **Aggressive**: detection check uses `aggressive_detection_range` (15u, config), subject to the same opposed-roll + LOS + lighting rules as scout detection (Story 010 defines the shared detection helpers). Player stealth state suppresses initiation (AC-2). Movement toward the threat must begin within 2 seconds of detection (AC-1) — wire the 2-second latency budget into the acceptance test with a tolerant clock.
- **Defensive**: proximity radius 5u (config `defensive_proximity_radius`); no combat initiation.
- **Passive**: flee at 1.5× exploration speed toward nearest party member until 15u (`flee_distance`) from the threat; zero-support companions hold position (AC-5).

### Tuning Knobs (external config)

`aggressive_detection_range` 15 (10–25), `aggressive_ability_reserve` 0.25 (0.0–0.5), `defensive_proximity_radius` 5 (3–8), `flee_distance` 15 (10–30).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 009: Combat-context mode translation (AllyTurn target selection, ability-use rules, player override)
- Story 010: Scout detection mechanics (opposed checks, cone, staged alert)
- Story 011: Formation positions during exploration movement

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1**: Aggressive + no stealth → movement toward threat ≤2s, continues to contact/combat
- **AC-2**: Aggressive + stealth → no initiation
- **AC-3**: Defensive → stays within 5u, no initiation
- **AC-4**: Passive → flees 1.5× speed to ≥15u
- **AC-5**: Passive + zero support abilities → holds position (assert no movement, no attacks)
- **AC-6**: New recruit defaults Defensive; mid-combat set rejected; save/load round-trip keeps mode
- **AC-7**: Scout+Aggressive → alert, not attack
- **Edge cases**: region transition preserves mode

---

## Test Evidence

**Story Type**: Logic
**Required evidence**:
- Logic: `tests/unit/party/behavior_modes_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (roster storage)
- Unlocks: Story 009
