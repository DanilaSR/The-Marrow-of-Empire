# Story 005: Death, Dying & Stabilization

> **Epic**: Tactical Combat Engine
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09

## Context

**GDD**: `design/gdd/tactical-combat-engine.md`
**Requirement**: `TR-combat-004`

**ADR Governing Implementation**: ADR-0009: Combat Engine
**ADR Decision Summary**: 0 HP → Dying (party), 3 rounds to stabilize (2 AP, adjacent). Enemy at 0 HP → dead. Permadeath on timeout.

**Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN party member at 5 HP taking 8 damage, THEN HP=0, status=DYING, death_rounds=3
- [ ] GIVEN dying member at round 2, WHEN adjacent ally spends 2 AP, THEN status=STABLE
- [ ] GIVEN dying member at round 3 with no stabilization, THEN character dies permanently
- [ ] GIVEN enemy at 3 HP taking 5 damage, THEN enemy is DEAD (no dying state)
- [ ] GIVEN dead party member, WHEN combat ends, THEN permadeath applied

## Implementation Notes

From ADR-0009: Dying triggers at 0 HP for party only. 3-round window. Stabilization: 2 AP, adjacent, auto-success. Round 3 expiry → permadeath. Enemy death instant at 0 HP. Dead removed from turn order immediately.

## Out of Scope

- Revive abilities
- Permadeath settings UI
- Death VFX
- Loot drops

## QA Test Cases

- **AC-1**: 5 HP → 8 dmg → HP=0, DYING, rounds=3.
- **AC-2**: Dying round 2 → ally spends 2 AP → STABLE.
- **AC-3**: Dying round 3, no stabilization → DEAD.
- **AC-4**: Enemy 3 HP → 5 dmg → DEAD (no dying).
- **AC-5**: Dead party member, combat ends → permadeath.

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/combat/death_dying_test.go`
**Status**: [ ] Not yet created

## Dependencies

- Depends on: Story 001, Story 003
- Unlocks: Story 006 (Combat Resolution)
