# ADR-0005: NPC Knowledge Graph Data Model

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Core / Scripting |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | NPC Knowledge Graph GDD, AI Dialogue Engine GDD, Paper Prototype Report |
| **Post-Cutoff APIs Used** | None — pure data model |
| **Verification Required** | YAML parsing in Go, Knowledge component storage in Entity Framework |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — Knowledge component on NPC entities) |
| **Enables** | ADR-0008 (LLM Inference — knowledge graph feeds prompt construction) |
| **Blocks** | AI Dialogue Engine, Question System — cannot function without NPC knowledge data |
| **Ordering Note** | Must be Accepted before AI Dialogue Engine implementation |

## Context

### Problem Statement
Every NPC in The Marrow of Empire has a knowledge graph — domains of expertise, specific facts they know, trust-gated secrets, and explicit knowledge boundaries. This data must be authored (as YAML `.kg` files), loaded at runtime, stored on NPC entities, and queried by the AI Dialogue Engine for prompt construction. How should the knowledge graph be structured and queried?

### Constraints
- Must scale to 100+ NPCs across 5-7 regions
- Facts must be authored as natural language (feeds LLM prompts directly)
- Trust gating: facts tagged with trust level 0-3, queried by current trust
- Domain matching: player question → NPC domain → relevant facts
- Authoring format must be human-writable (designers, not programmers)
- Paper prototype validated the 5-NPC fractured knowledge design

### Requirements
- Knowledge domains: named categories (craft_metallurgy, town_gossip, old_empire_lore, etc.)
- Fact types: FACT, RUMOR, OPINION, SECRET
- Entity references: facts reference other entities by GameEntityID
- Knowledge boundaries: explicit "knows_not" lists per NPC
- Discovery triggers: optional conditions that unlock facts

## Decision

**YAML `.kg` files per NPC, loaded into the Knowledge component at scene start. Facts are stored as struct slices, queried by domain + trust level.** YAML is human-readable and writable — designers can author NPC knowledge without a custom tool. The Knowledge component holds the parsed fact set; the AI Dialogue Engine queries it at runtime.

### Data Model

```go
// KnowledgeGraph is the parsed .kg file, stored in the Knowledge component.
type KnowledgeGraph struct {
    NPCID            string    `yaml:"npc_id"`
    NPCName          string    `yaml:"npc_name"`
    Domains          []string  `yaml:"domains"`
    Personality      string    `yaml:"personality"`
    SpeechStyle      string    `yaml:"speech_style"`
    KnowsNot         []string  `yaml:"knows_not"`
    KnowsNotEntities []string  `yaml:"knows_not_entities"`
    Facts            []Fact    `yaml:"facts"`
}

type FactType string
const (
    FactType_Fact     FactType = "FACT"
    FactType_Rumor    FactType = "RUMOR"
    FactType_Opinion  FactType = "OPINION"
    FactType_Secret   FactType = "SECRET"
)

type Fact struct {
    FactID            string       `yaml:"fact_id"`
    Domain            string       `yaml:"domain"`
    Type              FactType     `yaml:"type"`
    Content           string       `yaml:"content"`
    TrustRequired     int          `yaml:"trust_required"`     // 0-3
    References        []GameEntityID `yaml:"references"`
    DiscoveryTriggers []string     `yaml:"discovery_triggers"` // "player_has_item:X", "player_visited:Y"
    Contradicts       []string     `yaml:"contradicts"`        // fact_ids this contradicts
    LeadsTo           []Lead       `yaml:"leads_to"`
}

type Lead struct {
    EntityID  GameEntityID `yaml:"entity_id"`
    TopicHint string       `yaml:"topic_hint"`
}
```

### Query Interface

```go
// KnowledgeStore queries NPC knowledge for dialogue prompt construction.
type KnowledgeStore interface {
    // QueryFacts returns facts matching the given domain at or below the trust level,
    // filtered by satisfied discovery triggers.
    QueryFacts(npcID GameEntityID, domain string, trustLevel int, satisfiedTriggers []string) []Fact

    // GetDomains returns the NPC's knowledge domains.
    GetDomains(npcID GameEntityID) []string

    // GetPersonality returns the NPC's personality and speech style for prompt construction.
    GetPersonality(npcID GameEntityID) (personality, speechStyle string)

    // GetBoundaries returns what the NPC explicitly does not know.
    GetBoundaries(npcID GameEntityID) (domains, entities []string)
}
```

### Authoring Format (.kg file)

```yaml
npc_id: "greta_blacksmith"
npc_name: "Greta"
domains: ["craft_metallurgy", "trade_commerce"]
personality: "Practical, observant, proud of her craft."
speech_style: "Direct, uses smithing metaphors."
knows_not: ["old_empire_lore", "faction_politics"]
knows_not_entities: []
facts:
  - fact_id: "greta_ley_metal_001"
    domain: "craft_metallurgy"
    type: FACT
    content: "This fragment... it's an alloy I've never seen. Pale silver, warm to the touch."
    trust_required: 0
    references: [aldric_entity_id, old_ruins_entity_id]
    leads_to:
      - entity_id: orin_entity_id
        topic_hint: "Ask the hermit about Ley-touched metal"
```

### File Location

`assets/npc_knowledge/{region_id}/{npc_id}.kg` — loaded at scene start alongside the scene file.

## Alternatives Considered

### Alternative 1: YAML .kg files (CHOSEN)
- **Pros**: Human-readable and writable. No custom tooling required at MVP. Standard Go YAML library. Version-controllable. Designers can edit in any text editor.
- **Cons**: No validation at authoring time — errors caught at load. No GUI for relationship visualization. Scale concern at 100+ NPCs.
- **Rejection Reason**: N/A — chosen as the pragmatic MVP choice. A custom authoring tool is deferred to a future ADR if authoring burden becomes significant.

### Alternative 2: JSON .kg files
- **Pros**: Same benefits as YAML. Faster parsing in Go. No YAML library dependency.
- **Cons**: Less human-readable than YAML. No comments. More verbose (quotes, braces). Harder for non-programmers to author.
- **Rejection Reason**: YAML's readability and comment support outweigh JSON's parsing speed for an authoring format that humans write.

### Alternative 3: Embedded in Go code
- **Description**: Define NPC knowledge as Go struct literals in source files.
- **Pros**: Compile-time validation. No parsing overhead. Type-safe.
- **Cons**: Requires recompilation for content changes. Designers cannot edit. Knowledge content mixed with engine code.
- **Rejection Reason**: Content must be separable from code — designers must be able to write and iterate on NPC knowledge without a Go toolchain.

## Consequences

### Positive
- Human-writable format — designers author NPC knowledge directly
- Natural language facts feed LLM prompts without transformation
- Domain + trust + trigger query model validated by paper prototype
- Entity references enable automatic cross-referencing and thread creation

### Negative
- YAML parsing at load time (~10ms per NPC, 1s for 100 NPCs)
- No authoring-time validation — errors surface at load
- No GUI for visualizing the knowledge web across NPCs
- Fact content duplication risk — the same fact authored in multiple NPCs must be kept consistent manually

### Risks
- **Authoring burden at scale**: 100 NPCs × ~30 facts each = 3000 facts to author. Mitigation: region-by-region development; each region ships complete before the next begins.
- **Content staleness**: Fact references may point to entities that change during development. Mitigation: entity reference validation at load time — warn on unresolved references.
- **LLM prompt overflow**: Too many matching facts may exceed the 2048-token context window. Mitigation: `QueryFacts` returns max 5 facts; prompt constructor prioritizes by domain match quality.

## GDD Requirements Addressed

| GDD System | Requirement | How This ADR Addresses It |
|------------|-------------|--------------------------|
| NPC Knowledge Graph | CR-1: Knowledge Domain — named categories of expertise | `domains[]` field in KnowledgeGraph struct |
| NPC Knowledge Graph | CR-2: Knowledge Fact — fact_id, domain, type, content, trust_required, references, leads_to | Fact struct with all specified fields |
| NPC Knowledge Graph | CR-3: Fact Type Behavior — FACT/RUMOR/OPINION/SECRET | FactType enum with defined presentation rules |
| NPC Knowledge Graph | CR-4: Trust Gating — facts gated by trust level 0-3 | `QueryFacts` filters by `trust_required <= trustLevel` |
| NPC Knowledge Graph | CR-5: Knowledge Boundaries — explicit knows_not lists | `knows_not` (domains) and `knows_not_entities` (entity IDs) |
| NPC Knowledge Graph | CR-6: Authoring Format — .kg files per NPC | YAML format at `assets/npc_knowledge/{region}/{npc_id}.kg` |

## Performance Implications
- **CPU**: YAML parsing: ~10ms per NPC file. Fact query: <0.5ms (filter facts by domain + trust + triggers). Domain matching: <5ms (keyword overlap scoring).
- **Memory**: ~5KB per NPC knowledge graph (30 facts × ~150 bytes each). ~500KB for 100 NPCs.
- **Load Time**: Scene load includes .kg parsing: ~200ms for 20 NPCs in a region.
- **Network**: N/A.

## Migration Plan
First implementation. No existing knowledge data to migrate. Author Thornwell NPCs (MVP region) first. Add subsequent regions' NPCs as content is developed.

## Validation Criteria
- .kg file parses without error → Knowledge component populated with all facts
- QueryFacts at trust 0 returns only trust_required=0 facts
- QueryFacts with unsatisfied discovery trigger excludes that fact
- Entity reference validation: referenced entity ID exists in current region → fact loads normally; missing reference → warning logged, fact still loads
- Round-trip: authored .kg → parsed → serialized via Save/Load → reloaded → identical facts

## Related Decisions
- ADR-0001: Entity Architecture — Knowledge component stored on NPC entities
- ADR-0008: LLM Inference — knowledge graph facts feed prompt construction
- ADR-0010: Thread Investigation — entity references and leads_to create threads
