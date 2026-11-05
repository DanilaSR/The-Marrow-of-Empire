// Unit tests for Story KG-004: Entity References & Knowledge Boundaries
// (TR-kg-005, TR-kg-006, ADR-0005 CR-5).
//
// Evidence path: production/epics/npc-knowledge-graph/story-004-entity-references-boundaries.md
// Each test maps to one acceptance criterion (AC-1 .. AC-3). knows_not_entities
// entries are authored as decimal GameEntityID strings, matching the numeric
// reference convention in parser.go (ADR-0005: symbolic names in the authoring
// example are illustration only).
package knowledge_test

import (
	"reflect"
	"testing"

	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
)

// AC-1: GIVEN a fact referencing aldric_entity_id and old_ruins_entity_id,
// WHEN the fact is stored, THEN both entity references are queryable.
func TestReferences_StoredAndQueryable(t *testing.T) {
	// Arrange
	graph := knowledge.KnowledgeGraph{
		NPCID:   "greta_blacksmith",
		Domains: []string{"craft_metallurgy"},
		Facts: []knowledge.Fact{
			{
				FactID:        "greta_ley_metal_001",
				Domain:        "craft_metallurgy",
				Type:          knowledge.FactType_Fact,
				Content:       "This fragment is a pale silver alloy, warm to the touch.",
				TrustRequired: 0,
				References: []entity.GameEntityID{
					entity.GameEntityID(7),  // aldric
					entity.GameEntityID(12), // old ruins
				},
			},
		},
	}

	// Act
	facts := graph.Query("craft_metallurgy", 0)

	// Assert: the stored fact is queryable and carries both entity references.
	if len(facts) != 1 {
		t.Fatalf("Query returned %d facts, want 1", len(facts))
	}
	want := []entity.GameEntityID{entity.GameEntityID(7), entity.GameEntityID(12)}
	if !reflect.DeepEqual(facts[0].References, want) {
		t.Errorf("fact.References = %v, want %v (both entity references queryable)", facts[0].References, want)
	}
}

// AC-1 (validation, ADR-0005 validation criteria): GIVEN facts referencing
// both existing and missing entities, WHEN ValidateReferences is called with
// the region's entity existence check, THEN only the unresolved references
// are returned — each once, in first-occurrence order; resolved references
// are omitted. A nil graph validates clean.
func TestValidateReferences_ReturnsOnlyUnresolvedIDs(t *testing.T) {
	// Arrange
	graph := knowledge.KnowledgeGraph{
		Facts: []knowledge.Fact{
			{
				FactID: "greta_ley_metal_001",
				References: []entity.GameEntityID{
					entity.GameEntityID(7),  // resolved
					entity.GameEntityID(30), // veren — unresolved
				},
			},
			{
				FactID: "greta_ruins_002",
				References: []entity.GameEntityID{
					entity.GameEntityID(30), // same unresolved entity again
					entity.GameEntityID(12), // resolved
				},
			},
		},
	}
	exists := func(id entity.GameEntityID) bool {
		return id == entity.GameEntityID(7) || id == entity.GameEntityID(12)
	}

	// Act
	unresolved := graph.ValidateReferences(exists)

	// Assert: only veren, deduplicated across facts.
	want := []entity.GameEntityID{entity.GameEntityID(30)}
	if !reflect.DeepEqual(unresolved, want) {
		t.Errorf("ValidateReferences = %v, want %v (each unresolved ID once)", unresolved, want)
	}

	// Nil receiver is safe and validates clean.
	var nilGraph *knowledge.KnowledgeGraph
	if got := nilGraph.ValidateReferences(exists); got != nil {
		t.Errorf("ValidateReferences on nil graph = %v, want nil", got)
	}
}

// AC-2: GIVEN an NPC with knows_not=["old_empire_lore", "faction_politics"]
// and knows_not_entities=["30"], WHEN GetBoundaries is called, THEN both
// excluded domains and the excluded entity are returned.
func TestGetBoundaries_ReturnsExcludedDomainsAndEntities(t *testing.T) {
	// Arrange
	graph := knowledge.KnowledgeGraph{
		NPCID:            "greta_blacksmith",
		KnowsNot:         []string{"old_empire_lore", "faction_politics"},
		KnowsNotEntities: []string{"30"}, // veren
	}

	// Act
	domains, entities := graph.GetBoundaries()

	// Assert: both domains and the entity returned, in authoring order.
	wantDomains := []string{"old_empire_lore", "faction_politics"}
	if !reflect.DeepEqual(domains, wantDomains) {
		t.Errorf("GetBoundaries domains = %v, want %v", domains, wantDomains)
	}
	wantEntities := []string{"30"}
	if !reflect.DeepEqual(entities, wantEntities) {
		t.Errorf("GetBoundaries entities = %v, want %v", entities, wantEntities)
	}

	// Boundaries are copies: mutating the returned slices must not leak into
	// the graph.
	domains[0] = "mutated"
	if graph.KnowsNot[0] != "old_empire_lore" {
		t.Error("GetBoundaries returned the graph's own slice (mutation leaked)")
	}
}

// AC-3: GIVEN an NPC with knows_not=["old_empire_lore", "faction_politics"]
// and knows_not_entities=["30" (veren)], WHEN queried about an excluded domain
// or entity, THEN the boundary check rejects it — preventing the LLM from
// referencing it (ADR-0005 CR-5 prompt instruction). Known domains and
// entities, and graphs with no boundaries, remain within boundaries.
func TestIsWithinBoundaries_EnforcesDomainAndEntityExclusions(t *testing.T) {
	// Arrange
	graph := knowledge.KnowledgeGraph{
		NPCID:            "greta_blacksmith",
		KnowsNot:         []string{"old_empire_lore", "faction_politics"},
		KnowsNotEntities: []string{"30"}, // veren (numeric ID convention)
	}

	// Act & Assert
	// Excluded domain: the NPC may not answer old_empire_lore questions.
	if graph.IsWithinBoundaries("old_empire_lore", entity.GameEntityID(7)) {
		t.Error("IsWithinBoundaries(old_empire_lore, aldric) = true, want false (domain excluded)")
	}
	// Excluded entity: the NPC may not reference veren, even in an allowed domain.
	if graph.IsWithinBoundaries("craft_metallurgy", entity.GameEntityID(30)) {
		t.Error("IsWithinBoundaries(craft_metallurgy, veren) = true, want false (entity excluded)")
	}
	// Known domain and entity: within boundaries.
	if !graph.IsWithinBoundaries("craft_metallurgy", entity.GameEntityID(7)) {
		t.Error("IsWithinBoundaries(craft_metallurgy, aldric) = false, want true")
	}
	// A graph with no boundaries admits everything; a nil graph has no boundaries.
	empty := knowledge.KnowledgeGraph{}
	if !empty.IsWithinBoundaries("any_domain", entity.GameEntityID(99)) {
		t.Error("IsWithinBoundaries on boundary-less graph = false, want true")
	}
	var nilGraph *knowledge.KnowledgeGraph
	if !nilGraph.IsWithinBoundaries("any_domain", entity.GameEntityID(99)) {
		t.Error("IsWithinBoundaries on nil graph = false, want true")
	}
}
