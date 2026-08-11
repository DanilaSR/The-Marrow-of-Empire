// Unit tests for Story KG-003: Trust Gating & Query Interface (TR-kg-003).
//
// Evidence path: production/epics/npc-knowledge-graph/story-003-trust-gating-queries.md
// Each test maps to one acceptance criterion (AC-1 .. AC-5). AC-1, AC-2, AC-4,
// AC-5 exercise the trust-gated Query/QueryMulti methods on KnowledgeGraph;
// AC-3 (discovery trigger exclusion) is satisfied by QueryFacts, the only
// query that evaluates DiscoveryTriggers (query.go) — Query/QueryMulti have
// no trigger parameter and filter by domain + trust only.
package knowledge_test

import (
	"fmt"
	"reflect"
	"testing"

	"marrow-of-empire/npc_knowledge_graph"
)

// newTestGraph returns the story's canonical fixture: 10 trust-0 facts (5 per
// domain) and 5 trust-2 facts (3 craft, 2 gossip), in authoring order.
// Facts f001-f008 are craft_metallurgy, f009-f015 are town_gossip.
func newTestGraph() *knowledge.KnowledgeGraph {
	kg := &knowledge.KnowledgeGraph{
		NPCID:   "greta_blacksmith",
		Domains: []string{"craft_metallurgy", "town_gossip"},
	}
	id := 0
	add := func(domain string, trust, n int) {
		for i := 0; i < n; i++ {
			id++
			kg.Facts = append(kg.Facts, knowledge.Fact{
				FactID:        fmt.Sprintf("greta_f%03d", id),
				Domain:        domain,
				Type:          knowledge.FactType_Fact,
				Content:       "Fact about " + domain,
				TrustRequired: trust,
			})
		}
	}
	add("craft_metallurgy", 0, 5) // f001-f005
	add("town_gossip", 0, 5)      // f006-f010
	add("craft_metallurgy", 2, 3) // f011-f013
	add("town_gossip", 2, 2)      // f014-f015
	return kg
}

// factIDs extracts the FactID of each fact, in order, for slice comparisons.
func factIDs(facts []knowledge.Fact) []string {
	ids := make([]string, len(facts))
	for i, f := range facts {
		ids[i] = f.FactID
	}
	return ids
}

// AC-1: GIVEN an NPC with 10 trust-0 facts and 5 trust-2 facts, WHEN
// QueryFacts is called with trust_level=0, THEN only the 10 trust-0 facts are
// returned.
func TestQuery_TrustZero_ReturnsOnlyTrustZeroFacts(t *testing.T) {
	// Arrange
	kg := newTestGraph()

	// Act
	got := kg.Query("", 0)

	// Assert: exactly the 10 trust-0 facts, none gated by trust.
	if len(got) != 10 {
		t.Fatalf("Query(trust=0) returned %d facts, want 10: %v", len(got), factIDs(got))
	}
	for _, f := range got {
		if f.TrustRequired != 0 {
			t.Errorf("Query(trust=0) returned trust_required=%d fact %q; want only trust-0 facts", f.TrustRequired, f.FactID)
		}
	}
}

// AC-2: GIVEN the same NPC, WHEN QueryFacts is called with trust_level=2,
// THEN all 15 facts are returned. Trust level 3 likewise returns every fact
// (the task's "trust 3 -> all facts" rule), so the two results must agree.
func TestQuery_TrustTwoAndThree_ReturnAllFacts(t *testing.T) {
	// Arrange
	kg := newTestGraph()

	// Act
	gotTwo := kg.Query("", 2)
	gotThree := kg.Query("", 3)

	// Assert: both trust levels unlock all 15 facts, identically.
	if len(gotTwo) != 15 {
		t.Fatalf("Query(trust=2) returned %d facts, want 15: %v", len(gotTwo), factIDs(gotTwo))
	}
	if len(gotThree) != 15 {
		t.Fatalf("Query(trust=3) returned %d facts, want 15: %v", len(gotThree), factIDs(gotThree))
	}
	if !reflect.DeepEqual(factIDs(gotTwo), factIDs(gotThree)) {
		t.Errorf("Query(trust=2) = %v, Query(trust=3) = %v; want identical", factIDs(gotTwo), factIDs(gotThree))
	}
}

// AC-4: GIVEN QueryFacts with a domain filter, WHEN called with
// domain="craft_metallurgy", THEN only facts in that domain are returned.
func TestQuery_DomainFilter_ReturnsOnlyDomainFacts(t *testing.T) {
	// Arrange
	kg := newTestGraph()

	// Act
	got := kg.Query("craft_metallurgy", 2)

	// Assert: all 8 craft facts (5 trust-0 + 3 trust-2), nothing from gossip.
	if len(got) != 8 {
		t.Fatalf("Query(craft_metallurgy, trust=2) returned %d facts, want 8: %v", len(got), factIDs(got))
	}
	for _, f := range got {
		if f.Domain != "craft_metallurgy" {
			t.Errorf("Query returned fact %q in domain %q; want craft_metallurgy only", f.FactID, f.Domain)
		}
	}
}

// AC-5: GIVEN QueryFacts with an empty domain, WHEN called, THEN facts from
// all domains are returned.
func TestQuery_EmptyDomain_ReturnsFactsFromAllDomains(t *testing.T) {
	// Arrange
	kg := newTestGraph()

	// Act
	got := kg.Query("", 2)

	// Assert: all 15 facts, spanning both domains.
	if len(got) != 15 {
		t.Fatalf("Query(empty domain, trust=2) returned %d facts, want 15: %v", len(got), factIDs(got))
	}
	seen := map[string]bool{}
	for _, f := range got {
		seen[f.Domain] = true
	}
	if !seen["craft_metallurgy"] || !seen["town_gossip"] {
		t.Errorf("Query(empty domain) covered domains %v; want both craft_metallurgy and town_gossip", seen)
	}
}

// QueryMulti: GIVEN a union of domains, WHEN queried at trust level 2, THEN
// all facts from each domain are returned in authoring order, with no
// duplicates (each fact belongs to exactly one domain).
func TestQueryMulti_UnionAcrossDomains_ReturnsCombinedFacts(t *testing.T) {
	// Arrange
	kg := newTestGraph()

	// Act
	got := kg.QueryMulti([]string{"craft_metallurgy", "town_gossip"}, 2)

	// Assert: all 15 facts in authoring order — the union must equal the
	// graph's Facts exactly, with no duplicates and no reordering.
	if len(got) != len(kg.Facts) {
		t.Fatalf("QueryMulti(union, trust=2) returned %d facts, want %d: %v", len(got), len(kg.Facts), factIDs(got))
	}
	if !reflect.DeepEqual(factIDs(got), factIDs(kg.Facts)) {
		t.Errorf("QueryMulti returned %v, want authoring order %v", factIDs(got), factIDs(kg.Facts))
	}
}

// QueryMulti edge case: an empty or nil domain list matches nothing, so the
// result is nil regardless of trust level.
func TestQueryMulti_EmptyDomainList_ReturnsNil(t *testing.T) {
	// Arrange
	kg := newTestGraph()

	// Act
	gotNil := kg.QueryMulti(nil, 3)
	gotEmpty := kg.QueryMulti([]string{}, 3)

	// Assert: no facts from an empty union.
	if len(gotNil) != 0 {
		t.Errorf("QueryMulti(nil, trust=3) returned %d facts, want 0: %v", len(gotNil), factIDs(gotNil))
	}
	if len(gotEmpty) != 0 {
		t.Errorf("QueryMulti([], trust=3) returned %d facts, want 0: %v", len(gotEmpty), factIDs(gotEmpty))
	}
}

// AC-3: GIVEN a fact with discovery_trigger "player_has_item:ley_metal_fragment",
// WHEN the trigger is not satisfied, THEN the fact is excluded even at
// sufficient trust level. Trigger evaluation is QueryFacts' responsibility
// (query.go) — Query/QueryMulti have no trigger parameter.
func TestQueryFacts_UnsatisfiedTrigger_ExcludesFact(t *testing.T) {
	// Arrange
	kg := &knowledge.KnowledgeGraph{
		NPCID: "greta_blacksmith",
		Facts: []knowledge.Fact{{
			FactID:            "greta_ley_secret_001",
			Domain:            "craft_metallurgy",
			Type:              knowledge.FactType_Secret,
			Content:           "The Ley metal is mined from a vein beneath the temple.",
			TrustRequired:     2,
			DiscoveryTriggers: []string{"player_has_item:ley_metal_fragment"},
		}},
	}
	satisfied := knowledge.QueryConfig{
		TrustLevel:        3,
		SatisfiedTriggers: []string{"player_has_item:ley_metal_fragment"},
	}

	// Act
	gotWithout := knowledge.QueryFacts(kg, knowledge.QueryConfig{TrustLevel: 3})
	gotWith := knowledge.QueryFacts(kg, satisfied)

	// Assert: excluded without the trigger even at trust 3; included with it.
	if len(gotWithout) != 0 {
		t.Errorf("QueryFacts returned %d facts with unsatisfied trigger, want 0: %v", len(gotWithout), factIDs(gotWithout))
	}
	if len(gotWith) != 1 || gotWith[0].FactID != "greta_ley_secret_001" {
		t.Errorf("QueryFacts with satisfied trigger = %v, want [greta_ley_secret_001]", factIDs(gotWith))
	}
}
