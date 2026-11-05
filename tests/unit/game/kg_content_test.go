// Content validation for INT-2: the three borderland .kg files must parse
// and their fact references must resolve to the slice's entity ID set.
// Guards against authoring regressions (e.g. trigger words drifting into
// the numeric references list).
//
// Evidence path: prototypes/marrow-of-empire-vertical-slice (integration story INT-2)
package game_test

import (
	"testing"

	knowledge "marrow-of-empire/npc_knowledge_graph"
)

const kgDir = "../../../assets/npc_knowledge/borderland/"

var kgFiles = []string{"blacksmith.kg", "miner.kg", "scholar.kg"}

// knownEntityIDs are the slice's hand-assigned GameEntityIDs (game/talk.go).
var knownEntityIDs = map[uint64]bool{
	1001: true, // blacksmith
	1002: true, // miner
	1003: true, // scholar
	2001: true, // old mine
	2002: true, // inscription
}

func TestKGContent_ParsesAndResolvesReferences(t *testing.T) {
	for _, file := range kgFiles {
		graph, err := knowledge.ParseKGFile(kgDir + file)
		if err != nil {
			t.Fatalf("%s: parse failed: %v", file, err)
		}
		if graph.NPCName == "" || len(graph.Domains) == 0 || len(graph.Facts) == 0 {
			t.Fatalf("%s: graph incomplete: %+v", file, graph)
		}
		for _, f := range graph.Facts {
			for _, ref := range f.References {
				if !knownEntityIDs[uint64(ref)] {
					t.Fatalf("%s: fact %q references unknown entity %d", file, f.FactID, ref)
				}
			}
			for _, lead := range f.LeadsTo {
				if !knownEntityIDs[uint64(lead.EntityID)] {
					t.Fatalf("%s: fact %q leads to unknown entity %d", file, f.FactID, lead.EntityID)
				}
			}
		}
	}
}

func TestKGContent_ContradictionPairExists(t *testing.T) {
	// The slice's authored contradiction: the scholar's formula contradicts
	// the blacksmith's guild-oath opinion — the INT-4 revelation trigger.
	found := false
	for _, file := range kgFiles {
		graph, err := knowledge.ParseKGFile(kgDir + file)
		if err != nil {
			t.Fatalf("%s: parse failed: %v", file, err)
		}
		for _, f := range graph.Facts {
			if len(f.Contradicts) > 0 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no contradiction authored — the investigation chain's reveal has no trigger")
	}
}
