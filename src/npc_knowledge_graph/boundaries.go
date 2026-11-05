// Knowledge boundaries for the NPC Knowledge Graph (Story KG-004, TR-kg-005,
// TR-kg-006, ADR-0005 CR-5): the explicit "knows_not" domain and entity
// exclusions per NPC. The AI Dialogue Engine builds "You do NOT know about:
// X" prompt instructions from these boundaries, so out-of-bound knowledge
// never reaches the LLM prompt (ADR-0005 CR-5, story AC-3).
package knowledge

import (
	"strconv"

	"marrow-of-empire/entity"
)

// GetBoundaries returns the NPC's explicit knowledge exclusions (Story KG-004,
// TR-kg-005, ADR-0005 CR-5): the domains the NPC does not know (KnowsNot) and
// the entities the NPC does not know (KnowsNotEntities), in authoring order.
// The AI Dialogue Engine feeds these into prompt construction as "You do NOT
// know about: X" instructions.
//
// The returned slices are copies — mutating them does not mutate the graph.
// Returns two nil slices when g is nil or the graph declares no exclusions.
func (g *KnowledgeGraph) GetBoundaries() (domains, entities []string) {
	if g == nil {
		return nil, nil
	}
	domains = append([]string(nil), g.KnowsNot...)
	entities = append([]string(nil), g.KnowsNotEntities...)
	return domains, entities
}

// IsWithinBoundaries reports whether (domain, entityID) falls inside the NPC's
// knowledge boundaries (Story KG-004, TR-kg-006, ADR-0005 CR-5) — that is,
// whether the NPC is permitted to know and speak about that domain and that
// entity. It returns false when domain appears in KnowsNot (domain exclusion)
// or entityID appears in KnowsNotEntities (entity exclusion); otherwise true.
// Matching is exact on both: domain is compared as authored (e.g.
// "old_empire_lore"), and entityID is compared as its decimal string —
// knows_not_entities entries follow the numeric GameEntityID authoring
// convention from parser.go (symbolic names are illustration only).
//
// The AI Dialogue Engine consults this before placing an entity or domain in a
// prompt: an out-of-bound pair becomes the prompt instruction "You do NOT know
// about: X", so a boundary prevents the LLM from referencing excluded entities
// (story AC-3). A nil graph declares no boundaries and is always within.
func (g *KnowledgeGraph) IsWithinBoundaries(domain string, entityID entity.GameEntityID) bool {
	if g == nil {
		return true
	}
	for _, d := range g.KnowsNot {
		if d == domain {
			return false
		}
	}
	id := strconv.FormatUint(uint64(entityID), 10)
	for _, e := range g.KnowsNotEntities {
		if e == id {
			return false
		}
	}
	return true
}
