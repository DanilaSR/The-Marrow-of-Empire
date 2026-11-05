# Story 005: Knowledge Component Integration

> **Epic**: NPC Knowledge Graph
> **Status**: Complete | **Last Updated**: 2026-08-11
> **Manifest Version**: 2026-08-09 | **Depends on**: Story 001-004

## Context
**GDD**: `design/gdd/npc-knowledge-graph.md` | **Req**: All TR-kg-* | **ADR**: ADR-0005

## Acceptance Criteria

- [ ] GIVEN a scene load with NPCs, WHEN .kg files are parsed, THEN each NPC's Knowledge component is populated with their KnowledgeGraph
- [ ] GIVEN a Save operation, WHEN the Knowledge component is serialized, THEN all facts, domains, and boundaries are preserved
- [ ] GIVEN a Load operation, WHEN the Knowledge component is deserialized, THEN facts, domains, and boundaries match the saved state (note: .kg files are re-parsed on load; runtime modifications to knowledge are lost unless separately persisted)
- [ ] GIVEN the AI Dialogue Engine, WHEN it calls QueryFacts for an NPC, THEN the KnowledgeStore interface returns correctly filtered facts

## Implementation Notes
From ADR-0005: Knowledge component stores the parsed KnowledgeGraph. On scene load: parse .kg files → populate Knowledge components. Save/Load: KnowledgeGraph serialized via pod. Note: .kg files are the source of truth; runtime fact modifications (e.g., NPC learning new facts) require Persistent World State and are out of scope for MVP.
