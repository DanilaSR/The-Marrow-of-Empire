# Story 001: Topic Matcher & Domain Scoring

> **Epic**: Question System
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (4–6 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 3/3 passing
**Deviations**: ADVISORY — AC-1 entity-bonus reading ("mine" not in GDD keyword table, uses "old mine" entity detection per CR-2 step 1). ADVISORY — DomainScore.FromContext field added for CR-3 pronoun resolution.
**Test Evidence**: Logic — `tests/unit/question_system/topic_matcher_test.go` (21 tests, all PASS, p95=157µs)
**Code Review**: Complete — APPROVED

## Context

**GDD**: `design/gdd/question-system.md`
**Requirements**:
- `TR-q-001` — Free-text input with topic matching: keyword overlap + entity reference detection → domain relevance scoring
- `TR-q-002` — Domain keyword scoring: each domain has keyword set, question scored against all domains, threshold 0.3

**ADR Governing Implementation**: ADR-0005: NPC Knowledge Graph Data Model
**ADR Decision Summary**: NPC knowledge is authored in YAML .kg files with typed facts (FACT/RUMOR/OPINION/SECRET), domain categories, trust gating, and entity references by GameEntityID. The Question System queries the Knowledge Graph's domain and fact indices — it does not own knowledge data.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Pure Go string processing and scoring — no Kaiju APIs required. Entity registry lookup uses the game-layer entity registry (ADR-0001) for name detection in question text.

**Control Manifest Rules (Feature Layer)**:
- Required: Event-driven architecture — question matched events emitted via EventBus (ADR-0011)
- Guardrail: Topic matching <5ms (GDD performance AC)

---

## Acceptance Criteria

*From GDD `design/gdd/question-system.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN an NPC with domains `["craft_metallurgy", "trade_commerce"]`, WHEN the player types "What do you know about the ore from the old mine?", THEN the topic matcher scores `craft_metallurgy` above threshold based on keyword overlap ("ore", "mine") and routes matching facts to the AI Dialogue Engine.

- [ ] **AC-2**: GIVEN the same NPC, WHEN the player types "What happened to the old empire?", THEN no domain scores above threshold (NPC doesn't have `old_empire_lore`) and the system routes with fallback — all facts at current trust level with a deflection instruction.

- [ ] **Performance**: GIVEN the topic matcher with a full domain keyword set (9 domains × ~10 keywords each) and a 500-character question, WHEN run under the Go benchmark harness for 1000 iterations, THEN p95 matching time ≤ 5ms.

---

## Implementation Notes

*Derived from ADR-0005 and GDD CR-2 (Topic Matching), CR-3 (Question Context):*

### Topic Matcher is a Pure Function

```go
// MatchResult describes which domains matched and at what relevance.
type MatchResult struct {
    MatchedDomains []DomainScore   // domains above threshold, ranked
    EntityRefs     []GameEntityID  // entities detected in question text
    IsFallback     bool            // true when no domain scored above threshold
}

type DomainScore struct {
    Domain    string
    Score     float64
    Keywords  []string  // which keywords matched
}

// MatchTopics scores the player's question against an NPC's domain keyword sets.
// Pure function — no I/O. Entity registry passed as a look-up table.
func MatchTopics(question string, npcDomains []Domain, entityIndex EntityNameIndex) MatchResult
```

### Topic Matching Algorithm (CR-2)

1. **Entity reference detection**: Scan the question for entity names (NPCs, locations, items, factions) using the entity registry's name index. Case-insensitive, whole-word matching. Multi-word names use n-gram scanning. Detected entities → `EntityRefs` in the result, and add 1.0 bonus to relevance scores.

2. **Domain keyword matching**: For each of the NPC's domains, count keyword matches in the question (case-insensitive, word-boundary). Score: `score = (keyword_matches / total_keywords_in_domain) + (entity_reference_bonus × 0.5)`. Entity reference bonus: 1.0 if any entity name detected.

3. **Threshold**: Domains with score ≥ 0.3 are included in the result, ranked by score descending. If no domain scores above threshold → `IsFallback = true`, all domains included at low relevance.

4. **Conversation history context** (CR-3): If previous questions exist, unresolved references ("it", "that", "they") are resolved against the prior topic context. The last 3 questions (any NPC) are provided to the matcher.

### Domain Keyword Sets

Default keyword sets (authored in config, data-driven per coding standards):

| Domain | Keywords |
|--------|----------|
| `craft_metallurgy` | ore, metal, forge, smith, steel, iron, alloy, weapon, armor, blade, bronze, copper, silver |
| `herbalism_medicine` | herb, plant, poison, remedy, heal, wound, sickness, poultice, root, fungus |
| `town_gossip` | rumor, heard, people, town, happen, strange, recently, anyone, visitor, new |
| `military_security` | guard, patrol, attack, defend, danger, threat, safe, wall, gate, weapon, soldier |
| `old_empire_lore` | empire, ruin, ancient, old, Ley, imperial, history, past, fallen, technology, machine |
| `wilderness_survival` | forest, river, path, animal, beast, track, weather, hunt, forage, camp |
| `trade_commerce` | buy, sell, price, gold, trade, merchant, goods, supply, rare, valuable |
| `faction_politics` | faction, power, leader, alliance, enemy, rival, politics, control, influence |
| `personal_history` | you, your, past, before, born, family, childhood, name, why, how long |

Keywords per domain and the threshold are configurable (data-driven). The default keyword sets are from the GDD; custom domain keywords are authored per NPC in their .kg file.

### Fallback Behaviour
When `IsFallback = true`: all facts at current trust level are routed to the AI Dialogue Engine, with a system note: "The player asked about [question]. This is outside your main expertise. Deflect naturally and suggest who might know from your referral roster."

### Performance Budget
- Entity name scanning: hash-set lookup, <1ms for ~200-name registry
- Keyword matching: O(domains × keywords × question_words), <5ms for typical input
- Total: <5ms p95

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Question input UI state machine, topic suggestion generation, suggestion chip interaction, empty input validation
- AI Dialogue Engine: The topic matcher produces `MatchResult` + routes facts — it does NOT construct prompts or call the LLM
- Dialogue UI: Rendering the text input field, suggestion chips, or shake animation (Presentation layer epic)

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1**: Domain keyword scoring above threshold → matched facts routed
- **AC-2**: No domain above threshold → fallback with deflection instruction
- **Performance**: p95 matching ≤5ms under benchmark

---

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/question_system/topic_matcher_test.go` — must exist and pass

**Status**: [x] Created — `tests/unit/question_system/topic_matcher_test.go` (21 tests, all PASS, p95=157µs)

---

## Dependencies

- Depends on: NPC Knowledge Graph epic (KG-001 domain/fact model + KG-005 component integration; KG-002/003/004 not required for topic matcher implementation)
- Unlocks: Story 002 (Question Input & State Machine needs the topic matcher to route questions)
