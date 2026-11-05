# Story 004: Entity References & Knowledge Boundaries

> **Epic**: NPC Knowledge Graph
> **Status**: Ready | **Layer**: Core | **Type**: Logic | **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09 | **Depends on**: Story 001

## Context
**GDD**: `design/gdd/npc-knowledge-graph.md` | **Req**: `TR-kg-005`, `TR-kg-006` | **ADR**: ADR-0005

## Acceptance Criteria

- [ ] GIVEN a fact referencing aldric_entity_id and old_ruins_entity_id, WHEN the fact is stored, THEN both entity references are queryable
- [ ] GIVEN an NPC with knows_not=["old_empire_lore", "faction_politics"], WHEN GetBoundaries is called, THEN both excluded domains are returned
- [ ] GIVEN an NPC with knows_not_entities=[veren_secret_identity], WHEN queried about that entity, THEN the boundary prevents the LLM from referencing it
- [ ] GIVEN two NPCs with facts referencing the same entity, WHEN both facts are discovered, THEN Thread Investigation detects the entity overlap

## Implementation Notes
From ADR-0005: `References []GameEntityID` on each Fact. `KnowsNot []string` and `KnowsNotEntities []string` on KnowledgeGraph. Boundaries are passed to the AI Dialogue Engine for prompt construction — they become "You do NOT know about: X" instructions.
