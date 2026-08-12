package investigation_journal_test

import (
	"testing"

	"marrow-of-empire/investigation_journal"
	"marrow-of-empire/thread_investigation"
)

// ──────────────────────────────────────────────────────────────────────────────
// AC-2: Fact Log Building & Grouping
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildFactLog_EmptyPlayerKnowledge(t *testing.T) {
	groups := investigation_journal.BuildFactLog(nil, "chain_1")
	if groups != nil {
		t.Errorf("BuildFactLog(nil) = %v, want nil", groups)
	}
}

func TestBuildFactLog_NoMatchingChain(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Facts: []string{"f1"}},
		},
	}
	groups := investigation_journal.BuildFactLog(pk, "chain_2")
	if len(groups) != 0 {
		t.Errorf("BuildFactLog for non-existent chain has %d groups, want 0", len(groups))
	}
}

func TestBuildFactLog_GroupsBySourceNPC(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		DiscoveredFacts: []thread_investigation.DiscoveredFact{
			{FactID: "f1", SourceNPC: 10, DiscoveredAt: 100},
			{FactID: "f2", SourceNPC: 20, DiscoveredAt: 200},
			{FactID: "f3", SourceNPC: 10, DiscoveredAt: 300},
		},
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Facts: []string{"f1", "f2", "f3"}},
		},
	}
	groups := investigation_journal.BuildFactLog(pk, "chain_1")
	if len(groups) != 2 {
		t.Fatalf("BuildFactLog has %d groups, want 2 (NPC 10 and NPC 20)", len(groups))
	}
	// NPC 10 should have 2 facts.
	if len(groups[0].Facts) != 2 {
		t.Errorf("NPC 10 group has %d facts, want 2", len(groups[0].Facts))
	}
}

func TestBuildFactLog_SortsByDiscoveryTime(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		DiscoveredFacts: []thread_investigation.DiscoveredFact{
			{FactID: "f2", SourceNPC: 10, DiscoveredAt: 200},
			{FactID: "f1", SourceNPC: 10, DiscoveredAt: 100},
		},
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Facts: []string{"f1", "f2"}},
		},
	}
	groups := investigation_journal.BuildFactLog(pk, "chain_1")
	if len(groups) != 1 {
		t.Fatalf("BuildFactLog has %d groups, want 1", len(groups))
	}
	if groups[0].Facts[0].FactID != "f1" {
		t.Errorf("first fact = %q, want f1 (oldest first)", groups[0].Facts[0].FactID)
	}
	if groups[0].Facts[1].FactID != "f2" {
		t.Errorf("second fact = %q, want f2", groups[0].Facts[1].FactID)
	}
}

func TestBuildFactLog_ContradictionFlag(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		DiscoveredFacts: []thread_investigation.DiscoveredFact{
			{FactID: "f1", SourceNPC: 10, DiscoveredAt: 100},
			{FactID: "f2", SourceNPC: 20, DiscoveredAt: 200},
		},
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Facts: []string{"f1", "f2"}},
		},
		ContradictedFacts: []string{"f1"},
	}
	groups := investigation_journal.BuildFactLog(pk, "chain_1")
	if len(groups) != 2 {
		t.Fatalf("BuildFactLog has %d groups, want 2", len(groups))
	}
	// Find fact f1.
	var f1 *investigation_journal.FactEntry
	for i := range groups {
		for j := range groups[i].Facts {
			if groups[i].Facts[j].FactID == "f1" {
				f1 = &groups[i].Facts[j]
				break
			}
		}
	}
	if f1 == nil {
		t.Fatal("fact f1 not found")
	}
	if !f1.Contradicts {
		t.Error("f1.Contradicts = false, want true")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Trust Level Labels
// ──────────────────────────────────────────────────────────────────────────────

func TestTrustLevelLabel_AllLevels(t *testing.T) {
	tests := []struct {
		level    int
		expected string
	}{
		{0, "Stranger"},
		{1, "Acquaintance"},
		{2, "Trusted"},
		{3, "Confidant"},
		{99, "Unknown"},
		{-1, "Unknown"},
	}
	for _, tt := range tests {
		got := investigation_journal.TrustLevelLabel(tt.level)
		if got != tt.expected {
			t.Errorf("TrustLevelLabel(%d) = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Entity Reference Set
// ──────────────────────────────────────────────────────────────────────────────

func TestEntityRefSet_NewIsEmpty(t *testing.T) {
	s := investigation_journal.NewEntityRefSet()
	if s.Len() != 0 {
		t.Errorf("new EntityRefSet has %d chips, want 0", s.Len())
	}
}

func TestEntityRefSet_RecordsAndOrders(t *testing.T) {
	s := investigation_journal.NewEntityRefSet()
	s.RecordEntity(1, "Alpha", 100)
	s.RecordEntity(2, "Beta", 200)
	s.RecordEntity(3, "Gamma", 300)

	chips := s.Chips()
	if len(chips) != 3 {
		t.Fatalf("chips = %d, want 3", len(chips))
	}
	// Most recent first.
	if chips[0].Name != "Gamma" {
		t.Errorf("chips[0].Name = %q, want Gamma", chips[0].Name)
	}
	if chips[2].Name != "Alpha" {
		t.Errorf("chips[2].Name = %q, want Alpha", chips[2].Name)
	}
}

func TestEntityRefSet_ReRecordMovesToFront(t *testing.T) {
	s := investigation_journal.NewEntityRefSet()
	s.RecordEntity(1, "Alpha", 100)
	s.RecordEntity(2, "Beta", 200)
	s.RecordEntity(1, "Alpha", 300) // re-record

	chips := s.Chips()
	if len(chips) != 2 {
		t.Fatalf("chips = %d, want 2 (no duplicates)", len(chips))
	}
	if chips[0].Name != "Alpha" {
		t.Errorf("first chip = %q, want Alpha (re-recorded, moved to front)", chips[0].Name)
	}
}

func TestEntityRefSet_MaxEight(t *testing.T) {
	s := investigation_journal.NewEntityRefSet()
	for i := uint64(1); i <= 10; i++ {
		s.RecordEntity(i, "E", float64(i))
	}
	if s.Len() != 8 {
		t.Errorf("len = %d, want 8 (max cap)", s.Len())
	}
}

func TestEntityRefSet_ChipsReturnsCopy(t *testing.T) {
	s := investigation_journal.NewEntityRefSet()
	s.RecordEntity(1, "Alpha", 100)
	chips := s.Chips()
	chips[0].Name = "Modified"
	if s.Chips()[0].Name != "Alpha" {
		t.Error("Chips() does not return a defensive copy")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// State: EntitySelected
// ──────────────────────────────────────────────────────────────────────────────

func TestJournalState_EntitySelectedIsOpen(t *testing.T) {
	if !investigation_journal.StateEntitySelected.IsOpen() {
		t.Error("StateEntitySelected.IsOpen() = false, want true")
	}
}

func TestJournalState_EntitySelectedString(t *testing.T) {
	if got := investigation_journal.StateEntitySelected.String(); got != "EntitySelected" {
		t.Errorf("StateEntitySelected.String() = %q, want 'EntitySelected'", got)
	}
}

func TestJournalState_FourStatesDistinct(t *testing.T) {
	states := []investigation_journal.JournalState{
		investigation_journal.StateClosed,
		investigation_journal.StateOpen,
		investigation_journal.StateChainSelected,
		investigation_journal.StateEntitySelected,
	}
	seen := make(map[investigation_journal.JournalState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate state: %v", s)
		}
		seen[s] = true
	}
	if len(seen) != 4 {
		t.Errorf("expected 4 distinct states, got %d", len(seen))
	}
}
