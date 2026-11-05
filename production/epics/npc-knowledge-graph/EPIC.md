# Epic: NPC Knowledge Graph

> **Layer**: Core
> **GDD**: design/gdd/npc-knowledge-graph.md
> **Architecture Module**: NPC Knowledge Graph
> **Status**: Ready
> **Stories**: 5 stories

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Knowledge Domain & Fact Model | Logic | Ready | ADR-0005 |
| 002 | YAML .kg File Parser | Logic | Ready | ADR-0005 |
| 003 | Trust Gating & Query Interface | Logic | Ready | ADR-0005 |
| 004 | Entity References & Boundaries | Logic | Ready | ADR-0005 |
| 005 | Knowledge Component Integration | Integration | Ready | ADR-0005 |

## Overview

Implement the NPC knowledge data model: YAML `.kg` file format per NPC, knowledge
domains, fact types (FACT/RUMOR/OPINION/SECRET), trust gating (0-3), entity
references via GameEntityID, knowledge boundaries (knows_not), and the query
interface for the AI Dialogue Engine. Designers author NPC knowledge as YAML;
the system parses it at scene load and serves it to the AI Dialogue Engine at
runtime based on the player's current trust level.

## Governing ADRs

| ADR | Decision Summary | Engine Risk |
|-----|-----------------|-------------|
| ADR-0005: NPC Knowledge Graph | YAML .kg files, fact types, trust gating, domain matching | HIGH |

## GDD Requirements

| TR-ID | Requirement | ADR Coverage |
|-------|-------------|--------------|
| TR-kg-001 | Knowledge domains per NPC | ADR-0005 ✅ |
| TR-kg-002 | Fact types (FACT/RUMOR/OPINION/SECRET) | ADR-0005 ✅ |
| TR-kg-003 | Trust gating (0-3 levels) | ADR-0005 ✅ |
| TR-kg-004 | YAML .kg authoring format | ADR-0005 ✅ |
| TR-kg-005 | Entity references via GameEntityID | ADR-0005 ✅ |
| TR-kg-006 | Knowledge boundaries (knows_not) | ADR-0005 ✅ |

## Definition of Done

- .kg files parse without error for all MVP NPCs
- QueryFacts returns only facts at or below current trust level
- Domain matching correctly filters facts by player question topic
- Knowledge boundaries prevent LLM from claiming knowledge outside NPC domain
- Entity references link correctly to existing GameEntityIDs

## Next Step

Run `/create-stories npc-knowledge-graph`
