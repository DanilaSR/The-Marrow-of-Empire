# Epic: Character Stats & Progression

> **Layer**: Core
> **GDD**: design/gdd/character-stats-progression.md
> **Architecture Module**: Character Stats & Progression
> **Status**: Ready
> **Stories**: 5 stories

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Attribute Model & Modifiers | Logic | Ready | ADR-0006 |
| 002 | Derived Stats | Logic | Ready | ADR-0006 |
| 003 | Character Creation | Logic | Ready | ADR-0006 |
| 004 | XP & Level Progression | Logic | Ready | ADR-0006 |
| 005 | Stats Component Integration | Integration | Ready | ADR-0006 |

## Overview

Implement the six-attribute CRPG stat system (STR/DEX/CON/INT/WIS/CHA, 3-18),
modifier calculation (`floor((A-10)/2)`), derived stats (Health, AP, Initiative,
Evasion, Persuasion, Insight, Comprehension), point-buy character creation
(30 pts, max 16 start), and XP/level progression (1-20, linear thresholds).
No class system — attributes ARE the class. Stats component stored via Entity
Framework; read by Combat, Dialogue, and Investigation systems.

## Governing ADRs

| ADR | Decision Summary | Engine Risk |
|-----|-----------------|-------------|
| ADR-0006: Character Stats | Six attributes, modifier formula, derived stats, point-buy, XP progression | HIGH |

## GDD Requirements

| TR-ID | Requirement | ADR Coverage |
|-------|-------------|--------------|
| TR-stats-001 | Six attributes STR/DEX/CON/INT/WIS/CHA, 3-18 | ADR-0006 ✅ |
| TR-stats-002 | Modifier formula floor((A-10)/2) | ADR-0006 ✅ |
| TR-stats-003 | Derived stats (HP, AP, Initiative, Evasion) | ADR-0006 ✅ |
| TR-stats-004 | Point-buy creation (30 pts, max 16) | ADR-0006 ✅ |
| TR-stats-005 | XP/level progression (1-20) | ADR-0006 ✅ |

## Definition of Done

- All stories implemented, reviewed, and closed
- Six attributes display correctly with modifiers
- Derived stats recalculate on attribute change or level-up
- Character creation allocates 30 points correctly
- XP awards from threads and combat result in correct level progression

## Next Step

Run `/create-stories character-stats-progression`
