# Story 010: Scout System

> **Epic**: Party System
> **Status**: Blocked
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (5–7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

> ⛔ **BLOCKED**: `TR-party-004` has no governing ADR — registry GAP: "needs Party ADR". Run `/architecture-decision` to create the Party System ADR before implementation. Once the ADR is Accepted, flip this story to Ready.

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-004` — Scout system: single scout designation, detection range 10+WIS_mod×2, opposed check d20+WIS_mod vs d20+threat_stealth+terrain, forward 180° cone, darkness/fog disadvantage, staged alert

**ADR Governing Implementation**: N/A — GAP (registry: "needs Party ADR")
**ADR Decision Summary**: Scout detection needs a decision on: who evaluates perception checks (Party vs a shared detection helper), how terrain/lighting modifiers arrive from Living Environment (ADR-0007) and Camera (ADR-0003), and how the staged-alert state machine interacts with ADR-0009 combat initiation. The d20 resolution pattern itself mirrors ADR-0009's attack resolution.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Detection is a 180° cone — cone math uses `kaijuengine.com/matrix` (never gonum/mathgl). Staged alert triggers animation/VO events — presentation is out of scope (ADR-0012 / audio epic consume the events).

**Control Manifest Rules (Feature Layer)**:
- Required: All state mutations go through the owning system's interface, with events for notification (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Forbidden: Never hardcode gameplay values — all knobs external config (coding standards)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-4 (Scout mode), Formulas (Scout Detection, Base Detection), Edge Cases, scoped to this story:*

- [ ] **AC-1** (AC-PS-09a, WIS 16 detection): GIVEN a test map with a hidden threat (stealth 10) at a known position and a scout with WIS 16 (+3), WHEN the party approaches, THEN the scout detects the threat at `scout_detection_base + WIS_mod × scout_wis_scale` (16) units ±1, enters alert state (stops, alert animation), and the player receives a prompt (approach / recall / push forward). Verifiable via debug readout.

- [ ] **AC-2** (AC-PS-09b, WIS 10 detection): GIVEN the same setup with WIS 10 (+0), WHEN the party approaches, THEN detection occurs at 10 units ±1.

- [ ] **AC-3** (opposed check): GIVEN a detection evaluation, THEN detection succeeds only if `scout_perception > threat_stealth`, where `scout_perception = d20 + WIS_mod + scout_level_bonus` and `threat_stealth = d20 + threat_stealth_mod + terrain_cover_bonus`.

- [ ] **AC-4** (disadvantage conditions): GIVEN darkness, fog, heavy foliage, or a threat with full cover, THEN the scout rolls twice and takes the lower result.

- [ ] **AC-5** (180° cone): GIVEN a threat positioned behind the scout's facing, THEN the threat is not detected by the scout regardless of range.

- [ ] **AC-6** (staged alert + hold): GIVEN a detection, THEN the scout enters "alerted" state (stops, hand-raised animation, "Something ahead" voice line) and, if the player does nothing, the scout holds position.

- [ ] **AC-7** (designation rules): GIVEN any party, THEN at most one companion may be scout at a time; the player character cannot be the scout; scout mode is disabled for solo parties.

- [ ] **AC-8** (auto-designation): GIVEN no manual designation, THEN the scout defaults to the companion with highest WIS (ties → highest DEX; further ties → higher trust level; final tie → longest tenure in the party).

- [ ] **AC-9** (designation gating): GIVEN a scout-designation change attempt away from a rest point, THEN the attempt is rejected — designation changes only at rest points (CR-3); the current scout continues until the next rest.

- [ ] **AC-10** (AC-PS-14, combat isolation): GIVEN a scout companion 10 units ahead and hidden enemies between the scout and the party, WHEN combat triggers, THEN the scout deploys at their scout position (potentially isolated), the rest of the party deploys at formation positions, and initiative proceeds normally.

- [ ] **Edge case — base detection (non-scout)**: GIVEN a non-scout party member, THEN they detect threats at `base_detection_range` (8 units) with the same opposed-check rules, without the scout range bonus.

- [ ] **Edge case — aggressive detection**: GIVEN an Aggressive-mode companion, THEN hostile detection within `aggressive_detection_range` (15u) uses the same opposed-roll + LOS + lighting rules (shared helper with Story 008).

---

## Implementation Notes

*Derived from GDD CR-4, Formulas — to be confirmed against the Party ADR once it exists:*

### Detection Formulas (from GDD Formulas section)

```
scout_detection_range = scout_detection_base + WIS_mod × scout_wis_scale   // 10 + WIS_mod×2
scout_perception      = d20 + WIS_mod + scout_level_bonus
threat_stealth        = d20 + threat_stealth_mod + terrain_cover_bonus
success               = scout_perception > threat_stealth
base_detection_range  = 8 units (all party members, non-scout)
```

- All knobs external config: `scout_distance` 10 (5–20), `scout_detection_base` 10 (5–15), `scout_wis_scale` 2 (1–3), `base_detection_range` 8 (5–12), `scout_detection_arc` 180° (90–360).
- Determinism rule (testing standards): the d20 rolls must be injectable — a `Roller` interface, seeded dice in tests. No `math/rand` global state in the detection path.
- Disadvantage = roll twice, take lower (AC-4). Darkness/fog/foliage inputs come from Living Environment (ADR-0007) and region data via read-only queries; full cover from combat positioning rules (ADR-0009).
- Cone check uses `kaijuengine.com/matrix` vector math: facing dot-product gate within ±(arc/2).
- Scout advance: scout moves `scout_distance` units ahead of the party leader (CR-4) — coordinate with Story 011 formation positions.

### Staged Alert

`patrolling → detected(alerted) → player_prompt(approach | recall | push_forward) → hold` — holding is the default on no input (AC-6). The prompt UI and voice lines are presentation contracts; this story emits `party.scout_alert` with the threat reference.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 008: Aggressive-mode detection behavior (consumes the shared detection helper)
- Story 011: Formation positions, including scout advance placement
- Presentation: alert animations, voice lines, prompt UI (ADR-0012 / audio epic)

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1/AC-2**: Detection at 16u ±1 (WIS 16) and 10u ±1 (WIS 10) with seeded rolls
- **AC-3**: Opposed-check outcome matrix — success only when perception > stealth
- **AC-4**: Each disadvantage condition forces lower-of-two-rolls
- **AC-5**: Threat behind facing → no detection at any range
- **AC-6**: Alerted state holds with no player input
- **AC-7**: Single-scout invariant, PC ineligible, solo party → disabled
- **AC-8**: Auto-designation tiebreak chain (WIS → DEX → trust → tenure)
- **AC-9**: Off-rest-point designation change rejected
- **AC-10**: Combat trigger → scout at scout position, rest at formation
- **Edge cases**: non-scout 8u detection; aggressive-mode 15u shared-helper path

---

## Test Evidence

**Story Type**: Logic
**Required evidence**:
- Logic: `tests/unit/party/scout_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (roster storage), Story 011 (formation positions — BLOCKED on Party ADR), Living Environment epic (complete — light/terrain inputs)
- Unlocks: None
