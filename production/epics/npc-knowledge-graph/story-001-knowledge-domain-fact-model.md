# Story 001: Knowledge Domain & Fact Model

> **Epic**: NPC Knowledge Graph
> **Status**: Complete | **Layer**: Core | **Type**: Logic | **Estimate**: M (4-6 hours) | **Last Updated**: 2026-08-11
> **Manifest Version**: 2026-08-09

## Context
**GDD**: `design/gdd/npc-knowledge-graph.md` | **Req**: `TR-kg-001`, `TR-kg-002` | **ADR**: ADR-0005

## Acceptance Criteria

- [ ] GIVEN a KnowledgeGraph struct, WHEN domains are set to ["craft_metallurgy", "trade_commerce"], THEN both domains are stored and queryable
- [ ] GIVEN a FACT-type fact, WHEN the AI Dialogue Engine reads it, THEN the type indicates confident presentation
- [ ] GIVEN a RUMOR-type fact, WHEN read, THEN the type indicates caveated presentation ("I heard...")
- [ ] GIVEN an OPINION-type fact, WHEN read, THEN the type indicates subjective presentation
- [ ] GIVEN a SECRET-type fact, WHEN read, THEN the type indicates reluctant, trust-gated presentation
- [ ] GIVEN a fact with trust_required=2, WHEN queried at trust level 1, THEN the fact is excluded

## Implementation Notes
From ADR-0005: `KnowledgeGraph` struct with `Domains []string`, `Facts []Fact`. Fact struct with FactID, Domain, Type (FACT/RUMOR/OPINION/SECRET), Content, TrustRequired, References, DiscoveryTriggers, Contradicts, LeadsTo. Fact types determine how the AI Dialogue Engine presents the information.

## Out of Scope
- YAML .kg file parsing — Story 002
- AI Dialogue Engine integration / prompt construction — Story 003+
- NPC entity attachment — requires Entity Framework component registration
- Trust level calculation — belongs to Trust & Relationship system
- Fact query engine (domain + trust + trigger filtering) — Story 003+

## Dependencies
- **Entity Framework** (component store pattern): Knowledge component will be
  stored via `map[GameEntityID]KnowledgeComponent`. This story defines the
  data types; EF component registration must be complete before the Knowledge
  component is attached to NPC entities.
- **ADR-0005** (Accepted): Defines the data model this story implements —
  domains, fact types, trust gating, entity references.
- **ADR-0001** (Accepted): GameEntityID mapping and component store architecture
  that the Knowledge component plugs into.

## Performance Budget
No runtime cost for this story — struct definitions and type declarations only.
Performance budgets from ADR-0005 apply to downstream stories:
- Fact query (domain + trust + trigger filter): <0.5ms — Story 003+
- .kg YAML parse: ~10ms per NPC — Story 002

## Test Evidence
- **Unit tests**: `tests/unit/npc_knowledge_graph/knowledge_graph_test.go`
- Covers all 6 acceptance criteria:
  1. Domain storage and queryability
  2. FACT → confident presentation flag
  3. RUMOR → caveated presentation flag
  4. OPINION → subjective presentation flag
  5. SECRET → trust-gated presentation flag
  6. Trust level filtering (trust_required=2 excluded at trust level 1)
- **Type**: Logic — automated unit tests required (BLOCKING gate)

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 6/6 passing — all auto-verified via `go test ./tests/unit/npc_knowledge_graph/ -v`
**Deviations**: None — implementation matches ADR-0005 field-for-field, manifest versions match (2026-08-09)
**Test Evidence**: `tests/unit/npc_knowledge_graph/knowledge_graph_test.go` — 7 tests, all PASS
**Code Review**: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester, 2026-08-11). Non-blocking suggestions: add zero-value IsGated test, add KnowledgeGraph header round-trip test, clean up tautological assertions in FactType tests.
