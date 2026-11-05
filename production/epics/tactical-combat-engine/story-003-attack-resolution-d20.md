# Story 003: Attack Resolution (d20 System)

> **Epic**: Tactical Combat Engine
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09

## Context

**GDD**: `design/gdd/tactical-combat-engine.md`
**Requirement**: `TR-combat-003`

**ADR Governing Implementation**: ADR-0009: Combat Engine, ADR-0006: Character Stats
**ADR Decision Summary**: d20 + accuracy >= evasion → hit; nat20 = crit (2x damage); nat1 = auto-miss. Damage = weapon base + STR_mod (melee) or DEX_mod (ranged) - armor.

**Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN acc=12 vs eva=14, roll=3, THEN 12+3=15 ≥ 14 → HIT
- [ ] GIVEN acc=10 vs eva=20, roll=8, THEN 10+8=18 < 20 → MISS
- [ ] GIVEN roll=20, THEN critical hit, 2× base damage regardless of evasion
- [ ] GIVEN roll=1, THEN auto-miss regardless of accuracy
- [ ] GIVEN melee with STR 16 (+3) and weapon 4, THEN base damage = 7
- [ ] GIVEN ranged with DEX 14 (+2), weapon 5, armor 2, THEN final = 5

## Implementation Notes

From ADR-0009: `ResolveAttack(attacker, defender, weapon) AttackResult`. Deterministic seed for tests. Hit: d20 + accuracy >= evasion. Damage: weapon_base + STR_mod (melee) or DEX_mod (ranged) - armor (min 1). Crit: nat20 doubles weapon_base. Auto-miss: nat1. AttackResult: hit, damage, crit flag, roll value.

## Out of Scope

- Positioning bonuses (Story 004)
- Ability formulas
- Damage types
- Combat UI display

## QA Test Cases

- **AC-1**: acc=12, eva=14, roll=3 → HIT (15 ≥ 14).
- **AC-2**: acc=10, eva=20, roll=8 → MISS (18 < 20).
- **AC-3**: roll=20 → crit, damage = weapon_base × 2 + mod.
- **AC-4**: roll=1 → auto-miss.
- **AC-5**: STR 16, weapon 4 → base = 7.
- **AC-6**: DEX 14, weapon 5, armor 2 → final = 5.

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/combat/attack_resolution_test.go`
**Status**: [ ] Not yet created

## Dependencies

- Depends on: Story 001, Story 002, STAT-002
- Unlocks: Story 004, Story 005
