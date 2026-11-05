# Epic: Entity Framework

> **Layer**: Foundation
> **GDD**: design/gdd/entity-framework.md
> **Architecture Module**: Entity Framework
> **Status**: Ready
> **Stories**: 6 stories

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Entity Lifecycle | Logic | Ready | ADR-0001, ADR-0011 |
| 002 | Component Model | Logic | Ready | ADR-0001 |
| 003 | Spatial Index | Integration | Ready | ADR-0001 |
| 004 | Write Arbitration | Logic | Ready | ADR-0001 |
| 005 | Serialization | Integration | Ready | ADR-0001, ADR-0002 |
| 006 | Performance Validation | Logic | Ready | ADR-0001 |

## Overview

Implement the game-level entity system as a typed component layer over Kaiju
Engine's native `engine.Entity`. GameEntityID (uint64) → EntityId (string)
mapping, 12+ typed component stores, uniform grid spatial index, lifecycle
event bridge (Kaiju → game events), write arbitration by system priority,
and pod.Register serialization for save/load. This is the root of the
dependency graph — every other system depends on it.

## Governing ADRs

| ADR | Decision Summary | Engine Risk |
|-----|-----------------|-------------|
| ADR-0001: Entity Architecture | Game layer over Kaiju entities, typed component stores, spatial index | HIGH |
| ADR-0011: Event Bus | Synchronous event dispatch with system priority | HIGH |

## GDD Requirements

| TR-ID | Requirement | ADR Coverage |
|-------|-------------|--------------|
| TR-ef-001 | GameEntityID mapping to Kaiju EntityId | ADR-0001 ✅ |
| TR-ef-002 | Typed component stores with Go generics | ADR-0001 ✅ |
| TR-ef-003 | Uniform grid spatial index | ADR-0001 ✅ |
| TR-ef-004 | Lifecycle event bridge | ADR-0001, ADR-0011 ✅ |
| TR-ef-005 | Write arbitration by system priority | ADR-0001 ✅ |
| TR-ef-006 | pod.Register serialization | ADR-0001, ADR-0002 ✅ |

## Definition of Done

- All stories implemented, reviewed, and closed via `/story-done`
- All acceptance criteria from GDD verified
- Entity creation benchmark: <0.5ms
- Spatial query benchmark: <1ms for 500 entities
- Save/load round-trip: identical entity state after reload

## Next Step

Run `/create-stories entity-framework`
