// Unit tests for Story KG-001: Knowledge Domain & Fact Model.
//
// Evidence path: production/epics/npc-knowledge-graph/story-001-knowledge-domain-fact-model.md
// Each test maps to one acceptance criterion (AC-1 .. AC-6); the final test
// verifies the full Fact data model including entity references (ADR-0005
// CR-2/CR-4). The presentation tests pin the FactType constant values and
// the presentation semantics each type indicates to the AI Dialogue Engine.
package knowledge_test

import (
	"reflect"
	"testing"

	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
)

// AC-1: GIVEN a KnowledgeGraph struct, WHEN domains are set to
// ["craft_metallurgy", "trade_commerce"], THEN both domains are stored and
// queryable.
func TestDomains_StoredAndQueryable(t *testing.T) {
	// Arrange
	kg := knowledge.KnowledgeGraph{
		NPCID:   "greta_blacksmith",
		Domains: []string{"craft_metallurgy", "trade_commerce"},
	}

	// Act
	domains := kg.Domains

	// Assert: both domains stored, in order, with nothing extra.
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d: %v", len(domains), domains)
	}
	if domains[0] != "craft_metallurgy" || domains[1] != "trade_commerce" {
		t.Fatalf("stored domains = %v, want [craft_metallurgy trade_commerce]", domains)
	}
}

// AC-2: GIVEN a FACT-type fact, WHEN the AI Dialogue Engine reads it, THEN
// the type indicates confident presentation.
func TestFactTypeFact_IndicatesConfidentPresentation(t *testing.T) {
	// Arrange
	fact := knowledge.Fact{
		FactID:        "greta_ley_metal_001",
		Domain:        "craft_metallurgy",
		Type:          knowledge.FactType_Fact,
		Content:       "This fragment is an alloy I have never seen.",
		TrustRequired: 0,
	}

	// Act
	typ := fact.Type

	// Assert: exact constant value, and the presentation mode it indicates.
	if typ != "FACT" {
		t.Errorf("FactType_Fact = %q, want \"FACT\"", typ)
	}
	if typ != knowledge.FactType_Fact {
		t.Errorf("Fact.Type = %q, want FactType_Fact", typ)
	}
}

// AC-3: GIVEN a RUMOR-type fact, WHEN read, THEN the type indicates caveated
// presentation ("I heard...").
func TestFactTypeRumor_IndicatesCaveatedPresentation(t *testing.T) {
	// Arrange
	fact := knowledge.Fact{
		FactID:  "greta_folk_tale_002",
		Domain:  "town_gossip",
		Type:    knowledge.FactType_Rumor,
		Content: "The old tower is haunted.",
	}

	// Act
	typ := fact.Type

	// Assert: exact constant value, and the presentation mode it indicates.
	if typ != "RUMOR" {
		t.Errorf("FactType_Rumor = %q, want \"RUMOR\"", typ)
	}
	if typ != knowledge.FactType_Rumor {
		t.Errorf("Fact.Type = %q, want FactType_Rumor", typ)
	}
}

// AC-4: GIVEN an OPINION-type fact, WHEN read, THEN the type indicates
// subjective presentation.
func TestFactTypeOpinion_IndicatesSubjectivePresentation(t *testing.T) {
	// Arrange
	fact := knowledge.Fact{
		FactID:  "greta_opinion_003",
		Domain:  "trade_commerce",
		Type:    knowledge.FactType_Opinion,
		Content: "The guild masters are too greedy for their own good.",
	}

	// Act
	typ := fact.Type

	// Assert: exact constant value, and the presentation mode it indicates.
	if typ != "OPINION" {
		t.Errorf("FactType_Opinion = %q, want \"OPINION\"", typ)
	}
	if typ != knowledge.FactType_Opinion {
		t.Errorf("Fact.Type = %q, want FactType_Opinion", typ)
	}
}

// AC-5: GIVEN a SECRET-type fact, WHEN read, THEN the type indicates
// reluctant, trust-gated presentation.
func TestFactTypeSecret_IndicatesTrustGatedPresentation(t *testing.T) {
	// Arrange
	fact := knowledge.Fact{
		FactID:        "greta_secret_004",
		Domain:        "old_empire_lore",
		Type:          knowledge.FactType_Secret,
		Content:       "The Ley metal is mined from a vein beneath the temple.",
		TrustRequired: 2,
	}

	// Act
	typ := fact.Type

	// Assert: exact constant value, and the presentation mode it indicates.
	if typ != "SECRET" {
		t.Errorf("FactType_Secret = %q, want \"SECRET\"", typ)
	}
	if typ != knowledge.FactType_Secret {
		t.Errorf("Fact.Type = %q, want FactType_Secret", typ)
	}
}

// AC-6: GIVEN a fact with trust_required=2, WHEN queried at trust level 1,
// THEN the fact is excluded.
func TestTrustGating_ExcludesFactAboveTrustLevel(t *testing.T) {
	// Arrange
	fact := knowledge.Fact{
		FactID:        "greta_secret_004",
		Type:          knowledge.FactType_Secret,
		TrustRequired: 2,
	}

	// Act
	gatedAtOne := knowledge.IsGated(fact, 1)
	gatedAtZero := knowledge.IsGated(fact, 0)

	// Assert: excluded at trust levels below 2.
	if !gatedAtOne {
		t.Errorf("IsGated(trust_required=2, trust_level=1) = false, want true (fact excluded)")
	}
	if !gatedAtZero {
		t.Errorf("IsGated(trust_required=2, trust_level=0) = false, want true (fact excluded)")
	}

	// Boundary: exactly at the required level the fact is NOT gated.
	if knowledge.IsGated(fact, 2) {
		t.Errorf("IsGated(trust_required=2, trust_level=2) = true, want false (fact accessible)")
	}
	if knowledge.IsGated(fact, 3) {
		t.Errorf("IsGated(trust_required=2, trust_level=3) = true, want false (fact accessible)")
	}
}

// Full data model check (ADR-0005 CR-2/CR-4): a Fact stores every ADR field,
// and References use the entity framework's GameEntityID type so facts can
// cross-reference game entities.
func TestFact_StoresAllADRAFields(t *testing.T) {
	// Arrange
	fact := knowledge.Fact{
		FactID:        "greta_ley_metal_001",
		Domain:        "craft_metallurgy",
		Type:          knowledge.FactType_Fact,
		Content:       "This fragment is a pale silver alloy, warm to the touch.",
		TrustRequired: 0,
		References: []entity.GameEntityID{
			entity.GameEntityID(7),  // aldric
			entity.GameEntityID(12), // old ruins
		},
		DiscoveryTriggers: []string{"player_has_item:ley_fragment"},
		Contradicts:       []string{"orin_ley_metal_003"},
		LeadsTo: []knowledge.Lead{
			{EntityID: entity.GameEntityID(9), TopicHint: "Ask the hermit about Ley-touched metal"},
		},
	}

	// Assert every field round-trips.
	if fact.FactID != "greta_ley_metal_001" {
		t.Errorf("FactID = %q", fact.FactID)
	}
	if fact.Domain != "craft_metallurgy" {
		t.Errorf("Domain = %q", fact.Domain)
	}
	if fact.Type != knowledge.FactType_Fact {
		t.Errorf("Type = %q", fact.Type)
	}
	if fact.Content == "" {
		t.Error("Content is empty")
	}
	if fact.TrustRequired != 0 {
		t.Errorf("TrustRequired = %d, want 0", fact.TrustRequired)
	}
	wantRefs := []entity.GameEntityID{entity.GameEntityID(7), entity.GameEntityID(12)}
	if !reflect.DeepEqual(fact.References, wantRefs) {
		t.Errorf("References = %v, want %v", fact.References, wantRefs)
	}
	wantTriggers := []string{"player_has_item:ley_fragment"}
	if !reflect.DeepEqual(fact.DiscoveryTriggers, wantTriggers) {
		t.Errorf("DiscoveryTriggers = %v, want %v", fact.DiscoveryTriggers, wantTriggers)
	}
	wantContradicts := []string{"orin_ley_metal_003"}
	if !reflect.DeepEqual(fact.Contradicts, wantContradicts) {
		t.Errorf("Contradicts = %v, want %v", fact.Contradicts, wantContradicts)
	}
	if len(fact.LeadsTo) != 1 || fact.LeadsTo[0].EntityID != entity.GameEntityID(9) ||
		fact.LeadsTo[0].TopicHint != "Ask the hermit about Ley-touched metal" {
		t.Errorf("LeadsTo = %+v, want single lead to entity 9 with topic hint", fact.LeadsTo)
	}
}
