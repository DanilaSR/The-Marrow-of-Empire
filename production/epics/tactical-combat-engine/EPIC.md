# Epic: Tactical Combat Engine

> **Layer**: Feature | **GDD**: tactical-combat-engine.md | **Status**: Ready | **Stories**: 8 stories
> **Governing ADRs**: ADR-0009 | **Engine Risk**: HIGH

## Overview
Implement turn-based tactical combat: square grid (1 unit = 1 meter), AP system (4 + DEX_mod), initiative (DEX_mod + WIS_mod), d20 attack resolution (accuracy vs. evasion, nat20 crit, nat1 miss), positioning (flanking +2, cover +2/full, elevation +1), death/dying (3-round stabilization), and combat state machine (init → player turns → enemy turns → resolution). Pillar 3 delivery mechanism.

## GDD Requirements
| TR-ID | Requirement | ADR |
|-------|-------------|-----|
| TR-combat-001 | Square grid | ADR-0009 ✅ |
| TR-combat-002 | AP system | ADR-0009 ✅ |
| TR-combat-003 | d20 attack resolution | ADR-0009 ✅ |
| TR-combat-004 | Death/dying | ADR-0009 ✅ |
| TR-combat-005 | Positioning bonuses | ADR-0009 ✅ |
| TR-combat-006 | Combat state machine & initiative | ADR-0009 ✅ |
| TR-combat-007 | Combat resolution & system integration | ADR-0009, ADR-0007, ADR-0011 ✅ |
| TR-combat-008 | Enemy stat blocks & scaling | ADR-0009 ✅ |
| TR-combat-009 | Encounter budgets & validation | ADR-0009 ✅ |

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Combat State Machine & Initiative | Integration | Ready | ADR-0009 |
| 002 | Grid Movement & AP Economy | Logic | Ready | ADR-0009, ADR-0006 |
| 003 | Attack Resolution (d20 System) | Logic | Ready | ADR-0009, ADR-0006 |
| 004 | Positioning & Cover System | Logic | Ready | ADR-0009 |
| 005 | Death, Dying & Stabilization | Logic | Ready | ADR-0009 |
| 006 | Combat Resolution & System Integration | Integration | Ready | ADR-0009, ADR-0007, ADR-0011 |
| 007 | Enemy Stat Blocks & Scaling | Logic | Ready | ADR-0009 |
| 008 | Encounter Budgets & Validation | Logic | Ready | ADR-0009 |

## Next Step
Run `/story-readiness production/epics/tactical-combat-engine/story-001-combat-state-machine-initiative.md`
