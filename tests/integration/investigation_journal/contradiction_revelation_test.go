package investigation_journal_test

import (
	"testing"

	"marrow-of-empire/investigation_journal"
	"marrow-of-empire/thread_investigation"
)

// ──────────────────────────────────────────────────────────────────────────────
// AC-3: Contradiction Detection & Pairing
// ──────────────────────────────────────────────────────────────────────────────

func TestFindContradictions_Empty(t *testing.T) {
	pairs := investigation_journal.FindContradictions(nil, "chain_1")
	if pairs != nil {
		t.Errorf("FindContradictions(nil) = %v, want nil", pairs)
	}
}

func TestFindContradictions_NoContradictedFacts(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		DiscoveredFacts: []thread_investigation.DiscoveredFact{
			{FactID: "f1", SourceNPC: 10},
			{FactID: "f2", SourceNPC: 20},
		},
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Facts: []string{"f1", "f2"}},
		},
	}
	pairs := investigation_journal.FindContradictions(pk, "chain_1")
	if len(pairs) != 0 {
		t.Errorf("FindContradictions has %d pairs, want 0", len(pairs))
	}
}

func TestFindContradictions_PairsContradictedFacts(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		DiscoveredFacts: []thread_investigation.DiscoveredFact{
			{FactID: "f1", SourceNPC: 10},
			{FactID: "f2", SourceNPC: 20},
			{FactID: "f3", SourceNPC: 30},
		},
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Facts: []string{"f1", "f2", "f3"}},
		},
		ContradictedFacts: []string{"f1", "f2"},
	}
	pairs := investigation_journal.FindContradictions(pk, "chain_1")
	if len(pairs) != 1 {
		t.Fatalf("FindContradictions has %d pairs, want 1", len(pairs))
	}
	if pairs[0].FactA.FactID != "f1" || pairs[0].FactB.FactID != "f2" {
		t.Errorf("pair = (%s, %s), want (f1, f2)", pairs[0].FactA.FactID, pairs[0].FactB.FactID)
	}
}

func TestFindContradictions_MultiplePairs(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		DiscoveredFacts: []thread_investigation.DiscoveredFact{
			{FactID: "f1", SourceNPC: 10},
			{FactID: "f2", SourceNPC: 20},
			{FactID: "f3", SourceNPC: 30},
		},
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Facts: []string{"f1", "f2", "f3"}},
		},
		ContradictedFacts: []string{"f1", "f2", "f3"},
	}
	pairs := investigation_journal.FindContradictions(pk, "chain_1")
	if len(pairs) != 2 {
		t.Fatalf("FindContradictions has %d pairs, want 2 (f1-f2, f1-f3)", len(pairs))
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AC-2: Revelation Threshold Detection
// ──────────────────────────────────────────────────────────────────────────────

func TestCheckRevelationReady_False(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Status: "active"},
		},
	}
	if investigation_journal.CheckRevelationReady(pk, "chain_1") {
		t.Error("CheckRevelationReady(active) = true, want false")
	}
}

func TestCheckRevelationReady_True(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Status: thread_investigation.ChainStatusRevelationReady},
		},
	}
	if !investigation_journal.CheckRevelationReady(pk, "chain_1") {
		t.Error("CheckRevelationReady(revelation_ready) = false, want true")
	}
}

func TestCheckRevelationReady_NilPK(t *testing.T) {
	if investigation_journal.CheckRevelationReady(nil, "chain_1") {
		t.Error("CheckRevelationReady(nil) = true, want false")
	}
}

func TestBuildRevelationPrompt_NilWhenNotReady(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Status: "active"},
		},
	}
	if prompt := investigation_journal.BuildRevelationPrompt(pk, "chain_1"); prompt != nil {
		t.Error("BuildRevelationPrompt(active) should be nil")
	}
}

func TestBuildRevelationPrompt_ReturnsPromptWhenReady(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Status: thread_investigation.ChainStatusRevelationReady},
		},
		Revelations: []thread_investigation.Revelation{
			{ID: "rev_1", ChainID: "chain_1"},
		},
	}
	prompt := investigation_journal.BuildRevelationPrompt(pk, "chain_1")
	if prompt == nil {
		t.Fatal("BuildRevelationPrompt should return a prompt")
	}
	if prompt.ChainID != "chain_1" {
		t.Errorf("prompt.ChainID = %q, want chain_1", prompt.ChainID)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AC-3: Dismissal Tracking
// ──────────────────────────────────────────────────────────────────────────────

func TestDismissedRevelations_NewIsEmpty(t *testing.T) {
	d := investigation_journal.NewDismissedRevelations()
	if d.IsDismissed("chain_1") {
		t.Error("new DismissedRevelations should not have any dismissed")
	}
}

func TestDismissedRevelations_DismissAndCheck(t *testing.T) {
	d := investigation_journal.NewDismissedRevelations()
	d.Dismiss("chain_1")
	if !d.IsDismissed("chain_1") {
		t.Error("chain_1 should be dismissed after Dismiss()")
	}
	if d.IsDismissed("chain_2") {
		t.Error("chain_2 should not be dismissed")
	}
}

func TestDismissedRevelations_ResetClearsAll(t *testing.T) {
	d := investigation_journal.NewDismissedRevelations()
	d.Dismiss("chain_1")
	d.Dismiss("chain_2")
	d.Reset()
	if d.IsDismissed("chain_1") || d.IsDismissed("chain_2") {
		t.Error("Reset() should clear all dismissals")
	}
}
