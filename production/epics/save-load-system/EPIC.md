# Epic: Save/Load System

> **Layer**: Foundation
> **GDD**: design/gdd/save-load-system.md
> **Architecture Module**: Save/Load System
> **Status**: Ready
> **Stories**: 3 stories

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Save/Load Orchestration | Integration | Ready | ADR-0002 |
| 002 | Steam Cloud & Platform Paths | Integration | Ready | ADR-0002 |
| 003 | Corruption Recovery | Logic | Ready | ADR-0002 |

## Overview

Implement full-state game persistence via Kaiju `pod` serialization: save file
format (header + metadata + entity table + component stores + world state +
investigation), versioning with migration functions, checksum + backup for
corruption recovery, 20 manual + 1 auto + 1 quicksave slots, Steam Cloud
Auto-Cloud at launch with conflict resolution, and platform-standard save
paths.

## Governing ADRs

| ADR | Decision Summary | Engine Risk |
|-----|-----------------|-------------|
| ADR-0002: Save/Load | Full-state pod, versioning, checksum+backup, Steam Cloud | HIGH |

## GDD Requirements

| TR-ID | Requirement | ADR Coverage |
|-------|-------------|--------------|
| TR-save-001 | Full-state pod serialization | ADR-0002 ✅ |
| TR-save-002 | Save versioning with migrations | ADR-0002 ✅ |
| TR-save-003 | Steam Cloud integration | ADR-0002 ✅ |
| TR-save-004 | Platform-standard save paths | ADR-0002 ✅ |
| TR-save-005 | Checksum + backup recovery | ADR-0002 ✅ |

## Definition of Done

- All stories implemented, reviewed, and closed
- Save completes <2s for 500 entities
- Load completes <3s for 500 entities
- Save/load round-trip: identical entity state
- Corrupted save detected and backup restored
- Steam Cloud syncs on launch and exit
- Game functions without Steam (offline/GOG)

## Next Step

Run `/create-stories save-load-system`
