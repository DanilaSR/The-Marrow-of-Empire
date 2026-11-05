// INT-4d: the slice's investigation chain, proven headlessly. Registers the
// three .kg files' facts with the thread engine and replays the player's
// discovery sequence (Vek → Tomas → Mira), asserting that the revelation
// fires — the contradiction between Mira's formula and Vek's oath plus the
// connected facts must reach revelation_ready, exactly as the journal view
// checks it (BuildRevelationPrompt).
//
// Evidence path: prototypes/marrow-of-empire-vertical-slice (integration story INT-4)
package game_test

import (
	"testing"

	knowledge "marrow-of-empire/npc_knowledge_graph"
	threads "marrow-of-empire/thread_investigation"

	"marrow-of-empire/entity"
	"marrow-of-empire/investigation_journal"
)

const (
	entVek   entity.GameEntityID = 1001
	entTomas entity.GameEntityID = 1002
	entMira  entity.GameEntityID = 1003
)

// loadFacts parses all three .kg files and returns their facts.
func loadFacts(t *testing.T) []knowledge.Fact {
	t.Helper()
	var facts []knowledge.Fact
	for _, file := range []string{"blacksmith.kg", "miner.kg", "scholar.kg"} {
		graph, err := knowledge.ParseKGFile(kgDir + file)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		facts = append(facts, graph.Facts...)
	}
	return facts
}

// discover replays the player learning every fact from the NPC in order.
func discover(t *testing.T, e *threads.Engine, facts []knowledge.Fact, npc entity.GameEntityID) {
	t.Helper()
	for _, f := range facts {
		if f.FactID == "vek_ore_gone_wrong" || f.FactID == "vek_miner_gone_quiet" ||
			f.FactID == "vek_forgemaster_oath" {
			if npc == entVek {
				e.OnFactDiscovered(f.FactID, npc)
			}
			continue
		}
		if f.FactID == "tomas_flood_not_rain" || f.FactID == "tomas_slab_hidden" ||
			f.FactID == "tomas_lied_to_smith" {
			if npc == entTomas {
				e.OnFactDiscovered(f.FactID, npc)
			}
			continue
		}
		if npc == entMira {
			e.OnFactDiscovered(f.FactID, npc)
		}
	}
}

func TestRevelationChain_FiresAfterThreeConversations(t *testing.T) {
	e := threads.NewEngine(threads.WithAutoChainMerge(true))
	for _, f := range loadFacts(t) {
		e.RegisterFact(f)
	}

	// 1. Vek: the ore is wrong, the miner went quiet, the guild oath.
	discover(t, e, loadFacts(t), entVek)
	pk := e.Knowledge()
	if len(pk.ActiveThreads) == 0 {
		t.Fatal("Vek's facts must open threads")
	}
	if investigation_journal.BuildRevelationPrompt(&pk, activeChainID(&pk)) != nil {
		t.Fatal("revelation must not fire after one conversation")
	}

	// 2. Tomas: the flood, the hidden slab, the lie to the smith.
	discover(t, e, loadFacts(t), entTomas)

	// 3. Mira: the script's purposes, the smelting formula (contradicts
	// Vek's oath), the stone still warm.
	discover(t, e, loadFacts(t), entMira)

	pk = e.Knowledge()
	chainID := activeChainID(&pk)
	if chainID == "" {
		t.Fatal("no chain formed — auto-chaining must link the facts")
	}
	if !investigation_journal.CheckRevelationReady(&pk, chainID) {
		t.Fatalf("chain %q must be revelation_ready after the three conversations", chainID)
	}
	prompt := investigation_journal.BuildRevelationPrompt(&pk, chainID)
	if prompt == nil {
		t.Fatal("revelation prompt must be non-nil for a ready chain")
	}
	t.Logf("REVELATION: %s (chain %s, confront %d)", prompt.Conclusion, prompt.ChainID, prompt.TargetNPC)
}

func TestRevelationChain_ContradictionDetected(t *testing.T) {
	e := threads.NewEngine(threads.WithAutoChainMerge(true))
	for _, f := range loadFacts(t) {
		e.RegisterFact(f)
	}
	discover(t, e, loadFacts(t), entVek)
	discover(t, e, loadFacts(t), entMira) // Mira's formula contradicts Vek's oath

	pk := e.Knowledge()
	// The contradiction must flag the contradicted facts in the knowledge.
	found := false
	for _, id := range pk.ContradictedFacts {
		if id == "vek_forgemaster_oath" || id == "mira_smelting_formula" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the authored contradiction must flag its facts, got %v", pk.ContradictedFacts)
	}
}

// activeChainID returns the first active chain's ID, or "".
func activeChainID(pk *threads.PlayerKnowledge) string {
	for _, ch := range pk.Chains {
		if ch.Status == "active" || ch.Status == threads.ChainStatusRevelationReady {
			return ch.ID
		}
	}
	return ""
}
