# Story 003: Trust Gating & Query Interface

> **Epic**: NPC Knowledge Graph
> **Status**: Ready | **Layer**: Core | **Type**: Logic | **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09 | **Depends on**: Story 001

## Context
**GDD**: `design/gdd/npc-knowledge-graph.md` | **Req**: `TR-kg-003` | **ADR**: ADR-0005

## Acceptance Criteria

- [ ] GIVEN an NPC with 10 trust-0 facts and 5 trust-2 facts, WHEN QueryFacts is called with trust_level=0, THEN only the 10 trust-0 facts are returned
- [ ] GIVEN the same NPC, WHEN QueryFacts is called with trust_level=2, THEN all 15 facts are returned
- [ ] GIVEN a fact with discovery_trigger "player_has_item:ley_metal_fragment", WHEN the trigger is not satisfied, THEN the fact is excluded even at sufficient trust level
- [ ] GIVEN QueryFacts with a domain filter, WHEN called with domain="craft_metallurgy", THEN only facts in that domain are returned
- [ ] GIVEN QueryFacts with empty domain, WHEN called, THEN facts from all domains are returned

## Implementation Notes
From ADR-0005: `QueryFacts(npcID, domain, trustLevel, satisfiedTriggers) []Fact`. Filter: `fact.TrustRequired <= trustLevel` AND `all(fact.DiscoveryTriggers are in satisfiedTriggers)` AND `(domain == "" || fact.Domain == domain)`. Max 5 facts returned (prompt budget).
