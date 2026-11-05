# Story 009: Combat Mode Translation (TCE Contract)

> **Epic**: Party System
> **Status**: Blocked
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: M (5–7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

> ⛔ **BLOCKED**: `TR-party-003` has no governing ADR — registry GAP: "needs Party ADR or ADR-0009 extension". Run `/architecture-decision` (Party ADR, or extend ADR-0009) before implementation. Once the governing ADR is Accepted, flip this story to Ready.

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-003` — Companion AI behavior modes (Aggressive/Defensive/Passive), set during exploration, persist across save/load; TCE translates mode → target selection + ability-use in AllyTurn

**ADR Governing Implementation**: N/A — GAP (registry: "needs Party ADR or ADR-0009 extension")
**ADR Decision Summary**: The GDD assigns AI decision ownership to the Tactical Combat Engine (ADR-0009), with the Party System providing mode configuration. The governing ADR must formalize this split: Party owns mode storage + the translation *spec*; TCE owns decision execution in `AllyTurn`. ADR-0009's combat-AI budget (10–50ms per decision) constrains the translation.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Lives at the TCE boundary. No new Kaiju APIs; the contract is a Go interface over TCE's AllyTurn hook. Verify the TCE epic's AllyTurn signature and cooldown/ability data structures before implementing.

**Control Manifest Rules (Feature Layer)**:
- Required: All state mutations go through the owning system's interface, with events for notification (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Guardrail: Combat AI decision: 10–50ms (ADR-0009)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-5 (Companion AI Behavior), combat column, scoped to this story:*

- [ ] **AC-1** (AC-PS-08a, passive in combat): GIVEN a companion set to Passive mode, WHEN the party is attacked and no enemy targets the companion for the first 3 rounds, THEN the companion performs no attacks. Support abilities on the lowest-HP party member are permitted.

- [ ] **AC-2** (AC-PS-08b, defensive in combat): GIVEN a companion set to Defensive mode, WHEN the party is attacked but no enemy engages the companion (deals damage or enters melee range), THEN the companion stays within 5 units of the player character and does not initiate attacks.

- [ ] **AC-3** (aggressive targeting): GIVEN a companion in Aggressive mode, THEN each turn the companion targets the nearest enemy by path distance (ties broken by lowest current HP) and re-evaluates the target each turn.

- [ ] **AC-4** (aggressive ability reserve): GIVEN an Aggressive companion with abilities on cooldown, THEN abilities are used unless the ability consumes a limited resource and remaining reserves are below `aggressive_ability_reserve` (25%) — above that threshold abilities are used freely.

- [ ] **AC-5** (defensive targeting): GIVEN a Defensive companion, THEN the companion targets enemies within 5 units of the player character or enemies that dealt damage to any party member in the previous round, and does not pursue enemies retreating beyond 10 units from the player.

- [ ] **AC-6** (passive self-defense): GIVEN a Passive companion directly targeted (enemy declares attack against this companion), THEN the companion may attack that enemy in self-defense; support abilities target the lowest-HP party member within range.

- [ ] **AC-7** (player override): GIVEN any mode, WHEN the player issues an explicit command to a companion during combat, THEN the override persists until the ordered target dies or the player issues a new explicit command to that companion — after which the AI mode resumes as the default.

- [ ] **Edge case — mode cannot change mid-combat**: GIVEN an active encounter, WHEN a mode-change request arrives, THEN it is rejected (modes set only during exploration — Story 008 AC-6).

---

## Implementation Notes

*Derived from GDD CR-5 combat column — to be confirmed against the governing ADR once it exists:*

### Ownership Split (per GDD cross-document contract)

- **Party System owns**: mode storage (Story 008), the read interface TCE consumes, and the translation *specification* below (as data or a shared package the ADR fixes).
- **TCE owns**: decision execution inside `AllyTurn` — target selection, ability-use checks, movement. TCE reads the mode (read-only query, permitted by ADR-0011).
- The contract must be reflected in both GDDs per the GDD's cross-document note — flag any TCE GDD drift during implementation.

### Translation Spec

| Mode | Combat target rule | Ability rule |
|------|--------------------|--------------|
| Aggressive | Nearest enemy by path distance; ties → lowest current HP; re-evaluate each turn | Use on cooldown unless limited-resource ability AND reserves < `aggressive_ability_reserve` (0.25) |
| Defensive | Enemies within 5u of player OR damaged a party member last round; no pursuit beyond 10u of player | Per TCE ability rules |
| Passive | None (self-defense only when directly targeted) | Support abilities only, lowest-HP party member in range |

- Player override: explicit command state per companion; cleared when the ordered target dies or a new explicit command arrives (AC-7).
- Mode reads during combat come from the mode store; a mid-combat change request is rejected at the store (Story 008 owns the setter guard).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 008: Exploration behavior, mode persistence, defaults
- Story 007: Who is in the encounter roster (this story governs how they act)
- Tactical Combat Engine epic: AllyTurn mechanics, cooldowns, ability data

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1**: Passive + untargeted 3 rounds → zero attacks, support allowed on lowest-HP ally
- **AC-2**: Defensive + unengaged → stays within 5u of player, no initiation
- **AC-3**: Aggressive target selection — nearest by path distance, ties → lowest HP, re-evaluated per turn
- **AC-4**: Reserve rule — limited-resource abilities gated below 25%, cooldown abilities used freely
- **AC-5**: Defensive targets — 5u-of-player or damaged-party-last-round; no pursuit past 10u
- **AC-6**: Directly targeted Passive companion fights back; supports lowest-HP in range
- **AC-7**: Override persists until target death or new command, then mode resumes
- **Edge cases**: mid-combat mode set rejected

---

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/party/combat_mode_translation_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 007 (combat roster), Story 008 (mode model — BLOCKED on governing ADR), Tactical Combat Engine epic (complete)
- Unlocks: None
