# Story 011: Party Formation & Marching Order

> **Epic**: Party System
> **Status**: Blocked
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: S (2–4 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

> ⛔ **BLOCKED**: `TR-party-005` has no governing ADR — registry GAP: "needs Party ADR". Run `/architecture-decision` to create the Party System ADR before implementation. Once the ADR is Accepted, flip this story to Ready.

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-005` — Formation: default loose diamond at 3u spacing, trailing arc for other sizes, front/back rank config at rest points, combat deploys companions at formation positions ±1u

**ADR Governing Implementation**: N/A — GAP (registry: "needs Party ADR")
**ADR Decision Summary**: Formation positions feed three consumers — exploration movement (Living Environment / movement), combat initial deployment (ADR-0009), and the camera's party-centroid framing (ADR-0003, TR-cam-006). The Party ADR must decide who computes formation offsets and how the front/back rank config persists (Save/Load, ADR-0002).

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Offset computation is pure vector math — use `kaijuengine.com/matrix` (never gonum/mathgl). Deployment consumes the party Transform positions per ADR-0003's tracked-set contract (TR-cam-016).

**Control Manifest Rules (Feature Layer)**:
- Required: All state mutations go through the owning system's interface, with events for notification (ADR-0011)
- Forbidden: Never hardcode gameplay values — all knobs external config (coding standards)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-4 (Party Formation), scoped to this story:*

- [ ] **AC-1** (AC-PS-06, combat deployment): GIVEN a party of 4 with front rank (soldier, scholar) and back rank (hunter, player) set via the party management screen, WHEN combat starts, THEN by the start of round 1 each companion is within `formation_spacing` (±1 unit) of their assigned position relative to the player character. Front-rank companions deploy ahead; back-rank behind or to the side. Verifiable via debug position readout.

- [ ] **AC-2** (default loose diamond): GIVEN a party of 4 in exploration with no rank overrides, THEN the player character leads and companions follow in a loose diamond at `formation_spacing` (3u) behind the leader.

- [ ] **AC-3** (arc shapes by size): GIVEN party sizes other than 4, THEN companions fan out in a trailing arc — 2 companions: line; 3: triangle; 5: pentagon arc; 6: double-row arc.

- [ ] **AC-4** (rank assignment): GIVEN a companion at a rest point, THEN the player can set front/back rank per companion; at VS the default auto-assignment places melee-oriented companions in front and ranged/caster companions in back.

- [ ] **AC-5** (rank persistence): GIVEN rank assignments set at a rest point, THEN they persist across save/load.

- [ ] **AC-6** (scout advance): GIVEN a designated scout, THEN the scout moves `scout_distance` (10u) units ahead of the party leader during exploration (cross-contract with Story 010).

- [ ] **AC-7** (AC-PS-14, deployment with isolated scout): GIVEN combat triggers while the scout is ahead, THEN the rest of the party deploys at formation positions relative to the player character while the scout deploys at their scout position.

- [ ] **Edge case — solo party**: GIVEN a party of 1, THEN no formation offsets are computed; the PC moves alone.

- [ ] **Edge case — free-form editor deferred**: GIVEN any party state at VS, THEN the free-form grid editor is NOT available — front/back toggle only (Full Vision feature).

---

## Implementation Notes

*Derived from GDD CR-4 — to be confirmed against the Party ADR once it exists:*

### Formation Geometry

- Offsets are computed relative to the leader's position and facing; the leader is the player character (slot 0 in the roster).
- Default diamond at `formation_spacing` 3u (config, safe range 1–5); arc layouts per size per the GDD table (AC-3).
- Front rank deploys ahead of the PC, back rank behind/side (AC-1, AC-2).
- Compute with `kaijuengine.com/matrix` vector math. Offsets must be allocation-free in the per-frame path (camera reads party positions ~15/frame per TR-cam-016; recompute on membership change only, cache otherwise).

### Rank Configuration

- Per-companion rank field owned by the Party System, set at rest points only (Story 014 exposes the toggle; Story 013 gates availability).
- VS auto-assignment: melee-oriented → front, ranged/caster → back, derived from the companion's role/archetype data (read from Character Stats / NPC data).
- Persisted in the PartyState per ADR-0002 full-state saves (AC-5).

### Combat Deployment Contract

- At encounter start, Party supplies per-member deployment targets = formation offset + leader position; TCE (Story 007's roster consumer) places combatants. Tolerance ±1u verified by the start of round 1 (AC-1).
- Scout deploys at scout position, not formation position (AC-7, AC-PS-14).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 007: Who is in the roster at encounter start
- Story 010: Scout detection mechanics (this story only places the scout)
- Story 014: The rank-toggle UI on the party management screen
- Full Vision: free-form formation editor (deferred)

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1**: 4-member party, ranks set → each companion within ±1u of assigned position by round 1 start (debug readout)
- **AC-2**: 4-member default → loose diamond at 3u behind leader
- **AC-3**: Sizes 2/3/5/6 → line / triangle / pentagon arc / double-row arc
- **AC-4**: Auto-assignment melee→front, ranged/caster→back at VS
- **AC-5**: Rank config round-trips save/load
- **AC-6**: Scout advances 10u ahead
- **AC-7**: Combat with isolated scout → scout at scout position, rest at formation
- **Edge cases**: solo party computes no offsets; free-form editor absent at VS

---

## Test Evidence

**Story Type**: Logic
**Required evidence**:
- Logic: `tests/unit/party/formation_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (roster storage)
- Unlocks: Story 010 (scout advance placement)
