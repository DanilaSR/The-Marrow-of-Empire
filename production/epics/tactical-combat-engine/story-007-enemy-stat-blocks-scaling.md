# Story 007: Enemy Stat Blocks & Scaling

> **Epic**: Tactical Combat Engine
> **Status**: Ready
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09

## Context

**GDD**: `design/gdd/tactical-combat-engine.md`
**Requirement**: `TR-combat-008`

**ADR Governing Implementation**: ADR-0009: Combat Engine
**ADR Decision Summary**: Enemy stats in YAML, scaled by player level, with behavior tags.

**Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN enemy YAML, WHEN parsed, THEN 6 attributes, derived stats, and AP populated
- [ ] GIVEN player level 5, enemy base 3, WHEN scaled, THEN +1 to all attributes
- [ ] GIVEN enemy type "beast", WHEN loaded, THEN BEAST behavior tag set
- [ ] GIVEN enemy with claws (base 3, melee), WHEN attack calculated, THEN uses weapon stats
- [ ] GIVEN player level 12, enemy base 3, WHEN scaled, THEN capped at +4

## Implementation Notes

From ADR-0009: YAML at `assets/combat/enemies/{id}.yaml`. Format: attributes, base_level, faction, behavior_tags, weapon, abilities, xp_value. Scale: `min((player_level - base_level) / 2, 4)`. Each point = +1 all attributes. Tags: AGGRESSIVE, DEFENSIVE, SUPPORT, COWARD.

## Out of Scope

- AI decision-making
- Encounter budgets (Story 008)
- Boss mechanics
- Loot tables

## QA Test Cases

- **AC-1**: Valid YAML → all attributes + derived stats populated.
- **AC-2**: Player 5, base 3 → +1 all attributes.
- **AC-3**: Beast type → AGGRESSIVE tag.
- **AC-4**: Claws weapon → attack uses weapon values.
- **AC-5**: Player 12, base 3 → capped +4.

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/combat/enemy_stats_test.go`
**Status**: [ ] Not yet created

## Dependencies

- Depends on: Story 001, STAT-001
- Unlocks: Story 008 (Encounter Budgets)
