# Story 008: Encounter Budgets & Validation

> **Epic**: Tactical Combat Engine
> **Status**: Ready
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09

## Context

**GDD**: `design/gdd/tactical-combat-engine.md`
**Requirement**: `TR-combat-009`

**ADR Governing Implementation**: ADR-0009: Combat Engine
**ADR Decision Summary**: Difficulty by enemy XP vs party XP ratio. No filler fights.

**Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN party XP 3600, enemy budget 3000, THEN difficulty = HARD (83%)
- [ ] GIVEN party XP 8000, enemy budget 12000, THEN DEADLY warning (>150%)
- [ ] GIVEN encounter at 40% party power, THEN EASY, passes validation
- [ ] GIVEN 0 enemies, THEN error "encounter has no enemies"
- [ ] GIVEN party XP 2500, enemy budget 5000, THEN UNWINNABLE (>200%)

## Implementation Notes

From ADR-0009: Difficulty: EASY <50%, NORMAL 50-80%, HARD 80-150%, DEADLY 150-200%, UNWINNABLE >200%. Party XP = sum TotalXPForLevel. Enemy XP = sum xp_value. DEADLY → warning. UNWINNABLE → error (blocks ship). Encounter YAML at `assets/combat/encounters/{region}/{id}.yaml`.

## Out of Scope

- Dynamic difficulty
- Playtesting framework
- Designer tooling
- Reinforcement logic

## QA Test Cases

- **AC-1**: Party 3600, enemy 3000 → HARD (83%).
- **AC-2**: Party 8000, enemy 12000 → DEADLY warning.
- **AC-3**: 40% ratio → EASY pass.
- **AC-4**: 0 enemies → error.
- **AC-5**: Party 2500, enemy 5000 → UNWINNABLE.

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/combat/encounter_budget_test.go`
**Status**: [ ] Not yet created

## Dependencies

- Depends on: Story 007, STAT-003 (XP)
- Unlocks: Combat UI epic
