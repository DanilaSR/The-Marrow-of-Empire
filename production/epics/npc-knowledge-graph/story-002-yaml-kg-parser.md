# Story 002: YAML .kg File Parser

> **Epic**: NPC Knowledge Graph
> **Status**: Ready | **Layer**: Core | **Type**: Logic | **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09 | **Depends on**: Story 001

## Context
**GDD**: `design/gdd/npc-knowledge-graph.md` | **Req**: `TR-kg-004` | **ADR**: ADR-0005

## Acceptance Criteria

- [ ] GIVEN a valid .kg YAML file at `assets/npc_knowledge/thornwell/greta_blacksmith.kg`, WHEN parsed, THEN all domains, facts, personality, speech_style, and knows_not are populated
- [ ] GIVEN a .kg file with invalid YAML syntax, WHEN parsed, THEN an error is returned and the NPC loads with zero domains and zero facts (fallback personality)
- [ ] GIVEN a .kg file, WHEN parsed, THEN entity references in facts are validated against existing GameEntityIDs (warning logged for unresolved refs)
- [ ] GIVEN a .kg file with facts referencing discovery triggers, WHEN parsed, THEN triggers are stored but not evaluated at parse time

## Implementation Notes
From ADR-0005: Use Go's `gopkg.in/yaml.v3`. Parse at scene load. File path: `assets/npc_knowledge/{region_id}/{npc_id}.kg`. Invalid YAML → fallback personality ("confused, doesn't know what's happening"), error logged. Entity reference validation: check GameEntityID exists in current region → warn if not found, fact still loads.

## Out of Scope
- Writing .kg files (read-only parser)
- Fact query engine — Story 003
- Trust gating logic — Story 003
- Loading parsed graphs into Entity Framework Knowledge components — Story 005
- Region-level .kg file discovery and batch loading — Story 005

## Dependencies
- **KG-001** (Complete): KnowledgeGraph, Fact, FactType, Lead types defined in `src/npc_knowledge_graph/` — the parser populates these structs
- **ADR-0005** (Accepted): .kg file format specification, fallback personality rules, entity reference validation
- **ADR-0001** (Accepted): GameEntityID type for entity reference validation

## Performance Budget
- .kg YAML parse: target <10ms per NPC file (control manifest Core Layer guardrail)
- Entity reference validation: <1ms per fact (map lookup against active entity IDs)
- Scene load impact: ~200ms for 20 NPCs in a region (ADR-0005 performance budget)
- Memory: ~5KB per parsed KnowledgeGraph (30 facts × ~150 bytes each)

## Test Evidence
- **Unit tests**: `tests/unit/npc_knowledge_graph/kg_parser_test.go`
- Covers all 4 acceptance criteria:
  1. Valid .kg → all fields populated (domains, facts, personality, speech_style, knows_not)
  2. Invalid YAML → error returned + fallback personality
  3. Entity reference validation (warning on unresolved refs, fact still loads)
  4. Discovery triggers stored without evaluation
- **Type**: Logic — automated unit tests required (BLOCKING gate)
- Note: AC-1 references `assets/npc_knowledge/thornwell/greta_blacksmith.kg` — create a minimal test fixture file as part of implementation
